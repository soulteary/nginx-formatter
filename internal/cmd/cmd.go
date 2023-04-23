package cmd

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/soulteary/nginx-formatter/internal/define"
)

func InitArgv() (argvSrc string, argvDest string, argvIndent int, argvIndentChar string) {
	var inputDir string
	var outputDir string
	var indent int
	var indentChar string
	flag.StringVar(&inputDir, define.APP_ARGV_INPUT, define.DEFAULT_WORKDIR, "Input directory")
	flag.StringVar(&outputDir, define.APP_ARGV_OUTPUT, define.DEFAULT_WORKDIR, "Output directory")
	flag.IntVar(&indent, define.APP_ARGV_INDENT, define.DEFAULT_INDENT_SIZE, fmt.Sprintf("Indent size, defualt: %d", define.DEFAULT_INDENT_SIZE))
	flag.StringVar(&indentChar, define.APP_ARGV_CHAR, define.DEFAULT_INDENT_CHAR, fmt.Sprintf("Indent char, defualt: `%s`", define.DEFAULT_INDENT_CHAR))
	flag.Parse()

	if inputDir == "" {
		dir, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("No input directory specified, use the current working directory:", dir)
		argvSrc = dir
	} else {
		fmt.Println("Specify the working directory as:", inputDir)
		argvSrc = inputDir
	}

	if outputDir == "" {
		dir, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("No output directory specified, use the current working directory:", dir)
		argvDest = dir
	} else {
		err := os.MkdirAll(outputDir, 0750)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("Specify the output directory as:", inputDir)
		argvDest = outputDir
	}

	if indent <= 0 {
		fmt.Println("No output indent size specified, use the default value:", define.DEFAULT_INDENT_SIZE)
		argvIndent = define.DEFAULT_INDENT_SIZE
	} else {
		fmt.Println("Specify the indent size as:", inputDir)
		argvIndent = indent
	}

	if indentChar == "" {
		argvIndentChar = define.DEFAULT_INDENT_CHAR
		fmt.Printf("No output indent char specified, use the default value: `%s`\n", define.DEFAULT_INDENT_CHAR)
	} else {
		if !(indentChar == "\t" || indentChar == " " || indentChar == "\\s") {
			indentChar = define.DEFAULT_INDENT_CHAR
			fmt.Printf("Specify the indent char not support, use the default value: `%s`\n", define.DEFAULT_INDENT_CHAR)
		}
		argvIndentChar = indentChar
		fmt.Printf("Specify the indent char as: `%s`\n", indentChar)
	}
	fmt.Println()
	return argvSrc, argvDest, argvIndent, argvIndentChar
}
