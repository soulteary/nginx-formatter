package main

import (
	"fmt"
	"log"
	"os"

	"github.com/soulteary/nginx-formatter/internal/formatter"
	"github.com/soulteary/nginx-formatter/internal/updater"
	"github.com/soulteary/nginx-formatter/internal/version"
)

func main() {
	fmt.Printf("Nginx Formatter v%s\n\n", version.Version)

	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	err = updater.UpdateConfInDir(dir, formatter.Formatter)
	if err != nil {
		log.Fatal(err)
	}
}
