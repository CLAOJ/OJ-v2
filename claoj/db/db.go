package db

import (
	"log"

	"github.com/CLAOJ/claoj/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// Connect opens the GORM connection to MySQL using the configured DSN.
// CLAOJ uses the same MySQL database as Django — we do NOT run AutoMigrate
// to avoid touching the schema managed by Django migrations.
func Connect() {
	cfg := config.C.Database
	if cfg.DSN == "" {
		log.Println("db: no DSN configured, skipping connection")
		return
	}

	gormCfg := &gorm.Config{}
	if config.C.Server.Mode == "release" {
		gormCfg.Logger = logger.Default.LogMode(logger.Silent)
	} else {
		gormCfg.Logger = logger.Default.LogMode(logger.Info)
	}

	var err error
	DB, err = gorm.Open(mysql.New(mysql.Config{
		DSN:                       cfg.DSN,
		SkipInitializeWithVersion: false,
	}), gormCfg)
	if err != nil {
		log.Fatalf("db: failed to connect: %v", err)
	}

	sqlDB, _ := DB.DB()
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)

	log.Println("db: connected successfully")
}
