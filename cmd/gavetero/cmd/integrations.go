// Package cmd: the integrations subcommand. Copies the
// portable Gavetero skill into the install directory of the
// agent the user wants to use it with. Currently supported:
//
//	hermes      -> ~/.hermes/skills/gavetero/
//	opencode     -> ~/.config/opencode/skills/gavetero/
//	omp         -> ~/.config/omp/skills/gavetero/  (alias of opencode)
//
// The install is a plain file copy. The user does not need
// to restart the agent to pick up the skill (most agents
// load SKILL.md lazily on first use, but Hermes in
// particular announces the skill on session start).
package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newIntegrationsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "integrations <install|list> [target]",
		Short: "Install the Gavetero skill into an agent's skill directory",
		Long: `Manage Gavetero integrations with other agents.

Subcommands:

  gvt integrations install <hermes|opencode|omp>
      Copy skills/gavetero/SKILL.md into the agent's skill
      directory. The skill becomes available as a slash
      command (Hermes) or as a name-activatable tool
      (OpenCode, OMP).

  gvt integrations list
      List the install directories known to gavetero and
      whether the skill is currently present.`,
		Example: `  gvt integrations install opencode
  gvt integrations install hermes
  gvt integrations list`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "install":
				if len(args) < 2 {
					return errors.New("usage: gvt integrations install <hermes|opencode|omp>")
				}
				return runIntegrationsInstall(cmd.OutOrStdout(), args[1])
			case "list":
				return runIntegrationsList(cmd.OutOrStdout())
			default:
				return fmt.Errorf("unknown subcommand %q (want install or list)", args[0])
			}
		},
	}
	return cmd
}

func runIntegrationsInstall(stdout io.Writer, target string) error {
	dest, err := integrationTargetDir(target)
	if err != nil {
		return err
	}
	src, err := findSkillSource()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("create %s: %w", dest, err)
	}
	dstFile := filepath.Join(dest, "SKILL.md")
	body, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	if err := os.WriteFile(dstFile, body, 0644); err != nil {
		return fmt.Errorf("write %s: %w", dstFile, err)
	}
	fmt.Fprintf(stdout, "\u2713 Installed Gavetero skill to %s\n", dstFile)
	return nil
}

func runIntegrationsList(stdout io.Writer) error {
	targets := []struct {
		name string
		path string
	}{
		{"hermes", filepath.Join(homeDir(), ".hermes", "skills", "gavetero", "SKILL.md")},
		{"opencode", filepath.Join(homeDir(), ".config", "opencode", "skills", "gavetero", "SKILL.md")},
		{"omp", filepath.Join(homeDir(), ".config", "omp", "skills", "gavetero", "SKILL.md")},
	}
	fmt.Fprintln(stdout, "Gavetero skill install status:")
	fmt.Fprintln(stdout, "------------------------------")
	for _, t := range targets {
		_, err := os.Stat(t.path)
		state := "not installed"
		if err == nil {
			state = "installed"
		}
		fmt.Fprintf(stdout, "  %-10s %-12s %s\n", t.name, state, t.path)
	}
	return nil
}

func integrationTargetDir(target string) (string, error) {
	home := homeDir()
	switch target {
	case "hermes":
		return filepath.Join(home, ".hermes", "skills", "gavetero"), nil
	case "opencode", "omp":
		return filepath.Join(home, ".config", target, "skills", "gavetero"), nil
	default:
		return "", fmt.Errorf("unknown integration target %q (want hermes, opencode, or omp)", target)
	}
}

func findSkillSource() (string, error) {
	// The skill source lives in the repo at skills/gavetero/SKILL.md.
	// We resolve it relative to the running executable:
	//   <exe dir>/../../../skills/gavetero/SKILL.md
	// (the executable is <repo>/bin/gavetero when built with `make`,
	// and <repo>/cmd/gavetero/main when running `go run`).
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		// Try the install location first.
		candidate := filepath.Join(dir, "..", "..", "skills", "gavetero", "SKILL.md")
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate, nil
		}
		// Then the source-tree layout.
		candidate = filepath.Join(dir, "..", "..", "..", "skills", "gavetero", "SKILL.md")
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate, nil
		}
	}
	// Fallback: look for skills/gavetero/SKILL.md in the current dir
	// and a few parents (for `go test` running inside cmd/gavetero/).
	wd, err := os.Getwd()
	if err == nil {
		for i := 0; i < 5; i++ {
			candidate := filepath.Join(wd, "skills", "gavetero", "SKILL.md")
			if _, statErr := os.Stat(candidate); statErr == nil {
				return candidate, nil
			}
			wd = filepath.Dir(wd)
		}
	}
	return "", fmt.Errorf("could not find skills/gavetero/SKILL.md (run from the repo root, or build with `make build` so the skill is co-located with the binary)")
}

func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return home
}
