package cmd

import (
	"fmt"

	"github.com/soulteary/nginx-formatter/internal/define"
	"github.com/soulteary/nginx-formatter/internal/updater"
	"github.com/spf13/cobra"
)

// newFormatCmd builds the `format` subcommand, the canonical way to format
// Nginx configuration files or directories.
func newFormatCmd() *cobra.Command {
	var (
		input      string
		output     string
		indent     int
		indentChar string
		check      bool
		diff       bool
	)

	cmd := &cobra.Command{
		Use:   "format",
		Short: "Format Nginx configuration files in a directory or a single file",
		Long: "Format Nginx configuration files.\n\n" +
			"When --input points to a directory, every .conf file inside is formatted,\n" +
			"plus sites-enabled/default and sites-available/default (the Debian and\n" +
			"Ubuntu site file, which has no extension); symbolic links are reported\n" +
			"and skipped rather than followed.\n" +
			"When --input points to a file, only that file is formatted (any extension).\n\n" +
			"With --output empty, both modes format in place.\n\n" +
			"--input - reads the configuration from stdin and writes the result to stdout.\n\n" +
			"--check and --diff never write: they report what would change and exit 1 if\n" +
			"anything would, which is what a CI job wants.\n\n" +
			"The --output value has three meanings in single-file mode:\n" +
			"  empty            overwrite the input file in place\n" +
			"  existing dir     write to <output-dir>/<original-file-name>\n" +
			"  other            treat as a target file path (parent dir created if needed)",
		Example: `  # Format all .conf files in the current directory
  nginx-formatter format

  # A single positional path means the same as --input
  nginx-formatter format ./conf.d

  # Format a specific directory and write to a new directory
  nginx-formatter format -i ./conf.d -o ./dist

  # Single file: overwrite in place / write to a dir / write to a file
  nginx-formatter format -i ./nginx.conf
  nginx-formatter format -i ./nginx.conf -o ./dist
  nginx-formatter format -i ./nginx.conf -o ./dist/nginx.formatted.conf

  # Use 4-space indentation
  nginx-formatter format -i ./conf.d -n 4 -c space

  # CI: fail if anything is not formatted, without touching the tree
  nginx-formatter format -i ./conf.d --check
  nginx-formatter format -i ./conf.d --diff

  # Editor format-on-save: stdin to stdout
  cat nginx.conf | nginx-formatter format -i -`,
		SilenceUsage:  true,
		SilenceErrors: true,
		// One positional path is accepted and means the same as --input, the
		// shape gofmt/prettier/black users reach for. It used to be parsed and
		// then silently dropped, so `nginx-formatter format /etc/nginx` walked
		// away and reformatted the *working directory* instead, exit code 0.
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Resolve the positional path first: it is what --check and
			// --diff will be asked about.
			if len(args) == 1 {
				if input != "" {
					return fmt.Errorf("cannot use both --input %q and the positional path %q", input, args[0])
				}
				input = args[0]
			}
			if check && diff {
				return fmt.Errorf("--check and --diff cannot be combined; --diff already reports what would change")
			}
			mode := updater.ModeWrite
			switch {
			case diff:
				mode = updater.ModeDiff
			case check:
				mode = updater.ModeCheck
			}
			return runFormatMode(input, output, indent, indentChar, mode)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&input, "input", "i", define.DEFAULT_WORKDIR, "Input directory or file (default: current directory)")
	flags.StringVarP(&output, "output", "o", define.DEFAULT_WORKDIR, "Output directory or file path")
	flags.IntVarP(&indent, "indent", "n", define.DEFAULT_INDENT_SIZE, "Indent size")
	flags.StringVarP(&indentChar, "char", "c", define.DEFAULT_INDENT_CHAR, "Indent char (space/tab/`\\s`/`\\t`)")
	flags.BoolVar(&check, "check", false, "Do not write; list files that are not formatted and exit 1 if any")
	flags.BoolVar(&diff, "diff", false, "Do not write; print a unified diff of what would change and exit 1 if any")

	return cmd
}
