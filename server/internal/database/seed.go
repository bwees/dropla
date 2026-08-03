package database

import (
	"app/internal/database/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func seedNotes(db *gorm.DB) {
	notes := []models.Note{
		{ID: "welcome", Title: "Welcome", Body: "This note is stored in SQLite via GORM. Edit or delete it."},
	}
	db.Clauses(clause.OnConflict{DoNothing: true}).Create(&notes)
}
