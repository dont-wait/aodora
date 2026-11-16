package migration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
)

type legacyEntry struct {
	Version  int    `json:"version"`
	ID       string `json:"id"`
	Name     string `json:"name"`
	Checksum string `json:"checksum"`
}

type legacyManifest struct {
	Migrations []legacyEntry `json:"migrations"`
}

// The committed manifest is an immutable snapshot of the historical files,
// not something reconstructed from a possibly incomplete checkout at adoption.
// Old go-migrate stores no checksums: an operator must also reconcile provenance.
func verifyLegacyManifest(files fs.FS, all []Migration, version int) (string, error) {
	data, err := fs.ReadFile(files, "legacy-manifest.json")
	if err != nil {
		return "", fmt.Errorf("read legacy manifest: %w", err)
	}
	var manifest legacyManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return "", fmt.Errorf("decode legacy manifest: %w", err)
	}
	known := map[int]Migration{}
	for _, m := range all {
		if m.LegacyVersion >= 0 && m.LegacyVersion <= version {
			known[m.LegacyVersion] = m
		}
	}
	seen := map[int]bool{}
	for _, entry := range manifest.Migrations {
		if entry.Version > version {
			continue
		}
		m, ok := known[entry.Version]
		if seen[entry.Version] || !ok || m.ID != entry.ID || m.Name != entry.Name || m.checksum() != entry.Checksum {
			return "", fmt.Errorf("legacy migration %d is missing, duplicated or differs from the committed manifest; reconcile historical files before adoption", entry.Version)
		}
		seen[entry.Version] = true
	}
	if len(seen) != len(known) || !seen[version] {
		return "", fmt.Errorf("legacy manifest does not cover the applied history through version %d", version)
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}
