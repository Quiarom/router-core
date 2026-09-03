// Command router-core is the runtime CLI for local-first,
// read-only observation of legacy consumer routers.
//
// The CLI is built on Cobra. The command tree is defined in
// the cmd/ subpackage; this file is the thin process entrypoint
// and the bridge to the legacy runtime functions in serve.go.
package main

import (
	"errors"
	"os"

	"github.com/Quiarom/router-core/cmd/router-core/cmd"
)

func init() {
	// Register the serve runner so the cmd subpackage can call
	// into the legacy runServeCommand. This is the only place
	// where the cmd subpackage touches package main; the rest
	// of the application logic is in cmd/app.go.
	cmd.SetServeRunner(serveBridge)
}

// serveBridge is the function the cmd subpackage calls when the
// user runs `router-core serve`. It reconstructs the legacy
// os.Args-style slice and calls runServeCommand. The legacy
// function in serve.go is unchanged.
func serveBridge(args cmd.ServeArgs) error {
	argv := []string{"serve"}
	if args.Host != "" {
		argv = append(argv, "--host", args.Host)
	}
	if args.Addr != "" {
		argv = append(argv, "--addr", args.Addr)
	}
	if args.Timeout > 0 {
		argv = append(argv, "--timeout", args.Timeout.String())
	}
	if args.Mock {
		argv = append(argv, "--mock")
	}
	if args.MockFixture != "" {
		argv = append(argv, "--mock-fixture", args.MockFixture)
	}
	if args.PasswordStdin {
		argv = append(argv, "--password-stdin")
	}
	return runServeCommand(argv)
}

func main() {
	if err := cmd.Execute(); err != nil {
		// cmd.Execute() already rendered the error to stderr
		// in the canonical "router-core: <msg>" shape. We just
		// pick the exit code here. A *cmd.ExitError carries
		// a specific code; anything else is exit 1.
		var exitErr *cmd.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
		os.Exit(1)
	}
}

// run is the legacy shim kept for the existing golden test in
// main_test.go. It routes through the Cobra command tree so
// the same tests exercise the new path.
func run(args []string) error {
	root := cmd.NewRootCmd()
	root.SetArgs(args)
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	return root.Execute()
}
