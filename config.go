package main

import (
	"errors"

	"github.com/ermos/backuprunner"
	"github.com/ermos/dotenv"
)

type Config struct {
	// PostgreSQL configuration
	PGHost      string `env:"PGHOST" default:"localhost"`
	PGPort      string `env:"PGPORT" default:"5432"`
	PGUser      string `env:"PGUSER" default:"postgres"`
	PGPassword  string `env:"PGPASSWORD"`
	PGDatabase  string `env:"PGDATABASE" default:"postgres"`
	Compression bool   `env:"BACKUP_COMPRESSION" default:"true"`

	// pg_dump options
	PGDumpFormat  string `env:"PGDUMP_FORMAT" default:"custom"` // plain, custom, directory, tar
	PGDumpOptions string `env:"PGDUMP_OPTIONS"`                 // Additional pg_dump options

	backuprunner.Config
}

func Load() (*Config, error) {
	// Try to load .env file (optional, won't fail if not found)
	_ = dotenv.Parse(".env")

	var cfg Config
	if err := dotenv.LoadStruct(&cfg); err != nil {
		return nil, err
	}

	if err := backuprunner.ConfigValidate(&cfg.Config); err != nil {
		return nil, err
	}

	// Validate storage-specific requirements
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	// Validate pg_dump format
	validFormats := map[string]bool{"plain": true, "custom": true, "directory": true, "tar": true}
	if !validFormats[c.PGDumpFormat] {
		return errors.New("PGDUMP_FORMAT must be 'plain', 'custom', 'directory', or 'tar'")
	}

	return nil
}

// GetBackupExtension returns the file extension based on format and compression
func (c *Config) GetBackupExtension() string {
	switch c.PGDumpFormat {
	case "plain":
		if c.Compression {
			return ".sql.gz"
		}
		return ".sql"
	case "custom":
		return ".dump"
	case "tar":
		return ".tar"
	case "directory":
		return "" // directory format doesn't have extension
	default:
		return ".dump"
	}
}
