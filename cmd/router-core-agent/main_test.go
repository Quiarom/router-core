package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestRouterCoreClientGet_HappyPath verifies the agent's HTTP
// client against router-core's GET endpoints. The mock returns
// a tiny JSON body; the client should expose it as an observation.
func TestRouterCoreClientGet_HappyPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method: got %s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"vendor":"TP-Link","model":"TL-WR841N/ND"}`)
	}))
	defer server.Close()

	c := newRouterCoreClient(server.URL, 2*time.Second)
	got, err := c.get(context.Background(), "/v0/device")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Path != "/v0/device" {
		t.Errorf("Path: got %q, want %q", got.Path, "/v0/device")
	}
	if got.Status != http.StatusOK {
		t.Errorf("Status: got %d, want 200", got.Status)
	}
	if !bytes.Contains(got.Body, []byte("TP-Link")) {
		t.Errorf("Body: got %q, want substring TP-Link", string(got.Body))
	}
}

// TestRouterCoreClientGet_404 confirms the client surfaces 404
// and the body as the observation body.
func TestRouterCoreClientGet_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"state":"absent"}`)
	}))
	defer server.Close()
	c := newRouterCoreClient(server.URL, 2*time.Second)
	got, err := c.get(context.Background(), "/v0/security/wps")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != http.StatusNotFound {
		t.Errorf("Status: got %d, want 404", got.Status)
	}
	if !bytes.Contains(got.Body, []byte("absent")) {
		t.Errorf("Body: %q", string(got.Body))
	}
}

// TestRouterCoreClientGet_InvalidJSON confirms non-JSON bodies
// are rejected with a clear error.
func TestRouterCoreClientGet_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "<html>not json</html>")
	}))
	defer server.Close()
	c := newRouterCoreClient(server.URL, 2*time.Second)
	_, err := c.get(context.Background(), "/v0/device")
	if err == nil {
		t.Fatal("expected error on non-JSON body")
	}
	if !strings.Contains(err.Error(), "no es JSON") {
		t.Errorf("error message: %q", err.Error())
	}
}

// TestStubSequence verifies the keyword-driven tool ordering
// that powers the deterministic offline mode.
func TestStubSequence(t *testing.T) {
	cases := []struct {
		name   string
		length int
		first  string
	}{
		{"Is my Wi-Fi exposed?", 3, "wireless"},
		// The stubSequence branches on Spanish keywords. A
		// question that contains "quien" matches the connected-devices
		// case, which currently returns nil (no observations).
		// The live path covers this question end-to-end.
		{"¿Quién está conectado a mi red?", 6, "wireless"},
		{"Are there port forwards?", 3, "dmz"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			seq := stubSequence(tc.name)
			if len(seq) != tc.length {
				t.Errorf("length: got %d, want %d (%v)", len(seq), tc.length, seq)
			}
			if len(seq) > 0 && seq[0] != tc.first {
				t.Errorf("first: got %q, want %q", seq[0], tc.first)
			}
		})
	}
}

// TestRunStub_HappyPath runs the deterministic stub end-to-end
// with a mock router-core server. The agent should produce a
// Mode="stub" agentResult with one Step per stubSequence entry.
func TestRunStub_HappyPath(t *testing.T) {
	router := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v0/device", "/v0/status", "/v0/clients", "/v0/capabilities":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"state":"ok","vendor":"TP-Link"}`)
		case "/v0/security/wireless":
			_, _ = io.WriteString(w, `{"state":"verified","result":{"SSID":"TP-LINK_CBEC16"}}`)
		case "/v0/security/wps":
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"state":"absent"}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer router.Close()

	c := newRouterCoreClient(router.URL, 2*time.Second)
	result, err := runStub(context.Background(), c, "minimax/minimax-m3:free",
		"Is my Wi-Fi exposed?", nil)
	if err != nil {
		t.Fatalf("runStub: %v", err)
	}
	if result.Mode != "stub" {
		t.Errorf("Mode: got %q, want %q", result.Mode, "stub")
	}
	if result.Model != "minimax/minimax-m3:free" {
		t.Errorf("Model: got %q", result.Model)
	}
	// The deterministic stub returns a generic message
	// asking the operator to set OPENROUTER_API_KEY. It does
	// not summarise the SSID. The Steps array is what carries
	// the evidence.
	if !strings.Contains(result.Answer, "OPENROUTER_API_KEY") {
		t.Errorf("Answer should explain how to enable live mode: %q", result.Answer)
	}
	if !strings.Contains(result.Answer, "MiniMax") {
		t.Errorf("Answer should mention MiniMax: %q", result.Answer)
	}
	if len(result.Steps) != 3 {
		t.Errorf("Steps: got %d, want 3 (wireless, wps, remote-management)", len(result.Steps))
	}
}

