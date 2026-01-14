package main

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ermos/backuprunner"
)

type PGBackup struct {
	cfg     *Config
	storage backuprunner.Storage
}

func NewPGBackup(cfg *Config) *PGBackup {
	return &PGBackup{
		cfg: cfg,
	}
}

func (b *PGBackup) Name() string {
	return "PostgreSQL"
}

func (b *PGBackup) Config() (*backuprunner.Config, error) {
	return &b.cfg.Config, nil
}

func (b *PGBackup) ExtraConfigLogInfo() []string {
	return []string{
		fmt.Sprintf("PostgreSQL: %s:%s", b.cfg.PGHost, b.cfg.PGPort),
		fmt.Sprintf("Database: %s", b.cfg.PGDatabase),
		fmt.Sprintf("Compression: %v", b.cfg.Compression),
	}
}

func (b *PGBackup) SetStorage(s backuprunner.Storage) error {
	b.storage = s
	return nil
}

// TestConnection verifies PostgreSQL connectivity using pg_isready
func (b *PGBackup) TestConnection(ctx context.Context) error {
	log.Println("Testing PostgreSQL connection...")

	maxRetries := 10
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		cmd := exec.CommandContext(ctx, "pg_isready",
			"-h", b.cfg.PGHost,
			"-p", b.cfg.PGPort,
			"-U", b.cfg.PGUser,
		)

		err := cmd.Run()
		if err == nil {
			return nil
		}

		lastErr = err

		waitTime := time.Duration(i+1) * 2 * time.Second
		if waitTime > 30*time.Second {
			waitTime = 30 * time.Second
		}
		log.Printf("Failed to connect to PostgreSQL (attempt %d/%d): %v. Retrying in %s...", i+1, maxRetries, lastErr, waitTime)
		time.Sleep(waitTime)
	}

	return fmt.Errorf("failed to connect to PostgreSQL after %d attempts: %w", maxRetries, lastErr)
}

// Run executes a backup operation
func (b *PGBackup) Run(ctx context.Context) error {
	log.Println("Starting backup process...")

	// Step 1: Generate backup filename with timestamp
	backupName := b.generateBackupName()
	tempFile := filepath.Join(os.TempDir(), backupName)

	defer func(name string) {
		errOsRemove := os.Remove(name)
		if errOsRemove != nil {
			log.Printf("Warning: failed to remove temp file %s: %v", name, errOsRemove)
		}
	}(tempFile)

	// Step 2: Run pg_dump
	if err := b.runPgDump(ctx, tempFile); err != nil {
		return fmt.Errorf("failed to run pg_dump: %w", err)
	}

	// Step 3: Upload to storage
	if err := b.storage.Upload(ctx, tempFile, backupName); err != nil {
		return fmt.Errorf("failed to upload backup: %w", err)
	}

	log.Printf("Backup completed successfully: %s (storage: %s)", backupName, b.storage.Type())

	return nil
}

// runPgDump executes pg_dump command
func (b *PGBackup) runPgDump(ctx context.Context, outputPath string) error {
	log.Printf("Running pg_dump for database '%s'...", b.cfg.PGDatabase)

	args := []string{
		"-h", b.cfg.PGHost,
		"-p", b.cfg.PGPort,
		"-U", b.cfg.PGUser,
		"-d", b.cfg.PGDatabase,
	}

	// Add format option
	args = append(args, "-F", string(b.cfg.PGDumpFormat[0]))

	// For plain format without compression, output directly to file
	// For plain format with compression, we'll pipe through gzip
	// For other formats, output to file
	if b.cfg.PGDumpFormat != "plain" {
		args = append(args, "-f", outputPath)
	}

	// Add any additional custom options
	if b.cfg.PGDumpOptions != "" {
		extraArgs := strings.Fields(b.cfg.PGDumpOptions)
		args = append(args, extraArgs...)
	}

	cmd := exec.CommandContext(ctx, "pg_dump", args...)

	// Set PGPASSWORD environment variable
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", b.cfg.PGPassword))

	if b.cfg.PGDumpFormat == "plain" {
		// Handle plain format output
		if b.cfg.Compression {
			// Pipe through gzip
			return b.runPgDumpWithGzip(ctx, cmd, outputPath)
		}
		// Direct file output for plain without compression
		outFile, err := os.Create(outputPath)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer func(outFile *os.File) {
			errOutFileClose := outFile.Close()
			if err != nil {
				log.Printf("Warning: failed to close output file: %v", errOutFileClose)
			}
		}(outFile)

		cmd.Stdout = outFile
		cmd.Stderr = os.Stderr

		if err = cmd.Run(); err != nil {
			return fmt.Errorf("pg_dump failed: %w", err)
		}
	} else {
		// For custom, tar, directory formats
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("pg_dump failed: %w", err)
		}
	}

	log.Println("pg_dump completed successfully")
	return nil
}

// runPgDumpWithGzip pipes pg_dump output through gzip
func (b *PGBackup) runPgDumpWithGzip(ctx context.Context, cmd *exec.Cmd, outputPath string) error {
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer func(outFile *os.File) {
		errOutFileClose := outFile.Close()
		if err != nil {
			log.Printf("Warning: failed to close output file: %v", errOutFileClose)
		}
	}(outFile)

	gzWriter := gzip.NewWriter(outFile)
	defer func(gzWriter *gzip.Writer) {
		errWriterClose := gzWriter.Close()
		if errWriterClose != nil {
			log.Printf("Warning: failed to close gzip writer: %v", errWriterClose)
		}
	}(gzWriter)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	cmd.Stderr = os.Stderr

	if err = cmd.Start(); err != nil {
		return fmt.Errorf("failed to start pg_dump: %w", err)
	}

	if _, err = io.Copy(gzWriter, stdout); err != nil {
		return fmt.Errorf("failed to compress output: %w", err)
	}

	if err = cmd.Wait(); err != nil {
		return fmt.Errorf("pg_dump failed: %w", err)
	}

	return nil
}

// generateBackupName creates a unique backup filename
func (b *PGBackup) generateBackupName() string {
	timestamp := time.Now().UTC().Format("2006-01-02_15-04-05")
	ext := b.cfg.GetBackupExtension()
	return fmt.Sprintf("pg-backup_%s_%s%s", b.cfg.PGDatabase, timestamp, ext)
}
