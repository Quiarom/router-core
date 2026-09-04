package cmd

import (
	"github.com/spf13/cobra"
)

// newInspectCmd builds the `router-core inspect` subcommand.
func newInspectCmd() *cobra.Command {
	var host string
	var fixtures string
	var jsonOut bool
	var timeoutMS int

	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Print status, clients, and security observations",
		Long: `Read the router's /v0/status, /v0/clients, and the
per-capability security matrix, then print the result as
plain text (default) or JSON (with --json).

Capabilities the runtime cannot observe are reported as
` + "`unsupported_or_unverified`" + `, never silently as
` + "`false`" + ` or ` + "`disabled`" + `.`,
		Example: `  # inspect a real router (will prompt for admin password)
  router-core inspect --host 192.168.1.1

  # inspect a fixture (no prompt, no network)
  router-core inspect --fixtures fixtures/synthetic/tplink-wr841n-v8

  # JSON output for piping into jq
  router-core inspect --fixtures fixtures/synthetic/tplink-wr841n-v8 --json | jq .`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := inspectOptions{
				Host:     host,
				Fixtures: fixtures,
				JSON:     jsonOut,
				Timeout:  timeoutFromMS(timeoutMS),
			}
			return runInspect(opts)
		},
	}

	cmd.Flags().StringVar(&host, "host", "192.168.0.1",
		"local router address (RFC1918 literal)")
	cmd.Flags().StringVar(&fixtures, "fixtures", "",
		"path to a synthetic fixture; bypasses real router auth")
	cmd.Flags().BoolVar(&jsonOut, "json", false,
		"write JSON to stdout")
	cmd.Flags().IntVar(&timeoutMS, "timeout-ms", 5000,
		"per-request timeout in milliseconds (real router only)")

	return cmd
}
