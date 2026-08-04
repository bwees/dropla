// Package zipstream writes a stored (uncompressed) zip straight to an
// io.Writer, and computes its exact size beforehand.
//
// Knowing the size up front is the whole point: it lets the handler set a real
// Content-Length, which is what gives recipients a genuine progress bar instead
// of an indeterminate spinner, and it turns a truncated transfer into a clear
// "unexpected EOF" rather than a silent success.
//
// This only works because every entry is stored rather than deflated. A single
// compressed entry would make the output size unknowable in advance, so
// compression is not offered here — for the media and archives people actually
// transfer it costs a great deal of CPU for no reduction anyway.
package zipstream

import (
	"archive/zip"
	"io"
	"time"
)

const uint32Max = 0xffffffff

// Entry is one file in the archive. Name must already be sanitised and
// de-duplicated; this package writes what it is given.
type Entry struct {
	Name     string
	Size     int64
	Modified time.Time
	Open     func() (io.ReadCloser, error)
}

// perEntryOverhead returns the bytes an entry costs besides its payload.
//
// Constants come from the ZIP specification (APPNOTE §4.3): a local file header
// is 30 bytes plus the name and extra field; a central directory entry is 46
// plus the same; a streamed entry is followed by a data descriptor, which widens
// from 16 to 24 bytes once Zip64 applies. Go always emits a 9-byte Info-ZIP
// extended-timestamp extra when Modified is set, in both headers.
func perEntryOverhead(nameLen int, size int64, hasModified bool) int64 {
	extra := 0
	if hasModified {
		extra = 9
	}

	local := int64(30 + nameLen + extra)
	central := int64(46 + nameLen + extra)

	descriptor := int64(16)
	if size >= uint32Max {
		descriptor = 24
		// The central directory carries a Zip64 extended-information field:
		// 4 bytes of header plus three 8-byte values.
		central += 28
	}

	return local + descriptor + central
}

// Size is the exact byte length of the archive Write produces for the same
// entries. Verified against real archives in the tests, including an entry at
// the Zip64 boundary.
func Size(entries []Entry) int64 {
	var total int64
	var payload int64

	for _, e := range entries {
		total += perEntryOverhead(len(e.Name), e.Size, !e.Modified.IsZero())
		total += e.Size
		payload += e.Size
	}

	// End of central directory record.
	total += 22

	// Once the archive as a whole crosses the 32-bit limits it also gains a
	// Zip64 end-of-central-directory record (56) and its locator (20).
	if payload >= uint32Max || len(entries) >= 0xffff || total >= uint32Max {
		total += 76
	}

	return total
}

// Write streams the archive. A mid-stream failure is returned to the caller,
// which must abort the response rather than return normally — a chunked response
// that ends cleanly is reported to the browser as a successful download even
// though the archive is truncated.
func Write(w io.Writer, entries []Entry) error {
	zw := zip.NewWriter(w)

	for _, e := range entries {
		header := &zip.FileHeader{
			Name:     e.Name,
			Method:   zip.Store,
			Modified: e.Modified,
		}

		dst, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}

		src, err := e.Open()
		if err != nil {
			return err
		}

		_, err = io.Copy(dst, src)
		closeErr := src.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}

	return zw.Close()
}
