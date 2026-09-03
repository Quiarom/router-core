package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newVersionCmd builds the `router-core version` subcommand.
// It prints the version string set at build time via ldflags.
func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "router-core", version)
			return nil
		},
	}
}
