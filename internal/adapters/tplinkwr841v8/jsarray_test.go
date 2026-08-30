package tplinkwr841v8

import (
	"errors"
	"testing"

	"github.com/Quiarom/router-core/internal/domain"
)

func TestExtractArray(t *testing.T) {
	html := []byte(`var x = new Array(
		"one, two", 'escaped\'quote', /* comment */ 42,
		foo // comment
	);`)
	tokens, ok := ExtractArray(html, "x")
	if !ok || len(tokens) != 4 {
		t.Fatalf("ok=%v tokens=%#v", ok, tokens)
	}
	if got, ok := Str(tokens, 0); !ok || got != "one, two" {
		t.Fatalf("string=%q %v", got, ok)
	}
	if got, ok := Int(tokens, 2); !ok || got != 42 {
		t.Fatalf("int=%d %v", got, ok)
	}
	if _, ok := Str(tokens, 20); ok {
		t.Fatal("out of range accessor succeeded")
	}
	if tokens[3].Kind != TokenRaw {
		t.Fatalf("kind=%s", tokens[3].Kind)
	}
	if empty, ok := ExtractArray([]byte("var e=new Array();"), "e"); !ok || len(empty) != 0 {
		t.Fatal("empty array not recognized")
	}
	for _, input := range []string{
		`<html>no block</html>`,
		`var x = new Array("unterminated);`,
		`var x = new Array(1, /* unterminated);`,
	} {
		if _, ok := ExtractArray([]byte(input), "x"); ok {
			t.Fatalf("malformed input accepted: %s", input)
		}
	}
}

func TestClassify(t *testing.T) {
	if !errors.Is(Classify([]byte(`<form name="login"><input password></form>`)), domain.ErrUnauthenticated) {
		t.Fatal("login page not detected")
	}
	if !errors.Is(Classify(nil), domain.ErrUnexpectedResponse) {
		t.Fatal("empty page not rejected")
	}
	if IsLoginPage([]byte(`<html><body>login password help</body><script>var statusPara = new Array(1);</script></html>`)) {
		t.Fatal("status page mentioning login/password was misclassified")
	}
}
