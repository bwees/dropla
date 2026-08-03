package services

import (
	"errors"

	"app/internal/database/models"
	"app/internal/dtos"
	"app/internal/repositories"

	"gorm.io/gorm"
)

var ErrNoteNotFound = errors.New("note not found")

type NotesService struct {
	repo *repositories.NotesRepository
}

func NewNotesService(repo *repositories.NotesRepository) *NotesService {
	return &NotesService{repo: repo}
}

func (s *NotesService) List() ([]models.Note, error) {
	return s.repo.List()
}

func (s *NotesService) Get(id string) (models.Note, error) {
	note, err := s.repo.Get(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.Note{}, ErrNoteNotFound
	}
	return note, err
}

func (s *NotesService) Create(dto dtos.CreateNoteDto) (models.Note, error) {
	note := models.Note{Title: dto.Title, Body: dto.Body}
	if err := s.repo.Create(&note); err != nil {
		return models.Note{}, err
	}
	return note, nil
}

func (s *NotesService) Delete(id string) error {
	deleted, err := s.repo.Delete(id)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrNoteNotFound
	}
	return nil
}
