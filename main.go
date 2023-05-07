package main

import (
	"fmt"

	"github.com/soulteary/nginx-formatter/internal/checker"
	"github.com/soulteary/nginx-formatter/internal/cmd"
	"github.com/soulteary/nginx-formatter/internal/formatter"
	"github.com/soulteary/nginx-formatter/internal/server"
	"github.com/soulteary/nginx-formatter/internal/updater"
	"github.com/soulteary/nginx-formatter/internal/version"
)

func main() {
	fmt.Printf("Nginx Formatter %s\n\n", version.Version)

	src, dest, indent, char, web, port := cmd.InitArgv()
	if web {
		err := server.Launch(port, indent, char, formatter.Formatter)
		checker.FailToRun(err)
	} else {
		checker.InDockerAndWorkDirIsRoot(src)
		checker.InputDirExist(src)

		err := updater.UpdateConfInDir(src, dest, indent, char, formatter.Formatter)
		checker.FailToRun(err)
	}
}
