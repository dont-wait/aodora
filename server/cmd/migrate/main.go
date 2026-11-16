package main

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/your-org/aodora/server/internal/config"
	"github.com/your-org/aodora/server/internal/infrastructure/migration"
	"github.com/your-org/aodora/server/internal/logger"
)

func main() {
	command := flag.String("command", "up", "migration command: create, adopt-legacy, up, up-by-one, or down")
	path := flag.String("path", "migrations", "migration directory")
	name := flag.String("name", "", "name for a new UUID migration")
	fingerprint := flag.String("legacy-fingerprint", "", "reviewed legacy manifest SHA-256 for explicit adoption")
	operationTimeout := flag.Duration("operation-timeout", 5*time.Minute, "timeout per MongoDB operation; 0 disables it")
	flag.Parse()

	log := logger.New("info", "json")
	if *command == "create" {
		id, err := migration.Create(*path, *name)
		if err != nil {
			log.Error().Err(err).Msg("create migration")
			os.Exit(1)
		}
		log.Info().Str("id", id).Str("name", *name).Msg("created UUID migration")
		return
	}
	if err := config.LoadEnv(); err != nil {
		log.Fatal().Err(err).Msg("load environment")
	}
	mongoConfig, err := config.LoadMongoConfig()
	if err != nil {
		log.Error().Err(err).Msg("load mongo configuration")
		os.Exit(1)
	}
	databaseURL, err := mongoDatabaseURL(mongoConfig.URI, mongoConfig.DB)
	if err != nil {
		log.Error().Err(err).Msg("build mongo migration URL")
		os.Exit(1)
	}
	migrationsPath, err := filepath.Abs(*path)
	if err != nil {
		log.Error().Err(err).Msg("resolve migrations path")
		os.Exit(1)
	}
	if err := migration.Run(databaseURL, os.DirFS(migrationsPath), *command, migration.RunOptions{LegacyFingerprint: *fingerprint, OperationTimeout: *operationTimeout}); err != nil {
		log.Error().Err(err).Str("command", *command).Msg("run migration")
		os.Exit(1)
	}
	log.Info().Str("command", *command).Msg("migration completed")
}

func mongoDatabaseURL(uri, databaseName string) (string, error) {
	if strings.TrimSpace(databaseName) == "" {
		return "", errors.New("MONGO_DB must not be empty")
	}
	parsed, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("parse MONGO_URI: %w", err)
	}
	if parsed.Scheme != "mongodb" && parsed.Scheme != "mongodb+srv" {
		return "", fmt.Errorf("unsupported MongoDB URI scheme %q", parsed.Scheme)
	}
	parsed.Path = "/" + strings.TrimPrefix(databaseName, "/")
	return parsed.String(), nil
}
