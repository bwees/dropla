package services

import (
	"testing"

	"app/internal/database/models"
	"app/internal/dtos"
	"app/internal/repositories"
	"app/internal/storage"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newSharesService(t *testing.T) (*SharesService, *models.User) {
	t.Helper()

	t.Setenv("UPLOAD_DIR", t.TempDir())
	t.Setenv("DATABASE_DIR", t.TempDir())

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Session{}, &models.AccessToken{},
		&models.Share{}, &models.ShareFile{}, &models.PendingUpload{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	store, err := storage.New()
	if err != nil {
		t.Fatalf("storage: %v", err)
	}

	users := repositories.NewUsersRepository(db)
	shares := repositories.NewSharesRepository(db)
	svc := NewSharesService(shares, users, store, NewNotifyService(shares))

	user := &models.User{Username: "bwees", PasswordHash: "x"}
	if err := users.Create(user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	return svc, user
}

func pendingForNewShare(user *models.User, sessionID, title string) *models.PendingUpload {
	return &models.PendingUpload{
		ID:              uuid.NewString(),
		OwnerID:         user.ID,
		NewShareTitle:   title,
		UploadSessionID: sessionID,
		ByOwner:         true,
	}
}

// The point of deferring: a drop whose files never arrive must leave no Share.
func TestNoShareExistsUntilAFileLands(t *testing.T) {
	svc, user := newSharesService(t)
	session := uuid.NewString()

	pending := pendingForNewShare(user, session, "report.pdf")
	if err := svc.ReserveWithID(pending); err != nil {
		t.Fatalf("reserve: %v", err)
	}

	shares, err := svc.List(user.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(shares) != 0 {
		t.Fatalf("expected no share while the upload is in flight, got %d", len(shares))
	}

	if err := svc.ReleasePending(pending.ID); err != nil {
		t.Fatalf("release: %v", err)
	}
	if _, err := svc.ShareForUploadSession(user.ID, session); err != ErrShareNotFound {
		t.Fatalf("expected ErrShareNotFound for a drop that never landed, got %v", err)
	}
}

func TestFirstLandedFileCreatesTheShare(t *testing.T) {
	svc, user := newSharesService(t)
	session := uuid.NewString()

	pending := pendingForNewShare(user, session, "report.pdf")
	if err := svc.ReserveWithID(pending); err != nil {
		t.Fatalf("reserve: %v", err)
	}

	share, err := svc.ShareForPending(pending)
	if err != nil {
		t.Fatalf("share for pending: %v", err)
	}
	if share.Title != "report.pdf" || share.Mode != models.ShareModeSend {
		t.Fatalf("unexpected share: %+v", share)
	}
	if _, err := svc.Commit(pending, share.ID, "report.pdf", 12, "application/pdf"); err != nil {
		t.Fatalf("commit: %v", err)
	}

	found, err := svc.ShareForUploadSession(user.ID, session)
	if err != nil {
		t.Fatalf("share for session: %v", err)
	}
	if found.ID != share.ID {
		t.Fatalf("session resolved to %s, want %s", found.ID, share.ID)
	}
}

// Every file of one drop belongs in the same Share, so only the first creates it.
func TestLaterFilesJoinTheShareTheFirstCreated(t *testing.T) {
	svc, user := newSharesService(t)
	session := uuid.NewString()

	first := pendingForNewShare(user, session, "2 files")
	second := pendingForNewShare(user, session, "2 files")
	for _, p := range []*models.PendingUpload{first, second} {
		if err := svc.ReserveWithID(p); err != nil {
			t.Fatalf("reserve: %v", err)
		}
	}

	share, err := svc.ShareForPending(first)
	if err != nil {
		t.Fatalf("share for first: %v", err)
	}
	if _, err := svc.Commit(first, share.ID, "a.txt", 1, ""); err != nil {
		t.Fatalf("commit first: %v", err)
	}

	again, err := svc.ShareForPending(second)
	if err != nil {
		t.Fatalf("share for second: %v", err)
	}
	if again.ID != share.ID {
		t.Fatalf("second file made its own share %s, want %s", again.ID, share.ID)
	}

	shares, err := svc.List(user.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(shares) != 1 {
		t.Fatalf("expected one share for one drop, got %d", len(shares))
	}
}

// An upload into an existing Share still goes exactly where it was told to.
func TestPendingWithAShareIsUnaffected(t *testing.T) {
	svc, user := newSharesService(t)

	box, err := svc.Create(user.ID, dtos.CreateShareDto{Title: "Bids", Mode: string(models.ShareModeUploadBox)})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	pending := &models.PendingUpload{
		ID:              uuid.NewString(),
		OwnerID:         user.ID,
		ShareID:         box.ID,
		UploadSessionID: uuid.NewString(),
	}
	if err := svc.ReserveWithID(pending); err != nil {
		t.Fatalf("reserve: %v", err)
	}

	share, err := svc.ShareForPending(pending)
	if err != nil {
		t.Fatalf("share for pending: %v", err)
	}
	if share.ID != box.ID {
		t.Fatalf("resolved to %s, want %s", share.ID, box.ID)
	}
}

// A Share belongs to one account, so another account's id must not reach it.
func TestShareForUploadSessionIsOwnerScoped(t *testing.T) {
	svc, user := newSharesService(t)
	session := uuid.NewString()

	pending := pendingForNewShare(user, session, "report.pdf")
	if err := svc.ReserveWithID(pending); err != nil {
		t.Fatalf("reserve: %v", err)
	}
	share, err := svc.ShareForPending(pending)
	if err != nil {
		t.Fatalf("share for pending: %v", err)
	}
	if _, err := svc.Commit(pending, share.ID, "report.pdf", 12, ""); err != nil {
		t.Fatalf("commit: %v", err)
	}

	if _, err := svc.ShareForUploadSession("someone-else", session); err != ErrShareNotFound {
		t.Fatalf("expected ErrShareNotFound for another account, got %v", err)
	}
}
