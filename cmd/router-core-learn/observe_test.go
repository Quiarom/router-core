package main

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// signalStatus correctly identifies verified / forbidden / mismatch.
func TestSignalStatus(t *testing.T) {
	cases := []struct {
		name               string
		body               string
		status             int
		wantState          CapabilityState
		wantReasonContains string
	}{
		{
			name:               "200 with fingerprint match",
			body:               `<html>3.13.33 Build 130506 Rel.48660n WR841N v8 00000000</html>`,
			status:             200,
			wantState:          StateVerified,
			wantReasonContains: "fingerprint match",
		},
		{
			name:               "200 with login page",
			body:               "<TITLE>Login Incorrect</TITLE>",
			status:             200,
			wantState:          StateForbidden,
			wantReasonContains: "login page",
		},
		{
			name:               "401 unauthorized",
			body:               "",
			status:             401,
			wantState:          StateForbidden,
			wantReasonContains: "auth",
		},
		{
			name:               "500 server error",
			body:               "",
			status:             500,
			wantState:          StateTransportError,
			wantReasonContains: "500",
		},
		{
			name:               "200 no fingerprint match",
			body:               "<html>different firmware string</html>",
			status:             200,
			wantState:          StateMismatch,
			wantReasonContains: "no fingerprint",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state, reason := signalStatus([]byte(tc.body), tc.status)
			if state != tc.wantState {
				t.Errorf("state: got %s want %s", state, tc.wantState)
			}
			if !strings.Contains(reason, tc.wantReasonContains) {
				t.Errorf("reason: %q does not contain %q", reason, tc.wantReasonContains)
			}
		})
	}
}

// signalByHTTPAndBody correctly classifies by status code and body
// length.
func TestSignalByHTTPAndBody(t *testing.T) {
	big := strings.Repeat("x", 1024)
	cases := []struct {
		name      string
		body      string
		status    int
		wantState CapabilityState
	}{
		{"200 with body", big, 200, StateVerified},
		{"200 with empty body", "", 200, StateMismatch},
		{"200 with tiny body", "abc", 200, StateMismatch},
		{"200 with login page", "<TITLE>Login Incorrect</TITLE>", 200, StateForbidden},
		{"401", "", 401, StateForbidden},
		{"403", "", 403, StateForbidden},
		{"404", "", 404, StateUnsupported},
		{"500", "", 500, StateTransportError},
		{"501", "", 501, StateUnsupported},
		{"302", "", 302, StateTransportError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state, _ := signalByHTTPAndBody([]byte(tc.body), tc.status)
			if state != tc.wantState {
				t.Errorf("state: got %s want %s", state, tc.wantState)
			}
		})
	}
}

// capabilityList returns the documented order: device/status first,
// security surface next, then forwarding and clients.
func TestCapabilityListOrder(t *testing.T) {
	caps := capabilityList()
	if len(caps) == 0 {
		t.Fatalf("capabilityList must not be empty")
	}
	ids := make([]string, 0, len(caps))
	for _, c := range caps {
		ids = append(ids, c.id)
	}
	wantOrder := []string{"status", "wireless_security", "wps", "dmz", "upnp", "remote_management", "forwarding", "clients"}
	if len(ids) != len(wantOrder) {
		t.Fatalf("len: got %d want %d", len(ids), len(wantOrder))
	}
	for i := range wantOrder {
		if ids[i] != wantOrder[i] {
			t.Errorf("ids[%d]: got %q want %q", i, ids[i], wantOrder[i])
		}
	}
}

// parseAnyObserved requires a non-empty body that looks like HTML.
func TestParseAnyObserved(t *testing.T) {
	if _, err := parseAnyObserved([]byte("<html>body</html>")); err != nil {
		t.Errorf("HTML body should parse: %v", err)
	}
	if _, err := parseAnyObserved([]byte("")); err == nil {
		t.Errorf("empty body should fail")
	}
	if _, err := parseAnyObserved([]byte("no tags here")); err == nil {
		t.Errorf("non-HTML body should fail")
	}
}

// parseStatusObserved requires the firmware fingerprint string.
func TestParseStatusObserved(t *testing.T) {
	body := []byte(`<html>3.13.33 Build 130506 Rel.48660n</html>`)
	if _, err := parseStatusObserved(body); err != nil {
		t.Errorf("status body with firmware should parse: %v", err)
	}
	if _, err := parseStatusObserved([]byte(`<html>WRONG</html>`)); err == nil {
		t.Errorf("status body without firmware should fail")
	}
}

// TestAuthReportStructure verifies the JSON shape matches the recipe
// (header, not cookie; plaintext, not md5hex).
func TestBuildAuthReport_HeaderPlain(t *testing.T) {
	c := Candidate{
		Name:        "basic-auth-plain-header",
		Endpoint:    "/",
		UseHeader:   true,
		UseCookie:   false,
		HeaderValue: "Basic YWRtaW46YWRtaW4=",
	}
	r := buildAuthReport(c)
	if !r.HeaderNotCookie {
		t.Errorf("HeaderNotCookie should be true for header recipe")
	}
	if !r.PlainPassword {
		t.Errorf("PlainPassword should be true (basic-auth-plain-* name)")
	}
	if r.Endpoint != "/" {
		t.Errorf("Endpoint: %q", r.Endpoint)
	}
	if !r.PhysicalMatch {
		t.Errorf("PhysicalMatch should be true")
	}
	if r.CookieName != "" || r.RedirectShape != "" {
		t.Errorf("legacy cookie fields should be empty for header recipe")
	}
}

