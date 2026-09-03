package cmd

import (
	"github.com/spf13/cobra"
)

// newServeCmd builds the `router-core serve` subcommand.
// It starts a typed HTTP API on the loopback interface that
// exposes /v0/device, /v0/status, /v0/clients, /v0/capabilities,
// and /v0/security/<name> to AI agents and the dashboard.
func newServeCmd() *cobra.Command {
	var host string
	var addr string
	var timeoutMS int
	var mock bool
	var mockFixture string
	var passwordStdin bool

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Expose a typed read-only HTTP API on the loopback interface",
		Long: `Start the router-core HTTP server on a loopback address. The
server is read-only: it never mutates the router, and an
architecture test enforces no POST/PUT/DELETE in the runtime
path.

By default the runtime connects to a real router. With
--mock, it loads a fixture-backed adapter and never touches
the network, so you can develop without hardware.`,
		Example: `  # serve a real WR841N (prompts for admin password on the TTY)
  router-core serve --host 192.168.1.1

  # serve a fixture-backed adapter (no hardware, no key)
  router-core serve --mock

  # non-interactive password input (CI, systemd, container secrets)
  router-core serve --host 192.168.1.1 --password-stdin < secret.txt`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := serveOptions{
				Host:          host,
				Addr:          addr,
				Timeout:       timeoutFromMS(timeoutMS),
				Mock:          mock,
				MockFixture:   mockFixture,
				PasswordStdin: passwordStdin,
			}
			return runServe(opts)
		},
	}

	cmd.Flags().StringVar(&host, "host", "192.168.0.1",
		"local router address (RFC1918 literal)")
	cmd.Flags().StringVar(&addr, "addr", "127.0.0.1:8484",
		"loopback HTTP listen address")
	cmd.Flags().IntVar(&timeoutMS, "timeout-ms", 5000,
		"per-request timeout in milliseconds (real router only)")
	cmd.Flags().BoolVar(&mock, "mock", false,
		"run against a fixture-backed adapter (no network)")
	cmd.Flags().StringVar(&mockFixture, "mock-fixture", "",
		"path to a synthetic fixture (default: fixtures/synthetic/tplink-wr841n-v8)")
	cmd.Flags().BoolVar(&passwordStdin, "password-stdin", false,
		"read the admin password from stdin (refuses if stdin is a TTY)")

	return cmd
}
