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

const (
	DEFAULT_INDENT_SIZE = 2
	DEFAULT_INDENT_CHAR = " "
)

var FORMATTER_SRC = ""
var FORMATTER_DEST = ""
var FORMATTER_INDENT = DEFAULT_INDENT_SIZE
var FORMATTER_CHAR = DEFAULT_INDENT_CHAR

func InitArgv() {
	var inputDir string
	var outputDir string
	var indent int
	var indentChar string
	flag.StringVar(&inputDir, "input", "", "Input directory")
	flag.StringVar(&outputDir, "output", "", "Output directory")
	flag.IntVar(&indent, "indent", DEFAULT_INDENT_SIZE, fmt.Sprintf("Indent size, defualt: %d", DEFAULT_INDENT_SIZE))
	flag.StringVar(&indentChar, "char", DEFAULT_INDENT_CHAR, fmt.Sprintf("Indent char, defualt: `%s`", DEFAULT_INDENT_CHAR))
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
		err := os.MkdirAll(outputDir, 0750)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("Specify the output directory as:", inputDir)
		FORMATTER_DEST = outputDir
	}

	if indent <= 0 {
		fmt.Println("No output indent size specified, use the default value:", DEFAULT_INDENT_SIZE)
		FORMATTER_INDENT = DEFAULT_INDENT_SIZE
	} else {
		fmt.Println("Specify the indent size as:", inputDir)
		FORMATTER_INDENT = indent
	}

	if indentChar == "" {
		FORMATTER_CHAR = DEFAULT_INDENT_CHAR
		fmt.Printf("No output indent char specified, use the default value: `%s`\n", FORMATTER_CHAR)
	} else {
		if !(indentChar == "\t" || indentChar == " " || indentChar == "\\s") {
			indentChar = DEFAULT_INDENT_CHAR
			fmt.Printf("Specify the indent char not support, use the default value: `%s`\n", DEFAULT_INDENT_CHAR)
		}
		FORMATTER_CHAR = indentChar
		fmt.Printf("Specify the indent char as: `%s`\n", FORMATTER_CHAR)
	}
	fmt.Println()
}

func main() {
	fmt.Printf("Nginx Formatter v%s\n\n", version.Version)

	InitArgv()

	err := updater.UpdateConfInDir(FORMATTER_SRC, FORMATTER_DEST, FORMATTER_INDENT, FORMATTER_CHAR, formatter.Formatter)
	if err != nil {
		log.Fatal(err)
	}
}
