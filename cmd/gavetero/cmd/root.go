// Package cmd holds the Cobra command tree for the gavetero
// user-facing CLI.
//
// CLI constitution (carried from router-core/cmd):
//   - Cobra owns parsing, help, version, completion.
//   - SilenceUsage + SilenceErrors so the runtime owns error rendering.
//   - English help. Concise. Examples for each subcommand.
//   - results to stdout; prompts and diagnostics to stderr.
//   - no domain logic inside RunE. RunE delegates to app.go.
package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags="-X main.version=...".
var version = "0.1.0-dev"

// SetVersion overrides the version string. Called from main.go
// when ldflags inject a different value.
func SetVersion(v string) { version = v }

// NewRootCmd builds the gavetero root command and wires all
// subcommands. Exported so main.go and tests can build a fresh
// tree per test.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "gavetero",
		Aliases: []string{"gvt"},
		Short:   "Understand your home network, with the help of an LLM",
		Long: `gavetero turns a legacy consumer router into typed local observations
that you (or an AI agent) can investigate.

The runtime is read-only: it never changes the router. The
model decides what to inspect; the local runtime decides which
operations exist. Unknown is reported as unknown, never as
false.`,
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetVersionTemplate("gavetero {{.Version}}\n")
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)

	root.AddCommand(newVersionCmd())
	root.AddCommand(newSetupCmd())
	root.AddCommand(newAskCmd())
	root.AddCommand(newInspectCmd())
	root.AddCommand(newDoctorCmd())
	root.AddCommand(newIntegrationsCmd())

	return root
}

// ExecuteWithIO is the same as Execute but lets callers inject
// custom I/O and the args slice. Used by tests.
func ExecuteWithIO(out, errOut io.Writer, args ...string) error {
	root := NewRootCmd()
	root.SetOut(out)
	root.SetErr(io.Discard)
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		fmt.Fprintf(errOut, "gavetero: %s\n", err.Error())
		return err
	}
	return nil
}

// Execute is the main.go entry point.
func Execute() error {
	root := NewRootCmd()
	root.SetErr(io.Discard) // suppress cobra's duplicate error stream
	root.SilenceErrors = true
	root.SilenceUsage = true
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "gavetero: %s\n", err.Error())
		return err
	}
	return nil
}
