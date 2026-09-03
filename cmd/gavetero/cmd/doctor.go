// Package cmd: stub for the doctor subcommand. The implementation
// is added in subsequent commits; the stub exists today so the
// binary compiles and the root help shows the planned surface.
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "(stub) TODO",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.ErrOrStderr(), "gavetero: 'doctor' is not implemented in this build")
			return nil
		},
	}
}
