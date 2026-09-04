// Package cmd: the setup subcommand. Configures Gavetero's
// non-secret config under XDG, and stores the GMI Cloud
// API key in the OS credential store (Secret Service on
// Linux) when available.
//
// Resolution order for the API key at runtime:
//
//  1. GMI_SERVING_API_KEY environment variable (CI, headless)
//  2. keyring entry "gavetero/gmi-cloud-api-key" (set by setup)
//  3. (none) -> gvt ask prints a clean actionable error
//
// The setup subcommand never accepts --api-key <secret>
// (which would put the secret on argv) and never writes the
// key in plaintext to disk.
package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
	"golang.org/x/term"
)

const (
	// keyringService is the OS keyring "service" name.
	keyringService = "gavetero"
	// keyringAccount is the OS keyring "account" name.
	keyringAccount = "gmi-cloud-api-key"
	// envVarName is the env var override.
	envVarName = "GMI_SERVING_API_KEY"
)

func newSetupCmd() *cobra.Command {
	var apiKeyStdin bool

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Configure Gavetero (non-secret config + GMI Cloud API key)",
		Long: `Run this once on a new machine. It:

  - writes ~/.config/gavetero/config.toml (non-secret only);
  - stores the GMI Cloud API key in the OS credential store
    (Secret Service on Linux, Keychain on macOS, Credential
    Manager on Windows);
  - prints where the key lives and how to remove it later.

The GMI key is never written in plaintext. If no secure
credential store is available, the wizard explains the
fallback (GMI_SERVING_API_KEY env var) and does not silently
write the key to disk.

The wizard never accepts --api-key <secret> as a flag.
That would put the secret on the process command line.`,
		Example: `  gvt setup
  printf '%s' "$KEY" | gvt setup --api-key-stdin`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSetup(cmd.OutOrStdout(), cmd.ErrOrStderr(), apiKeyStdin)
		},
	}
	cmd.Flags().BoolVar(&apiKeyStdin, "api-key-stdin", false,
		"read the API key from stdin (avoids argv, supports piping)")
	return cmd
}

// configFile represents the non-secret config written under XDG.
// We do not store any credentials here.
type configFile struct {
	RouterHost    string `toml:"router_host,omitempty"`
	Model         string `toml:"model,omitempty"`
	FallbackModel string `toml:"fallback_model,omitempty"`
	Output        string `toml:"output,omitempty"`
}

func defaultConfig() configFile {
	return configFile{
		RouterHost:    "", // empty = auto-detect default gateway
		Model:         "MiniMaxAI/MiniMax-M3",
		FallbackModel: "MiniMaxAI/MiniMax-M2.7",
		Output:        "human",
	}
}

func runSetup(stdout, stderr io.Writer, apiKeyStdin bool) error {
	out := cmdWriter{stdout, stderr}

	out.println("Gavetero setup")
	out.println("==============")
	out.println("")

	// 1. Read the API key.
	key, err := readAPIKey(stdout, stderr, apiKeyStdin)
	if err != nil {
		return err
	}
	if key == "" {
		return errors.New("no API key provided; rerun when ready")
	}

	// 2. Store the key. Prefer the OS credential store; explain
	//    and degrade gracefully if it is not available.
	if err := keyring.Set(keyringService, keyringAccount, key); err != nil {
		out.println("")
		out.println("Could not store the key in the OS credential store:")
		out.println("  " + err.Error())
		out.println("")
		out.println("Falling back: set the API key in your environment instead:")
		out.println("  export " + envVarName + "=\"<your-key>\"")
		out.println("")
		out.println("Add that line to your shell rc (~/.bashrc, ~/.zshrc)")
		out.println("so the variable is set in every new terminal.")
	} else {
		out.println("")
		out.println("✓ Saved securely to the OS credential store")
		out.println("  service: " + keyringService)
		out.println("  account: " + keyringAccount)
	}

	// 3. Write the non-secret config to XDG.
	cfg := defaultConfig()
	cfgPath, err := writeConfig(cfg)
	if err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	out.println("")
	out.println("✓ Wrote non-secret config to " + cfgPath)

	// 4. Final summary.
	out.println("")
	out.println("Ready. Try:")
	out.println("  gvt doctor")
	out.println("  gvt inspect")
	out.println("  gvt ask \"Is my network exposed?\"")
	return nil
}

// readAPIKey returns the API key from stdin, the TTY, or
// surfaces a clear error explaining how to provide one.
func readAPIKey(stdout, stderr io.Writer, apiKeyStdin bool) (string, error) {
	if apiKeyStdin {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("read stdin: %w", err)
		}
		key := strings.TrimSpace(string(b))
		if key == "" {
			return "", errors.New("stdin was empty; provide a non-empty API key")
		}
		return key, nil
	}

	// Interactive path: we need a real TTY. We try /dev/tty
	// directly so the user can run `gvt ask` (which has a
	// non-TTY stdin) and still set up the key interactively.
	tty, err := os.Open("/dev/tty")
	if err != nil {
		// Surface a single, clear, actionable error. Do not
		// fall back to a plain stdin read: that would print
		// the typed characters back to the terminal, which is
		// the bug this fix replaces.
		return "", errors.New(
			"no interactive TTY available. " +
				"Use --api-key-stdin to read the key from a pipe: " +
				"printf 'YOUR_KEY' | gvt setup --api-key-stdin",
		)
	}
	defer tty.Close()

	fmt.Fprintln(stderr, "AI")
	fmt.Fprintln(stderr, "  Provider    GMI Cloud")
	fmt.Fprintln(stderr, "  Model       "+defaultConfig().Model)
	fmt.Fprintln(stderr, "")
	fmt.Fprintln(stderr, "GMI Cloud API key:")
	fmt.Fprintln(stderr, "(input is hidden; press Enter when done)")
	pw, err := term.ReadPassword(int(tty.Fd()))
	if err != nil {
		return "", fmt.Errorf("read password from TTY: %w", err)
	}
	fmt.Fprintln(stderr, "")
	return strings.TrimSpace(string(pw)), nil
}

func writeConfig(cfg configFile) (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	gavDir := filepath.Join(dir, "gavetero")
	if err := os.MkdirAll(gavDir, 0700); err != nil {
		return "", err
	}
	cfgPath := filepath.Join(gavDir, "config.toml")
	b, err := toml.Marshal(cfg)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(cfgPath, b, 0600); err != nil {
		return "", err
	}
	return cfgPath, nil
}

// cmdWriter sends user-facing messages: titles to stdout,
// subtitles and progress to stderr. This matches the project
// output contract (results -> stdout, diagnostics -> stderr).
type cmdWriter struct {
	out io.Writer
	err io.Writer
}

func (w cmdWriter) println(s string) {
	fmt.Fprintln(w.out, s)
}
