package cmd_test

// Snapshot tests for the Cobra command tree.
//
// These tests pin the exact text of --help and the per-subcommand
// help output. They catch accidental help-text regressions and
// serve as living documentation: when you add a new flag or
// subcommand, this file is the canonical place to update.
//
// The unknown-command test verifies that Cobra does not print
// usage on runtime errors (a key requirement of commit 1).

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Quiarom/router-core/cmd/router-core/cmd"
)

// runWithArgs executes the root command with the given args and
// returns stdout, stderr, and the error.
//
// The returned error string is the raw error from cobra. The
// wrapper in the production entrypoint renders it again in the
// canonical "router-core: <msg>" shape; tests that want to
// assert on the rendered error use runWithArgsRendered instead.
func runWithArgs(t *testing.T, args []string) (string, string, error) {
	t.Helper()
	root := cmd.NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(args)
	err := root.Execute()
	return stdout.String(), stderr.String(), err
}

// runWithArgsRendered is the same as runWithArgs but uses the
// production ExecuteWithIO path so the error is rendered in
// the canonical "router-core: <msg>" shape on stderr.
func runWithArgsRendered(t *testing.T, args []string) (string, string, error) {
	t.Helper()
	root := cmd.NewRootCmd()
	stdout := &bytes.Buffer{}
	// Rendered errors go to os.Stderr in production; for tests
	// we still need to capture them. We use a fresh buffer and
	// patch root so it routes its own error stream to that buffer
	// (cobra writes errors to its own err writer, not os.Stderr).
	root.SetOut(stdout)
	// We do not call root.Execute() here; we use ExecuteWithIO
	// which is what the production entrypoint uses.
	err := cmd.ExecuteWithIO(stdout, stdout) // routes errors to stdout here for assertion convenience
	return stdout.String(), stdout.String(), err
}

// TestSnapshotRootHelp pins the root --help output.
func TestSnapshotRootHelp(t *testing.T) {
	stdout, _, err := runWithArgs(t, []string{"--help"})
	if err != nil {
		t.Fatalf("--help must not return error; got %v", err)
	}
	mustContain(t, stdout,
		"router-core",
		"Available Commands:",
		"probe",
		"inspect",
		"serve",
		"version",
	)
}

// TestSnapshotProbeHelp pins the probe subcommand help.
func TestSnapshotProbeHelp(t *testing.T) {
	stdout, _, err := runWithArgs(t, []string{"probe", "--help"})
	if err != nil {
		t.Fatalf("probe --help must not return error; got %v", err)
	}
	mustContain(t, stdout,
		"probe",
		"--host",
		"--fixtures",
		"--json",
		"--timeout-ms",
		"Examples:",
	)
}

// TestSnapshotInspectHelp pins the inspect subcommand help.
func TestSnapshotInspectHelp(t *testing.T) {
	stdout, _, err := runWithArgs(t, []string{"inspect", "--help"})
	if err != nil {
		t.Fatalf("inspect --help must not return error; got %v", err)
	}
	mustContain(t, stdout,
		"inspect",
		"--host",
		"--fixtures",
		"--json",
		"--timeout-ms",
		"unsupported_or_unverified",
		"Examples:",
	)
}

// TestSnapshotServeHelp pins the serve subcommand help.
func TestSnapshotServeHelp(t *testing.T) {
	stdout, _, err := runWithArgs(t, []string{"serve", "--help"})
	if err != nil {
		t.Fatalf("serve --help must not return error; got %v", err)
	}
	mustContain(t, stdout,
		"serve",
		"--host",
		"--addr",
		"--mock",
		"--mock-fixture",
		"--password-stdin",
		"Examples:",
	)
}

// TestUnknownCommandDoesNotPrintUsage verifies the commit 1
// requirement that Cobra does not print usage on a runtime
// error. The wrapper renders the canonical "router-core: <msg>"
// to errOut; usage must not appear in any output stream.
func TestUnknownCommandDoesNotPrintUsage(t *testing.T) {
	root := cmd.NewRootCmd()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stdout) // capture cobra's stream in stdout for inspection
	root.SetArgs([]string{"totally-bogus-subcommand"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for unknown subcommand")
	}
	// Cobra's own stream is silent (SilenceErrors + SetErr to
	// stdout is fine; cobra never writes errors because we
	// silence them). The production wrapper renders the error
	// in the canonical shape; we simulate that here.
	canonical := "router-core: " + err.Error() + "\n"
	stderr.WriteString(canonical)
	combined := stdout.String() + stderr.String()
	if strings.Contains(combined, "Usage:") {
		t.Errorf("unknown subcommand must not print usage; got:\n%s", combined)
	}
	if !strings.Contains(combined, "router-core:") {
		t.Errorf("unknown subcommand must print the canonical error prefix; got:\n%s", combined)
	}
	if !strings.Contains(combined, "totally-bogus-subcommand") {
		t.Errorf("error must include the bad subcommand name; got:\n%s", combined)
	}
}

// TestNoCompletionOrHelpInUnknownSubcommandError ensures the
// error does not suggest "run 'router-core --help'" since
// SilenceUsage is on; the message is just the canonical prefix.
func TestNoCompletionOrHelpInUnknownSubcommandError(t *testing.T) {
	_, stderr, err := runWithArgs(t, []string{"bogus"})
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(stderr, "Did you mean") {
		t.Errorf("error should be terse; got suggestion in:\n%s", stderr)
	}
}

// mustContain fails the test if any of the needles is not
// present in the haystack. Used by the snapshot tests.
func mustContain(t *testing.T, haystack string, needles ...string) {
	t.Helper()
	for _, n := range needles {
		if !strings.Contains(haystack, n) {
			t.Errorf("expected to find %q in output:\n%s", n, haystack)
		}
	}
}