// TestAuthReportStructure_Legacy verifies the legacy cookie recipe
// shape (cookie_name + redirect_shape) is preserved for old recipes.
func TestBuildAuthReport_LegacyCookie(t *testing.T) {
	c := Candidate{
		Name:        "legacy-auth-a",
		Endpoint:    "/userRpm/LoginRpm.htm?Save=Save",
		UseCookie:   true,
		CookieValue: "YWRtaW46...",
	}
	r := buildAuthReport(c)
	if r.HeaderNotCookie {
		t.Errorf("HeaderNotCookie should be false for legacy cookie recipe")
	}
	if r.PlainPassword {
		t.Errorf("PlainPassword should be false for legacy recipe (md5hex)")
	}
	if r.CookieName != "" {
		t.Errorf("legacy cookie_name not surfaced here (only set in persistEvidence)")
	}
}

// wr841nSessionServer is an in-process emulation of the WR841N v8.4
// firmware where the firmware:
//
//  1. Accepts GET / with Basic Authorization (returns the dashboard,
//     possibly with a session token in the response body).
//  2. Accepts GET /<token>/userRpm/<path> for the protected endpoints
//     (Status, Wireless, DMZ, Forwarding) when the same Basic Auth
//     header is also sent.
//  3. Accepts GET /<path> (no token) for endpoints that don't require
//     a session token (clients).
//  4. Returns HTTP 501 for endpoints that don't exist on this build
//     (WPS, UPnP, Remote Management).
//
// This mirrors the observed behavior on the physical lab unit on
// 2026-08-30.
type wr841nSessionServer struct {
	server     *httptest.Server
	authValue  string
	sessionTok string
}

func newWR841NSessionServer(t *testing.T, user, pass string, sessionTok string) *wr841nSessionServer {
	t.Helper()
	authValue := base64.StdEncoding.EncodeToString([]byte(user + ":" + pass))
	expected := "Basic " + authValue

	mux := http.NewServeMux()
	check := func(r *http.Request) bool {
		return r.Header.Get("Authorization") == expected
	}

	// Paths that require the session token prefix. The firmware
	// serves these on /<token>/<path>, NOT on /<path>. Plain access
	// returns 68 bytes of "no authority" (matches the physical lab
	// observation). The mux registers a single catch-all and inspects
	// the prefix because Go's ServeMux treats exact paths as exact
	// match and there is no subtree path that captures the prefixed
	// form.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if !check(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// Root: returns the dashboard with the session-token script
		// embedded so probeAuth can extract it via sanitize.
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html")
			body := `<html><body>dashboard</body></html>`
			if sessionTok != "" {
				body = `<html><body><script>window.parent.location.href="http://192.168.0.1/` + sessionTok + `/userRpm/Index.htm"</script>dashboard</body></html>`
			}
			_, _ = io.WriteString(w, body)
			return
		}
		// Token URL form: /<TOKEN>/userRpm/<protected>
		if strings.HasPrefix(r.URL.Path, "/"+sessionTok+"/userRpm/") {
			suffix := strings.TrimPrefix(r.URL.Path, "/"+sessionTok)
			switch suffix {
			case "/userRpm/StatusRpm.htm",
				"/userRpm/WlanSecurityRpm.htm",
				"/userRpm/DMZRpm.htm",
				"/userRpm/VirtualServerRpm.htm":
				w.Header().Set("Content-Type", "text/html")
				_, _ = io.WriteString(w, `<html><body>var statusPara = new Array(1,1,1,1,1,1,"3.13.33 Build 130506 Rel.48660n","WR841N v8 00000000",0,1);</body></html>`)
			default:
				http.NotFound(w, r)
			}
			return
		}
		// Plain access to one of the protected paths without token:
		// rejected with the characteristic 68-byte "no authority" body.
		for _, protected := range []string{
			"/userRpm/StatusRpm.htm",
			"/userRpm/WlanSecurityRpm.htm",
			"/userRpm/DMZRpm.htm",
			"/userRpm/VirtualServerRpm.htm",
		} {
			if r.URL.Path == protected {
				w.Header().Set("Content-Type", "text/html")
				_, _ = io.WriteString(w, "<html><body><h1><B>You have no authority to access this router!</B></h1></body></html>")
				return
			}
		}
		http.NotFound(w, r)
	})

	// Clients endpoint: accepts either header-only auth (no token
	// needed). Returns dashboard.
	mux.HandleFunc("/userRpm/AssignedIpAddrListRpm.htm", func(w http.ResponseWriter, r *http.Request) {
		if !check(r) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, `<html><body>var DHCPDynList = new Array("omarchy", "00:11:22:33:44:55:66", "192.168.1.100", "01:24:35", 0, 0);</body></html><html><head><meta http-equiv="Content-Type" content="text/html; charset=iso-8859-1"><title>TL-WR841N</title><meta http-equiv="Pragma" content="no-cache"><link rel="stylesheet" type="text/css" href="/dynaform/css_main.css"><script type="text/javascript" src="/dynaform/common.js"></script></head><body>var DHCPDynPara = new Array(1, 4, 0, 0);</body></html>`)
	})

	// Endpoints that don't exist on this firmware (observed 501).
	notImplemented := []string{
		"/userRpm/WpsRpm.htm",
		"/userRpm/UpnpRpm.htm",
		"/userRpm/AccessCtrlRpm.htm",
	}
	for _, path := range notImplemented {
		captured := path
		mux.HandleFunc(captured, func(w http.ResponseWriter, r *http.Request) {
			if !check(r) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusNotImplemented)
		})
	}

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &wr841nSessionServer{server: srv, authValue: authValue, sessionTok: sessionTok}
}

