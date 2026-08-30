package internal_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourceContainsNoMutatingHTTPCalls(t *testing.T) {
	err := filepath.Walk("..", func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || strings.HasSuffix(path, "_test.go") {
			return err
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		text := string(body)
		for _, forbidden := range []string{"http.Post", `NewRequest("POST"`} {
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
