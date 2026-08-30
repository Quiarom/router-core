package internal_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourceContainsNoMutatingHTTPCalls(t *testing.T) {
	root := repoRoot(t)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if filepath.Base(path) == ".git" {
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
