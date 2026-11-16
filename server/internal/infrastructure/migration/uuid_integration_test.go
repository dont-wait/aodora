//go:build integration

package migration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratedb "github.com/golang-migrate/migrate/v4/database/mongodb"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func uuidTestDatabase(t *testing.T) (*mongo.Database, string) {
	t.Helper()
	raw := os.Getenv("MONGO_TEST_URI")
	if raw == "" {
		t.Fatal("MONGO_TEST_URI is required for integration tests")
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	u.Path = "/aodora_uuid_test_" + bson.NewObjectID().Hex()
	client, err := mongo.Connect(options.Client().ApplyURI(u.String()))
	if err != nil {
		t.Fatal(err)
	}
	db := client.Database(strings.TrimPrefix(u.Path, "/"))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := db.Drop(ctx); err != nil {
			t.Error(err)
		}
		_ = client.Disconnect(ctx)
	})
	return db, u.String()
}

func uuidBaseFiles() fstest.MapFS {
	files := fstest.MapFS{}
	addTestMigration(files, "000041", "base", nil, `[{"create":"events"},{"insert":"events","documents":[{"_id":"legacy"}]}]`, `[{"drop":"events"}]`)
	addTestLegacyManifest(files)
	return files
}

func addTestLegacyManifest(files fstest.MapFS) {
	all, err := Load(files)
	if err != nil {
		panic(err)
	}
	manifest := legacyManifest{}
	for _, m := range all {
		if m.LegacyVersion >= 0 {
			manifest.Migrations = append(manifest.Migrations, legacyEntry{m.LegacyVersion, m.ID, m.Name, m.checksum()})
		}
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		panic(err)
	}
	files["legacy-manifest.json"] = &fstest.MapFile{Data: data}
}

func adoptForTest(t *testing.T, uri string, files fs.FS) {
	t.Helper()
	data, err := fs.ReadFile(files, "legacy-manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	if err := Run(uri, files, "adopt-legacy", RunOptions{LegacyFingerprint: hex.EncodeToString(digest[:])}); err != nil {
		t.Fatal(err)
	}
}

func applyWithLegacyRunner(t *testing.T, uri string, files fs.FS) {
	t.Helper()
	source, err := iofs.New(files, ".")
	if err != nil {
		t.Fatal(err)
	}
	runner, err := migrate.NewWithSourceInstance("iofs", source, uri)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = runner.Close() }()
	if err := runner.Up(); err != nil {
		t.Fatal(err)
	}
}

