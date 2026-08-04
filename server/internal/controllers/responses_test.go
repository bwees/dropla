package controllers

import (
	"net/http/httptest"
	"testing"
)

// These two guards exist because fuego's behaviour on non-200 typed routes is
// surprising in ways that fail silently rather than loudly.

// A 201 that is labelled text/plain makes generated clients return the raw body
// string instead of a parsed object, so callers see fields as undefined with no
// error anywhere. Every handler overriding the status code must set the type.
func TestForceJSONSetsAJSONContentType(t *testing.T) {
	rec := httptest.NewRecorder()

	forceJSON(rec)

	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", got)
	}
}

// Announcing a trailer commits a response to a chunked body, which a 204 cannot
// have. Node-based proxies reject the combination and terminate, which reads as
// the API server having crashed.
func TestStripTrailerRemovesTheTrailerHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Header().Set("Trailer", "Server-Timing")

	stripTrailer(rec)

	if got := rec.Header().Get("Trailer"); got != "" {
		t.Fatalf("Trailer = %q, want it removed", got)
	}
}
