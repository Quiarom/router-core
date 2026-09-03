package cmd

import "time"

// probeOptions carries the resolved flag values for `router-core probe`.
// Application logic receives this struct and does not depend on Cobra.
type probeOptions struct {
	Host     string
	Fixtures string
	JSON     bool
	Timeout  time.Duration
}

// inspectOptions carries the resolved flag values for `router-core inspect`.
type inspectOptions struct {
	Host     string
	Fixtures string
	JSON     bool
	Timeout  time.Duration
}

// serveOptions carries the resolved flag values for `router-core serve`.
type serveOptions struct {
	Host          string
	Addr          string
	Timeout       time.Duration
	Mock          bool
	MockFixture   string
	PasswordStdin bool
}

// timeoutFromMS converts a millisecond flag value to a Duration.
// Centralized so future flags (e.g. --connect-timeout-ms) reuse the
// same conversion.
func timeoutFromMS(ms int) time.Duration {
	return time.Duration(ms) * time.Millisecond
}
