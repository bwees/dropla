package zipstream

import (
	"archive/zip"
	"bytes"
	"io"
	"strings"
	"testing"
	"time"
)

func entry(name string, size int64, modified time.Time) Entry {
	return Entry{
		Name:     name,
		Size:     size,
		Modified: modified,
		Open: func() (io.ReadCloser, error) {
			return io.NopCloser(io.LimitReader(zeroes{}, size)), nil
		},
	}
}

type zeroes struct{}

func (zeroes) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}

// The predicted size must match the real archive byte for byte, because a
// Content-Length that is even one byte wrong makes every download fail.
func TestSizeMatchesWrittenArchive(t *testing.T) {
	mod := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name    string
		entries []Entry
	}{
		{"empty", nil},
		{"single small", []Entry{entry("a.txt", 11, mod)}},
		{"several", []Entry{
			entry("a.txt", 11, mod),
			entry("nested/b.bin", 4096, mod),
			entry("c", 0, mod),
		}},
		{"long name", []Entry{entry(strings.Repeat("x", 200)+".dat", 1234, mod)}},
		{"no modified time", []Entry{{
			Name: "plain.txt",
			Size: 5,
			Open: func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader("hello")), nil },
		}}},
		{"utf8 name", []Entry{entry("réunion-日本.txt", 42, mod)}},
		{"many entries", func() []Entry {
			out := make([]Entry, 300)
			for i := range out {
				out[i] = entry(string(rune('a'+i%26))+"/f.bin", int64(i), mod)
			}
			return out
		}()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			predicted := Size(tc.entries)

			var buf bytes.Buffer
			if err := Write(&buf, tc.entries); err != nil {
				t.Fatalf("write: %v", err)
			}

			if int64(buf.Len()) != predicted {
				t.Fatalf("predicted %d bytes, wrote %d (off by %d)",
					predicted, buf.Len(), int64(buf.Len())-predicted)
			}

			// The archive must also actually open, not merely be the right length.
			r, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
			if err != nil {
				t.Fatalf("resulting archive does not open: %v", err)
			}
			if len(r.File) != len(tc.entries) {
				t.Fatalf("expected %d entries, archive has %d", len(tc.entries), len(r.File))
			}
		})
	}
}

// A truncated archive must never look complete: the end-of-central-directory
// record is written last, so losing any tail removes it.
func TestTruncatedArchiveIsAlwaysDetectable(t *testing.T) {
	mod := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)
	entries := []Entry{entry("a.bin", 5000, mod), entry("b.bin", 9000, mod)}

	var buf bytes.Buffer
	if err := Write(&buf, entries); err != nil {
		t.Fatalf("write: %v", err)
	}
	full := buf.Bytes()

	for _, fraction := range []float64{0.3, 0.7, 0.99} {
		cut := int(float64(len(full)) * fraction)
		if _, err := zip.NewReader(bytes.NewReader(full[:cut]), int64(cut)); err == nil {
			t.Fatalf("a %.0f%% truncated archive opened without error", fraction*100)
		}
	}

	// Even one byte short must fail.
	short := len(full) - 1
	if _, err := zip.NewReader(bytes.NewReader(full[:short]), int64(short)); err == nil {
		t.Fatal("archive missing its final byte opened without error")
	}
}

func TestZip64WidensDescriptorAtTheBoundary(t *testing.T) {
	small := perEntryOverhead(5, uint32Max-1, true)
	large := perEntryOverhead(5, uint32Max, true)

	// 8 more descriptor bytes plus a 28-byte central-directory extra field.
	if large-small != 8+28 {
		t.Fatalf("expected Zip64 to add 36 bytes of overhead, added %d", large-small)
	}
}
