package checker_test

import (
	"errors"
	"os"
	"testing"

	"github.com/soulteary/nginx-formatter/internal/checker"
)

// TestInDockerAndWorkDirIsRoot covers the refusal path. It used to exit 0,
// which told CI the run had succeeded when nothing had been formatted.
func TestInDockerAndWorkDirIsRoot(t *testing.T) {
	_, inDocker := os.Stat("/.dockerenv")

	t.Run("a normal directory is always fine", func(t *testing.T) {
		if err := checker.InDockerAndWorkDirIsRoot("/app"); err != nil {
			t.Errorf("expected no error for /app, got %v", err)
		}
	})

	t.Run("root inside a container is refused", func(t *testing.T) {
		if inDocker != nil {
			t.Skip("not running inside a container: /.dockerenv is absent")
		}
		err := checker.InDockerAndWorkDirIsRoot("/")
		if !errors.Is(err, checker.ErrDockerRootWorkDir) {
			t.Errorf("expected ErrDockerRootWorkDir, got %v", err)
		}
	})

	t.Run("root outside a container is not this check's business", func(t *testing.T) {
		if inDocker == nil {
			t.Skip("running inside a container")
		}
		if err := checker.InDockerAndWorkDirIsRoot("/"); err != nil {
			t.Errorf("expected no error outside a container, got %v", err)
		}
	})
}
