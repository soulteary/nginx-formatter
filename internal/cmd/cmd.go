package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/soulteary/nginx-formatter/internal/checker"
	"github.com/soulteary/nginx-formatter/internal/define"
	"github.com/soulteary/nginx-formatter/internal/formatter"
	"github.com/soulteary/nginx-formatter/internal/server"
	"github.com/soulteary/nginx-formatter/internal/updater"
	"github.com/soulteary/nginx-formatter/internal/version"
	"github.com/spf13/cobra"
)

// resolveOutputDefault decides the effective output value when the output
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

// resolveIndentChar normalizes and validates the indent char, falling back to
// the default when unsupported. It prints an informational message describing
// the effective choice.
func resolveIndentChar(indentChar string) string {
	if indentChar == "" {
		fmt.Printf("No output indent char specified, use the default value: `%s`\n", define.DISPLAY_INDENT_CHARS[define.DEFAULT_INDENT_CHAR])
		return define.DEFAULT_INDENT_CHAR
	}

	switch indentChar {
	case "space":
		indentChar = " "
	case "tab":
		indentChar = "\t"
	}

	if indentChar != "\t" && indentChar != " " && indentChar != "\\s" {
		fmt.Printf("Specify the indent char not support, use the default value: `%s`\n", define.DISPLAY_INDENT_CHARS[define.DEFAULT_INDENT_CHAR])
		indentChar = define.DEFAULT_INDENT_CHAR
	}

	if display, ok := define.DISPLAY_INDENT_CHARS[indentChar]; ok {
		fmt.Printf("Specify the indent char as: `%s`\n", display)
	} else {
		fmt.Printf("Specify the indent char as: `%s`\n", indentChar)
	}
	return indentChar
}

// resolveIndent normalizes the indent size, falling back to the default when
// a non-positive value is provided.
func resolveIndent(indent int) int {
	if indent <= 0 {
		fmt.Println("No output indent size specified, use the default value:", define.DEFAULT_INDENT_SIZE)
		return define.DEFAULT_INDENT_SIZE
	}
	fmt.Println("Specify the indent size as:", indent)
	return indent
}

// resolvePort validates the WebUI port, falling back to the default when it is
// out of the accepted range.
func resolvePort(port int) int {
	if port <= 1024 || port >= 65535 {
		fmt.Println("Please set the port above 1024 and the port within 65535")
		fmt.Printf("use the default value: `%d`\n", define.DEFAULT_PORT)
		return define.DEFAULT_PORT
	}
	return port
}

// runFormat resolves inputs and formats the target file or directory, reusing
// the existing updater logic.
func runFormat(input string, output string, indent int, indentChar string) error {
	var src string
	if input == "" {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}
		fmt.Println("No input directory specified, use the current working directory:", dir)
		src = dir
	} else {
		fmt.Println("Specify the working directory as:", input)
		src = input
	}

	dest, err := resolveOutputDefault(src, output)
	if err != nil {
		return err
	}
	if output == "" {
		if dest == "" {
			fmt.Println("No output specified, will overwrite the input file in place")
		} else {
			fmt.Println("No output directory specified, use the current working directory:", dest)
		}
	} else {
		fmt.Println("Specify the output directory as:", output)
	}

	indent = resolveIndent(indent)
	indentChar = resolveIndentChar(indentChar)
	fmt.Println()

	checker.InDockerAndWorkDirIsRoot(src)

	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return updater.UpdateConfInDir(src, dest, indent, indentChar, formatter.Formatter)
	}
	return updater.UpdateConfFile(src, dest, indent, indentChar, formatter.Formatter)
}

// runServe launches the WebUI, reusing the existing server logic.
func runServe(port int, indent int, indentChar string) error {
	indent = resolveIndent(indent)
	indentChar = resolveIndentChar(indentChar)
	port = resolvePort(port)
	fmt.Printf("Enable WebUI, please visit http://localhost:%d\n", port)
	fmt.Println()

	return server.Launch(port, indent, indentChar, formatter.Formatter)
}

