package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// These tests cover the pure helpers in learn.go without using an
// httptest server. Integration with the actual probe binary is
// covered separately by the binary-level end-to-end test.

// TestBuildCandidates verifies the five evidence-backed recipes.
func TestBuildCandidates(t *testing.T) {
	cands := buildCandidates("admin", "hunter2")
	if len(cands) != 5 {
		t.Fatalf("expected 5 candidates, got %d", len(cands))
	}

	expectedNames := []string{
		candidateAName,
		candidateBName,
		candidateCName,
		"basic-auth-plain-cookie",
		"basic-auth-plain-header",
	}
	for i, name := range expectedNames {
		if cands[i].Name != name {
			t.Errorf("candidate %d name: got %q want %q", i, cands[i].Name, name)
		}
	}

	if cands[0].CookieValue != cands[2].CookieValue {
		t.Errorf("candidate A and C must share cookie value; A=%q C=%q", cands[0].CookieValue, cands[2].CookieValue)
	}
	if !strings.HasPrefix(cands[1].CookieValue, "Basic%20") {
		t.Errorf("candidate B must use Basic%%20 prefix: %q", cands[1].CookieValue)
	}
	if cands[3].CookieValue != cands[4].HeaderValue[len("Basic "):] {
		t.Errorf("candidate D and E must share plain value; D=%q E-header=%q", cands[3].CookieValue, cands[4].HeaderValue)
	}

	if cands[0].Endpoint != loginPath {
		t.Errorf("candidate A endpoint: %q", cands[0].Endpoint)
	}
	if cands[1].Endpoint != loginPath {
		t.Errorf("candidate B endpoint: %q", cands[1].Endpoint)
	}
	if cands[2].Endpoint != "/" || cands[3].Endpoint != "/" || cands[4].Endpoint != "/" {
		t.Errorf("candidates C, D, E must target root /; got %q %q %q", cands[2].Endpoint, cands[3].Endpoint, cands[4].Endpoint)
	}

	if !cands[0].UseCookie {
		t.Errorf("candidate A must use cookie")
	}
	if !cands[4].UseHeader {
		t.Errorf("candidate E must use header")
	}
	if cands[4].HeaderValue == "" || !strings.HasPrefix(cands[4].HeaderValue, "Basic ") {
		t.Errorf("candidate E header value must start with 'Basic ': %q", cands[4].HeaderValue)
	}
}

// TestMatchResponse verifies the endpoint-specific success detection.
func TestMatchResponse(t *testing.T) {
	cases := []struct {
		name   string
		c      Candidate
		status int
		body   string
		want   bool
	}{
		{
			name:   "candidate C succeeds on 200 dashboard",
			c:      Candidate{Name: "c", Endpoint: "/"},
			status: 200,
			body:   "<html><body>dashboard</body></html>",
			want:   true,
		},
		{
			name:   "candidate C fails on 200 Login Incorrect",
			c:      Candidate{Name: "c", Endpoint: "/"},
			status: 200,
			body:   "<TITLE>Login Incorrect</TITLE>",
			want:   false,
		},
		{
			name:   "candidate C fails on 401",
			c:      Candidate{Name: "c", Endpoint: "/"},
			status: 401,
			body:   "<TITLE>Login Incorrect</TITLE>",
			want:   false,
		},
		{
			name:   "legacy PA fails without token redirect",
			c:      Candidate{Name: "a", Endpoint: loginPath},
			status: 200,
			body:   "<form>login</form>",
			want:   false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := matchResponse(tc.c, tc.status, tc.body); got != tc.want {
				t.Errorf("matchResponse(%+v, %d, %q) = %v, want %v", tc.c, tc.status, tc.body, got, tc.want)
			}
		})
	}
}

// TestExtractFingerprint verifies the firmware/hardware detection.
func TestExtractFingerprint(t *testing.T) {
	body := []byte(`var statusPara = new Array(
  "", "", "", "", "", "",
  "3.13.33 Build 130506 Rel.48660n",
  "WR841N v8 00000000",
  "0", "1"
);`)
	fp := extractFingerprint(body)
	if fp.Firmware != expectedFirmware {
		t.Errorf("firmware: got %q want %q", fp.Firmware, expectedFirmware)
	}
	if fp.Hardware != expectedHardware {
		t.Errorf("hardware: got %q want %q", fp.Hardware, expectedHardware)
	}
}