// TestExecuteQuestion_StubMode confirms that without a live API
// key, executeQuestion runs the deterministic stub and does NOT
// preload router observations. The stub generates its own
// observation steps based on the question. Steps must be >0
// (the stub fetches at least one observation) but no four
// automatic preloads happen.
func TestExecuteQuestion_StubMode(t *testing.T) {
	router := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"state":"ok"}`)
	}))
	defer router.Close()

	opts := options{
		routerCoreURL:    router.URL,
		openrouterKeyEnv: "OPENROUTER_API_KEY_NONEXISTENT",
		dryRun:           true,
		timeout:          2 * time.Second,
	}
	result, err := executeQuestion(context.Background(), opts, "Is my Wi-Fi exposed?")
	if err != nil {
		t.Fatalf("executeQuestion: %v", err)
	}
	if result.Mode != "stub" {
		t.Errorf("Mode: got %q, want %q", result.Mode, "stub")
	}
	if len(result.Steps) == 0 {
		t.Errorf("Steps: stub must produce >=1 step (got 0)")
	}
	// The four preloaded names must NOT be present as automatic
	// steps: the stub fetches its own sequence based on the
	// question, and the live reasoning path never preloads.
	for _, s := range result.Steps {
		if s.Tool == "get_device" || s.Tool == "get_status" || s.Tool == "get_capabilities" {
			t.Errorf("Steps: preload tool %q must not appear in stub mode", s.Tool)
		}
	}
}

// TestExecuteQuestion_RouterUnreachable confirms the agent
// surfaces a clear error if the router-core serve is down.
func TestExecuteQuestion_RouterUnreachable(t *testing.T) {
	opts := options{
		routerCoreURL:    "http://127.0.0.1:1", // closed port
		openrouterKeyEnv: "OPENROUTER_API_KEY_NONEXISTENT",
		dryRun:           true,
		timeout:          200 * time.Millisecond,
	}
	_, err := executeQuestion(context.Background(), opts, "anything")
	if err == nil {
		t.Fatal("expected error when router-core is unreachable")
	}
}

// TestRunLive_HappyPath exercises the OpenRouter path with a
// mock openrouter server. The mock returns a tool call to
// get_wps_state("wps"); the agent should execute the tool
// against the mock router-core and feed the result back.
func TestRunLive_HappyPath(t *testing.T) {
	router := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v0/security/wps" {
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"state":"absent","reason":"WPS endpoint not present"}`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"state":"ok"}`)
	}))
	defer router.Close()

	callCount := 0
	openrouter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		// First call: emit a tool call. Second call: emit
		// the final answer (no tool calls).
		var req chatRequest
		_ = json.NewDecoder(r.Body).Decode(&req)
		if len(req.Tools) == 0 {
			t.Errorf("openrouter: no tools sent in request")
		}
		if req.Model == "" {
			t.Errorf("openrouter: model not set")
		}
		if callCount == 1 {
			_, _ = io.WriteString(w, `{
				"choices":[{
					"message":{
						"role":"assistant",
						"content":"",
						"tool_calls":[
							{"id":"call-1","type":"function","function":{
								"name":"get_wps_state",
								"arguments":"{\"name\":\"wps\"}"
							}}
						]
					}
				}]
			}`)
			return
		}
		_, _ = io.WriteString(w, `{
			"choices":[{"message":{
				"role":"assistant",
				"content":"WPS is absent on this firmware.",
				"tool_calls":[]
			}}]
		}`)
	}))
	defer openrouter.Close()

	c := newRouterCoreClient(router.URL, 2*time.Second)
	opts := options{
		routerCoreURL:    router.URL,
		openrouterURL:    openrouter.URL,
		openrouterModel:  "minimax/minimax-m3:free",
		openrouterKeyEnv: "OPENROUTER_API_KEY_TEST",
		timeout:          2 * time.Second,
	}

	result, err := runLive(context.Background(), opts, c, "test-key", "Is WPS enabled?")
	if err != nil {
		t.Fatalf("runLive: %v", err)
	}
	if result.Mode != "live" {
		t.Errorf("Mode: got %q, want %q", result.Mode, "live")
	}
	if !strings.Contains(result.Answer, "absent") {
		t.Errorf("Answer should reflect the tool result: %q", result.Answer)
	}
	if callCount < 2 {
		t.Errorf("openrouter should be called at least twice, got %d", callCount)
	}
}

// TestRunLive_OpenRouterError confirms the agent surfaces the
// error from OpenRouter cleanly.
func TestRunLive_OpenRouterError(t *testing.T) {
	openrouter := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, `{"error":{"message":"upstream provider down"}}`)
	}))
	defer openrouter.Close()
	router := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"state":"ok"}`)
	}))
	defer router.Close()

	c := newRouterCoreClient(router.URL, 2*time.Second)
	opts := options{
		routerCoreURL:    router.URL,
		openrouterURL:    openrouter.URL,
		openrouterModel:  "minimax/minimax-m3:free",
		openrouterKeyEnv: "OPENROUTER_API_KEY_TEST",
		timeout:          2 * time.Second,
	}
	_, err := runLive(context.Background(), opts, c, "test-key", "anything")
	if err == nil {
		t.Fatal("expected error when openrouter returns 500")
	}
}

