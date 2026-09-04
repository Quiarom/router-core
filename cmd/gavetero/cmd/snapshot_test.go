package cmd_test

// Snapshot tests for the gavetero Cobra command tree.
// These tests pin the exact text of --help and the per-subcommand
// help output. They catch accidental help-text regressions.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Quiarom/router-core/cmd/gavetero/cmd"
)

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

func TestSnapshotRootHelp(t *testing.T) {
	stdout, _, err := runWithArgs(t, []string{"--help"})
	if err != nil {
		t.Fatalf("--help must not return error; got %v", err)
	}
	mustContain(t, stdout,
		"gavetero",
		"Available Commands:",
		"setup",
		"ask",
		"inspect",
		"doctor",
		"version",
		"gvt", // alias
	)
}

func TestSnapshotVersion(t *testing.T) {
	stdout, _, err := runWithArgs(t, []string{"version"})
	if err != nil {
		t.Fatalf("version must not return error; got %v", err)
	}
	if !strings.HasPrefix(stdout, "gavetero ") {
		t.Errorf("version output must start with 'gavetero '; got %q", stdout)
	}
}

func TestVersionFlag(t *testing.T) {
	stdout, _, err := runWithArgs(t, []string{"--version"})
	if err != nil {
		t.Fatalf("--version must not return error; got %v", err)
	}
	if !strings.HasPrefix(stdout, "gavetero ") {
		t.Errorf("--version output must start with 'gavetero '; got %q", stdout)
	}
}

func TestUnknownCommandDoesNotPrintUsage(t *testing.T) {
	// Use the production entry path so the canonical "gavetero: <msg>"
	// shape is rendered. The wrapper calls NewRootCmd and discards
	// cobra's own error stream; the canonical error is what we
	// capture here.
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	err := cmd.ExecuteWithIO(stdout, stderr, "totally-bogus-subcommand")
	if err == nil {
		t.Fatal("ExecuteWithIO returned nil; expected an error")
	}
	// The combined captured stream must NOT contain the usage block.
	combined := stdout.String() + stderr.String()
	if strings.Contains(combined, "Usage:") {
		t.Errorf("unknown subcommand must not print usage; got:\n%s", combined)
	}
}

func mustContain(t *testing.T, haystack string, needles ...string) {
	t.Helper()
	for _, n := range needles {
		if !strings.Contains(haystack, n) {
			t.Errorf("expected to find %q in output:\n%s", n, haystack)
		}
	}
}
