// Package cmd: stub for the setup subcommand. The implementation
// is added in subsequent commits; the stub exists today so the
// binary compiles and the root help shows the planned surface.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "(stub) TODO",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.ErrOrStderr(), "gavetero: 'setup' is not implemented in this build")
			return nil
		},
	}
}
