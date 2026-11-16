package migration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/fstest"
)

const (
	testUUIDA = "ffffffff-ffff-4fff-8fff-ffffffffffff"
	testUUIDB = "11111111-1111-4111-8111-111111111111"
)

func addTestMigration(files fstest.MapFS, id, name string, deps []string, up, down string) {
	stem := id + "_" + name
	files[stem+".up.json"] = &fstest.MapFile{Data: []byte(up)}
	files[stem+".down.json"] = &fstest.MapFile{Data: []byte(down)}
	if uuidPattern.MatchString(id) {
		data, _ := json.Marshal(metadata{DependsOn: deps})
		files[stem+".meta.json"] = &fstest.MapFile{Data: data}
	}
}

func TestLoadUUIDDependenciesAndHistoricalIDs(t *testing.T) {
	files := fstest.MapFS{}
	addTestMigration(files, "000041", "base", nil, "[]", "[]")
	addTestMigration(files, "000078", "next", nil, "[]", "[]")
	addTestMigration(files, testUUIDA, "first_uuid", []string{legacyID(78)}, "[]", "[]")
	addTestMigration(files, testUUIDB, "second_uuid", []string{testUUIDA}, "[]", "[]")
	all, err := Load(files)
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, m := range all {
		ids = append(ids, m.ID)
		if !uuidPattern.MatchString(m.ID) {
			t.Fatalf("non UUID identity %s", m.ID)
		}
	}
	if !reflect.DeepEqual(ids, []string{legacyID(41), legacyID(78), testUUIDA, testUUIDB}) {
		t.Fatalf("wrong dependency order: %v", ids)
	}
}

func TestLoadRejectsAmbiguousOrBrokenMigrations(t *testing.T) {
	for _, name := range []string{"duplicate numeric", "duplicate UUID", "missing pair", "missing metadata", "missing parent", "cycle", "invalid commands", "invalid ID"} {
		t.Run(name, func(t *testing.T) {
			files := fstest.MapFS{}
			addTestMigration(files, testUUIDA, "a", nil, "[]", "[]")
			switch name {
			case "duplicate numeric":
				addTestMigration(files, "000001", "a", nil, "[]", "[]")
				addTestMigration(files, "1", "b", nil, "[]", "[]")
			case "duplicate UUID":
				addTestMigration(files, testUUIDA, "another", nil, "[]", "[]")
			case "missing pair":
				delete(files, testUUIDA+"_a.down.json")
			case "missing metadata":
				delete(files, testUUIDA+"_a.meta.json")
			case "missing parent":
				addTestMigration(files, testUUIDA, "a", []string{testUUIDB}, "[]", "[]")
			case "cycle":
				addTestMigration(files, testUUIDA, "a", []string{testUUIDB}, "[]", "[]")
				addTestMigration(files, testUUIDB, "b", []string{testUUIDA}, "[]", "[]")
			case "invalid commands":
				addTestMigration(files, testUUIDA, "a", nil, "{}", "[]")
			case "invalid ID":
				addTestMigration(files, "not-a-uuid", "bad", nil, "[]", "[]")
			}
			if _, err := Load(files); err == nil {
				t.Fatal("expected validation failure")
			}
		})
	}
}

func TestCreateOnSeparateBranchesNeverUsesNextInteger(t *testing.T) {
	var ids []string
	for i := 0; i < 2; i++ {
		dir := t.TempDir()
		for _, direction := range []string{"up", "down"} {
			if err := os.WriteFile(filepath.Join(dir, "000078_base."+direction+".json"), []byte("[]"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		id, err := Create(dir, "same_feature")
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
		all, err := Load(os.DirFS(dir))
		if err != nil {
			t.Fatal(err)
		}
		if len(all) != 2 || all[1].ID != id || !reflect.DeepEqual(all[1].DependsOn, []string{legacyID(78)}) {
			t.Fatalf("wrong generated catalog: %+v", all)
		}
	}
	if ids[0] == ids[1] || !uuidPattern.MatchString(ids[0]) || !uuidPattern.MatchString(ids[1]) {
		t.Fatalf("bad UUIDs: %v", ids)
	}
	if _, err := Create(t.TempDir(), "../escape"); err == nil {
		t.Fatal("unsafe name accepted")
	}
}

func TestPrimaryURLsOverrideSecondaryAndPreserveMigrationOptions(t *testing.T) {
	driver, client, name, err := primaryURLs("mongodb://localhost/test?readPreference=secondary&maxStalenessSeconds=120&readPreferenceTags=dc:west&x-migrations-collection=custom")
	if err != nil {
		t.Fatal(err)
	}
	if name != "test" || !strings.Contains(driver, "readPreference=primary") || !strings.Contains(driver, "x-migrations-collection=custom") || strings.Contains(client, "x-") || strings.Contains(client, "readPreferenceTags") || strings.Contains(client, "maxStaleness") {
		t.Fatalf("unsafe URLs driver=%s client=%s", driver, client)
	}
	if _, _, _, err := primaryURLs("mongodb://localhost/test?x-advisory-locking=false"); err == nil {
		t.Fatal("disabled lock accepted")
	}
}