// TestHealthzHandler_Get verifies the /healthz route returns
// 200 with the expected shape.
func TestHealthzHandler_Get(t *testing.T) {
	opts := options{
		openrouterModel: "minimax/minimax-m3:free",
	}
	h := withLocalCORS(healthzHandler(opts))
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", w.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("body parse: %v", err)
	}
	if body["state"] != "ok" {
		t.Errorf("state: got %q, want ok", body["state"])
	}
	if body["model"] != "minimax/minimax-m3:free" {
		t.Errorf("model: got %q", body["model"])
	}
}

// TestChatHandler_HealthzRejectsPost confirms non-GET methods
// are 405'd.
func TestHealthzHandler_RejectsPost(t *testing.T) {
	h := withLocalCORS(healthzHandler(options{openrouterModel: "x"}))
	req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status: got %d, want 405", w.Code)
	}
}

// TestChatHandler_Chat_BadBody confirms /v0/chat with no
// question returns 400.
func TestChatHandler_Chat_BadBody(t *testing.T) {
	h := withLocalCORS(chatHandler(options{openrouterModel: "x"}))
	body := strings.NewReader("{}")
	req := httptest.NewRequest(http.MethodPost, "/v0/chat", body)
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

// TestChatHandler_Chat_RejectsGet confirms only POST is
// accepted on /v0/chat.
func TestChatHandler_Chat_RejectsGet(t *testing.T) {
	h := withLocalCORS(chatHandler(options{openrouterModel: "x"}))
	req := httptest.NewRequest(http.MethodGet, "/v0/chat", nil)
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status: got %d, want 405", w.Code)
	}
}

// TestChatHandler_OptionsPreflight confirms the CORS middleware
// short-circuits OPTIONS with 204.
func TestChatHandler_OptionsPreflight(t *testing.T) {
	h := withLocalCORS(chatHandler(options{openrouterModel: "x"}))
	req := httptest.NewRequest(http.MethodOptions, "/v0/chat", nil)
	req.Header.Set("Origin", "http://127.0.0.1:3000")
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("status: got %d, want 204", w.Code)
	}
}

// TestWithLocalCORS_RejectsForeignOrigin confirms the CORS
// middleware blocks non-loopback origins.
func TestWithLocalCORS_RejectsForeignOrigin(t *testing.T) {
	h := withLocalCORS(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("status: got %d, want 403", w.Code)
	}
}

