package migration

import (
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Migration struct {
	ID            string
	Name          string
	LegacyVersion int // -1 for UUID migrations
	DependsOn     []string
	Up, Down      []byte
}

type metadata struct {
	DependsOn []string `json:"depends_on"`
}

var (
	uuidPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-8][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	namePattern = regexp.MustCompile(`^[a-z0-9]+(?:[_-][a-z0-9]+)*$`)
)

func formatUUID(b []byte) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// Keep numeric files immutable and map each historical version to a UUIDv5.
func legacyID(version int) string {
	h := sha1.New()
	_, _ = h.Write(make([]byte, 16)) // nil UUID namespace
	_, _ = fmt.Fprintf(h, "aodora:legacy-migration:%d", version)
	b := h.Sum(nil)[:16]
	b[6], b[8] = (b[6]&0x0f)|0x50, (b[8]&0x3f)|0x80
	return formatUUID(b)
}

func newUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6], b[8] = (b[6]&0x0f)|0x40, (b[8]&0x3f)|0x80
	return formatUUID(b), nil
}

// Load orders UUID files by dependencies, never by a largest numeric version.
func Load(files fs.FS) ([]Migration, error) {
	entries, err := fs.ReadDir(files, ".")
	if err != nil {
		return nil, err
	}
	byID, stems := map[string]*Migration{}, map[string]string{}
	for _, entry := range entries {
		if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".up.json") && !strings.HasSuffix(entry.Name(), ".down.json")) {
			continue
		}
		up, suffix := strings.HasSuffix(entry.Name(), ".up.json"), ".down.json"
		if up {
			suffix = ".up.json"
		}
		stem := strings.TrimSuffix(entry.Name(), suffix)
		key, name, ok := strings.Cut(stem, "_")
		if !ok || name == "" {
			return nil, fmt.Errorf("invalid migration filename %s", entry.Name())
		}
		id, version := key, -1
		if !uuidPattern.MatchString(key) {
			n, err := strconv.ParseUint(key, 10, 63)
			if err != nil || uint64(int(n)) != n {
				return nil, fmt.Errorf("migration %s must use a UUID or historical numeric version", entry.Name())
			}
			version, id = int(n), legacyID(int(n))
		}
		m := byID[id]
		if m == nil {
			m = &Migration{ID: id, Name: name, LegacyVersion: version}
			byID[id], stems[id] = m, stem
		} else if stems[id] != stem {
			return nil, fmt.Errorf("duplicate migration identity %s: %s and %s", id, stems[id], stem)
		}
		data, err := fs.ReadFile(files, entry.Name())
		if err != nil {
			return nil, err
		}
		var commands []bson.D
		if !strings.HasPrefix(strings.TrimSpace(string(data)), "[") {
			return nil, fmt.Errorf("migration %s must contain a command array", entry.Name())
		}
		if err := bson.UnmarshalExtJSON(data, true, &commands); err != nil {
			return nil, fmt.Errorf("decode migration %s: %w", entry.Name(), err)
		}
		if up {
			m.Up = data
		} else {
			m.Down = data
		}
	}
	var legacy []*Migration
	for id, m := range byID {
		if m.Up == nil || m.Down == nil {
			return nil, fmt.Errorf("migration %s requires both up and down files", stems[id])
		}
		if m.LegacyVersion >= 0 {
			legacy = append(legacy, m)
			continue
		}
		data, err := fs.ReadFile(files, stems[id]+".meta.json")
		if err != nil {
			return nil, fmt.Errorf("read dependencies for %s: %w", id, err)
		}
		var meta metadata
		decoder := json.NewDecoder(strings.NewReader(string(data)))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&meta); err != nil {
			return nil, fmt.Errorf("decode dependencies for %s: %w", id, err)
		}
		m.DependsOn = meta.DependsOn
		sort.Strings(m.DependsOn)
	}
	sort.Slice(legacy, func(i, j int) bool { return legacy[i].LegacyVersion < legacy[j].LegacyVersion })
	for i := 1; i < len(legacy); i++ {
		legacy[i].DependsOn = []string{legacy[i-1].ID}
	}
	var ordered []Migration
	visiting, visited := map[string]bool{}, map[string]bool{}
	var visit func(string) error
	visit = func(id string) error {
		if visiting[id] {
			return fmt.Errorf("migration dependency cycle at %s", id)
		}
		if visited[id] {
			return nil
		}
		m, ok := byID[id]
		if !ok {
			return fmt.Errorf("missing migration dependency %s", id)
		}
		visiting[id] = true
		for _, dep := range m.DependsOn {
			if err := visit(dep); err != nil {
				return err
			}
		}
		visiting[id], visited[id] = false, true
		ordered = append(ordered, *m)
		return nil
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if err := visit(id); err != nil {
			return nil, err
		}
	}
	return ordered, nil
}

func (m Migration) checksum() string {
	h := sha256.New()
	for _, data := range []string{m.ID, m.Name, strings.Join(m.DependsOn, ","), string(m.Up), string(m.Down)} {
		_, _ = h.Write([]byte(data))
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// Each migration owns its metadata file, so branches never edit a shared counter.
func Create(directory, name string) (string, error) {
	if !namePattern.MatchString(name) {
		return "", fmt.Errorf("migration name must use lowercase letters, digits, underscores or hyphens")
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", err
	}
	all, err := Load(os.DirFS(directory))
	if err != nil {
		return "", err
	}
	parents := map[string]bool{}
	for _, m := range all {
		for _, dep := range m.DependsOn {
			parents[dep] = true
		}
	}
	meta := metadata{DependsOn: []string{}}
	for _, m := range all {
		if !parents[m.ID] {
			meta.DependsOn = append(meta.DependsOn, m.ID)
		}
	}
	sort.Strings(meta.DependsOn)
	id, err := newUUID()
	if err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return "", err
	}
	stem := filepath.Join(directory, id+"_"+name)
	var created []string
	for _, f := range []struct{ suffix, content string }{{".up.json", "[]\n"}, {".down.json", "[]\n"}, {".meta.json", string(data) + "\n"}} {
		path := stem + f.suffix
		if err := writeExclusive(path, f.content); err != nil {
			for _, p := range created {
				_ = os.Remove(p)
			}
			return "", err
		}
		created = append(created, path)
	}
	return id, nil
}

func writeExclusive(path, content string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	_, writeErr := f.WriteString(content)
	closeErr := f.Close()
	if writeErr != nil {
		_ = os.Remove(path)
		return writeErr
	}
	return closeErr
}
