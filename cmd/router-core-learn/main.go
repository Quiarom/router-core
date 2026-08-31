// Command router-core-learn is an EXPERIMENTAL probe that tests two
// evidence-backed candidate authentication recipes against the physical
// TP-Link TL-WR841N v8.4 stock firmware.
//
// This is not part of the runtime adapter. It does not modify
// internal/transport or internal/adapters. It exists to produce the
// first physical evidence needed to design Phase 3 correctly.
//
// Exit codes:
//
//	0  success; physical capture persisted and fingerprint matched
//	1  unexpected internal error
//	2  no candidate authentication recipe matched (protocol mismatch)
//	3  transport error (router unreachable, timeout, refused)
//	4  capture OK but physical fingerprint diverged (requires review)
//	5  invalid usage (bad flags, missing password, etc.)
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const probeVersion = "0.1.0"

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "router-core-learn:", err)
		os.Exit(exitCodeFor(err))
	}
}

// newRootCmd builds the Cobra command tree. Returns the root command
// so tests can introspect it.
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "router-core-learn",
		Short: "TP-Link WR841N physical capture probe",
		Long: `router-core-learn is an experimental probe that tests evidence-backed
legacy WR841N authentication recipes against a physical router at a
local RFC1918 address.

It is NOT part of the runtime adapter. It exists only to produce the
first physical evidence for Phase 3 design.

Run "router-core-learn learn --host 192.168.1.1" to start.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newLearnCmd())
	root.AddCommand(newObserveCmd())
	root.AddCommand(newProbeLoginPageCmd())
	root.AddCommand(newVersionCmd())

	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the probe version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "router-core-learn %s\n", probeVersion)
		},
	}
}

// exitCodeFor maps probe errors to the documented exit codes.
func exitCodeFor(err error) int {
	if err == nil {
		return 0
	}
	type coder interface{ ExitCode() int }
	if c, ok := err.(coder); ok {
		return c.ExitCode()
	}
	return 1
}
