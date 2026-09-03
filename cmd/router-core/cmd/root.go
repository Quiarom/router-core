// Package cmd holds the Cobra command tree for the router-core
// binary.
//
// CLI constitution (commit 1):
//   - Cobra owns parsing, help, version, completion.
//   - SilenceUsage: true so runtime errors don't print usage.
//   - SilenceErrors: true so the package owns error rendering.
//   - English help. Concise. Examples for each subcommand.
//   - stdout: results. stderr: prompts and diagnostics.
//   - No domain logic inside RunE. RunE only builds the typed
//     options struct and calls the application function in app.go.
package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags="-X main.version=...".
// Default is used when the binary is built without ldflags.
var version = "0.1.0-dev"

// ExitError carries a process exit code through cobra. It is
// returned by RunE and translated to os.Exit in main.go.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string { return e.Err.Error() }
func (e *ExitError) Unwrap() error { return e.Err }

// SetVersion overrides the version string. Used by main.go
// when it wants to share its own ldflags-injected version.
func SetVersion(v string) { version = v }

// NewRootCmd builds the router-core root command and wires all
// subcommands. Exported so main.go and tests can build a fresh
// tree per test.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "router-core",
		Short: "Local-first, read-only observation layer for legacy routers",
		Long: `router-core gives a legacy consumer router a typed local API
and an evidence-aware AI agent. No firmware replacement, no
cloud, no telemetry. Every observation is a fact the runtime
actually read; unknown is reported as unknown, never as false.`,
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		// No RunE: a bare `router-core` with no subcommand prints
		// help and exits 0. This is friendlier for new users.
	}
	root.SetVersionTemplate("router-core {{.Version}}\n")
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)

	root.AddCommand(newProbeCmd())
	root.AddCommand(newInspectCmd())
	root.AddCommand(newServeCmd())
	root.AddCommand(newVersionCmd())

	return root
}

// Execute is the main.go entry point.
func Execute() error {
	root := NewRootCmd()
	root.SetOut(os.Stdout)
	root.SetErr(io.Discard) // suppress cobra's duplicate error stream
	root.SilenceErrors = true
	root.SilenceUsage = true
	if err := root.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "router-core: %s\n", err.Error())
		return err
	}
	return nil
}

// ExecuteWithIO is the same as Execute but lets callers inject
// custom I/O (used by tests and by the runLegacy shim).
func ExecuteWithIO(out, errOut io.Writer) error {
	cmd := NewRootCmd()
	cmd.SetOut(out)
	cmd.SetErr(errOut)
	if err := cmd.Execute(); err != nil {
		// cobra did not print anything (SilenceErrors=true).
		// Render here with a consistent shape.
		fmt.Fprintf(errOut, "router-core: %s\n", err.Error())
		return err
	}
	return nil
}