func TestUUIDAdoptsRepositoryLegacyHistoryWithoutReplaying(t *testing.T) {
	db, uri := uuidTestDatabase(t)
	files := os.DirFS("../../../migrations")
	applyWithLegacyRunner(t, uri, files)
	adoptForTest(t, uri, files)
	ctx := context.Background()
	collections := []string{"products", "orders", "customers", "tailoring_requests"}
	before := map[string]int64{}
	for _, name := range collections {
		n, err := db.Collection(name).CountDocuments(ctx, bson.M{})
		if err != nil {
			t.Fatal(err)
		}
		before[name] = n
	}
	for i := 0; i < 2; i++ {
		if err := Run(uri, files, "up"); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range collections {
		n, err := db.Collection(name).CountDocuments(ctx, bson.M{})
		if err != nil || n != before[name] {
			t.Fatalf("adoption changed %s: count=%d before=%d err=%v", name, n, before[name], err)
		}
	}
	all, err := Load(files)
	if err != nil {
		t.Fatal(err)
	}
	n, err := db.Collection(historyCollection).CountDocuments(ctx, bson.M{"_id": bson.M{"$ne": "state"}})
	if err != nil || n != int64(len(all)) {
		t.Fatalf("incomplete adoption: %d records, %v", n, err)
	}
}

func TestUUIDFreshDatabaseMatchesLegacyRunner(t *testing.T) {
	oldDB, oldURI := uuidTestDatabase(t)
	newDB, newURI := uuidTestDatabase(t)
	repository := os.DirFS("../../../migrations")
	entries, err := fs.ReadDir(repository, ".")
	if err != nil {
		t.Fatal(err)
	}
	files := fstest.MapFS{}
	for _, entry := range entries {
		if _, err := source.Parse(entry.Name()); err != nil {
			continue
		}
		data, err := fs.ReadFile(repository, entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		files[entry.Name()] = &fstest.MapFile{Data: data}
	}
	applyWithLegacyRunner(t, oldURI, files)
	if err := Run(newURI, files, "up"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, name := range []string{"health_checks", "categories", "products", "customers", "orders", "carts", "reviews", "tailoring_requests", "sessions"} {
		read := func(db *mongo.Database) []bson.M {
			t.Helper()
			cursor, err := db.Collection(name).Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = cursor.Close(ctx) }()
			var documents []bson.M
			if err := cursor.All(ctx, &documents); err != nil {
				t.Fatal(err)
			}
			return documents
		}
		if !reflect.DeepEqual(read(oldDB), read(newDB)) {
			t.Fatalf("legacy and UUID runners produced different data in %s", name)
		}
		collectionOptions := func(db *mongo.Database) any {
			t.Helper()
			cursor, err := db.ListCollections(ctx, bson.M{"name": name})
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = cursor.Close(ctx) }()
			var rows []bson.M
			if err := cursor.All(ctx, &rows); err != nil || len(rows) != 1 {
				t.Fatalf("read %s options: rows=%d err=%v", name, len(rows), err)
			}
			return rows[0]["options"]
		}
		if !reflect.DeepEqual(collectionOptions(oldDB), collectionOptions(newDB)) {
			t.Fatalf("legacy and UUID runners produced different validators in %s", name)
		}
	}
}

func TestUUIDAdoptsLegacyAndAppliesLateMergedSibling(t *testing.T) {
	db, uri := uuidTestDatabase(t)
	files := uuidBaseFiles()
	applyWithLegacyRunner(t, uri, files)
	adoptForTest(t, uri, files)
	addTestMigration(files, testUUIDA, "branch_a", []string{legacyID(41)}, `[{"insert":"events","documents":[{"_id":"a"}]}]`, `[{"delete":"events","deletes":[{"q":{"_id":"a"},"limit":1}]}]`)
	if err := Run(uri, files, "up"); err != nil {
		t.Fatal(err)
	}
	// B sorts before A, but was merged after A was already applied.
	addTestMigration(files, testUUIDB, "branch_b", []string{legacyID(41)}, `[{"insert":"events","documents":[{"_id":"b"}]}]`, `[{"delete":"events","deletes":[{"q":{"_id":"b"},"limit":1}]}]`)
	if err := Run(uri, files, "up"); err != nil {
		t.Fatal(err)
	}
	if err := Run(uri, files, "up"); err != nil {
		t.Fatalf("idempotent up: %v", err)
	}
	ctx := context.Background()
	count, err := db.Collection("events").CountDocuments(ctx, bson.M{})
	if err != nil || count != 3 {
		t.Fatalf("late merge lost/repeated migration: count=%d err=%v", count, err)
	}
	if err := Run(uri, files, "down"); err != nil {
		t.Fatal(err)
	}
	if err := db.Collection("events").FindOne(ctx, bson.M{"_id": "b"}).Err(); err != mongo.ErrNoDocuments {
		t.Fatalf("down did not undo last actually applied migration: %v", err)
	}
	if err := db.Collection("events").FindOne(ctx, bson.M{"_id": "a"}).Err(); err != nil {
		t.Fatalf("down undid wrong sibling: %v", err)
	}
	// Checksum protects a migration already recorded in the UUID ledger.
	original := files[testUUIDA+"_branch_a.up.json"]
	files[testUUIDA+"_branch_a.up.json"] = &fstest.MapFile{Data: []byte("[]")}
	if err := Run(uri, files, "up"); err == nil || !strings.Contains(err.Error(), "missing or modified") {
		t.Fatalf("modified applied migration accepted: %v", err)
	}
	files[testUUIDA+"_branch_a.up.json"] = original
	if err := Run(uri, files, "up-by-one"); err != nil {
		t.Fatal(err)
	}
	var state migrationState
	if err := db.Collection(historyCollection).FindOne(ctx, bson.M{"_id": "state"}).Decode(&state); err != nil {
		t.Fatal(err)
	}
	if state.Head != testUUIDB || state.Dirty || state.LegacyVersion != 41 {
		t.Fatalf("wrong state %+v", state)
	}
}

func TestUUIDFailureLeavesDirtyHistoryAndDoesNotRepeatWrites(t *testing.T) {
	db, uri := uuidTestDatabase(t)
	files := uuidBaseFiles()
	addTestMigration(files, testUUIDA, "failure", []string{legacyID(41)}, `[{"insert":"events","documents":[{"_id":"partial"}]},{"insert":"events","documents":[{"_id":"legacy"}]}]`, `[]`)
	if err := Run(uri, files, "up"); err == nil {
		t.Fatal("expected duplicate-key failure")
	}
	if err := Run(uri, files, "up"); err == nil || !strings.Contains(err.Error(), "history is dirty") {
		t.Fatalf("dirty migration retried: %v", err)
	}
	count, err := db.Collection("events").CountDocuments(context.Background(), bson.M{"_id": "partial"})
	if err != nil || count != 1 {
		t.Fatalf("partial write was repeated: count=%d err=%v", count, err)
	}
}

func TestUUIDMigrationReadsHistoryAfterAcquiringSharedLock(t *testing.T) {
	db, uri := uuidTestDatabase(t)
	files := uuidBaseFiles()
	uri += "&x-advisory-lock-timeout-interval=1"
	driver, err := (&migratedb.Mongo{}).Open(uri)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = driver.Close() }()
	if err := driver.Lock(); err != nil {
		t.Fatal(err)
	}
	locked := true
	defer func() {
		if locked {
			_ = driver.Unlock()
		}
	}()
	result := make(chan error, 1)
	go func() { result <- Run(uri, files, "up") }()
	select {
	case err := <-result:
		t.Fatalf("runner bypassed shared migration lock: %v", err)
	case <-time.After(200 * time.Millisecond):
	}
	n, err := db.Collection(historyCollection).CountDocuments(context.Background(), bson.M{})
	if err != nil || n != 0 {
		t.Fatalf("history touched outside migration lock: %d %v", n, err)
	}
	if err := driver.Unlock(); err != nil {
		t.Fatal(err)
	}
	locked = false
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("migration did not resume after lock release")
	}
}

