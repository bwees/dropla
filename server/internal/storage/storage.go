// Package storage keeps uploaded blobs on disk. Files are addressed only by the
// Share and file UUIDs, never by their original names, so nothing a visitor
// supplies ever reaches the filesystem.
package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"app/internal/config"

	"go.uber.org/fx"
)

type Storage struct {
	root string
}

func New() (*Storage, error) {
	root := config.UploadDir()
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	return &Storage{root: root}, nil
}

// Root is where tusd keeps its own state, alongside the committed blobs.
func (s *Storage) Root() string { return s.root }

func (s *Storage) ShareDir(shareID string) string {
	return filepath.Join(s.root, "shares", shareID)
}

func (s *Storage) Path(shareID, fileID string) string {
	return filepath.Join(s.ShareDir(shareID), fileID)
}

// TusDir holds in-progress uploads. Kept separate from committed blobs so the
// abandoned-upload sweep never has to distinguish them by name.
func (s *Storage) TusDir() string {
	return filepath.Join(s.root, "incoming")
}

// Commit moves a completed upload into its Share. A rename is attempted first
// since it is atomic within a filesystem, falling back to a copy when the upload
// directory spans devices.
func (s *Storage) Commit(src, shareID, fileID string) error {
	if err := os.MkdirAll(s.ShareDir(shareID), 0o755); err != nil {
		return err
	}

	dst := s.Path(shareID, fileID)
	if err := os.Rename(src, dst); err == nil {
		return nil
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(dst)
		return err
	}
	return os.Remove(src)
}

// Write streams a reader straight to a Share, used by the CLI path where there
// is no tus upload to commit.
func (s *Storage) Write(shareID, fileID string, r io.Reader) (int64, error) {
	if err := os.MkdirAll(s.ShareDir(shareID), 0o755); err != nil {
		return 0, err
	}

	dst := s.Path(shareID, fileID)
	f, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return 0, err
	}

	written, err := io.Copy(f, r)
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(dst)
		return 0, err
	}
	return written, nil
}

func (s *Storage) Open(shareID, fileID string) (*os.File, error) {
	return os.Open(s.Path(shareID, fileID))
}

func (s *Storage) Remove(shareID, fileID string) error {
	if err := os.Remove(s.Path(shareID, fileID)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *Storage) RemoveShare(shareID string) error {
	if err := os.RemoveAll(s.ShareDir(shareID)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// FreeBytes reports free space on the upload volume. Quotas have no global
// ceiling, so they can over-commit the disk and an admin needs to see this.
func (s *Storage) FreeBytes() (int64, error) {
	return freeBytes(s.root)
}

var Module = fx.Provide(New)
