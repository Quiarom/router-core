package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Quiarom/router-core/internal/adapters/tplinkwr841v8"
	"github.com/Quiarom/router-core/internal/transport"
)

// wr841nServerForServe emulates the WR841N v8.4 firmware for the
// serve binary's runtime tests. The Basic Auth recipe matches
// ADR 0005.
type wr841nServerForServe struct {
	server     *httptest.Server
	authValue  string
	sessionTok string
}

func newWR841NForServe(t *testing.T, user, pass string, sessionTok string) *wr841nServerForServe {
	t.Helper()
	authValue := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
	expected := "Basic " + authValue

	mux := http.NewServeMux()
	mux.HandleFunc("/userRpm/AssignedIpAddrListRpm.htm", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != expected {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, `<html><body><script>var DHCPDynList = new Array("omarchy", "00:11:22:33:44:55:66", "192.168.1.100", "01:24:35", 0, 0); var DHCPDynPara = new Array(1, 4, 0, 0);</script></body></html>`)
	})
	// Catch-all: any /userRpm/<path> with the right Basic Auth
	// returns a dashboard body. Production paths use the adapter's
	// authedFetchWithFallback, which exercises both bare and
	// token-prefixed URLs against the real firmware.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != expected {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/":
			body := `<html><body>dashboard</body></html>`
			if sessionTok != "" {
				body = `<html><body><script>window.parent.location.href="http://192.168.0.1/` + sessionTok + `/userRpm/Index.htm"</script>dashboard</body></html>`
			}
			w.Header().Set("Content-Type", "text/html")
			_, _ = io.WriteString(w, body)
		case "/userRpm/StatusRpm.htm":
			w.Header().Set("Content-Type", "text/html")
			_, _ = io.WriteString(w, `<html><body>var statusPara = new Array(1,1,1,1,1,1,"3.13.33 Build 130506 Rel.48660n","WR841N v8 00000000",0,1);</body></html>`)
		default:
			w.Header().Set("Content-Type", "text/html")
			_, _ = io.WriteString(w, `<html><body>var dataPara = new Array(1,1,1,1,1,1,1,1,1,1);</body></html>`)
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &wr841nServerForServe{server: srv, authValue: authValue, sessionTok: sessionTok}
}

// listenLoopback opens a random TCP port on the loopback interface
// so serve tests never bind a fixed port.
func listenLoopback() (net.Listener, error) {
	return net.Listen("tcp", "127.0.0.1:0")
}

func runServeWithAdapter(t *testing.T, adapter *tplinkwr841v8.Adapter, sessionTok string) (string, func()) {
	t.Helper()
	if sessionTok != "" {
		adapter.SessionTokenForTest(sessionTok)
	}
	mux := http.NewServeMux()
	store := &sessionStore{password: "hunter2"}
	registerRoutes(mux, adapter, store)

	ln, err := listenLoopback()
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := &http.Server{Handler: mux}
	go func() { _ = srv.Serve(ln) }()
	return "http://" + ln.Addr().String(), func() { _ = srv.Close() }
}

func TestServe_Healthz(t *testing.T) {
	srv := newWR841NForServe(t, "admin", "hunter2", "")
	host := strings.TrimPrefix(srv.server.URL, "http://")
	a := tplinkwr841v8.New(host, transport.WithTimeout(2*time.Second))
	if err := a.Login(context.Background(), "admin", "hunter2"); err != nil {
		t.Fatalf("Login: %v", err)
	}
	addr, teardown := runServeWithAdapter(t, a, "")
	defer teardown()
	resp, err := http.Get(addr + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status: %d want 200", resp.StatusCode)
	}
}

func TestServe_Status(t *testing.T) {
	srv := newWR841NForServe(t, "admin", "hunter2", "ABCDEFGHIJKLMNOP")
	host := strings.TrimPrefix(srv.server.URL, "http://")
	a := tplinkwr841v8.New(host, transport.WithTimeout(2*time.Second))
	if err := a.Login(context.Background(), "admin", "hunter2"); err != nil {
		t.Fatalf("Login: %v", err)
	}
	// SessionTokenForTest is the only test-side escape hatch to
	// populate the URL-embedded token; the v8.4 firmware does not
	// return one from /, so production code paths obtain it from
	// the operator or the login response.
	a.SessionTokenForTest("ABCDEFGHIJKLMNOP")
	addr, teardown := runServeWithAdapter(t, a, "ABCDEFGHIJKLMNOP")
	defer teardown()
	resp, err := http.Get(addr + "/v0/status")
	if err != nil {
		t.Fatalf("GET /v0/status: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status: %d want 200, body: %s", resp.StatusCode, readAll(t, resp.Body))
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["reachable"] == nil {
		t.Errorf("body missing reachable: %v", body)
	}
}

func TestServe_Clients(t *testing.T) {
	srv := newWR841NForServe(t, "admin", "hunter2", "")
	host := strings.TrimPrefix(srv.server.URL, "http://")
	a := tplinkwr841v8.New(host, transport.WithTimeout(2*time.Second))
	if err := a.Login(context.Background(), "admin", "hunter2"); err != nil {
		t.Fatalf("Login: %v", err)
	}
	// The clients endpoint accepts the bare path with Basic Auth
	// header; no session token is needed.
	addr, teardown := runServeWithAdapter(t, a, "")
	defer teardown()
	resp, err := http.Get(addr + "/v0/clients")
	if err != nil {
		t.Fatalf("GET /v0/clients: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status: %d want 200, body: %s", resp.StatusCode, readAll(t, resp.Body))
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["state"] != "verified" {
		t.Errorf("state: %v want verified", body["state"])
	}
}

func readAll(t *testing.T, r io.Reader) string {
	t.Helper()
	body, _ := io.ReadAll(r)
	return string(body)
}

func TestServe_UnsupportedEndpoints(t *testing.T) {
	srv := newWR841NForServe(t, "admin", "hunter2", "")
	host := strings.TrimPrefix(srv.server.URL, "http://")
	a := tplinkwr841v8.New(host, transport.WithTimeout(2*time.Second))
	if err := a.Login(context.Background(), "admin", "hunter2"); err != nil {
		t.Fatalf("Login: %v", err)
	}
	addr, teardown := runServeWithAdapter(t, a, "")
	defer teardown()
	// WPS, UPnP, and Remote Management are not present on the
	// WR841N v8.4 build (501 observed 2026-08-31). serve must
	// respond 404 with the structured payload, not 200+false.
	for _, p := range []string{
		"/v0/security/wps",
		"/v0/security/upnp",
		"/v0/security/remote-management",
	} {
		t.Run(p, func(t *testing.T) {
			resp, err := http.Get(addr + p)
			if err != nil {
				t.Fatalf("GET %s: %v", p, err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != 404 {
				t.Errorf("status: %d want 404", resp.StatusCode)
			}
			var body map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if body["state"] != "unsupported_or_unverified" {
				t.Errorf("state: %v want unsupported_or_unverified", body["state"])
			}
		})
	}
}

func TestServe_RejectsNonLoopbackAddr(t *testing.T) {
	if isLoopbackAddr("0.0.0.0:8484") {
		t.Errorf("0.0.0.0 should not be accepted as loopback")
	}
	if isLoopbackAddr("192.168.1.1:8484") {
		t.Errorf("RFC1918 should not be accepted as loopback")
	}
	if !isLoopbackAddr("127.0.0.1:8484") {
		t.Errorf("127.0.0.1 must be accepted as loopback")
	}
	if !isLoopbackAddr("localhost:8484") {
		t.Errorf("localhost must be accepted as loopback")
	}
}
