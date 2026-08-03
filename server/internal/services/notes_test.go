package services

import (
	"errors"
	"testing"

	"app/internal/database/models"
	"app/internal/dtos"
	"app/internal/repositories"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func newTestService(t *testing.T) *NotesService {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.Note{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return NewNotesService(repositories.NewNotesRepository(db))
}

func TestNotesServiceCRUD(t *testing.T) {
	svc := newTestService(t)

	created, err := svc.Create(dtos.CreateNoteDto{Title: "First", Body: "hello"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected created note to have an ID")
	}
	if created.Title != "First" || created.Body != "hello" {
		t.Fatalf("unexpected note contents: %+v", created)
	}

	got, err := svc.Get(created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got != created {
		t.Fatalf("Get returned %+v, want %+v", got, created)
	}

	notes, err := svc.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("expected 1 note, got %d", len(notes))
	}

	if err := svc.Delete(created.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := svc.Get(created.ID); !errors.Is(err, ErrNoteNotFound) {
		t.Fatalf("expected ErrNoteNotFound after delete, got %v", err)
	}
	if err := svc.Delete(created.ID); !errors.Is(err, ErrNoteNotFound) {
		t.Fatalf("expected ErrNoteNotFound on second delete, got %v", err)
	}
}
