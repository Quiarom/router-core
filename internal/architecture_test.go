package internal_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourceContainsNoMutatingHTTPCalls(t *testing.T) {
	err := filepath.Walk("../..", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if filepath.Base(path) == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
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
