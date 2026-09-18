package updater_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/soulteary/nginx-formatter/internal/updater"
)

// TestScanFilesCoversSitesAvailableDefault covers the Debian/Ubuntu layout.
// sites-available/default is the most common site file on those systems and
// the only one with no extension, so the ".conf"-only filter skipped it
// silently.
func TestScanFilesCoversSitesAvailableDefault(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		full := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(body), 0600); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	write("nginx.conf", "a {\nb;\n}\n")
	write("conf.d/site.conf", "a {\nb;\n}\n")
	write("sites-available/default", "a {\nb;\n}\n")
	// Named "default" but not in an nginx sites directory: still not ours.
	write("html/default", "not a config\n")
	// A different extensionless name in a sites directory is also left alone.
	write("sites-available/example.com", "a {\nb;\n}\n")

	files, err := updater.ScanFiles(dir)
	if err != nil {
		t.Fatalf("ScanFiles: %v", err)
	}
	got := make([]string, 0, len(files))
	for _, f := range files {
		got = append(got, filepath.ToSlash(f))
	}

	for _, want := range []string{"nginx.conf", "conf.d/site.conf", "sites-available/default"} {
		if !slices.Contains(got, want) {
			t.Errorf("expected %s to be scanned, got %v", want, got)
		}
	}
	for _, unwanted := range []string{"html/default", "sites-available/example.com"} {
		if slices.Contains(got, unwanted) {
			t.Errorf("%s should not be scanned, got %v", unwanted, got)
		}
	}
}

// TestScanFilesCoversSitesEnabledDefault covers the enabled side of the same
// layout for the case where it is a real file rather than the usual symlink
// (symlinks are skipped, and the target is formatted through its own path).
func TestScanFilesCoversSitesEnabledDefault(t *testing.T) {
	dir := t.TempDir()
	full := filepath.Join(dir, "sites-enabled", "default")
	if err := os.MkdirAll(filepath.Dir(full), 0750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(full, []byte("a {\nb;\n}\n"), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	files, err := updater.ScanFiles(dir)
	if err != nil {
		t.Fatalf("ScanFiles: %v", err)
	}
	if len(files) != 1 || filepath.ToSlash(files[0]) != "sites-enabled/default" {
		t.Errorf("expected sites-enabled/default, got %v", files)
	}
}
