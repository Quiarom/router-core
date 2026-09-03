// Package cmd: the version subcommand. It prints the version
// string set at build time via -ldflags. The same pattern as
// router-core version.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "gavetero %s\n", version)
			return nil
		},
	}
}
