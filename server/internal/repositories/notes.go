package repositories

import (
	"app/internal/database/models"

	"gorm.io/gorm"
)

type NotesRepository struct {
	db *gorm.DB
}

func NewNotesRepository(db *gorm.DB) *NotesRepository {
	return &NotesRepository{db: db}
}

func (r *NotesRepository) List() ([]models.Note, error) {
	notes := []models.Note{}
	if err := r.db.Order("title").Find(&notes).Error; err != nil {
		return nil, err
	}
	return notes, nil
}

func (r *NotesRepository) Get(id string) (models.Note, error) {
	var note models.Note
	err := r.db.First(&note, "id = ?", id).Error
	return note, err
}

func (r *NotesRepository) Create(note *models.Note) error {
	return r.db.Create(note).Error
}

func (r *NotesRepository) Delete(id string) (bool, error) {
	result := r.db.Delete(&models.Note{}, "id = ?", id)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}
