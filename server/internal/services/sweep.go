package services

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"app/internal/config"
	"app/internal/repositories"
	"app/internal/storage"

	"go.uber.org/fx"
)

// SweepService owns every scheduled cleanup. Two jobs share one ticker:
//
//  1. Shares whose expiry plus the grace period has passed. Expiry closes the
//     public link immediately; the data survives the grace window so an Upload Box
//     that lapses before its Owner collects does not eat the files.
//  2. Abandoned tus uploads.
//
// Every job is idempotent, and the sweep also runs once at startup so an instance
// that was down for a week catches up rather than waiting for the first tick.
type SweepService struct {
	shares   *repositories.SharesRepository
	sessions *repositories.SessionsRepository
	storage  *storage.Storage
}

func NewSweepService(
	shares *repositories.SharesRepository,
	sessions *repositories.SessionsRepository,
	store *storage.Storage,
) *SweepService {
	return &SweepService{shares: shares, sessions: sessions, storage: store}
}

func (s *SweepService) Start(lc fx.Lifecycle) {
	ctx, cancel := context.WithCancel(context.Background())

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				s.RunOnce()

				ticker := time.NewTicker(config.SweepInterval())
				defer ticker.Stop()

				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						s.RunOnce()
					}
				}
			}()
			return nil
		},
		OnStop: func(context.Context) error {
			cancel()
			return nil
		},
	})
}

func (s *SweepService) RunOnce() {
	s.sweepExpiredShares()
	s.sweepAbandonedUploads()
	if err := s.sessions.DeleteExpired(); err != nil {
		slog.Warn("sweep: expired sessions", "error", err)
	}
}

func (s *SweepService) sweepExpiredShares() {
	shares, err := s.shares.SweepableShares(config.ShareGracePeriod())
	if err != nil {
		slog.Warn("sweep: expired shares", "error", err)
		return
	}

	for _, share := range shares {
		if err := s.storage.RemoveShare(share.ID); err != nil {
			slog.Warn("sweep: remove blobs", "share", share.ID, "error", err)
			continue
		}
		if err := s.shares.Delete(share.ID); err != nil {
			slog.Warn("sweep: delete share", "share", share.ID, "error", err)
			continue
		}
		slog.Info("sweep: deleted expired share past its grace period", "share", share.ID)
	}
}

// sweepAbandonedUploads keys expiry off the upload's own file mtime, not the
// pending_upload row's creation time — an upload someone is actively resuming has
// a recent mtime but an old row, and deleting it would destroy work in progress.
func (s *SweepService) sweepAbandonedUploads() {
	cutoff := time.Now().Add(-config.AbandonedUploadAfter())

	pending, err := s.shares.PendingUploadsOlderThan(cutoff)
	if err != nil {
		slog.Warn("sweep: pending uploads", "error", err)
		return
	}

	for _, p := range pending {
		binPath := filepath.Join(s.storage.TusDir(), p.ID)

		if info, err := os.Stat(binPath); err == nil && info.ModTime().After(cutoff) {
			continue
		}

		_ = os.Remove(binPath)
		_ = os.Remove(binPath + ".info")

		if err := s.shares.DeletePendingUpload(p.ID); err != nil {
			slog.Warn("sweep: delete pending upload", "upload", p.ID, "error", err)
			continue
		}
		slog.Info("sweep: removed abandoned upload", "upload", p.ID)
	}
}
