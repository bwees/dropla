package controllers

import (
	"net/http"
	"net/url"
	"strings"

	"app/internal/config"
	"app/internal/dtos"

	"github.com/go-fuego/fuego"
)

type ConfigController struct{}

func NewConfigController() *ConfigController { return &ConfigController{} }

func (c *ConfigController) Register(api *fuego.Server) {
	fuego.Get(api, "/config", func(ctx fuego.ContextNoBody) (dtos.InstanceConfigDto, error) {
		return dtos.InstanceConfigDto{
			MaxRequestBytes: maxRequestBytes(ctx.Request()),
		}, nil
	}, fuego.OptionOperationID("getInstanceConfig"))
}

// maxRequestBytes reports the ceiling for the origin a request arrived through,
// which the client cannot determine for itself: the same instance is reachable
// both through a proxy that caps request bodies and directly, and only the
// server knows which name belongs to which.
//
// Any origin that is not the configured public one is reported as uncapped. That
// direction of error is the recoverable one — an oversized request is refused
// visibly, whereas guessing a cap would silently throttle every private
// transfer.
func maxRequestBytes(r *http.Request) int64 {
	base := config.PublicBaseURL()
	if base == "" {
		return 0
	}

	public, err := url.Parse(base)
	if err != nil || public.Host == "" {
		return 0
	}
	if !strings.EqualFold(requestHost(r), public.Host) {
		return 0
	}
	return config.PublicMaxRequestBytes()
}

// requestHost prefers the forwarded host because the tus handler is configured
// to respect forwarded headers, and the two must agree on which origin a request
// arrived through or uploads would be split on one path and not the other.
func requestHost(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-Host"); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	return r.Host
}
