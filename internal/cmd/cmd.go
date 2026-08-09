package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/soulteary/nginx-formatter/internal/checker"
	"github.com/soulteary/nginx-formatter/internal/define"
)

// resolveOutputDefault decides the effective output value when the -output
// flag is empty:
//   - output != ""                    -> returned as-is
//   - output == "" and src is a file  -> returns "" so UpdateConfFile overwrites in place
//   - output == "" and src is a dir   -> falls back to the current working directory
func resolveOutputDefault(src string, output string) (string, error) {
	if output != "" {
		return output, nil
	}

	if info, err := os.Stat(src); err == nil && !info.IsDir() {
		return "", nil
	}

	return os.Getwd()
}

func InitArgv() (argvSrc string, argvDest string, argvIndent int, argvIndentChar string, argvWeb bool, argvPort int) {
	var inputDir string
	flag.StringVar(&inputDir, define.APP_ARGV_INPUT, define.DEFAULT_WORKDIR, "Input directory")
	var outputDir string
	flag.StringVar(&outputDir, define.APP_ARGV_OUTPUT, define.DEFAULT_WORKDIR, "Output directory")
	var indent int
	flag.IntVar(&indent, define.APP_ARGV_INDENT, define.DEFAULT_INDENT_SIZE, fmt.Sprintf("Indent size, defualt: %d", define.DEFAULT_INDENT_SIZE))
	var indentChar string
	flag.StringVar(&indentChar, define.APP_ARGV_CHAR, define.DEFAULT_INDENT_CHAR, fmt.Sprintf("Indent char, defualt: `%s`", define.DEFAULT_INDENT_CHAR))

	var web bool
	flag.BoolVar(&web, define.APP_ARGV_WEB, define.DEFAULT_WEB, fmt.Sprintf("Enable WebUI, defualt: `%v`", define.DEFAULT_WEB))
	var port int
	flag.IntVar(&port, define.APP_ARGV_PORT, define.DEFAULT_PORT, fmt.Sprintf("WebUI Port, defualt: `%d`", define.DEFAULT_PORT))

	flag.Parse()

	if inputDir == "" {
		dir, err := os.Getwd()
		checker.FailToRun(err)
		fmt.Println("No input directory specified, use the current working directory:", dir)
		argvSrc = dir
	} else {
		fmt.Println("Specify the working directory as:", inputDir)
		argvSrc = inputDir
	}

	if outputDir == "" {
		dest, err := resolveOutputDefault(argvSrc, outputDir)
		checker.FailToRun(err)
		if dest == "" {
			fmt.Println("No output specified, will overwrite the input file in place")
		} else {
			fmt.Println("No output directory specified, use the current working directory:", dest)
		}
		argvDest = dest
	} else {
		fmt.Println("Specify the output directory as:", outputDir)
		argvDest = outputDir
	}

	if indent <= 0 {
		fmt.Println("No output indent size specified, use the default value:", define.DEFAULT_INDENT_SIZE)
		argvIndent = define.DEFAULT_INDENT_SIZE
	} else {
		fmt.Println("Specify the indent size as:", indent)
		argvIndent = indent
	}

	if indentChar == "" {
		argvIndentChar = define.DEFAULT_INDENT_CHAR
		fmt.Printf("No output indent char specified, use the default value: `%s`\n", define.DISPLAY_INDENT_CHARS[define.DEFAULT_INDENT_CHAR])
	} else {
		if indentChar != "\t" && indentChar != " " && indentChar != "\\s" {
			indentChar = define.DEFAULT_INDENT_CHAR
			fmt.Printf("Specify the indent char not support, use the default value: `%s`\n", define.DISPLAY_INDENT_CHARS[define.DEFAULT_INDENT_CHAR])
		}
		argvIndentChar = indentChar
		display, ok := define.DISPLAY_INDENT_CHARS[indentChar]
		if ok {
			fmt.Printf("Specify the indent char as: `%s`\n", display)
		} else {
			fmt.Printf("Specify the indent char as: `%s`\n", indentChar)
		}
	}

	if web {
		argvWeb = true
		if port <= 1024 || port >= 65535 {
			fmt.Println("Please set the port above 1024 and the port within 65535")
			fmt.Printf("use the default value: `%d`\n", define.DEFAULT_PORT)
			argvPort = define.DEFAULT_PORT
		} else {
			argvPort = port
			fmt.Printf("Specify the indent char as: `%d`\n", port)
		}
		fmt.Printf("Enable WebUI, please visit http://localhost:%d\n", port)
	} else {
		argvWeb = false
	}

	fmt.Println()
	return argvSrc, argvDest, argvIndent, argvIndentChar, argvWeb, argvPort
}
