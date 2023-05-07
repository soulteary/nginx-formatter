package main

import (
	"fmt"
	"log"
	"os"

	"github.com/soulteary/nginx-formatter/internal/cmd"
	"github.com/soulteary/nginx-formatter/internal/formatter"
	"github.com/soulteary/nginx-formatter/internal/server"
	"github.com/soulteary/nginx-formatter/internal/updater"
	"github.com/soulteary/nginx-formatter/internal/version"
)

func main() {
	fmt.Printf("Nginx Formatter v%s\n\n", version.Version)

	src, dest, indent, char, web, port := cmd.InitArgv()
	if web {
		err := server.Launch(port, indent, char, formatter.Formatter)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		if _, err := os.Stat("/.dockerenv"); err == nil && src == "/" {
			fmt.Println("To run using the Docker model, you need to specify a run directory other than the root directory.")
			fmt.Println("example:")
			fmt.Println("  docker run --rm -it -v `pwd`:/app soulteary/nginx-formatter -input=/app")
			os.Exit(0)
		}

		if _, err := os.Stat(src); err != nil {
			fmt.Println("The directory you specified does not exist, please check the path parameters and try again.")
			fmt.Println("Input directory:", src)
			os.Exit(0)
		}

		err := updater.UpdateConfInDir(src, dest, indent, char, formatter.Formatter)
		if err != nil {
			log.Fatal(err)
		}
	}
}