// newRootCmd builds the root command, mounts the semantic subcommands, and
// keeps the legacy single-dash long flags as hidden compatibility flags.
func newRootCmd() *cobra.Command {
	var (
		legacyInput  string
		legacyOutput string
		legacyIndent int
		legacyChar   string
		legacyWeb    bool
		legacyPort   int
	)

	rootCmd := &cobra.Command{
		Use:   "nginx-formatter",
		Short: "A small, fast Nginx configuration formatter with CLI and WebUI",
		Long: "Nginx Formatter is a small, fast Nginx configuration formatter.\n" +
			"It supports formatting a directory or a single file (CLI) and a browser-based WebUI.",
		Version:       version.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		Example: `  # Format all .conf files in the current directory
  nginx-formatter format

  # Start the WebUI
  nginx-formatter serve

  # Print version
  nginx-formatter version`,
		// Print the startup banner for every command except `version`,
		// whose output already carries the version number.
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if cmd.Name() != "version" {
				fmt.Printf("Nginx Formatter %s\n\n", version.Version)
			}
		},
		// The root command keeps backward compatibility with the legacy
		// flags: when -web is set it routes to serve logic, otherwise it
		// runs format logic (including the legacy -input/-output/... flags).
		RunE: func(cmd *cobra.Command, args []string) error {
			if legacyWeb {
				return runServe(legacyPort, legacyIndent, legacyChar)
			}
			return runFormat(legacyInput, legacyOutput, legacyIndent, legacyChar)
		},
	}

	flags := rootCmd.Flags()
	flags.StringVar(&legacyInput, define.APP_ARGV_INPUT, define.DEFAULT_WORKDIR, "Input directory or file (legacy)")
	flags.StringVar(&legacyOutput, define.APP_ARGV_OUTPUT, define.DEFAULT_WORKDIR, "Output directory or file (legacy)")
	flags.IntVar(&legacyIndent, define.APP_ARGV_INDENT, define.DEFAULT_INDENT_SIZE, "Indent size (legacy)")
	flags.StringVar(&legacyChar, define.APP_ARGV_CHAR, define.DEFAULT_INDENT_CHAR, "Indent char (legacy)")
	flags.BoolVar(&legacyWeb, define.APP_ARGV_WEB, define.DEFAULT_WEB, "Enable WebUI (legacy)")
	flags.IntVar(&legacyPort, define.APP_ARGV_PORT, define.DEFAULT_PORT, "WebUI port (legacy)")

	for _, name := range []string{
		define.APP_ARGV_INPUT, define.APP_ARGV_OUTPUT, define.APP_ARGV_INDENT,
		define.APP_ARGV_CHAR, define.APP_ARGV_WEB, define.APP_ARGV_PORT,
	} {
		_ = flags.MarkHidden(name)
	}

	rootCmd.AddCommand(newFormatCmd(), newServeCmd(), newVersionCmd())
	return rootCmd
}

// legacyFlags are the single-dash long flags kept for backward compatibility.
var legacyFlags = map[string]struct{}{
	define.APP_ARGV_INPUT:  {},
	define.APP_ARGV_OUTPUT: {},
	define.APP_ARGV_INDENT: {},
	define.APP_ARGV_CHAR:   {},
	define.APP_ARGV_WEB:    {},
	define.APP_ARGV_PORT:   {},
}

// normalizeLegacyArgs rewrites legacy single-dash long flags (e.g. `-input`,
// `-input=/app`) into their pflag-compatible double-dash form (`--input`).
// pflag only recognizes long flags with a double dash, so this shim preserves
// backward compatibility with old scripts and Docker usage.
func normalizeLegacyArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		if len(arg) > 1 && arg[0] == '-' && arg[1] != '-' {
			body := arg[1:]
			name := body
			if idx := strings.IndexByte(body, '='); idx >= 0 {
				name = body[:idx]
			}
			if _, ok := legacyFlags[name]; ok {
				out = append(out, "-"+arg)
				continue
			}
		}
		out = append(out, arg)
	}
	return out
}

// Execute builds and runs the root command, applying the legacy-flag shim.
func Execute() error {
	root := newRootCmd()
	root.SetArgs(normalizeLegacyArgs(os.Args[1:]))
	return root.Execute()
}
