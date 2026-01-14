package main

import (
	"log"

	"github.com/ermos/backuprunner"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting PostgreSQL Backup Service...")

	// Load configuration
	cfg, err := Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	err = backuprunner.Run(NewPGBackup(cfg))
	if err != nil {
		log.Fatalf("Backup runner failed: %v", err)
	}
}
