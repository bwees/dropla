package controllers

import (
	"net/http"
	"testing"
)

func request(host, forwardedHost string) *http.Request {
	r := &http.Request{Host: host, Header: http.Header{}}
	if forwardedHost != "" {
		r.Header.Set("X-Forwarded-Host", forwardedHost)
	}
	return r
}

func TestMaxRequestBytesCapsOnlyThePublicOrigin(t *testing.T) {
	t.Setenv("PUBLIC_BASE_URL", "https://dropla.example.io")
	t.Setenv("PUBLIC_MAX_REQUEST_BYTES", "104857600")

	cases := []struct {
		name          string
		host          string
		forwardedHost string
		want          int64
	}{
		{"public host is capped", "dropla.example.io", "", 104857600},
		{"private host is uncapped", "dropla.example.lab", "", 0},
		{"host match ignores case", "DROPLA.EXAMPLE.IO", "", 104857600},
		{"forwarded host wins over the direct one", "dropla.svc.cluster.local", "dropla.example.io", 104857600},
		{"forwarded chain uses the original client hop", "dropla.svc.cluster.local", "dropla.example.io, proxy.internal", 104857600},
		{"forwarded private host stays uncapped", "dropla.example.io", "dropla.example.lab", 0},
	}

	for _, c := range cases {
		if got := maxRequestBytes(request(c.host, c.forwardedHost)); got != c.want {
			t.Errorf("%s: maxRequestBytes(host=%q forwarded=%q) = %d, want %d",
				c.name, c.host, c.forwardedHost, got, c.want)
		}
	}
}

// An unset PUBLIC_BASE_URL means there is no proxied origin to cap, so every
// request must be reported as uncapped however the cap itself is configured.
func TestMaxRequestBytesUncappedWithoutAPublicBaseURL(t *testing.T) {
	t.Setenv("PUBLIC_BASE_URL", "")
	t.Setenv("PUBLIC_MAX_REQUEST_BYTES", "104857600")

	if got := maxRequestBytes(request("dropla.example.io", "")); got != 0 {
		t.Errorf("maxRequestBytes = %d, want 0", got)
	}
}

func TestMaxRequestBytesUncappedWhenNoCapIsConfigured(t *testing.T) {
	t.Setenv("PUBLIC_BASE_URL", "https://dropla.example.io")
	t.Setenv("PUBLIC_MAX_REQUEST_BYTES", "")

	if got := maxRequestBytes(request("dropla.example.io", "")); got != 0 {
		t.Errorf("maxRequestBytes = %d, want 0", got)
	}
}
