package updater_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/soulteary/nginx-formatter/internal/formatter"
	"github.com/soulteary/nginx-formatter/internal/updater"
)

// withBOM prefixes a UTF-8 byte order mark, spelled as an escape so no editor
// or patch tool can lose it on the way into this file.
func withBOM(s string) string { return "\ufeff" + s }

// hasBOMOnDisk reports whether the file still starts with a byte order mark,
// which is the only thing that settles what actually happened to it.
func hasBOMOnDisk(t *testing.T, path string) bool {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return strings.HasPrefix(string(b), "\ufeff")
}

func captureOut(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	saved := os.Stdout
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		var sb strings.Builder
		buf := make([]byte, 4096)
		for {
			n, readErr := r.Read(buf)
			sb.Write(buf[:n])
			if readErr != nil {
				break
			}
		}
		done <- sb.String()
	}()
	fn()
	os.Stdout = saved
	_ = w.Close()
	return <-done
}

// TestBOMNoticeRequiresAWrite is the whole justification for announcing this
// at all: the formatter is editing the user's file, so it says so. A notice
// printed before the work is attempted says so on the paths where no edit
// happens. A file that does not parse is left exactly as it was, byte order
// mark included, while the log claimed it had been removed — so someone
// reading it believes a file was repaired when it was not.
func TestBOMNoticeRequiresAWrite(t *testing.T) {
	dir := t.TempDir()
	broken := filepath.Join(dir, "broken.conf")
	if err := os.WriteFile(broken, []byte(withBOM("server {\nlisten 80;\n")), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	out := captureOut(t, func() {
		if err := updater.UpdateConfFile(broken, "", 2, " ", formatter.Formatter); err == nil {
			t.Error("expected the unbalanced brace to be reported")
		}
	})

	if !hasBOMOnDisk(t, broken) {
		t.Fatal("the file was rewritten even though it could not be parsed")
	}
	if strings.Contains(out, "UTF-8 BOM") {
		t.Errorf("the file still has its BOM, but the run announced one:\n%s", out)
	}
}

// TestBOMNoticeInPlace is the other direction: when the mark really is removed
// from the user's own file, that has to be said.
func TestBOMNoticeInPlace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "x.conf")
	if err := os.WriteFile(path, []byte(withBOM("server {\nlisten 80;\n}\n")), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	out := captureOut(t, func() {
		if err := updater.UpdateConfFile(path, "", 2, " ", formatter.Formatter); err != nil {
			t.Fatalf("UpdateConfFile: %v", err)
		}
	})

	if hasBOMOnDisk(t, path) {
		t.Fatal("the BOM survived an in-place format")
	}
	if !strings.Contains(out, "removed it") {
		t.Errorf("the removal was not reported:\n%s", out)
	}

	// Nothing to report on the second pass.
	again := captureOut(t, func() {
		if err := updater.UpdateConfFile(path, "", 2, " ", formatter.Formatter); err != nil {
			t.Fatalf("UpdateConfFile: %v", err)
		}
	})
	if strings.Contains(again, "UTF-8 BOM") {
		t.Errorf("the notice fired again with nothing to remove:\n%s", again)
	}
}

// TestBOMNoticeWithSeparateOutputDoesNotClaimTheInput covers the mode whose
// entire contract is that the input is left alone. The formatted copy has no
// mark, but the original keeps its own — so a notice saying the named file was
// edited is wrong about the one file the user asked not to touch.
func TestBOMNoticeWithSeparateOutputDoesNotClaimTheInput(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "in")
	dst := filepath.Join(dir, "out")
	if err := os.MkdirAll(src, 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	input := filepath.Join(src, "x.conf")
	if err := os.WriteFile(input, []byte(withBOM("server {\nlisten 80;\n}\n")), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	out := captureOut(t, func() {
		if err := updater.UpdateConfInDir(src, dst, 2, " ", formatter.Formatter); err != nil {
			t.Fatalf("UpdateConfInDir: %v", err)
		}
	})

	if !hasBOMOnDisk(t, input) {
		t.Error("the input file was modified even though a separate output directory was given")
	}
	if hasBOMOnDisk(t, filepath.Join(dst, "x.conf")) {
		t.Error("the formatted copy still carries a BOM")
	}
	if strings.Contains(out, "removed it") {
		t.Errorf("the run claimed to have edited the input, which it did not touch:\n%s", out)
	}
	if !strings.Contains(out, "the input is unchanged") {
		t.Errorf("the notice does not say which file was written:\n%s", out)
	}
}

// TestBOMNoticeSingleFileToOtherTarget is the same guarantee in single-file
// mode, where -o names a file rather than a directory.
func TestBOMNoticeSingleFileToOtherTarget(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "x.conf")
	target := filepath.Join(dir, "y.conf")
	if err := os.WriteFile(input, []byte(withBOM("server {\nlisten 80;\n}\n")), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	out := captureOut(t, func() {
		if err := updater.UpdateConfFile(input, target, 2, " ", formatter.Formatter); err != nil {
			t.Fatalf("UpdateConfFile: %v", err)
		}
	})

	if !hasBOMOnDisk(t, input) {
		t.Error("the input file was modified even though a separate target was given")
	}
	if hasBOMOnDisk(t, target) {
		t.Error("the written target still carries a BOM")
	}
	if strings.Contains(out, "removed it") {
		t.Errorf("the run claimed to have edited the input, which it did not touch:\n%s", out)
	}
	if !strings.Contains(out, "the input is unchanged") {
		t.Errorf("the notice does not say which file was written:\n%s", out)
	}
}

// TestBOMWithCRLFIsTheRealisticCase pairs the two things a Windows editor does
// to a config: it writes a byte order mark and it writes CRLF line endings. A
// file with one almost always has the other, yet the two are handled by
// different passes -- the BOM in the parser, the line endings in the formatter
// -- so nothing was checking that they compose. Stripping the mark must not
// cost the file its line endings, and preserving the line endings must not
// carry the mark back in.
func TestBOMWithCRLFIsTheRealisticCase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "windows.conf")
	if err := os.WriteFile(path, []byte(withBOM("server {\r\nlisten 80;\r\n}\r\n")), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := updater.UpdateConfFile(path, "", 2, " ", formatter.Formatter); err != nil {
		t.Fatalf("UpdateConfFile: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if hasBOMOnDisk(t, path) {
		t.Error("the BOM survived")
	}
	if !strings.Contains(string(got), "\r\n") {
		t.Errorf("the file lost its CRLF line endings: %q", got)
	}
	if strings.Contains(strings.ReplaceAll(string(got), "\r\n", ""), "\n") {
		t.Errorf("line endings came back mixed: %q", got)
	}
	if !strings.Contains(string(got), "  listen 80;") {
		t.Errorf("the file was not actually formatted: %q", got)
	}
}
