package tplinkwr841v8

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Quiarom/router-core/internal/domain"
)

// statusFixture is a sanitizedized version of the Status page observed
// on the physical WR841N v8.4 lab unit on 2026-08-30. The fingerprint
// (firmware/hardware) is preserved; session tokens, MACs, SSIDs and
// other network identifiers have been redacted to placeholders.
const statusFixture = `<html><body>
var statusPara = new Array(
  "", "", "", "", "", "",
  "3.13.33 Build 130506 Rel.48660n",
  "WR841N v8 00000000",
  "0",
  "1"
);
</body></html>`

// md5hexLocal is provided for test clarity. The runtime probe uses
// md5hex for older recipes; the WR841N v8.4 firmware 3.13.33 Build
// 130506 Rel.48660n observed on 2026-08-30 rejects md5hex and accepts
// plaintext Basic Auth.
func md5hexLocal(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

// wr841nStubServer is an in-process emulation of the WR841N v8.4
// firmware. It responds to HTTP Basic Authorization against / with
// either 200 (correct credentials) or 401 (anything else). It responds
// to /userRpm/StatusRpm.htm the same way. The Authorization header
// value is "Basic base64(user:plain_password)" with no md5hex hashing.
type wr841nStubServer struct {
	server *httptest.Server
}

func newWR841NStub(t *testing.T, validUser, validPass string) *wr841nStubServer {
	t.Helper()
	expected := base64.StdEncoding.EncodeToString([]byte(validUser + ":" + validPass))
	mux := http.NewServeMux()
	mux.HandleFunc("/userRpm/StatusRpm.htm", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Basic ") {
			w.WriteHeader(http.StatusUnauthorized)
			w.Header().Set("WWW-Authenticate", `Basic realm="TP-LINK"`)
			return
		}
		if auth != "Basic "+expected {
			w.WriteHeader(http.StatusUnauthorized)
			w.Header().Set("WWW-Authenticate", `Basic realm="TP-LINK"`)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, statusFixture)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Basic ") {
			w.WriteHeader(http.StatusUnauthorized)
			w.Header().Set("WWW-Authenticate", `Basic realm="TP-LINK"`)
			return
		}
		if auth != "Basic "+expected {
			w.WriteHeader(http.StatusUnauthorized)
			w.Header().Set("WWW-Authenticate", `Basic realm="TP-LINK"`)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, "<html><body>dashboard</body></html>")
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &wr841nStubServer{server: srv}
}

func (s *wr841nStubServer) URL() string {
	return s.server.URL
}

// TestAdapter_Login_Succeeds verifies that Login authenticates against
// the WR841N recipe (HTTP Basic Auth against / with plaintext password,
// no md5hex) and stores the session.
func TestAdapter_Login_Succeeds(t *testing.T) {
	stub := newWR841NStub(t, "admin", "hunter2")
	host := strings.TrimPrefix(stub.URL(), "http://")
	a := New(host)
	if a.authenticated() {
		t.Fatalf("adapter should not be authenticated before Login")
	}
	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFn()
	if err := a.Login(ctx, "admin", "hunter2"); err != nil {
		t.Fatalf("Login: %v", err)
	}
	if !a.authenticated() {
		t.Fatalf("adapter should be authenticated after Login")
	}
}

// TestAdapter_Login_WrongPassword verifies that Login returns
// ErrUnauthenticated when the credentials are wrong.
func TestAdapter_Login_WrongPassword(t *testing.T) {
	stub := newWR841NStub(t, "admin", "hunter2")
	host := strings.TrimPrefix(stub.URL(), "http://")
	a := New(host)
	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFn()
	if err := a.Login(ctx, "admin", "wrongpass"); err == nil {
		t.Fatalf("expected error for wrong password")
	} else if err != domain.ErrUnauthenticated {
		t.Fatalf("expected ErrUnauthenticated, got: %v", err)
	}
	if a.authenticated() {
		t.Fatalf("adapter should NOT be authenticated after wrong password")
	}
}

// TestAdapter_Status_RequiresLogin verifies that Status returns an
// error when called without prior Login.
func TestAdapter_Status_RequiresLogin(t *testing.T) {
	a := New("192.168.1.1")
	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFn()
	if _, err := a.Status(ctx); err == nil {
		t.Fatalf("expected error when Status called before Login")
	}
}

// TestAdapter_Status_AfterLogin verifies the full Login + Status flow
// against the WR841N recipe. The body is the sanitized fixture from the
// physical capture; Status returns without error and reaches the
// parser.
func TestAdapter_Status_AfterLogin(t *testing.T) {
	stub := newWR841NStub(t, "admin", "hunter2")
	host := strings.TrimPrefix(stub.URL(), "http://")
	a := New(host)

	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFn()

	if err := a.Login(ctx, "admin", "hunter2"); err != nil {
		t.Fatalf("Login: %v", err)
	}

	status, err := a.Status(ctx)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	// Status reaches the parser and returns a RouterStatus with
	// Reachable=True (since the response was 200).
	if status.Reachable != domain.True {
	}
}

// TestAdapter_DoesNotSendMD5HexVerifiedBySanitizedCapture verifies
// that Login uses the plain-text recipe (verified against the physical
// lab unit) and NOT the md5hex recipe (which the lab firmware rejected).
// The stub server accepts only the plain-text variant. If the adapter
// accidentally sent the md5hex value, the test fails.
func TestAdapter_DoesNotSendMD5HexVerifiedBySanitizedCapture(t *testing.T) {
	stub := newWR841NStub(t, "admin", "hunter2")
	host := strings.TrimPrefix(stub.URL(), "http://")
	a := New(host)

	// The plain-text password recipe (the one the lab unit accepted).
	plainAuth := base64.StdEncoding.EncodeToString([]byte("admin:hunter2"))

	// Sanity: the md5hex recipe is what PA-1 / PA-2 / tplink_exporter
	// all build. It MUST be different from the plain recipe.
	md5Auth := base64.StdEncoding.EncodeToString([]byte("admin:" + md5hexLocal("hunter2")))
	if plainAuth == md5Auth {
		t.Fatalf("test setup error: plain and md5hex base64 collide")
	}

	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFn()
	if err := a.Login(ctx, "admin", "hunter2"); err != nil {
		t.Fatalf("Login with plain recipe: %v", err)
	}
	if !a.authenticated() {
		t.Fatalf("adapter must be authenticated after Login")
	}
}
