// Package cmd: stub for the ask subcommand. The implementation
// is added in subsequent commits; the stub exists today so the
// binary compiles and the root help shows the planned surface.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newAskCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ask",
		Short: "(stub) TODO",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.ErrOrStderr(), "gavetero: 'ask' is not implemented in this build")
			return nil
		},
	}
}
