package main

import (
	"log"

	"github.com/CLAOJ/claoj-go/config"
	"github.com/CLAOJ/claoj-go/db"
	"github.com/CLAOJ/claoj-go/db/migrations"
)

func main() {
	// Load configuration
	config.Load()

	// Connect to database
	db.Connect()

	// Run migration
	log.Println("Running migration: Create Roles and Permissions")
	if err := migrations.Migrate001CreateRolesAndPermissions(); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Migration completed successfully!")
}
