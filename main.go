package main

import (
	"github.com/soulteary/nginx-formatter/internal/formatter"
	"github.com/soulteary/nginx-formatter/internal/updater"
)

func main() {
	err := updater.UpdateConfInDir(".", formatter.Formatter)
	if err != nil {
		panic(err)
	}
}
