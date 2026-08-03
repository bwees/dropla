package database

import (
	"os"
	"path/filepath"

	"app/internal/config"
	"app/internal/database/models"

	"github.com/glebarez/sqlite"
	"go.uber.org/fx"
	"gorm.io/gorm"
)

func NewDatabase() *gorm.DB {
	dataDir := config.DataDir()
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		panic("failed to create data directory: " + err.Error())
	}

	dsn := filepath.Join(dataDir, "app.db")

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}

	if err := db.AutoMigrate(&models.Note{}); err != nil {
		panic("failed to migrate database: " + err.Error())
	}

	seedNotes(db)

	return db
}

var Module = fx.Provide(
	NewDatabase,
)
