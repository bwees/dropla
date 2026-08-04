package services

import (
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strings"
	"time"

	"app/internal/config"
	"app/internal/database/models"
	"app/internal/repositories"

	"github.com/nicholas-fedor/shoutrrr"
	"github.com/nicholas-fedor/shoutrrr/pkg/types"
)

type NotifyService struct {
	shares *repositories.SharesRepository
}

func NewNotifyService(shares *repositories.SharesRepository) *NotifyService {
	return &NotifyService{shares: shares}
}

// UploadBoxArrival tells the Owner that files landed, using whatever shoutrrr URL
// they configured — ntfy, Discord, Telegram, Slack, smtp and so on.
//
// Filenames are deliberately omitted. The Owner may see them in the app, but a
// notification travels through a third-party service, and they opted into that
// service rather than into publishing their Share listing.
//
// Dispatch is fire-and-forget so a slow or dead target can never delay or fail an
// upload. There are no retries; instead the last error is stored on the Share and
// surfaced in the dashboard, because silent notification failure is the classic
// version of this bug — a mistyped URL must not look like "nobody uploaded".
func (s *NotifyService) UploadBoxArrival(share *models.Share, fileCount int, label string) {
	if !share.NotifyEnabled || strings.TrimSpace(share.NotifyURL) == "" {
		return
	}

	go func() {
		message := fmt.Sprintf("%d file%s arrived in %q",
			fileCount, plural(fileCount), share.Title)
		if label != "" {
			message += fmt.Sprintf(" — from %q (self-declared)", label)
		}

		// An absolute link needs PUBLIC_BASE_URL: a background dispatch has no
		// browser origin to compose one from, so omit rather than guess.
		if base := config.PublicBaseURL(); base != "" {
			message += "\n" + base + "/shares/" + share.ID
		}

		if err := s.send(share.NotifyURL, message); err != nil {
			slog.Warn("notification failed", "share", share.ID, "error", err)
			_ = s.shares.SetNotifyError(share.ID, err.Error())
			return
		}
		_ = s.shares.SetNotifyError(share.ID, "")
	}()
}

func (s *NotifyService) send(rawURL, message string) error {
	if err := guardSSRF(rawURL); err != nil {
		return err
	}

	// Bound the attempt so a hanging target cannot pin the goroutine forever.
	sender, err := shoutrrr.CreateSenderWithOptions(
		types.SenderOptions{Timeout: 15 * time.Second},
		rawURL,
	)
	if err != nil {
		return err
	}

	errs := sender.Send(message, nil)
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// guardSSRF refuses generic webhook targets pointed at link-local or loopback
// addresses. Only authenticated account holders can set these URLs, so this is
// not an anonymous attack surface — but it still stops a non-admin using dropla
// to reach a cloud metadata endpoint.
func guardSSRF(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "generic" && scheme != "webhook" {
		return nil
	}

	host := parsed.Hostname()
	if host == "" {
		return nil
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		// A name that does not resolve will fail at send time anyway.
		return nil
	}
	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return fmt.Errorf("refusing to send to %s: link-local or loopback address", host)
		}
	}
	return nil
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
