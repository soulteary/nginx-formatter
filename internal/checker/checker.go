package checker

import (
	"fmt"
	"log"
	"os"
)

func InDockerAndWorkDirIsRoot(src string) {
	if _, err := os.Stat("/.dockerenv"); err == nil && src == "/" {
		fmt.Println("To run using the Docker model, you need to specify a run directory other than the root directory.")
		fmt.Println("example:")
		fmt.Println("  docker run --rm -it -v `pwd`:/app soulteary/nginx-formatter -input=/app")
		os.Exit(0)
	}
}

func InputDirExist(src string) {
	if _, err := os.Stat(src); err != nil {
		fmt.Println("The directory you specified does not exist, please check the path parameters and try again.")
		fmt.Println("Input directory:", src)
		os.Exit(0)
	}
}

func FailToRun(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
