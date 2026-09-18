package cmd

import (
	"fmt"
	"net"
	"os"
	"strconv"
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
//   - output == "" and src is a dir   -> returns src so the directory is formatted in place
//
// Directory mode used to fall back to the working directory, which meant
// `format -i ./conf.d` left ./conf.d untouched and instead wrote a copy of the
// tree into the caller's cwd, silently overwriting any same-named file there.
// Both modes now mean the same thing when no output is given: format where the
// input is.
func resolveOutputDefault(src string, output string) (string, error) {
	if output != "" {
		return output, nil
	}

	if info, err := os.Stat(src); err == nil && !info.IsDir() {
		return "", nil
	}

	return src, nil
}

// resolveIndentChar normalizes and validates the indent char, falling back to
// the default when unsupported. It prints an informational message describing
// the effective choice.
func resolveIndentChar(indentChar string) string {
	if indentChar == "" {
		fmt.Printf("No output indent char specified, use the default value: `%s`\n", define.DISPLAY_INDENT_CHARS[define.DEFAULT_INDENT_CHAR])
		return define.DEFAULT_INDENT_CHAR
	}

	// Normalize the documented spellings to the real character before
	// validating. "\\s" and "\\t" are the two-character escape forms the README
	// and --help advertise; leaving them un-normalized would write the literal
	// text (e.g. "\\s\\s") into the config as indentation.
	switch indentChar {
	case "space", "\\s":
		indentChar = " "
	case "tab", "\\t":
		indentChar = "\t"
	}

	if indentChar != "\t" && indentChar != " " {
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

// minPort is the lowest port the WebUI will bind. Ports at or below 1024 are
// privileged on Unix, and this tool has no business asking for them.
const minPort = 1025

// maxPort is the highest port number there is.
const maxPort = 65535

// resolvePort validates the WebUI port, falling back to the default when it is
// out of the accepted range.
func resolvePort(port int) int {
	// The guard used to read "port >= 65535", which rejected 65535 itself even
	// though the message promised everything "within 65535".
	if port < minPort || port > maxPort {
		fmt.Printf("Please set the port between %d and %d\n", minPort, maxPort)
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
			fmt.Println("No output directory specified, will format the input directory in place:", dest)
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
func runServe(host string, port int, indent int, indentChar string) error {
	indent = resolveIndent(indent)
	indentChar = resolveIndentChar(indentChar)
	port = resolvePort(port)

	// Report the address actually bound. An empty host means every interface,
	// so saying "localhost" there would understate the exposure.
	if host == "" {
		fmt.Printf("Enable WebUI on all interfaces, please visit http://localhost:%d\n", port)
	} else {
		fmt.Printf("Enable WebUI, please visit http://%s\n", net.JoinHostPort(host, strconv.Itoa(port)))
	}
	fmt.Println()

	return server.Launch(host, port, indent, indentChar, formatter.Formatter)
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
		legacyHost   string
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
				return runServe(legacyHost, legacyPort, legacyIndent, legacyChar)
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
	flags.StringVar(&legacyHost, define.APP_ARGV_HOST, define.DEFAULT_HOST, "Address to bind (legacy)")

	for _, name := range []string{
		define.APP_ARGV_INPUT, define.APP_ARGV_OUTPUT, define.APP_ARGV_INDENT,
		define.APP_ARGV_CHAR, define.APP_ARGV_WEB, define.APP_ARGV_PORT,
		define.APP_ARGV_HOST,
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
	define.APP_ARGV_HOST:   {},
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
