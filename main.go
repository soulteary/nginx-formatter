package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/soulteary/nginx-formatter/internal/formatter"
	"github.com/soulteary/nginx-formatter/internal/updater"
	"github.com/soulteary/nginx-formatter/internal/version"
)

var FORMATTER_SRC = ""
var FORMATTER_DEST = ""
var FORMATTER_INDENT = 2
var FORMATTER_CHAR = " "

func InitArgv() {
	var inputDir string
	var outputDir string
	var indent int
	var indentChar string
	flag.StringVar(&inputDir, "input", "", "Input directory")
	flag.StringVar(&outputDir, "output", "", "Output directory")
	flag.IntVar(&indent, "indent", 2, "Indent size")
	flag.StringVar(&indentChar, "char", " ", "Indent char")
	flag.Parse()

	if inputDir == "" {
		dir, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("No input directory specified, use the current working directory:", dir)
		FORMATTER_SRC = dir
	} else {
		fmt.Println("Specify the working directory as:", inputDir)
		FORMATTER_SRC = inputDir
	}

	if outputDir == "" {
		dir, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("No output directory specified, use the current working directory:", dir)
		FORMATTER_DEST = dir
	} else {
		err := os.MkdirAll(outputDir, 0755)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	if indent <= 0 {
		FORMATTER_INDENT = 2
	} else {
		FORMATTER_INDENT = indent
	}

	if indentChar == "" {
		FORMATTER_CHAR = " "
	} else {
		if indentChar != "\t" || indentChar != " " {
			indentChar = " "
		}
		FORMATTER_CHAR = indentChar
	}
}

func main() {
	fmt.Printf("Nginx Formatter v%s\n\n", version.Version)

	InitArgv()

	err := updater.UpdateConfInDir(FORMATTER_SRC, FORMATTER_DEST, FORMATTER_INDENT, FORMATTER_CHAR, formatter.Formatter)
	if err != nil {
		log.Fatal(err)
	}
}
