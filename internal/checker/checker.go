package checker

import (
	"errors"
	"log"
	"os"
)

// ErrDockerRootWorkDir is returned when the tool is asked to format "/" from
// inside a container, which is almost always a forgotten volume mount rather
// than a real request.
var ErrDockerRootWorkDir = errors.New(
	"refusing to format the container root directory.\n" +
		"Specify a run directory other than \"/\", for example:\n" +
		"  docker run --rm -it -v `pwd`:/app soulteary/nginx-formatter format -i /app")

// InDockerAndWorkDirIsRoot reports the refusal as an error rather than acting
// on it.
//
// It used to print the hint and call os.Exit(0). Exit code 0 means success, so
// a CI job or script saw a clean run when in fact nothing had been formatted —
// the one outcome a check like this must not produce.
func InDockerAndWorkDirIsRoot(src string) error {
	if _, err := os.Stat("/.dockerenv"); err == nil && src == "/" {
		return ErrDockerRootWorkDir
	}
	return nil
}

func FailToRun(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
