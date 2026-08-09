package cmd

import (
	"github.com/soulteary/nginx-formatter/internal/define"
	"github.com/spf13/cobra"
)

// newServeCmd builds the `serve` subcommand, the canonical replacement for the
// legacy -web flag.
func newServeCmd() *cobra.Command {
	var (
		port       int
		indent     int
		indentChar string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start the browser-based WebUI",
		Long: "Start the browser-based WebUI so you can format Nginx configuration in your browser.\n\n" +
			"--indent and --char set the default indentation the WebUI applies when formatting.",
		Example: `  # Start the WebUI on the default port
  nginx-formatter serve

  # Start the WebUI on a custom port
  nginx-formatter serve -p 8123`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServe(port, indent, indentChar)
		},
	}

	flags := cmd.Flags()
	flags.IntVarP(&port, "port", "p", define.DEFAULT_PORT, "WebUI port")
	flags.IntVarP(&indent, "indent", "n", define.DEFAULT_INDENT_SIZE, "Default indent size the WebUI applies")
	flags.StringVarP(&indentChar, "char", "c", define.DEFAULT_INDENT_CHAR, "Default indent char the WebUI applies (space/tab/`\\s`/`\\t`)")

	return cmd
}
