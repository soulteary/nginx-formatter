package cmd

import (
	"fmt"

	"github.com/soulteary/nginx-formatter/internal/version"
	"github.com/spf13/cobra"
)

// newVersionCmd builds the `version` subcommand, replacing the version banner
// that used to be printed unconditionally.
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Example: `  # Print the version
  nginx-formatter version`,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("Nginx Formatter %s\n", version.Version)
			return nil
		},
	}
}
