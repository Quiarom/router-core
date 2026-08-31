package tplinkwr841v8

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Quiarom/router-core/internal/domain"
)

// statusFixture is a sanitized version of the Status page observed
// on the physical WR841N v8.4 lab unit. The fingerprint
// (firmware/hardware) is preserved.
const statusFixture = `<html><body>
var statusPara = new Array(
  "", "", "", "", "", "",
  "3.15.9 Build 140724 Rel.63227n",
  "WR841N v8 00000000",
  "0",
  "1"
);
</body></html>`

// wr841nStubServer is an in-process emulation of the WR841N v8.4
// firmware. It responds to HTTP Basic Authorization against / with
// 200 (correct credentials) or 401 (wrong credentials). It responds
// to /userRpm/StatusRpm.htm with the sanitized status fixture when
// the Authorization header is correct and the Referer header
// matches the parent frameset URL (the recipe verified live
// 2026-08-31). With a missing or wrong Referer, the firmware
// returns the 68-byte "no authority" rejection.
type wr841nStubServer struct {
	server *httptest.Server
}

func newWR841NStub(t *testing.T, validUser, validPass string) *wr841nStubServer {
	t.Helper()
	expected := base64.StdEncoding.EncodeToString([]byte(validUser + ":" + validPass))
	mux := http.NewServeMux()
	mux.HandleFunc("/userRpm/StatusRpm.htm", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Basic "+expected {
			w.WriteHeader(http.StatusUnauthorized)
			w.Header().Set("WWW-Authenticate", `Basic realm="TP-LINK"`)
			return
		}
		// The v8.4 firmware only returns the dashboard body when
		// the Referer points to the parent frameset page.
		// Without it, the firmware returns the 68-byte rejection.
		if r.Header.Get("Referer") == "" {
			w.Header().Set("Content-Type", "text/html")
			_, _ = io.WriteString(w, "<html><body><h1><B>You have no authority to access this router!</B></h1></body></html>")
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, statusFixture)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
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

func TestAdapter_Status_RequiresLogin(t *testing.T) {
	a := New("192.168.1.1")
	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFn()
	if _, err := a.Status(ctx); err == nil {
		t.Fatalf("expected error when Status called before Login")
	}
}

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
	if status.Reachable != domain.True {
		t.Errorf("status.Reachable: got %v want True", status.Reachable)
	}
}

func TestAdapter_Identify_AfterLogin(t *testing.T) {
	stub := newWR841NStub(t, "admin", "hunter2")
	host := strings.TrimPrefix(stub.URL(), "http://")
	a := New(host)

	ctx, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFn()

	if err := a.Login(ctx, "admin", "hunter2"); err != nil {
		t.Fatalf("Login: %v", err)
	}

	info, err := a.Identify(ctx)
	if err != nil {
		t.Fatalf("Identify: %v", err)
	}
	if info.Authenticated != domain.True {
		t.Errorf("Authenticated: got %v want True", info.Authenticated)
	}
	if info.FirmwareVersion.Value() == "" {
		t.Errorf("FirmwareVersion: got empty string, want a real value")
	}
	if info.HardwareVersion.Value() == "" {
		t.Errorf("HardwareVersion: got empty string, want a real value")
	}
}
