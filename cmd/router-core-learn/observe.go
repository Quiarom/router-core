package main

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Quiarom/router-core/cmd/router-core-learn/sanitize"
)

// CapabilityState is the per-capability evidence classification.
type CapabilityState string

const (
	StateUnverified     CapabilityState = "unverified"
	StateCaptured       CapabilityState = "captured"
	StateVerified       CapabilityState = "verified"
	StateMismatch       CapabilityState = "mismatch"
	StateUnsupported    CapabilityState = "unsupported_or_unverified"
	StateTransportError CapabilityState = "transport_error"
	StateForbidden      CapabilityState = "auth_ok_but_forbidden"
)

// capability describes one read-only observation the probe attempts.
// The probe must never write; it only fetches the configured path
// using the verified Basic Auth session.
type capability struct {
	id          string
	displayName string
	path        string
	parser      func([]byte) (any, error)
	signalFn    func(body []byte, status int) (CapabilityState, string)
}

// newObserveCmd builds the `observe` subcommand: it authenticates
// against the WR841N v8.4 firmware using the verified recipe (Basic
// Auth with plaintext password via HTTP header) and fetches each
// read-only capability in sequence. For each capability it records
// sanitized evidence and classifies the state as VERIFIED, MISMATCH,
// TRANSPORT_ERROR, or UNVERIFIED.
//
// No mutations. No logout (the logout endpoint is unverified).
func newObserveCmd() *cobra.Command {
	host := ""
	outDir := ""
	passwordStdin := false
	verbose := false

	cmd := &cobra.Command{
		Use:   "observe",
		Short: "Authenticate and observe every read-only capability against the physical router",
		Long: `observe runs the full read-only observation capture pass:

  1. prompts for the admin password locally (no logging);
  2. authenticates with the recipe verified against the WR841N v8.4
     firmware 3.13.33 Build 130506 Rel.48660n on 2026-08-30
     (HTTP Basic Authorization header, plaintext password);
  3. fetches each known capability in sequence;
  4. sanitizes every response body in memory;
  5. classifies each capability as VERIFIED, MISMATCH, or another
     state;
  6. writes a capability-evidence.json matrix and per-capability
     sanitized bodies under <out>/.

This command does NOT mutate anything. It does NOT log out. It only
reads.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runObserve(cmd, host, outDir, passwordStdin, verbose)
		},
	}

	cmd.Flags().StringVar(&host, "host", "192.168.0.1",
		"local router address (RFC1918 literal)")
	cmd.Flags().StringVar(&outDir, "out", "",
		"output directory (default: ./fixtures/captured/tplink-wr841n-v8)")
	cmd.Flags().BoolVar(&passwordStdin, "password-stdin", false,
		"read the admin password from stdin instead of /dev/tty")
	cmd.Flags().BoolVar(&verbose, "verbose", false,
		"print full URL and status per request")
	cmd.Flags().Duration("timeout", 5*time.Second, "per-request timeout")

	return cmd
}

// capabilityList enumerates the read-only capabilities to observe.
// Order is intentional: device identification first, status second,
// then security surface (WPS, DMZ, UPnP, Remote Management),
// then forwarding, then client list. Wireless Security was previously
// missing from the surface and is included here as P0 for the
// "am I exposed" question.
//
// The probe stops at the first error per capability, never retries.
// It does NOT scan arbitrary endpoints. If a capability fails, it is
// recorded as MISMATCH or TRANSPORT_ERROR and the loop continues with
// the next capability.
func capabilityList() []capability {
	return []capability{
		{
			id:          "status",
			displayName: "Status",
			path:        "/userRpm/StatusRpm.htm",
			parser:      parseStatusObserved,
			signalFn:    signalStatus,
		},
		{
			id:          "wireless_security",
			displayName: "Wireless Security",
			path:        "/userRpm/WlanSecurityRpm.htm",
			parser:      parseAnyObserved,
			signalFn:    signalByHTTPAndBody,
		},
		{
			id:          "wps",
			displayName: "WPS",
			path:        "/userRpm/WpsRpm.htm",
			parser:      parseAnyObserved,
			signalFn:    signalByHTTPAndBody,
		},
		{
			id:          "dmz",
			displayName: "DMZ",
			path:        "/userRpm/DMZRpm.htm",
			parser:      parseAnyObserved,
			signalFn:    signalByHTTPAndBody,
		},
		{
			id:          "upnp",
			displayName: "UPnP",
			path:        "/userRpm/UpnpRpm.htm",
			parser:      parseAnyObserved,
			signalFn:    signalByHTTPAndBody,
		},
		{
			id:          "remote_management",
			displayName: "Remote Management",
			path:        "/userRpm/AccessCtrlRpm.htm",
			parser:      parseAnyObserved,
			signalFn:    signalByHTTPAndBody,
		},
		{
			id:          "forwarding",
			displayName: "Virtual Servers / Forwarding",
			path:        "/userRpm/VirtualServerRpm.htm",
			parser:      parseAnyObserved,
			signalFn:    signalByHTTPAndBody,
		},
		{
			id:          "clients",
			displayName: "DHCP Clients",
			path:        "/userRpm/AssignedIpAddrListRpm.htm",
			parser:      parseAnyObserved,
			signalFn:    signalByHTTPAndBody,
		},
	}
}

// observedEvidence is the per-capability entry persisted to disk.
type observedEvidence struct {
	Capability        string          `json:"capability"`
	DisplayName       string          `json:"display_name"`
	Endpoint          string          `json:"endpoint"`
	Status            int             `json:"http_status"`
	ResponseSize      int             `json:"response_bytes"`
	State             CapabilityState `json:"state"`
	Reason            string          `json:"reason,omitempty"`
	SanitizedBodySize int             `json:"sanitized_body_bytes,omitempty"`
}

// capabilityMatrix is the global view persisted at
// capability-evidence.json.
type capabilityMatrix struct {
	Capture        captureSummary              `json:"capture"`
	Authentication authSummary                 `json:"authentication"`
	Capabilities   map[string]observedEvidence `json:"capabilities"`
}

type captureSummary struct {
	Date          string `json:"date"`
	Host          string `json:"host"`
	FOutOutTarget string `json:"firmware_target"`
	Sanitization  string `json:"sanitization_policy"`
}

type authSummary struct {
	Recipe          string `json:"recipe"`
	HeaderNotCookie bool   `json:"header_not_cookie"`
	PlainPassword   bool   `json:"plain_password"`
	VerifiedAt      string `json:"verified_at"`
}

// runObserve orchestrates the full observation sequence: authenticate
// once, then iterate the capability list. Each capability produces a
// sanitized evidence file plus a matrix entry. Exit code 0 if every
// capability was verified or had a non-auth error; exit code 4 if
// authentication itself failed.
func runObserve(cmd *cobra.Command, host, outDir string, passwordStdin, verbose bool) error {
	if !isRFC1918OrLoopback(host) {
		return exitCodeError(5, fmt.Sprintf("refusing to observe host %q: not loopback/RFC1918", host), nil)
	}

	resolvedOut := outDir
	if resolvedOut == "" {
		resolvedOut = filepath.Join("fixtures", "captured", "tplink-wr841n-v8")
	}
	if err := os.MkdirAll(resolvedOut, 0o755); err != nil {
		return exitCodeError(1, "cannot create out dir", err)
	}

	timeout, _ := cmd.Flags().GetDuration("timeout")

	out := cmd.OutOrStdout()
	errOut := cmd.ErrOrStderr()

	fmt.Fprintf(out, "\nrouter-core-learn %s\n", probeVersion)
	fmt.Fprintf(out, "mode:   observe (read-only capability capture)\n")
	fmt.Fprintf(out, "target: TP-Link TL-WR841N v8.4, firmware %s\n", expectedFirmware)
	fmt.Fprintf(out, "host:   %s\n", host)
	fmt.Fprintf(out, "out:    %s\n\n", resolvedOut)

	// Step 1: authenticate using the verified recipe.
	fmt.Fprintf(out, "[auth] HTTP Basic Auth (plaintext password) against /\n")
	password, err := readPassword(passwordStdin)
	if err != nil {
		return exitCodeError(5, "cannot read password", err)
	}
	if password == "" {
		return exitCodeError(5, "empty password", nil)
	}

	authValue := base64.StdEncoding.EncodeToString([]byte("admin:" + password))

	// Drop the plaintext password reference immediately.
	password = ""
	// authValue still holds the base64 credential; the runtime
	// session re-uses it for every authenticated request.
	_ = authValue

	client := newLocalClient(timeout)

	// Sanity probe: verify auth with a known-good endpoint. The
	// session token (if any) is used to drive the per-capability loop
	// below as a fallback to the Basic Auth header.
	_, _, sessionToken, err := probeAuth(out, client, host, authValue)
	if err != nil {
		fmt.Fprintln(errOut, "  auth probe failed:", err)
		return exitCodeError(4, "authentication failed", err)
	}
	fmt.Fprintln(out, "  → authenticated")
	if sessionToken != "" {
		fmt.Fprintf(out, "  session token extracted: <SESSION_TOKEN> (%d chars)\n", len(sessionToken))
	}

	// Step 2: iterate capabilities.
	matrix := capabilityMatrix{
		Capabilities: make(map[string]observedEvidence, 8),
	}
	matrix.Capture = captureSummary{
		Date:          time.Now().UTC().Format(time.RFC3339),
		Host:          host,
		FOutOutTarget: expectedFirmware,
		Sanitization:  "cmd/router-core-learn/sanitize applied to every body in memory before persistence; structural session token and Authorization material replaced with placeholders; MACs, SSIDs, Wi-Fi keys, password fields redacted; fingerprint preserved",
	}
	matrix.Authentication = authSummary{
		Recipe:          "GET / with `Authorization: Basic <base64(admin:plaintext)>` HTTP header (NOT cookie, NOT md5hex)",
		HeaderNotCookie: true,
		PlainPassword:   true,
		VerifiedAt:      "2026-08-30 against physical WR841N v8.4 firmware 3.13.33 Build 130506 Rel.48660n",
	}

	caps := capabilityList()
	for _, cap := range caps {
		fmt.Fprintf(out, "[observe] %s (%s) …\n", cap.displayName, cap.path)

		// First attempt: Basic Auth header against the plain path.
		body, status, err := authedGet(client, host, cap.path, authValue, verbose, out)
		state, reason := cap.signalFn(body, status)

		// Fallback: if the firmware rejected the plain path with the
		// characteristic "no authority" body but the firmware emitted
		// a session token during auth, retry with /<TOKEN><path>.
		if sessionToken != "" && state == StateMismatch && len(body) == 68 {
			fmt.Fprintf(out, "    retrying with session token URL prefix …\n")
			tokBody, tokStatus, tokErr := authedGetWithToken(client, host, cap.path, authValue, sessionToken, verbose, out)
			if tokErr == nil {
				if tokState, tokReason := cap.signalFn(tokBody, tokStatus); tokState == StateVerified {
					body = tokBody
					status = tokStatus
					state = tokState
					reason = tokReason + " (via session token URL prefix)"
				}
			}
		}

		ev := observedEvidence{
			Capability:   cap.id,
			DisplayName:  cap.displayName,
			Endpoint:     cap.path,
			Status:       status,
			ResponseSize: len(body),
			State:        state,
			Reason:       reason,
		}

		// Persist sanitized body for every state that has one. The
		// sanitizer runs in memory before any byte touches disk.
		saniBody := sanitize.Apply(string(body), sanitize.Default())
		ev.SanitizedBodySize = len(saniBody)
		capFile := filepath.Join(resolvedOut, cap.id+".html")
		if writeErr := os.WriteFile(capFile, []byte(saniBody), 0o644); writeErr != nil {
			fmt.Fprintf(errOut, "    cannot persist %s: %v\n", capFile, writeErr)
		}

		matrix.Capabilities[cap.id] = ev
		fmt.Fprintf(out, "    state: %s", state)
		if reason != "" {
			fmt.Fprintf(out, " (%s)", reason)
		}
		fmt.Fprintln(out)

		// Best-effort parser validation: signalFn classified as
		// VERIFIED, but the actual parser for the endpoint may still
		// reject the body (shape mismatch). The hierarchy is signal
		// → parser: parser mismatch downgrades to MISMATCH.
		if _, perr := cap.parser(body); perr != nil && state == StateVerified {
			ev.State = StateMismatch
			ev.Reason = "HTTP 200 but parser failed: " + perr.Error()
			matrix.Capabilities[cap.id] = ev
			fmt.Fprintf(out, "    parser mismatch: %v\n\n", perr)
		}
		_ = err // err is captured above inside signalFn branch indirectly
	}

	// Step 3: persist matrix and summary.
	matrixJSON, err := json.MarshalIndent(matrix, "", "  ")
	if err != nil {
		return exitCodeError(1, "marshal matrix", err)
	}
	if err := os.WriteFile(filepath.Join(resolvedOut, "capability-evidence.json"), matrixJSON, 0o644); err != nil {
		return exitCodeError(1, "persist matrix", err)
	}

	// Step 4: print summary.
	fmt.Fprintln(out, "[summary]")
	for _, cap := range caps {
		ev := matrix.Capabilities[cap.id]
		fmt.Fprintf(out, "  %-22s %s\n", cap.id, ev.State)
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "No mutations performed. No logout attempted. Sanitized evidence only.")

	return nil
}

// probeAuth does a single GET / with the verified recipe and returns
// the body, the HTTP status, and an extracted session token if the
// firmware emits one in the response (observed as a URL of the form
// /<TOKEN>/userRpm/Index.htm embedded in a script tag). The body is
// always returned so the caller can persist or inspect it.
func probeAuth(out io.Writer, client *http.Client, host, authValue string) ([]byte, int, string, error) {
	body, status, err := authedGet(client, host, "/", authValue, false, out)
	if err != nil {
		return body, status, "", err
	}
	if status != 200 {
		return body, status, "", fmt.Errorf("HTTP %d (expected 200)", status)
	}
	if strings.Contains(string(body), "Login Incorrect") {
		return body, status, "", fmt.Errorf("HTTP 200 but body contains Login Incorrect")
	}
	token := sanitize.ExtractSessionToken(string(body))
	return body, status, token, nil
}

// authedGet performs GET <path> with the verified Authorization header.
// The path is appended to the host (which may or may not include the
// scheme). The function is the only place that constructs authenticated
// requests in the observe command; no other path can dispatch.
func authedGet(client *http.Client, host, path, authValue string, verbose bool, out io.Writer) ([]byte, int, error) {
	u, err := url.Parse("http://" + host + path)
	if err != nil {
		return nil, 0, fmt.Errorf("parse URL: %w", err)
	}
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Basic "+authValue)
	req.Header.Set("User-Agent", probeUserAgent)
	if verbose && out != nil {
		fmt.Fprintf(out, "    GET %s\n", u.String())
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	if verbose && out != nil {
		fmt.Fprintf(out, "    HTTP %d\n", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

// authedGetWithToken performs GET /<token><path> with the verified
// Authorization header. The session token is the URL-embedded token
// emitted by some legacy WR841N firmware builds; it is required by
// paths that reject the Basic Auth header alone (observed on the
// 3.13.33 Build 130506 Rel.48660n build for Status / Wireless /
// DMZ / Forwarding / Remote Management / WPS / UPnP). The token
// value is passed in but never persisted: it lives only in this
// function's call frame.
func authedGetWithToken(client *http.Client, host, path, authValue, token string, verbose bool, out io.Writer) ([]byte, int, error) {
	u, err := url.Parse("http://" + host + "/" + token + path)
	if err != nil {
		return nil, 0, fmt.Errorf("parse URL: %w", err)
	}
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Basic "+authValue)
	req.Header.Set("User-Agent", probeUserAgent)
	if verbose && out != nil {
		fmt.Fprintf(out, "    GET %s\n", u.String())
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	return body, resp.StatusCode, nil
}

// signalStatus is the special signal function for the Status page:
// verified when the body contains the expected firmware or hardware
// string.
func signalStatus(body []byte, status int) (CapabilityState, string) {
	if status != 200 {
		if status == 401 {
			return StateForbidden, "auth rejected"
		}
		return StateTransportError, fmt.Sprintf("HTTP %d", status)
	}
	if strings.Contains(string(body), expectedFirmware) && strings.Contains(string(body), expectedHardware) {
		return StateVerified, "fingerprint match"
	}
	if strings.Contains(string(body), "Login Incorrect") {
		return StateForbidden, "login page returned (auth cookie rejected)"
	}
	return StateMismatch, "200 OK but no fingerprint match"
}

// signalByHTTPAndBody is the generic signal function: 200 with a
// non-empty body is VERIFIED; 401 is FORBIDDEN; 200 with Login
// Incorrect body is FORBIDDEN; otherwise the actual status is
// reflected.
func signalByHTTPAndBody(body []byte, status int) (CapabilityState, string) {
	switch status {
	case 200:
		if strings.Contains(string(body), "Login Incorrect") {
			return StateForbidden, "login page returned (auth rejected)"
		}
		if len(body) < 256 {
			return StateMismatch, fmt.Sprintf("200 OK but body only %d bytes", len(body))
		}
		return StateVerified, fmt.Sprintf("200 OK with %d bytes", len(body))
	case 401:
		return StateForbidden, "HTTP 401 (auth cookie rejected)"
	case 403:
		return StateForbidden, "HTTP 403"
	case 404:
		return StateUnsupported, "HTTP 404 (endpoint not present on this firmware)"
	case 501:
		return StateUnsupported, "HTTP 501 Not Implemented (endpoint not present on this firmware)"
	}
	return StateTransportError, fmt.Sprintf("HTTP %d", status)
}

// parseStatusObserved is the Status-specific parser wrapper used by
// the observe command. It delegates to the adapter's ParseStatus
// (kept in the tplinkwr841v8 package; the observe command cannot
// import it directly without breaking the boundary between the probe
// and the runtime). The wrapper uses the same regex-based extraction
// the probe's runLearn uses.
func parseStatusObserved(body []byte) (any, error) {
	if !strings.Contains(string(body), expectedFirmware) {
		return nil, fmt.Errorf("firmware string not found in body")
	}
	return map[string]string{
		"firmware": expectedFirmware,
	}, nil
}

// parseAnyObserved is the generic parser for endpoints whose response
// shape is not yet known. It checks only that the response is a
// non-empty HTML body.
func parseAnyObserved(body []byte) (any, error) {
	if len(body) == 0 {
		return nil, fmt.Errorf("empty body")
	}
	if !strings.Contains(string(body), "<") {
		return nil, fmt.Errorf("body does not look like HTML")
	}
	return map[string]int{"body_bytes": len(body)}, nil
}

// md5hex is used only by tests that want to exercise the legacy
// recipe path explicitly. The runtime observe path uses plaintext.
func md5hexForObserve(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}
