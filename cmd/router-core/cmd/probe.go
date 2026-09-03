package cmd

import (
	"github.com/spf13/cobra"
)

// newProbeCmd builds the `router-core probe` subcommand.
// It delegates to probeApplication in app.go (no Cobra logic
// inside RunE: RunE only wires flags into the application
// function).
func newProbeCmd() *cobra.Command {
	var host string
	var fixtures string
	var jsonOut bool
	var timeoutMS int

	cmd := &cobra.Command{
		Use:   "probe",
		Short: "Print the router's device identity (vendor, model, firmware)",
		Long: `Read the router's /v0/device endpoint and print the
result as plain text (default) or as JSON (with --json).

If --fixtures is set, the runtime loads a fixture-backed
adapter and never touches the network. Otherwise it expects
the admin password on stdin (or via --password-stdin).`,
		Example: `  # replay against a sanitized fixture (no hardware, no key)
  router-core probe --fixtures fixtures/synthetic/tplink-wr841n-v8

  # talk to a real TP-Link WR841N on 192.168.1.1
  router-core probe --host 192.168.1.1

  # machine-readable output
  router-core probe --fixtures fixtures/synthetic/tplink-wr841n-v8 --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := probeOptions{
				Host:     host,
				Fixtures: fixtures,
				JSON:     jsonOut,
				Timeout:  timeoutFromMS(timeoutMS),
			}
			return runProbe(opts)
		},
	}

	cmd.Flags().StringVar(&host, "host", "192.168.0.1",
		"local router address (RFC1918 literal, e.g. 192.168.1.1)")
	cmd.Flags().StringVar(&fixtures, "fixtures", "",
		"path to a synthetic fixture; bypasses real router auth")
	cmd.Flags().BoolVar(&jsonOut, "json", false,
		"write JSON to stdout (results only; diagnostics stay on stderr)")
	cmd.Flags().IntVar(&timeoutMS, "timeout-ms", 5000,
		"per-request timeout in milliseconds (real router only)")

	return cmd
}
