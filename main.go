package main

import (
	"github.com/soulteary/nginx-formatter/internal/checker"
	"github.com/soulteary/nginx-formatter/internal/cmd"
)

func main() {
	checker.FailToRun(cmd.Execute())
}