func TestExtractFingerprint_NoMatch(t *testing.T) {
	fp := extractFingerprint([]byte(`<html>empty</html>`))
	if fp.Firmware != "" || fp.Hardware != "" {
		t.Fatalf("expected empty, got %+v", fp)
	}
}

// TestIsRFC1918OrLoopback covers the host guard.
func TestIsRFC1918OrLoopback(t *testing.T) {
	ok := []string{"192.168.1.1", "10.0.0.1", "172.16.0.1", "127.0.0.1", "localhost", "192.168.1.1:8080"}
	no := []string{"8.8.8.8", "example.com", "1.1.1.1", "0.0.0.0", "172.32.0.1"}
	for _, h := range ok {
		if !isRFC1918OrLoopback(h) {
			t.Errorf("expected allowed: %q", h)
		}
	}
	for _, h := range no {
		if isRFC1918OrLoopback(h) {
			t.Errorf("expected refused: %q", h)
		}
	}
}

// TestProbeErr covers the typed error contract.
func TestProbeErr(t *testing.T) {
	base := errors.New("underlying")
	e := exitCodeError(2, "no candidate matched", base)
	if e.ExitCode() != 2 {
		t.Errorf("exit code: %d", e.ExitCode())
	}
	if !strings.Contains(e.Error(), "no candidate matched") {
		t.Errorf("error string missing msg: %q", e.Error())
	}
	if !strings.Contains(e.Error(), "underlying") {
		t.Errorf("error string missing cause: %q", e.Error())
	}
	if e.Unwrap() != base {
		t.Errorf("Unwrap: got %v want %v", e.Unwrap(), base)
	}
}

func TestProbeErr_NoCause(t *testing.T) {
	e := exitCodeError(5, "invalid usage", nil)
	if e.Error() != "invalid usage" {
		t.Errorf("error string: %q", e.Error())
	}
	if e.Unwrap() != nil {
		t.Errorf("Unwrap should be nil")
	}
}

// TestPersistMismatchEvidence verifies sanitization before disk.
func TestPersistMismatchEvidence(t *testing.T) {
	tmpDir := t.TempDir()
	body := []byte(`<TITLE>Login Incorrect</TITLE>
var secret = "ABCDEFGHIJKLMNOP";`)

	if err := persistMismatchEvidence(tmpDir, "basic-auth-root", 401, body, "192.168.1.1"); err != nil {
		t.Fatalf("persistMismatchEvidence: %v", err)
	}

	path := filepath.Join(tmpDir, "login-response.html")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if strings.Contains(string(got), "ABCDEFGHIJKLMNOP") {
		t.Errorf("real session token leaked: %q", string(got))
	}
}

func TestPersistMismatchEvidence_EmptyBody(t *testing.T) {
	tmpDir := t.TempDir()
	if err := persistMismatchEvidence(tmpDir, "x", 0, nil, "192.168.1.1"); err != nil {
		t.Fatalf("empty body: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmpDir, "login-response.html")); !os.IsNotExist(err) {
		t.Errorf("expected no file for empty body, got %v", err)
	}
}

// TestReadPasswordFromReader covers the stdin fallback path.
func TestReadPasswordFromReader(t *testing.T) {
	r := strings.NewReader("hunter2\n")
	got, err := readPasswordFromReader("stdin", r, 2*time.Second)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got != "hunter2" {
		t.Errorf("got %q want hunter2", got)
	}
}

// TestReadPasswordFromReader_Timeout ensures the timeout fires.
func TestReadPasswordFromReader_Timeout(t *testing.T) {
	blocker := &blockingReader{}
	_, err := readPasswordFromReader("stdin", blocker, 50*time.Millisecond)
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("expected timeout, got: %v", err)
	}
}

type blockingReader struct{}

func (b *blockingReader) Read(p []byte) (int, error) {
	<-never()
	return 0, io.EOF
}

func never() <-chan struct{} {
	ch := make(chan struct{})
	return ch
}

// TestExitCodeFor verifies the error-to-exit-code mapping.
func TestExitCodeFor(t *testing.T) {
	if exitCodeFor(nil) != 0 {
		t.Errorf("nil error should map to 0")
	}
	if exitCodeFor(errors.New("plain")) != 1 {
		t.Errorf("plain error should map to 1")
	}
	e := exitCodeError(3, "x", nil)
	if exitCodeFor(e) != 3 {
		t.Errorf("probeErr code: got %d want 3", exitCodeFor(e))
	}
}