func TestLegacyAdoptionRequiresCompleteFilesAndExplicitFingerprint(t *testing.T) {
	db, uri := uuidTestDatabase(t)
	files := uuidBaseFiles()
	addTestMigration(files, "000078", "next", nil, `[{"insert":"events","documents":[{"_id":"next"}]}]`, `[{"delete":"events","deletes":[{"q":{"_id":"next"},"limit":1}]}]`)
	addTestLegacyManifest(files)
	applyWithLegacyRunner(t, uri, files)
	if err := Run(uri, files, "up"); err == nil || !strings.Contains(err.Error(), "explicit reconciliation") {
		t.Fatalf("legacy history silently adopted: %v", err)
	}
	if err := Run(uri, files, "adopt-legacy", RunOptions{LegacyFingerprint: "wrong"}); err == nil {
		t.Fatal("unreviewed fingerprint accepted")
	}
	up, down := files["000041_base.up.json"], files["000041_base.down.json"]
	delete(files, "000041_base.up.json")
	delete(files, "000041_base.down.json")
	if err := Run(uri, files, "adopt-legacy", RunOptions{LegacyFingerprint: "wrong"}); err == nil || !strings.Contains(err.Error(), "committed manifest") {
		t.Fatalf("missing earlier migration accepted: %v", err)
	}
	files["000041_base.up.json"], files["000041_base.down.json"] = up, down
	files["000041_base.up.json"] = &fstest.MapFile{Data: []byte(`[{"create":"different_schema"}]`)}
	if err := Run(uri, files, "adopt-legacy", RunOptions{LegacyFingerprint: "wrong"}); err == nil || !strings.Contains(err.Error(), "committed manifest") {
		t.Fatalf("conflicting branch history accepted: %v", err)
	}
	files["000041_base.up.json"] = up
	n, err := db.Collection(historyCollection).CountDocuments(context.Background(), bson.M{})
	if err != nil || n != 0 {
		t.Fatalf("failed adoption wrote history: %d %v", n, err)
	}
	adoptForTest(t, uri, files)
	if err := Run(uri, files, "up"); err != nil {
		t.Fatal(err)
	}
	n, err = db.Collection("events").CountDocuments(context.Background(), bson.M{})
	if err != nil || n != 2 {
		t.Fatalf("adoption replayed schema/seed: %d %v", n, err)
	}
}

func TestUUIDEmptyTemplateIsNotRecordedAsApplied(t *testing.T) {
	db, uri := uuidTestDatabase(t)
	dir := t.TempDir()
	id, err := Create(dir, "unfinished")
	if err != nil {
		t.Fatal(err)
	}
	if err := Run(uri, os.DirFS(dir), "up"); err == nil || !strings.Contains(err.Error(), "empty up commands") {
		t.Fatalf("empty template accepted: %v", err)
	}
	n, err := db.Collection(historyCollection).CountDocuments(context.Background(), bson.M{"_id": id})
	if err != nil || n != 0 {
		t.Fatalf("empty template recorded as applied: %d %v", n, err)
	}
}
