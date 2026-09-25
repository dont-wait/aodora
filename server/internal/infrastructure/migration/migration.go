package migration

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mongodb"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func Run(databaseURL string, migrations fs.FS, command string) error {
	source, err := iofs.New(migrations, ".")
	if err != nil {
		return fmt.Errorf("open migrations: %w", err)
	}
	runner, err := migrate.NewWithSourceInstance("iofs", source, databaseURL)
	if err != nil {
		_ = source.Close()
		return fmt.Errorf("create migration runner: %w", err)
	}
	defer func() { _, _ = runner.Close() }()

	var migrationErr error
	switch command {
	case "up":
		migrationErr = runner.Up()
	case "up-by-one":
		migrationErr = runner.Steps(1)
	case "down":
		migrationErr = runner.Steps(-1)
	default:
		return fmt.Errorf("unsupported migration command %q", command)
	}
	if errors.Is(migrationErr, migrate.ErrNoChange) {
		return nil
	}
	if migrationErr != nil {
		return fmt.Errorf("run migrations: %w", migrationErr)
	}
	return nil
}
