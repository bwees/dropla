package database

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"app/internal/config"
	"app/internal/database/models"

	"github.com/glebarez/sqlite"
	"go.uber.org/fx"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDatabase() *gorm.DB {
	dir := config.DatabaseDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		panic("failed to create database directory: " + err.Error())
	}

	// WAL matters here: uploads, downloads and the cleanup sweeps all touch the
	// database concurrently, and the default journal mode serialises them.
	dsn := filepath.Join(dir, "dropla.db") + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"

	// A missing row is a normal outcome throughout the repository layer, which
	// maps it to a nil result, so GORM must not log it as an error.
	gormLogger := logger.New(log.New(os.Stderr, "", log.LstdFlags), logger.Config{
		SlowThreshold:             time.Second,
		LogLevel:                  logger.Warn,
		IgnoreRecordNotFoundError: true,
	})

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{Logger: gormLogger})
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}

	if err := db.AutoMigrate(
		&models.User{},
		&models.Session{},
		&models.AccessToken{},
		&models.Share{},
		&models.ShareFile{},
		&models.PendingUpload{},
	); err != nil {
		panic("failed to migrate database: " + err.Error())
	}

	return db
}

var Module = fx.Provide(
	NewDatabase,
)