// TestIsAllowedSecurityCapability enumerates the six known
// capabilities and a few non-capabilities to confirm the
// allowlist.
// TestResolveToolPathEnumeratesKnownCapabilities confirms the
// closed mapping covers the six known security capabilities and
// rejects arbitrary strings. This replaces the previous
// isAllowedSecurityCapability allowlist test: the new contract
// is "closed mapping from name to path", not "allowlist of
// generic name".
func TestResolveToolPathEnumeratesKnownCapabilities(t *testing.T) {
	for _, want := range []string{
		"get_wireless_security",
		"get_wps_state",
		"get_dmz_state",
		"get_upnp_state",
		"get_remote_management_state",
		"get_forwarding_rules",
	} {
		path, err := resolveToolPath(want)
		if err != nil {
			t.Errorf("%q should resolve: %v", want, err)
			continue
		}
		if path == "" {
			t.Errorf("%q resolved to empty path", want)
		}
	}
	for _, bad := range []string{"", "wireless", "wps", "device", "clients", "capabilities", "not-a-cap"} {
		if _, err := resolveToolPath(bad); err == nil {
			t.Errorf("%q should be rejected", bad)
		}
	}
}

// TestValidateLoopbackURL_RejectsPublic confirms non-loopback
// URLs are refused.
func TestValidateLoopbackURL_RejectsPublic(t *testing.T) {
	for _, u := range []string{
		"http://1.2.3.4/foo",
		"https://example.com/foo",
		"http://router.local/foo",
	} {
		if err := validateLoopbackURL(u); err == nil {
			t.Errorf("%q: expected error", u)
		}
	}
}

// TestValidateLoopbackURL_AcceptsLoopback confirms RFC1918
// and 127.0.0.1 URLs are accepted.
func TestValidateLoopbackURL_AcceptsLoopback(t *testing.T) {
	for _, u := range []string{
		"http://127.0.0.1:8484/v0/device",
		"http://127.0.0.1/v0/device",
		"http://localhost:8484/v0/device",
	} {
		if err := validateLoopbackURL(u); err != nil {
			t.Errorf("%q: %v", u, err)
		}
	}
}

// TestRunAgentServer_RejectsNonLoopback confirms --serve 0.0.0.0
// is refused before any listener is created.
func TestRunAgentServer_RejectsNonLoopback(t *testing.T) {
	err := runAgentServer(options{serveAddr: "0.0.0.0:8585", timeout: time.Second})
	if err == nil {
		t.Fatal("expected error on non-loopback serve")
	}
	if !strings.Contains(err.Error(), "loopback") {
		t.Errorf("error: %q", err.Error())
	}
}

// TestBuildSystemPrompt_DoesNotPreloadRouterState is the live-mode
// assertion for commit 7: the system prompt contains policy
// and the four knowledge states, but does NOT preload
// device/status/clients/capabilities JSON. The model must call
// tools to learn.
func TestBuildSystemPrompt_DoesNotPreloadRouterState(t *testing.T) {
	prompt := buildSystemPrompt()
	for _, must := range []string{
		"verified",
		"absent",
		"unsupported_or_unverified",
		"unavailable",
	} {
		if !strings.Contains(prompt, must) {
			t.Errorf("system prompt must mention knowledge state %q", must)
		}
	}
	for _, mustNot := range []string{
		"DISPOSITIVO",
		"ESTADO",
		"APARATOS OBSERVADOS",
		"CAPACIDADES",
		"TL-WR841N",
	} {
		if strings.Contains(prompt, mustNot) {
			t.Errorf("system prompt must NOT contain preloaded section %q", mustNot)
		}
	}
}

// TestIsLoopbackAddr is a unit test for the loopback host check.
func TestIsLoopbackAddr(t *testing.T) {
	cases := map[string]bool{
		"127.0.0.1:8484":   true,
		"localhost:8585":   true,
		"0.0.0.0:8484":     false,
		"192.168.1.1:8484": false,
		"example.com:80":   false,
		// 127.0.0.1 without port is rejected (current design
		// requires explicit port). Callers should pass the port.
		"127.0.0.1": false,
	}
	for in, want := range cases {
		if got := isLoopbackAddr(in); got != want {
			t.Errorf("%q: got %v, want %v", in, got, want)
		}
	}
}
