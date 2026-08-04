package controllers

import "testing"

func TestSanitiseFilenameNeutralisesTraversal(t *testing.T) {
	cases := map[string]string{
		"../../etc/passwd":     "passwd",
		"/absolute/path.txt":   "path.txt",
		`C:\windows\evil.exe`:  "evil.exe",
		"..":                   "file",
		".":                    "file",
		"":                     "file",
		"....//....//boot":     "boot",
		"normal.pdf":           "normal.pdf",
		"réunion-日本.txt":       "réunion-日本.txt",
		"trailing.  ":          "trailing",
		"evil.exe.":            "evil.exe",
		"a/b/c/d.zip":          "d.zip",
		"nested\\..\\..\\x.md": "x.md",
	}

	for input, want := range cases {
		if got := sanitiseFilename(input); got != want {
			t.Errorf("sanitiseFilename(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSanitiseFilenameStripsNulAndControlBytes(t *testing.T) {
	if got := sanitiseFilename("bad\x00name.txt"); got != "badname.txt" {
		t.Errorf("NUL survived: %q", got)
	}
	if got := sanitiseFilename("tab\tnew\nline.txt"); got != "tabnewline.txt" {
		t.Errorf("control bytes survived: %q", got)
	}
}

func TestSanitiseFilenameEscapesWindowsDeviceNames(t *testing.T) {
	for _, name := range []string{"CON", "con.txt", "NUL", "lpt1.log", "aux"} {
		got := sanitiseFilename(name)
		if got == name {
			t.Errorf("reserved device name %q passed through unchanged", name)
		}
	}
	if got := sanitiseFilename("console.txt"); got != "console.txt" {
		t.Errorf("%q is not reserved and should be untouched, got %q", "console.txt", got)
	}
}

func TestFilenameIsSuspicious(t *testing.T) {
	suspicious := []string{"../x", "a/b", `a\b`, "", "..", ".", "x\x00y", "CON"}
	for _, name := range suspicious {
		if !filenameIsSuspicious(name) {
			t.Errorf("expected %q to be rejected", name)
		}
	}

	fine := []string{"report.pdf", "réunion.txt", "my file (1).zip", "a.b.c.tar.gz"}
	for _, name := range fine {
		if filenameIsSuspicious(name) {
			t.Errorf("expected %q to be accepted", name)
		}
	}
}

// Go permits duplicate zip entries, and sanitising can itself create collisions,
// so names must be de-duplicated deterministically.
func TestNameDeduperResolvesCollisions(t *testing.T) {
	d := newNameDeduper()

	got := []string{
		d.unique("report.pdf"),
		d.unique("report.pdf"),
		d.unique("report.pdf"),
		d.unique("notes"),
		d.unique("notes"),
	}
	want := []string{"report.pdf", "report (1).pdf", "report (2).pdf", "notes", "notes (1)"}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("entry %d: got %q, want %q (all: %v)", i, got[i], want[i], got)
		}
	}
}

func TestNameDeduperDoesNotCollideWithAnExistingRenamedName(t *testing.T) {
	d := newNameDeduper()

	first := d.unique("a.txt")
	preTaken := d.unique("a (1).txt")
	second := d.unique("a.txt")

	if first == second || second == preTaken {
		t.Fatalf("collision: %q, %q, %q", first, preTaken, second)
	}
}

func TestContentDispositionAlwaysForcesDownload(t *testing.T) {
	got := contentDisposition("réunion \"quoted\".pdf")

	if !contains(got, "attachment;") {
		t.Errorf("expected attachment disposition, got %q", got)
	}
	if !contains(got, "filename*=UTF-8''") {
		t.Errorf("expected an RFC 5987 encoded filename, got %q", got)
	}
	if contains(got, `"réunion "quoted".pdf"`) {
		t.Errorf("unescaped quotes leaked into the header: %q", got)
	}
}

// Visitor-supplied markup must never be served as markup from our own origin.
func TestContentTypeNeverRendersHostileMarkup(t *testing.T) {
	cases := map[string]string{
		"text/html":                "application/octet-stream",
		"image/svg+xml":            "application/octet-stream",
		"application/xhtml+xml":    "application/octet-stream",
		"video/mp4":                "video/mp4",
		"application/pdf":          "application/pdf",
		"":                         "application/octet-stream",
		"application/octet-stream": "application/octet-stream",
	}

	for declared, want := range cases {
		if got := contentTypeFor(declared, "file.bin"); got != want {
			t.Errorf("contentTypeFor(%q) = %q, want %q", declared, got, want)
		}
	}

	if got := contentTypeFor("", "page.html"); got != "application/octet-stream" {
		t.Errorf("an .html extension must not become text/html, got %q", got)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool {
		for i := 0; i+len(needle) <= len(haystack); i++ {
			if haystack[i:i+len(needle)] == needle {
				return true
			}
		}
		return false
	})()
}