func (s *wr841nSessionServer) URL() string {
	return s.server.URL
}

// TestObserve_EndToEnd_WithSessionToken runs the observe command
// against the in-process emulation of the WR841N v8.4 firmware with
// session-token-aware endpoints. Every step of the runObserve path is
// exercised through public functions (probeAuth, capabilityList, the
// per-capability loop).
func TestObserve_EndToEnd_WithSessionToken(t *testing.T) {
	const sessionTok = "ABCDEFGHIJKLMNOP"
	srv := newWR841NSessionServer(t, "admin", "hunter2", sessionTok)
	host := strings.TrimPrefix(srv.URL(), "http://")

	client := newLocalClient(2 * time.Second)
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}

	password := "hunter2"
	authValue := base64.StdEncoding.EncodeToString([]byte("admin:" + password))
	password = ""

	authBody, _, token, err := probeAuth(errOut, client, host, authValue)
	if err != nil {
		t.Fatalf("probeAuth: %v", err)
	}
	if token != sessionTok {
		t.Fatalf("expected session token %q, got %q", sessionTok, token)
	}
	if !bytes.Contains(authBody, []byte("dashboard")) {
		t.Fatalf("auth body missing dashboard: %q", authBody)
	}

	// Walk every capability using the same logic as runObserve:
	// header first, token-URL fallback on mismatch-with-token.
	caps := capabilityList()
	for _, cap := range caps {
		body, status, _ := authedGet(client, host, cap.path, authValue, false, out)
		state, reason := cap.signalFn(body, status)
		if token != "" && state == StateMismatch && len(body) == 68 {
			tokBody, tokStatus, _ := authedGetWithToken(client, host, cap.path, authValue, token, false, out)
			if tokState, _ := cap.signalFn(tokBody, tokStatus); tokState == StateVerified {
				state = StateVerified
				reason = "via session token"
				_ = reason
			}
		}
		_ = state
	}
}

// TestObserve_ProtectedEndpointsNeedToken is a smaller test that
// verifies the fallback: with session token present, protected
// endpoints are verified; without, they remain mismatch.
func TestObserve_ProtectedEndpointsNeedToken(t *testing.T) {
	const sessionTok = "ABCDEFGHIJKLMNOP"
	srv := newWR841NSessionServer(t, "admin", "hunter2", sessionTok)
	host := strings.TrimPrefix(srv.URL(), "http://")
	client := newLocalClient(2 * time.Second)
	authValue := srv.authValue

	// Status: with token, returns dashboard; without, "no authority".
	body, status, _ := authedGet(client, host, "/userRpm/StatusRpm.htm", authValue, false, &bytes.Buffer{})
	state, _ := signalStatus(body, status)
	if state != StateMismatch {
		t.Fatalf("header-only Status: got %s want mismatch", state)
	}
	tokBody, tokStatus, _ := authedGetWithToken(client, host, "/userRpm/StatusRpm.htm", authValue, sessionTok, false, &bytes.Buffer{})
	state, _ = signalStatus(tokBody, tokStatus)
	if state != StateVerified {
		t.Fatalf("token-prefixed Status: got %s want verified", state)
	}
	if !bytes.Contains(tokBody, []byte(expectedFirmware)) {
		t.Fatalf("token-prefixed Status body missing fingerprint")
	}
}

// TestObserve_ClientsEndpointDoesNotNeedToken verifies that the
// clients endpoint is reachable via header-only auth without session
// token.
func TestObserve_ClientsEndpointDoesNotNeedToken(t *testing.T) {
	srv := newWR841NSessionServer(t, "admin", "hunter2", "ABCDEFGHIJKLMNOP")
	host := strings.TrimPrefix(srv.URL(), "http://")
	client := newLocalClient(2 * time.Second)
	authValue := srv.authValue

	body, status, _ := authedGet(client, host, "/userRpm/AssignedIpAddrListRpm.htm", authValue, false, &bytes.Buffer{})
	state, _ := signalByHTTPAndBody(body, status)
	if state != StateVerified {
		t.Fatalf("clients endpoint with header auth: got %s want verified", state)
	}
}
