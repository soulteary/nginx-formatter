package updater_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoDirectStdoutWrites is a source check rather than a behavioural one,
// because the failure it guards is invisible from inside the package: a new
// progress line written with fmt.Printf works perfectly and is simply not
// covered by --quiet. Nothing fails, nothing is logged, and the leak is only
// noticed by whoever was piping stdout.
//
// It has already happened three times -- the output-directory exclusion, the
// "already formatted" short-circuit, and the byte order mark notice were each
// added by a separate change, each with its own fmt.Printf. A test per line
// only ever catches the lines someone thought to test.
//
// fmt.Errorf and fmt.Sprintf are untouched by this; they do not write anywhere.
func TestNoDirectStdoutWrites(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}

	banned := []string{"fmt.Printf(", "fmt.Println(", "fmt.Print("}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name) // #nosec G304 -- a fixed listing of this package's own sources
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		for i, line := range strings.Split(string(src), "\n") {
			for _, bad := range banned {
				if strings.Contains(line, bad) {
					t.Errorf("%s:%d writes to stdout directly and so escapes --quiet; use fmt.Fprintf(Out, ...)\n\t%s",
						name, i+1, strings.TrimSpace(line))
				}
			}
		}
	}
}
