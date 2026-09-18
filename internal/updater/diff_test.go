package updater_test

import (
	"strings"
	"testing"

	"github.com/soulteary/nginx-formatter/internal/updater"
)

func TestUnifiedDiff(t *testing.T) {
	t.Run("identical input produces nothing", func(t *testing.T) {
		if got := updater.UnifiedDiff("x.conf", "a\nb\n", "a\nb\n"); got != "" {
			t.Errorf("expected an empty diff, got %q", got)
		}
	})

	t.Run("a changed line shows both sides with context", func(t *testing.T) {
		got := updater.UnifiedDiff("x.conf", "server {\nlisten 80;\n}\n", "server {\n  listen 80;\n}\n")
		for _, want := range []string{
			"--- x.conf", "+++ x.conf", "@@ ",
			"-listen 80;", "+  listen 80;", " server {", " }",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("diff missing %q:\n%s", want, got)
			}
		}
	})

	t.Run("added and removed lines", func(t *testing.T) {
		got := updater.UnifiedDiff("x", "a\nb\nc\n", "a\nc\nd\n")
		if !strings.Contains(got, "-b") {
			t.Errorf("removal not reported:\n%s", got)
		}
		if !strings.Contains(got, "+d") {
			t.Errorf("addition not reported:\n%s", got)
		}
	})

	t.Run("distant changes become separate hunks", func(t *testing.T) {
		before := "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n13\n14\n15\n"
		after := "1x\n2\n3\n4\n5\n6\n7\n8\n9\n10\n11\n12\n13\n14\n15x\n"
		got := updater.UnifiedDiff("x", before, after)
		if n := strings.Count(got, "@@ "); n != 2 {
			t.Errorf("expected 2 hunks, got %d:\n%s", n, got)
		}
		// The untouched middle must be collapsed away.
		if strings.Contains(got, " 8") {
			t.Errorf("the unchanged middle was not collapsed:\n%s", got)
		}
	})

	t.Run("empty sides", func(t *testing.T) {
		if got := updater.UnifiedDiff("x", "", "a\n"); !strings.Contains(got, "+a") {
			t.Errorf("addition to an empty file not reported:\n%s", got)
		}
		if got := updater.UnifiedDiff("x", "a\n", ""); !strings.Contains(got, "-a") {
			t.Errorf("emptying a file not reported:\n%s", got)
		}
	})
}
