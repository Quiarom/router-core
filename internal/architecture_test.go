package internal_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSourceContainsNoMutatingHTTPCalls enforces the read-only
// invariant at the code-shape level: no source file under the
// router runtime paths may issue a POST/PUT/DELETE.
//
// Scope: cmd/router-core/, cmd/router-core-learn/, internal/.
// Excluded: cmd/router-core-agent/ (the Phase 5 reasoning layer
// POSTs to OpenRouter, not to the router; the router is reached
// only via the typed /v0/ HTTP surface).
func TestSourceContainsNoMutatingHTTPCalls(t *testing.T) {
	root := repoRoot(t)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			name := filepath.Base(path)
			if name == ".git" || name == "fixtures" {
				return filepath.SkipDir
			}
			// The agent binary is allowed to POST to the LLM
			// provider; the runtime is not allowed to POST to
			// the router. Keep the invariant scoped.
			if pathContainsSegment(path, "cmd", "router-core-agent") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		text := string(body)
		for _, forbidden := range []string{
			"http.Post", "http.PostForm", ".PostForm(",
			`NewRequest("POST"`, `NewRequest("PUT"`, `NewRequest("DELETE"`,
			"http.MethodPost", "http.MethodPut", "http.MethodDelete",
		} {
			if strings.Contains(text, forbidden) {
				t.Errorf("%s contains %s", path, forbidden)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// pathContainsSegment reports whether the slash-joined segments
// of path include every segment in want, in order.
func pathContainsSegment(path string, want ...string) bool {
	parts := strings.Split(path, string(os.PathSeparator))
	i := 0
	for _, p := range parts {
		if i < len(want) && p == want[i] {
			i++
		}
	}
	return i == len(want)
}

// repoRoot walks up from the test's directory to the module root so the scan
// covers this repository and nothing outside it.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above the test directory")
		}
		dir = parent
	}
}
