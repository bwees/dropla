// Package config is the single source of truth for environment-derived
// configuration. Every setting is read here so defaults live in one place.
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envBytes(key string, fallback int64) int64 {
	n, err := strconv.ParseInt(os.Getenv(key), 10, 64)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}

func envDuration(key string, fallback time.Duration) time.Duration {
	d, err := time.ParseDuration(os.Getenv(key))
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}

func Port() string { return env("PORT", "3000") }

// DatabaseDir and UploadDir are separate settings by design: blobs are bulk data
// and must be able to live on different storage from the database.
func DatabaseDir() string { return env("DATABASE_DIR", "./data/db") }
func UploadDir() string   { return env("UPLOAD_DIR", "./data/uploads") }

func WebStaticPath() string { return os.Getenv("WEB_STATIC_PATH") }

// PublicBaseURL is the externally reachable origin of this instance. When set,
// Shares expose a second link built from it, so an instance reached over a
// private network can still hand out URLs that work from outside. Unset is fully
// supported: the UI then offers one link, and notifications omit theirs.
func PublicBaseURL() string {
	return strings.TrimRight(os.Getenv("PUBLIC_BASE_URL"), "/")
}

// DefaultUserQuotaBytes is the per-account budget when a user has no override.
// Zero means unlimited, which is the shipped default so a fresh single-user
// install is not surprised by a ceiling.
func DefaultUserQuotaBytes() int64 { return envBytes("DEFAULT_USER_QUOTA_BYTES", 0) }

// MaxUploadBytes caps a single file, independent of quotas. Zero means no limit.
func MaxUploadBytes() int64 { return envBytes("MAX_UPLOAD_BYTES", 0) }

// ShareGracePeriod is how long blobs survive after a Share expires. Expiry
// closes the public link; deletion of the data happens only after this window,
// so an Upload Box that lapses before its Owner collects does not eat the files.
func ShareGracePeriod() time.Duration { return envDuration("SHARE_GRACE_PERIOD", 168*time.Hour) }

// AbandonedUploadAfter must comfortably exceed any legitimate pause-and-resume.
func AbandonedUploadAfter() time.Duration { return envDuration("ABANDONED_UPLOAD_AFTER", 48*time.Hour) }

func SweepInterval() time.Duration { return envDuration("SWEEP_INTERVAL", time.Hour) }

// EmptyShareAfter is how long a Share with no files and no uploads in flight
// survives. Drop-first creation means a Share can exist having never held
// anything, so these are swept rather than left forever.
func EmptyShareAfter() time.Duration { return envDuration("EMPTY_SHARE_AFTER", time.Hour) }
