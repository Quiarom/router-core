// Package sidecars embeds the router-core and router-core-agent
// binaries that gavetero spawns as sidecars. This makes
// gavetero a single-binary distribution: the user only
// installs one thing, and the sidecars travel inside it.
//
// The binaries are placed in this directory at build time
// by the Makefile. The embed pattern requires them to be
// present on disk before `go build`, not at runtime.
package sidecars

import (
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// Binaries is the embedded FS containing router-core and
// router-core-agent. They are placed here at build time by
// the Makefile.
var Binaries embed.FS

//go:embed router-core
//go:embed router-core-agent
var _placeholder embed.FS // satisfy the "embed imported" check

// Get returns the absolute path to a sidecar extracted from
// the embedded FS to a temp directory. The first call for
// either name extracts both sidecars; subsequent calls return
// the cached paths.
func Get(name string) (string, error) {
	if state.dir == "" {
		if err := extract(); err != nil {
			return "", err
		}
	}
	switch name {
	case "router-core":
		return state.corePath, nil
	case "router-core-agent":
		return state.agentPath, nil
	}
	return "", fmt.Errorf("unknown sidecar %q", name)
}

type pkgState struct {
	dir       string
	corePath  string
	agentPath string
}

var state pkgState

func extract() error {
	core, err := Binaries.ReadFile("router-core")
	if err != nil {
		return fmt.Errorf("read embedded router-core: %w (build with `make build`)", err)
	}
	agent, err := Binaries.ReadFile("router-core-agent")
	if err != nil {
		return fmt.Errorf("read embedded router-core-agent: %w (build with `make build`)", err)
	}
	dir, err := os.MkdirTemp("", "gavetero-sidecars-")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	corePath := filepath.Join(dir, "router-core")
	agentPath := filepath.Join(dir, "router-core-agent")
	if err := writeAndChmod(corePath, core); err != nil {
		_ = os.RemoveAll(dir)
		return err
	}
	if err := writeAndChmod(agentPath, agent); err != nil {
		_ = os.RemoveAll(dir)
		return err
	}
	state = pkgState{dir: dir, corePath: corePath, agentPath: agentPath}
	runtime.AddCleanup(&dir, func(d string) { _ = os.RemoveAll(d) }, dir)
	return nil
}

func writeAndChmod(path string, data []byte) error {
	if err := os.WriteFile(path, data, 0o755); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := os.Chmod(path, 0o755); err != nil {
		return fmt.Errorf("chmod %s: %w", path, err)
	}
	return nil
}

// Read exposes the raw embedded bytes for tests.
func Read(name string) ([]byte, error) {
	switch name {
	case "router-core":
		return Binaries.ReadFile("router-core")
	case "router-core-agent":
		return Binaries.ReadFile("router-core-agent")
	}
	return nil, fmt.Errorf("unknown sidecar %q", name)
}

// satisfy unused import
var _ = io.EOF
