package cmd

import (
	"github.com/soulteary/nginx-formatter/internal/define"
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
	)

	cmd := &cobra.Command{
		Use:   "format",
		Short: "Format Nginx configuration files in a directory or a single file",
		Long: "Format Nginx configuration files.\n\n" +
			"When --input points to a directory, every .conf file inside is formatted.\n" +
			"When --input points to a file, only that file is formatted (any extension).\n\n" +
			"The --output value has three meanings in single-file mode:\n" +
			"  empty            overwrite the input file in place\n" +
			"  existing dir     write to <output-dir>/<original-file-name>\n" +
			"  other            treat as a target file path (parent dir created if needed)",
		Example: `  # Format all .conf files in the current directory
  nginx-formatter format

  # Format a specific directory and write to a new directory
  nginx-formatter format -i ./conf.d -o ./dist

  # Single file: overwrite in place / write to a dir / write to a file
  nginx-formatter format -i ./nginx.conf
  nginx-formatter format -i ./nginx.conf -o ./dist
  nginx-formatter format -i ./nginx.conf -o ./dist/nginx.formatted.conf

  # Use 4-space indentation
  nginx-formatter format -i ./conf.d -n 4 -c space`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFormat(input, output, indent, indentChar)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&input, "input", "i", define.DEFAULT_WORKDIR, "Input directory or file (default: current directory)")
	flags.StringVarP(&output, "output", "o", define.DEFAULT_WORKDIR, "Output directory or file path")
	flags.IntVarP(&indent, "indent", "n", define.DEFAULT_INDENT_SIZE, "Indent size")
	flags.StringVarP(&indentChar, "char", "c", define.DEFAULT_INDENT_CHAR, "Indent char (space/tab/`\\s`/`\\t`)")

	return cmd
}
