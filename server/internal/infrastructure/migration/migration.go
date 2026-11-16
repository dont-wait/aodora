package migration

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4/database"
	migratedb "github.com/golang-migrate/migrate/v4/database/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

const historyCollection = "schema_migrations_uuid"

type migrationState struct {
	Head          string `bson:"head"`
	Dirty         bool   `bson:"dirty"`
	Pending       string `bson:"pending"`
	Direction     string `bson:"direction"`
	LegacyVersion int    `bson:"legacy_version"`
}

type appliedMigration struct {
	ID       string `bson:"_id"`
	Previous string `bson:"previous"`
	Checksum string `bson:"checksum"`
}

func Run(databaseURL string, files fs.FS, command string) error {
	if command != "up" && command != "up-by-one" && command != "down" {
		return fmt.Errorf("unsupported migration command %q", command)
	}
	all, err := Load(files)
	if err != nil {
		return fmt.Errorf("load migrations: %w", err)
	}
	driverURL, clientURL, databaseName, err := primaryURLs(databaseURL)
	if err != nil {
		return err
	}
	// Retain the shared advisory lock from go-migrate, but track each UUID
	// instead of handing numeric versions to its source runner.
	driver, err := (&migratedb.Mongo{}).Open(driverURL)
	if err != nil {
		return fmt.Errorf("open migration driver: %w", err)
	}
	defer func() { _ = driver.Close() }()
	if err := driver.Lock(); err != nil {
		return fmt.Errorf("lock migrations: %w", err)
	}
	defer func() { _ = driver.Unlock() }()
	client, err := mongo.Connect(options.Client().ApplyURI(clientURL).SetReadPreference(readpref.Primary()))
	if err != nil {
		return fmt.Errorf("open migration history: %w", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = client.Disconnect(ctx)
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	db := client.Database(databaseName)
	history := db.Collection(historyCollection)
	state, err := loadState(ctx, driver, history, all)
	if err != nil {
		return err
	}
	applied, err := readApplied(ctx, history, all, state.Head)
	if err != nil {
		return err
	}
	if command == "down" {
		if state.Head == "" {
			return nil
		}
		for _, m := range all {
			if m.ID != state.Head {
				continue
			}
			if err := checkDownCompatibility(ctx, db, m.Down); err != nil {
				return err
			}
			if err := setDirty(ctx, history, m.ID, "down"); err != nil {
				return err
			}
			if err := executeCommands(ctx, db, m.Down); err != nil {
				return fmt.Errorf("roll back migration %s (%s): %w; history is dirty", m.ID, m.Name, err)
			}
			if _, err := history.DeleteOne(ctx, bson.M{"_id": m.ID}); err != nil {
				return err
			}
			return finish(ctx, history, applied[m.ID].Previous)
		}
		return fmt.Errorf("applied migration %s is missing from the checkout", state.Head)
	}
	for _, m := range all {
		if _, exists := applied[m.ID]; exists {
			continue
		}
		for _, dep := range m.DependsOn {
			if _, ok := applied[dep]; !ok {
				return fmt.Errorf("migration %s requires unapplied migration %s", m.ID, dep)
			}
		}
		if err := setDirty(ctx, history, m.ID, "up"); err != nil {
			return err
		}
		if err := executeCommands(ctx, db, m.Up); err != nil {
			return fmt.Errorf("apply migration %s (%s): %w; history is dirty", m.ID, m.Name, err)
		}
		record := appliedMigration{ID: m.ID, Previous: state.Head, Checksum: m.checksum()}
		if _, err := history.InsertOne(ctx, record); err != nil {
			return err
		}
		if err := finish(ctx, history, m.ID); err != nil {
			return err
		}
		state.Head, applied[m.ID] = m.ID, record
		if command == "up-by-one" {
			break
		}
	}
	return nil
}

func primaryURLs(raw string) (string, string, string, error) {
	uri, err := url.Parse(raw)
	if err != nil {
		return "", "", "", fmt.Errorf("parse migration URL: %w", err)
	}
	name := strings.TrimPrefix(uri.Path, "/")
	if (uri.Scheme != "mongodb" && uri.Scheme != "mongodb+srv") || name == "" {
		return "", "", "", errors.New("migration URL must be MongoDB with a database name")
	}
	query := uri.Query()
	for key := range query {
		switch strings.ToLower(key) {
		case "readpreference", "readpreferencetags", "maxstalenessseconds":
			query.Del(key)
		case "x-advisory-locking":
			if query.Get(key) != "true" {
				return "", "", "", errors.New("UUID migrations require advisory locking")
			}
		case "x-transaction-mode":
			if query.Get(key) != "false" {
				return "", "", "", errors.New("UUID migrations execute DDL outside transactions; x-transaction-mode must be false or omitted")
			}
		case "w":
			if query.Get(key) == "0" {
				return "", "", "", errors.New("migrations require acknowledged writes")
			}
		}
	}
	query.Set("readPreference", "primary")
	uri.RawQuery = query.Encode()
	driverURL := uri.String()
	for key := range query {
		if strings.HasPrefix(key, "x-") {
			query.Del(key)
		}
	}
	uri.RawQuery = query.Encode()
	return driverURL, uri.String(), name, nil
}

func loadState(ctx context.Context, driver database.Driver, history *mongo.Collection, all []Migration) (migrationState, error) {
	var state migrationState
	legacyVersion, legacyDirty, err := driver.Version()
	if err != nil {
		return state, err
	}
	if legacyDirty {
		return state, fmt.Errorf("legacy migration %d is dirty; repair it before using UUID migrations", legacyVersion)
	}
	err = history.FindOne(ctx, bson.M{"_id": "state"}).Decode(&state)
	if err == nil {
		if state.Dirty {
			return state, fmt.Errorf("migration history is dirty: %s %s; inspect and repair the failed operation before retrying", state.Direction, state.Pending)
		}
		if legacyVersion != state.LegacyVersion {
			return state, errors.New("legacy migration version changed after UUID adoption; do not mix old and new runners")
		}
		return state, nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return state, err
	}
	var legacy []Migration
	for _, m := range all {
		if m.LegacyVersion >= 0 && m.LegacyVersion <= legacyVersion {
			legacy = append(legacy, m)
		}
	}
	if legacyVersion >= 0 && (len(legacy) == 0 || legacy[len(legacy)-1].LegacyVersion != legacyVersion) {
		return state, fmt.Errorf("legacy version %d is missing from migration files; refusing to guess applied history", legacyVersion)
	}
	state = migrationState{LegacyVersion: legacyVersion, Dirty: true, Direction: "import"}
	if _, err := history.InsertOne(ctx, bson.M{"_id": "state", "legacy_version": legacyVersion, "dirty": true, "direction": "import"}); err != nil {
		return state, err
	}
	for _, m := range legacy {
		if _, err := history.InsertOne(ctx, appliedMigration{ID: m.ID, Previous: state.Head, Checksum: m.checksum()}); err != nil {
			return state, err
		}
		state.Head = m.ID
	}
	if err := finish(ctx, history, state.Head); err != nil {
		return state, err
	}
	state.Dirty, state.Direction = false, ""
	return state, nil
}

func readApplied(ctx context.Context, history *mongo.Collection, all []Migration, head string) (map[string]appliedMigration, error) {
	cursor, err := history.Find(ctx, bson.M{"_id": bson.M{"$ne": "state"}})
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()
	var rows []appliedMigration
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	known := map[string]Migration{}
	for _, m := range all {
		known[m.ID] = m
	}
	applied := map[string]appliedMigration{}
	for _, row := range rows {
		m, ok := known[row.ID]
		if !ok || row.Checksum != m.checksum() {
			return nil, fmt.Errorf("applied migration %s is missing or modified; restore its original files", row.ID)
		}
		applied[row.ID] = row
	}
	seen := map[string]bool{}
	for id := head; id != ""; {
		row, ok := applied[id]
		if !ok || seen[id] {
			return nil, fmt.Errorf("migration history chain is broken at %s", id)
		}
		seen[id], id = true, row.Previous
	}
	if len(seen) != len(applied) {
		return nil, errors.New("migration history contains disconnected applied records")
	}
	return applied, nil
}

func setDirty(ctx context.Context, history *mongo.Collection, id, direction string) error {
	_, err := history.UpdateOne(ctx, bson.M{"_id": "state"}, bson.M{"$set": bson.M{"dirty": true, "pending": id, "direction": direction}})
	return err
}

func finish(ctx context.Context, history *mongo.Collection, head string) error {
	_, err := history.UpdateOne(ctx, bson.M{"_id": "state"}, bson.M{"$set": bson.M{"head": head, "dirty": false, "pending": "", "direction": ""}})
	return err
}

func executeCommands(ctx context.Context, db *mongo.Database, data []byte) error {
	var commands []bson.D
	if err := bson.UnmarshalExtJSON(data, true, &commands); err != nil {
		return err
	}
	for _, command := range commands {
		var result struct {
			WriteErrors []struct {
				Code    int    `bson:"code"`
				Message string `bson:"errmsg"`
			} `bson:"writeErrors"`
			WriteConcernError *struct {
				Code    int    `bson:"code"`
				Message string `bson:"errmsg"`
			} `bson:"writeConcernError"`
		}
		if err := db.RunCommand(ctx, command).Decode(&result); err != nil {
			return err
		}
		if len(result.WriteErrors) != 0 {
			return fmt.Errorf("migration command write error %d: %s", result.WriteErrors[0].Code, result.WriteErrors[0].Message)
		}
		if result.WriteConcernError != nil {
			return fmt.Errorf("migration command write concern error %d: %s", result.WriteConcernError.Code, result.WriteConcernError.Message)
		}
	}
	return nil
}
