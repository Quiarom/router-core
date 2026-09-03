// bridge.go: the bridge between the cmd subpackage and the
// runtime functions that live in package main.
//
// The cmd subpackage owns Cobra. The HTTP server setup and
// password read live in package main (serve.go). To avoid a
// circular import (Go forbids importing the parent package
// from a subpackage), the cmd subpackage declares typed
// argument structs and a runtime setter that package main
// fills at startup time. This is the "register a callback"
// pattern.
package cmd

import (
	"fmt"
	"time"
)

// ServeArgs is the typed argument struct for `router-core serve`.
// It is exported (capitalized) because package main references
// it when it calls SetServeRunner.
type ServeArgs struct {
	Host          string
	Addr          string
	Timeout       time.Duration
	Mock          bool
	MockFixture   string
	PasswordStdin bool
}

// runServeFn is the function signature package main registers
// at startup via SetServeRunner.
type runServeFn func(ServeArgs) error

// serveRunner is the registered serve runner. nil means "serve
// is not wired yet"; runServe returns an error in that case.
var serveRunner runServeFn

// SetServeRunner is called by package main during init() to
// register the actual runServeCommand bridge.
func SetServeRunner(fn runServeFn) { serveRunner = fn }

// runServe dispatches to the registered runner. It is called
// from serve.go's RunE.
func runServe(opts serveOptions) error {
	if serveRunner == nil {
		return &ExitError{
			Code: 1,
			Err:  fmt.Errorf("serve subcommand is not wired in this build"),
		}
	}
	return serveRunner(ServeArgs{
		Host:          opts.Host,
		Addr:          opts.Addr,
		Timeout:       opts.Timeout,
		Mock:          opts.Mock,
		MockFixture:   opts.MockFixture,
		PasswordStdin: opts.PasswordStdin,
	})
}
