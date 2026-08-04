package controllers

import (
	"fmt"
	"path"
	"strings"
	"unicode"
)

// Filename handling is security-relevant, not cosmetic. Go's archive/zip writes
// whatever name it is given — including "../../etc/passwd" — and its reader does
// not reject such names either, because ErrInsecurePath is off by default. Names
// arrive from uploaders and from the CLI, so they are attacker-controlled and are
// sanitised at ingest rather than only when an archive is built.

var windowsReserved = map[string]bool{
	"con": true, "prn": true, "aux": true, "nul": true,
	"com1": true, "com2": true, "com3": true, "com4": true,
	"com5": true, "com6": true, "com7": true, "com8": true, "com9": true,
	"lpt1": true, "lpt2": true, "lpt3": true, "lpt4": true,
	"lpt5": true, "lpt6": true, "lpt7": true, "lpt8": true, "lpt9": true,
}

// sanitiseFilename reduces a supplied name to a safe single path segment,
// following RFC 6266 §4.3. It always returns something usable.
func sanitiseFilename(name string) string {
	// Take the last segment only, treating both separators as such, so neither
	// "a/b/c" nor "C:\dir\evil" can escape.
	name = strings.ReplaceAll(name, "\\", "/")
	name = path.Base(path.Clean("/" + name))

	name = strings.Map(func(r rune) rune {
		if r == 0 || unicode.IsControl(r) {
			return -1
		}
		return r
	}, name)

	name = strings.TrimSpace(name)
	// Trailing dots and spaces are silently stripped by Windows, which turns
	// "evil.exe." into "evil.exe".
	name = strings.TrimRight(name, ". ")

	// path.Base collapses "..", "." and "/" to "/", so that has to be caught too.
	if name == "" || name == "." || name == ".." || name == "/" {
		return "file"
	}

	stem := strings.ToLower(name)
	if dot := strings.IndexByte(stem, '.'); dot > 0 {
		stem = stem[:dot]
	}
	if windowsReserved[stem] {
		name = "_" + name
	}

	// Keep names short enough for every common filesystem.
	if len(name) > 200 {
		ext := path.Ext(name)
		if len(ext) > 20 {
			ext = ""
		}
		name = name[:200-len(ext)] + ext
	}

	return name
}

// filenameIsSuspicious reports whether a supplied name needed more than cosmetic
// cleaning. The CLI rejects such names loudly rather than silently normalising
// them, so a script doing something wrong finds out.
func filenameIsSuspicious(name string) bool {
	if name == "" {
		return true
	}
	if strings.ContainsAny(name, "/\\") || strings.ContainsRune(name, 0) {
		return true
	}
	if name == "." || name == ".." {
		return true
	}
	return sanitiseFilename(name) != name
}

// nameDeduper keeps archive entry names unique. Go permits duplicate entries, and
// sanitising can itself create collisions, so this runs after sanitising.
type nameDeduper struct {
	seen map[string]int
}

func newNameDeduper() *nameDeduper {
	return &nameDeduper{seen: map[string]int{}}
}

func (d *nameDeduper) unique(name string) string {
	n, clash := d.seen[name]
	if !clash {
		// Zero, not one: the count records how many suffixed variants exist so far,
		// so the first collision becomes "(1)" rather than "(2)".
		d.seen[name] = 0
		return name
	}

	ext := path.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for {
		n++
		candidate := fmt.Sprintf("%s (%d)%s", stem, n, ext)
		if _, taken := d.seen[candidate]; !taken {
			d.seen[name] = n
			d.seen[candidate] = 0
			return candidate
		}
	}
}

func asciiFallback(name string) string {
	var b strings.Builder
	for _, r := range name {
		if r < 128 && r != '"' && r != '\\' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	if out := b.String(); out != "" {
		return out
	}
	return "file"
}

func percentEncode(s string) string {
	const safe = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_.~"

	var b strings.Builder
	for _, byteValue := range []byte(s) {
		if strings.IndexByte(safe, byteValue) >= 0 {
			b.WriteByte(byteValue)
		} else {
			fmt.Fprintf(&b, "%%%02X", byteValue)
		}
	}
	return b.String()
}
