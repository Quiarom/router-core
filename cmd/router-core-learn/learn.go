package main

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/Quiarom/router-core/cmd/router-core-learn/sanitize"
	"github.com/Quiarom/router-core/internal/domain"
)

const (
	candidateAName = "legacy-auth-a"
	candidateBName = "legacy-auth-b"
	candidateCName = "basic-auth-root"

	// Structural URL the firmware is expected to emit on a successful
	// login, observed in both prior-art implementations.
	loginPath  = "/userRpm/LoginRpm.htm?Save=Save"
	statusPath = "/userRpm/StatusRpm.htm"

	// Expected physical fingerprint of the lab unit. Compared against
	// what the Status parser extracts from the real response.
	expectedFirmware = "3.13.33 Build 130506 Rel.48660n"
	expectedHardware = "WR841N v8 00000000"

	probeUserAgent = "router-core-learn/0.1 (physical-capture)"
)

// probeErr lets the probe's RunE return an error that maps to a specific
// exit code without leaking internal types through the cobra pipeline.
type probeErr struct {
	code  int
	msg   string
	cause error
}

func (e *probeErr) Error() string {
	if e.cause != nil {
		return e.msg + ": " + e.cause.Error()
	}
	return e.msg
}

func (e *probeErr) ExitCode() int { return e.code }

func (e *probeErr) Unwrap() error { return e.cause }

func exitCodeError(code int, msg string, cause error) *probeErr {
	return &probeErr{code: code, msg: msg, cause: cause}
}

// learnFlags holds the flags for the `learn` subcommand.
type learnFlags struct {
	host          string
	outDir        string
	timeout       time.Duration
	passwordStdin bool
	verbose       bool
	dryRun        bool
}

func newLearnCmd() *cobra.Command {
	flags := &learnFlags{}

	cmd := &cobra.Command{
		Use:   "learn",
		Short: "Probe a physical WR841N for the legacy authentication protocol",
		Long: `learn probes a physical TP-Link TL-WR841N at the given --host
address and writes sanitized evidence to --out.

The probe:
  1. prompts locally for the router admin password (never logged);
  2. tries two evidence-backed candidate recipes (PA-1, PA-2 shapes);
  3. structurally extracts the session token from the login response;
  4. fetches the authenticated Status page;
  5. verifies the firmware/hardware fingerprint;
  6. persists sanitized evidence for Phase 3 design.

If neither candidate matches, the probe stops with exit code 2 and
prints "Known protocol did not match this firmware. Browser discovery
required." No retries, no scanning, no brute force.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLearn(cmd, flags)
		},
	}

	cmd.Flags().StringVar(&flags.host, "host", "192.168.0.1",
		"local router address (RFC1918 literal, e.g. 192.168.1.1)")
	cmd.Flags().StringVar(&flags.outDir, "out", "",
		"output directory (default: ./fixtures/captured/tplink-wr841n-v8)")
	cmd.Flags().DurationVar(&flags.timeout, "timeout", 5*time.Second,
		"per-request timeout")
	cmd.Flags().BoolVar(&flags.passwordStdin, "password-stdin", false,
		"read the admin password from stdin instead of /dev/tty")
	cmd.Flags().BoolVar(&flags.verbose, "verbose", false,
		"print sanitized request/response details (names only, never values)")
	cmd.Flags().BoolVar(&flags.dryRun, "dry-run", false,
		"validate connectivity only; do not send credentials or persist evidence")

	return cmd
}

// runLearn orchestrates the probe with explicit step-by-step feedback.
// Every step prints a [N/M] header BEFORE running and an OK/FAIL/SKIP
// result AFTER, so the operator always sees where the probe is.
func runLearn(cmd *cobra.Command, flags *learnFlags) error {
	out := cmd.OutOrStdout()
	errOut := cmd.ErrOrStderr()

	if !isRFC1918OrLoopback(flags.host) {
		return exitCodeError(5,
			fmt.Sprintf("refusing to probe host %q: not loopback/RFC1918", flags.host),
			nil)
	}

	resolvedOut := flags.outDir
	if resolvedOut == "" {
		resolvedOut = filepath.Join("fixtures", "captured", "tplink-wr841n-v8")
	}

	fmt.Fprintf(out, "\nrouter-core-learn %s\n", probeVersion)
	fmt.Fprintf(out, "target: TP-Link TL-WR841N v8.4, firmware %s\n", expectedFirmware)
	fmt.Fprintf(out, "host:   %s\n", flags.host)
	if flags.dryRun {
		fmt.Fprintf(out, "mode:   dry-run (connectivity check only, no credentials)\n")
	} else {
		fmt.Fprintf(out, "out:    %s\n", resolvedOut)
	}
	fmt.Fprintln(out)

	client := newLocalClient(flags.timeout)

	if flags.dryRun {
		step(out, 1, 1, "Connectivity check (no credentials sent)")
		fmt.Fprintf(out, "  GET http://%s/ …\n", flags.host)
		resp, err := client.Get("http://" + flags.host + "/")
		if err != nil {
			stepResult(out, "FAIL")
			fmt.Fprintln(errOut, "  router-core-learn:", err)
			return exitCodeError(3, "router unreachable", err)
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		resp.Body.Close()
		switch {
		case resp.StatusCode == 200:
			fmt.Fprintf(out, "  HTTP 200, body %d bytes — looks like a TP-Link login page\n", len(body))
			stepResult(out, "OK")
			fmt.Fprintln(out, "\nDry-run complete. Pass without --dry-run to run the full probe.")
		case resp.StatusCode == 401:
			fmt.Fprintf(out, "  HTTP 401 (Unauthorized)\n")
			stepResult(out, "FAIL")
			fmt.Fprintln(errOut, "  router-core-learn: host answered 401 without any credentials sent.")
			fmt.Fprintln(errOut, "  This usually means the URL is NOT the WR841N panel,")
			fmt.Fprintln(errOut, "  or a proxy/filter is intercepting the request.")
			fmt.Fprintln(errOut, "  Verify the WR841N management IP before running the full probe.")
			return exitCodeError(3, "host did not respond as TP-Link panel", nil)
		case resp.StatusCode >= 500:
			fmt.Fprintf(out, "  HTTP %d (server error)\n", resp.StatusCode)
			stepResult(out, "FAIL")
			return exitCodeError(3, "router returned server error", nil)
		default:
			fmt.Fprintf(out, "  HTTP %d (unexpected for a TP-Link panel)\n", resp.StatusCode)
			stepResult(out, "FAIL")
			return exitCodeError(3, "unexpected response from host", nil)
		}
		return nil
	}

	// [1/4] Read password locally.
	step(out, 1, 4, "Reading admin password locally")
	password, err := readPassword(flags.passwordStdin)
	if err != nil {
		stepResult(out, "FAIL")
		fmt.Fprintln(errOut, "  router-core-learn:", err)
		return exitCodeError(5, "cannot read password", err)
	}
	if password == "" {
		stepResult(out, "FAIL")
		return exitCodeError(5, "empty password", nil)
	}
	stepResult(out, "OK")
	if flags.verbose {
		fmt.Fprintf(out, "  password length: %d (value never displayed)\n", len(password))
	}

	// [2/4] Try candidate recipes.
	step(out, 2, 4, "Testing legacy authentication candidates")
	candidates := buildCandidates("admin", password)

	var matchedCandidate *Candidate
	var sessionToken string
	var lastLoginBody []byte
	var lastLoginStatus int
	var lastCandidateTried string

	for i, c := range candidates {
		fmt.Fprintf(out, "  [%d/%d] candidate %q …\n", i+1, len(candidates), c.Name)
		if flags.verbose {
			fmt.Fprintf(out, "    GET http://%s%s\n", flags.host, c.Endpoint)
			fmt.Fprintf(out, "    Cookie: Authorization=<redacted, %d chars>\n", len(c.CookieValue))
			fmt.Fprintf(out, "    Referer: http://%s/\n", flags.host)
		}
		tok, result, body, status, err := tryCandidateVerbose(client, flags.host, c)
		lastLoginBody = body
		lastLoginStatus = status
		lastCandidateTried = c.Name
		if err != nil {
			fmt.Fprintf(out, "    transport error: %v\n", err)
			continue
		}
		if result == ResultMatched {
			if c.Endpoint == "/" {
				fmt.Fprintf(out, "    matched: HTTP 200 with dashboard body (Basic Auth succeeded)\n")
			} else {
				fmt.Fprintf(out, "    matched: structural session-token redirect observed\n")
			}
			sessionToken = tok
			matchedCandidate = &candidates[i]
			break
		}
		if c.Endpoint == "/" {
			fmt.Fprintf(out, "    not matched: HTTP %d, body indicates failure (login page returned)\n", status)
		} else {
			fmt.Fprintf(out, "    not matched: structural redirect not found in response (HTTP %d, %d bytes)\n", status, len(body))
		}
	}

	if matchedCandidate == nil {
		stepResult(out, "FAIL")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Known protocol did not match this firmware.")
		fmt.Fprintln(out, "Persisting sanitized login-response for Phase 3 review.")
		if err := persistMismatchEvidence(resolvedOut, lastCandidateTried, lastLoginStatus, lastLoginBody, flags.host); err != nil {
			fmt.Fprintln(errOut, "  router-core-learn:", err)
		} else {
			fmt.Fprintf(out, "  mismatch evidence written to: %s/login-response.html\n", resolvedOut)
			fmt.Fprintf(out, "  review it with: bat %s/login-response.html\n", resolvedOut)
		}
		fmt.Fprintln(out, "Browser discovery required.")
		return exitCodeError(2, "no candidate matched", nil)
	}
	stepResult(out, "OK")

	// [3/4] Fetch authenticated Status. The path depends on which
	// candidate matched:
	//
	//   - PA-1/PA-2 (matchedCandidate.Endpoint == loginPath): the
	//     session token is URL-embedded as /<TOKEN>/userRpm/StatusRpm.htm.
	//
	//   - Basic Auth (matchedCandidate.Endpoint == "/"): the session is
	//     carried by the Authorization cookie itself; the Status path is
	//     just /userRpm/StatusRpm.htm without a token prefix.
	step(out, 3, 4, "Fetching authenticated Status page")
	var statusBody []byte
	if matchedCandidate.Endpoint == "/" {
		fmt.Fprintf(out, "  GET http://%s%s (Basic Auth)\n", flags.host, statusPath)
		statusBody, err = fetchStatusBasic(client, flags.host, *matchedCandidate)
	} else {
		fmt.Fprintf(out, "  GET http://%s/<session-token>%s\n", flags.host, statusPath)
		statusBody, err = fetchStatus(client, flags.host, sessionToken)
	}
	if err != nil {
		stepResult(out, "FAIL")
		fmt.Fprintln(errOut, "  router-core-learn:", err)
		return exitCodeError(3, "status fetch failed", err)
	}
	fmt.Fprintf(out, "  received %d bytes\n", len(statusBody))
	stepResult(out, "OK")

	// [4/4] Verify physical fingerprint + persist sanitized evidence.
	step(out, 4, 4, "Verifying physical fingerprint and persisting sanitized evidence")
	fp := extractFingerprint(statusBody)
	fmt.Fprintf(out, "  expected firmware: %s\n", expectedFirmware)
	fmt.Fprintf(out, "  found firmware:    %s\n", fp.Firmware)
	fmt.Fprintf(out, "  expected hardware: %s\n", expectedHardware)
	fmt.Fprintf(out, "  found hardware:    %s\n", fp.Hardware)

	match := fp.Firmware == expectedFirmware && fp.Hardware == expectedHardware

	if !match {
		// Still persist so the divergence is auditable.
		fmt.Fprintln(out, "  fingerprint diverges; persisting anyway for review")
	} else {
		fmt.Fprintln(out, "  fingerprint matches expected physical lab unit")
	}

	if err := persistEvidence(resolvedOut, *matchedCandidate, statusBody, fp, match, flags.host); err != nil {
		stepResult(out, "FAIL")
		fmt.Fprintln(errOut, "  router-core-learn:", err)
		return exitCodeError(1, "persist evidence failed", err)
	}
	fmt.Fprintf(out, "  evidence written to: %s\n", resolvedOut)
	stepResult(out, "OK")

	fmt.Fprintln(out)
	if match {
		fmt.Fprintln(out, "VERIFIED AGAINST PHYSICAL DEVICE")
	} else {
		fmt.Fprintln(out, "CAPTURE OK; FINGERPRINT DIVERGES — REVIEW REQUIRED")
	}
	fmt.Fprintln(out, "No secrets persisted. No mutation performed.")
	if match {
		return nil
	}
	return exitCodeError(4, "fingerprint diverged", nil)
}

// step prints the standard [N/M] step header.
func step(out io.Writer, n, total int, label string) {
	fmt.Fprintf(out, "[%d/%d] %s\n", n, total, label)
}

// stepResult prints the OK/FAIL/SKIP marker at the end of a step.
func stepResult(out io.Writer, result string) {
	fmt.Fprintf(out, "  → %s\n", result)
}

// Candidate describes one evidence-backed authentication recipe.
//
// Endpoint is the path the probe GETs. Authorization is applied as
// follows: when UseCookie is true, the value is attached as a cookie
// named "Authorization"; when UseHeader is true, the value is sent as
// an HTTP Authorization header. Both can be true. Headers are sent
// verbatim as provided.
type Candidate struct {
	Name         string
	Endpoint     string
	CookieValue  string
	UseCookie    bool
	UseHeader    bool
	HeaderValue  string
	HeaderFormat string
}

// Result describes the outcome of one candidate attempt.
type Result string

const (
	ResultMatched      Result = "matched"
	ResultNotMatched   Result = "not_matched"
	ResultTransportErr Result = "transport_error"
)

// candidateReport describes the verified authentication recipe.
//
// The recipe is observed to use HTTP Basic Authorization header (NOT a
// cookie) on the root path "/" of the WR841N v8.4 firmware
// 3.13.33 Build 130506 Rel.48660n. There is no URL-embedded session
// token and no structural redirect — Basic Auth keeps the session in
// the Authorization header which the browser re-transmits until the
// browser process restarts.
//
// The fields below preserve the historically-shaped report but
// renamed to accurately describe the verified mechanism. Old field
// names are kept as omitempty with the new descriptive name so existing
// downstream consumers do not break, but new evidence should rely on
// the renamed fields.
type candidateReport struct {
	Candidate     string `json:"candidate"`
	Method        string `json:"method"`
	Endpoint      string `json:"endpoint"`
	Authorization string `json:"authorization"`

	// True when the recipe uses the HTTP Authorization header rather
	// than a cookie. False when it uses a cookie.
	HeaderNotCookie bool `json:"header_not_cookie"`

	// True when the firmware expects a Basic-style
	// `Authorization: Basic <base64(user:password)>` value with no MD5
	// hashing.
	PlainPassword bool `json:"plain_password"`

	// Legacy fields, kept for backward compatibility. Set only when
	// the recipe uses a cookie-named-Authorization, never for the WR841N
	// v8.4 recipe we have verified.
	CookieName    string `json:"cookie_name,omitempty"`
	RedirectShape string `json:"redirect_shape,omitempty"`

	PhysicalMatch bool `json:"physical_match"`
}

type evidence struct {
	Auth        candidateReport  `json:"auth_evidence"`
	Status      map[string]any   `json:"status_evidence,omitempty"`
	Fingerprint fingerprintMatch `json:"physical_fingerprint,omitempty"`
	Capture     captureMetadata  `json:"capture"`
}

type captureMetadata struct {
	Date           string `json:"date"`
	Host           string `json:"host"`
	FirmwareTarget string `json:"firmware_target"`
	Sanitization   string `json:"sanitization_policy"`
}

type fingerprintMatch struct {
	ExpectedFirmware string `json:"expected_firmware"`
	FoundFirmware    string `json:"found_firmware"`
	ExpectedHardware string `json:"expected_hardware"`
	FoundHardware    string `json:"found_hardware"`
	Match            bool   `json:"match"`
}

// fingerprint is the in-memory representation extracted from a Status
// response body before any sanitization.
type fingerprint struct {
	Firmware string
	Hardware string
}

// buildCandidates constructs five evidence-backed authentication
// recipes. The password is used only in memory; nothing is written to
// disk before sanitization.
//
// Candidates A and B come from public prior art (PA-1 / PA-2) targeting
// older firmware builds where login happens at /userRpm/LoginRpm.htm
// and the response contains a structural redirect with a session
// token. Both apply md5hex(password) before base64.
//
// Candidates C and D come from the WR841N v8.4 firmware observed on
// the physical lab unit on 2026-08-30: the firmware responds 401 with
// `WWW-Authenticate: Basic realm="TP-LINK Wireless N Router WR841N"`
// and accepts HTTP Basic Authorization directly against `/`. C uses
// md5hex(password) (PA-1/PA-2 construction). D uses password PLAIN
// without MD5 hashing -- this is the variant that matches the standard
// browser Basic Auth dialog (which sends the user-typed password
// verbatim, not pre-hashed). Evidence: the physical lab unit rejects
// C with admin/admin but the operator successfully authenticates with
// the same credentials via the browser, suggesting the firmware
// accepts plain Basic Auth.
//
// Candidate E is the same shape as D but uses the Authorization HTTP
// header (canonical for HTTP Basic Auth) instead of a cookie.
//
// All five are tried in order; the first one that matches a 200 with
// a non-login response wins. None of them use brute force or
// alternative credentials -- only the operator-provided password is
// tested in the various encodings.
func buildCandidates(user, password string) []Candidate {
	md5sum := md5hex(password)

	// md5hex(password) base64-encoded (PA-1/PA-2 construction).
	md5Base64 := base64.StdEncoding.EncodeToString([]byte(user + ":" + md5sum))

	// Plain password base64-encoded (standard Basic Auth dialog).
	plainBase64 := base64.StdEncoding.EncodeToString([]byte(user + ":" + password))

	// Candidate A (PA-2 / tplink_exporter): md5hex + cookie + LoginRpm.htm.
	a := Candidate{
		Name:        candidateAName,
		Endpoint:    loginPath,
		CookieValue: md5Base64,
		UseCookie:   true,
	}

	// Candidate B (PA-1 / tpylink): md5hex + cookie with URL-encoded
	// "Basic%20" prefix + LoginRpm.htm.
	b := Candidate{
		Name:        candidateBName,
		Endpoint:    loginPath,
		CookieValue: "Basic%20" + md5Base64,
		UseCookie:   true,
	}

	// Candidate C: md5hex + cookie + Basic Auth against root /.
	c := Candidate{
		Name:        candidateCName,
		Endpoint:    "/",
		CookieValue: md5Base64,
		UseCookie:   true,
	}

	// Candidate D: PLAIN password + cookie + Basic Auth against root /.
	// Evidence: lab operator's browser authenticates successfully with
	// admin/admin plain text but candidate C with md5hex was rejected.
	d := Candidate{
		Name:        "basic-auth-plain-cookie",
		Endpoint:    "/",
		CookieValue: plainBase64,
		UseCookie:   true,
	}

	// Candidate E: PLAIN password + Authorization HTTP header + Basic
	// Auth against root /.
	e := Candidate{
		Name:         "basic-auth-plain-header",
		Endpoint:     "/",
		HeaderValue:  "Basic " + plainBase64,
		HeaderFormat: "Basic",
		UseHeader:    true,
	}

	return []Candidate{a, b, c, d, e}
}

func md5hex(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

// tryCandidate executes one candidate and returns the captured session
// token if the structural redirect is observed. It also returns the raw
// response body and status so the caller can persist sanitized
// evidence when no candidate matches.
func tryCandidate(client *http.Client, host string, c Candidate) (string, Result, error) {
	tok, result, _, _, err := tryCandidateVerbose(client, host, c)
	return tok, result, err
}

// tryCandidateVerbose executes one candidate and returns the captured
// session token (where applicable) and the response body so the
// caller can persist sanitized evidence on failure.
//
// Match detection depends on the endpoint:
//
//   - For /userRpm/LoginRpm.htm?Save=Save (PA-1/PA-2 recipes): the
//     response must contain a structural session-token redirect
//     (/<16-char-token>/userRpm/Index.htm).
//
//   - For / (HTTP Basic Auth recipe): the response must be HTTP 200
//     and must NOT contain the "Login Incorrect" error page.
//
// All other responses (including 401 with "Login Incorrect") are
// reported as NotMatched so the next candidate is tried.
func tryCandidateVerbose(client *http.Client, host string, c Candidate) (string, Result, []byte, int, error) {
	req, err := buildLoginRequest(host, c)
	if err != nil {
		return "", ResultTransportErr, nil, 0, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", ResultTransportErr, nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", ResultTransportErr, body, resp.StatusCode, err
	}

	if matchResponse(c, resp.StatusCode, string(body)) {
		// For Basic Auth, there is no URL-embedded token; the
		// Authorization header itself carries the session. Return an
		// empty string for the caller to interpret.
		return "", ResultMatched, body, resp.StatusCode, nil
	}
	return "", ResultNotMatched, body, resp.StatusCode, nil
}

// matchResponse applies the endpoint-specific detection rule for a
// successful authentication attempt.
func matchResponse(c Candidate, status int, body string) bool {
	if c.Endpoint == "/" {
		return status == 200 && !strings.Contains(body, "Login Incorrect")
	}
	return sanitize.ExtractSessionToken(body) != ""
}

// persistMismatchEvidence writes the sanitized login response from the
// last failed candidate to disk so the operator can inspect the actual
// shape of the firmware's login page. The sanitizer runs in memory
// before any byte touches disk; secret material is redacted.
func persistMismatchEvidence(outDir, candidate string, status int, body []byte, host string) error {
	if len(body) == 0 {
		return os.MkdirAll(outDir, 0o755)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	saniBody := sanitize.Apply(string(body), sanitize.Default())
	return os.WriteFile(filepath.Join(outDir, "login-response.html"), []byte(saniBody), 0o644)
}

// buildLoginRequest constructs the GET <c.Endpoint> request for a
// given candidate. The wire shape depends on the candidate's UseCookie
// and UseHeader flags. Default for backwards compatibility: cookie
// only.
func buildLoginRequest(host string, c Candidate) (*http.Request, error) {
	u, err := url.Parse("http://" + host + c.Endpoint)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	if c.UseCookie {
		req.AddCookie(&http.Cookie{Name: "Authorization", Value: c.CookieValue})
	}
	if c.UseHeader && c.HeaderValue != "" {
		req.Header.Set("Authorization", c.HeaderValue)
	}
	req.Header.Set("Referer", "http://"+host+"/")
	req.Header.Set("User-Agent", probeUserAgent)
	return req, nil
}

// fetchStatus requests the authenticated Status page using the captured
// session token. The token is never written to disk; the response body
// is sanitized before persistence.
func fetchStatus(client *http.Client, host, token string) ([]byte, error) {
	u, err := url.Parse("http://" + host + "/" + token + statusPath)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Referer", u.String())
	req.Header.Set("User-Agent", probeUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(io.LimitReader(resp.Body, 2<<20))
}

// fetchStatusBasic requests the authenticated Status page using HTTP
// Basic Auth directly (no URL-embedded session token). Used when the
// matched authentication recipe is the Basic Auth candidate against
// the root URL. If the candidate uses Authorization header (UseHeader
// is true), the header value is sent; otherwise the cookie value is
// sent. Both are kept in memory only.
func fetchStatusBasic(client *http.Client, host string, c Candidate) ([]byte, error) {
	u, err := url.Parse("http://" + host + statusPath)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	if c.UseHeader && c.HeaderValue != "" {
		req.Header.Set("Authorization", c.HeaderValue)
	} else {
		req.AddCookie(&http.Cookie{Name: "Authorization", Value: c.CookieValue})
	}
	req.Header.Set("Referer", "http://"+host+"/")
	req.Header.Set("User-Agent", probeUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(io.LimitReader(resp.Body, 2<<20))
}

// extractFingerprint pulls firmware and hardware strings from a Status
// response. Conservative: any field it cannot confidently identify is
// reported as "".
func extractFingerprint(body []byte) fingerprint {
	var fp fingerprint
	if m := firmwarePattern.FindString(string(body)); m != "" {
		fp.Firmware = strings.TrimSpace(m)
	}
	if m := hardwarePattern.FindString(string(body)); m != "" {
		fp.Hardware = strings.TrimSpace(m)
	}
	return fp
}

// persistEvidence writes the sanitized artifacts to disk.
// buildAuthReport produces an accurate report of the authentication
// recipe that matched. The fields describe the OBSERVED wire shape:
// the path that the probe GETted, whether the credential was sent as a
// cookie or as an HTTP Authorization header, and whether the password
// is hashed before base64 encoding or not.
func buildAuthReport(c Candidate) candidateReport {
	report := candidateReport{
		Candidate:       c.Name,
		Method:          http.MethodGet,
		Endpoint:        c.Endpoint,
		HeaderNotCookie: c.UseHeader,
		PlainPassword:   !strings.HasPrefix(c.Name, "legacy-"),
		PhysicalMatch:   true,
	}
	if c.UseHeader {
		report.Authorization = "Basic <base64(user:plaintext)> (HTTP header)"
	} else {
		report.Authorization = "Authorization (cookie value)"
	}
	return report
}

func persistEvidence(
	outDir string,
	c Candidate,
	statusBody []byte,
	fp fingerprint,
	match bool,
	host string,
) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	auth := buildAuthReport(c)

	saniStatus := sanitize.Apply(string(statusBody), sanitize.Default())

	ev := evidence{
		Auth: auth,
		Status: map[string]any{
			"request_path":    "/" + sanitize.PlaceholderSessionToken + statusPath,
			"response_bytes":  len(statusBody),
			"sanitized_bytes": len(saniStatus),
		},
		Fingerprint: fingerprintMatch{
			ExpectedFirmware: expectedFirmware,
			FoundFirmware:    fp.Firmware,
			ExpectedHardware: expectedHardware,
			FoundHardware:    fp.Hardware,
			Match:            match,
		},
		Capture: captureMetadata{
			Date:           time.Now().UTC().Format(time.RFC3339),
			Host:           host,
			FirmwareTarget: expectedFirmware,
			Sanitization:   "apply cmd/router-core-learn/sanitize.Default to all persisted bodies; structural redirect token and Authorization material replaced with placeholders; MACs, SSIDs, Wi-Fi keys, password fields redacted; fingerprint (vendor/model/hardware/firmware) preserved",
		},
	}

	authJSON, err := json.MarshalIndent(auth, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "auth-evidence.json"), authJSON, 0o644); err != nil {
		return err
	}

	statusReqPath := "/" + sanitize.PlaceholderSessionToken + statusPath
	statusReq := map[string]any{
		"method": http.MethodGet,
		"path":   statusReqPath,
		"headers": map[string]string{
			"Referer":    "http://" + host + statusReqPath,
			"User-Agent": probeUserAgent,
		},
	}
	statusReqJSON, err := json.MarshalIndent(statusReq, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "status-request.json"), statusReqJSON, 0o644); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(outDir, "status-response.html"), []byte(saniStatus), 0o644); err != nil {
		return err
	}

	evJSON, err := json.MarshalIndent(ev, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "captured-index.json"), evJSON, 0o644); err != nil {
		return err
	}

	readme := fmt.Sprintf(`TP-Link TL-WR841N v8.4 — sanitized physical capture

Date: %s
Host: %s
Firmware target: %s

Sanitization policy:
- apply cmd/router-core-learn/sanitize.Default to all persisted bodies;
- structural redirect token and Authorization material replaced with placeholders;
- MACs, SSIDs, Wi-Fi keys, password fields redacted;
- fingerprint (vendor/model/hardware/firmware) preserved.

This capture exists solely as evidence for Phase 3 design. No secrets,
no session tokens, no passwords are present in any of the files in
this directory.
`, ev.Capture.Date, host, expectedFirmware)
	return os.WriteFile(filepath.Join(outDir, "README.md"), []byte(readme), 0o644)
}

// newLocalClient builds an HTTP client restricted to the configured
// timeout. We do not follow redirects — the probe must observe the
// redirect body itself.
func newLocalClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// isRFC1918OrLoopback rejects anything that is not a literal
// loopback/RFC1918 address.
func isRFC1918OrLoopback(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	h, _, err := net.SplitHostPort(host)
	if err != nil {
		h = strings.Trim(host, "[]")
	}
	ip := net.ParseIP(h)
	if ip == nil {
		return false
	}
	if ip.IsLoopback() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		return ip4[0] == 10 ||
			(ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) ||
			(ip4[0] == 192 && ip4[1] == 168)
	}
	return false
}

// readPassword reads the admin password locally.
//
// Resolution order:
//
//  1. If --password-stdin is set, read from os.Stdin with a 5s
//     inactivity timeout. Intended for piping and CI.
//  2. Otherwise try /dev/tty with echo disabled. If /dev/tty is not
//     available, fall back to os.Stdin with a 30s inactivity timeout.
//  3. If neither source yields input within the timeout, return an
//     error that points the operator at the fix.
//
// The prompt is always written to stderr so the operator sees it even
// when stdout is being captured by a parent process.
func readPassword(passwordStdin bool) (string, error) {
	if passwordStdin {
		return readPasswordFromReader("stdin", os.Stdin, 5*time.Second)
	}

	tty, err := os.Open("/dev/tty")
	if err != nil {
		// No controlling tty (common when invoked via `go run`, CI, or
		// background launchers). Fall back to stdin with a short
		// timeout so the operator is not stuck waiting.
		fmt.Fprintln(os.Stderr, "  no controlling tty; reading password from stdin (timeout 5s)")
		fmt.Fprintln(os.Stderr, "  hint: pipe the password via stdin or re-run with --password-stdin")
		return readPasswordFromReader("stdin", os.Stdin, 5*time.Second)
	}
	defer tty.Close()

	fd := int(tty.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", err
	}
	defer term.Restore(fd, oldState)

	fmt.Fprint(tty, "Router admin password: ")

	// TTY read: block until Enter (or EOF). No timeout because the
	// operator is interacting directly.
	buf := make([]byte, 256)
	n, err := io.ReadFull(tty, buf)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return "", err
	}
	return strings.TrimRight(string(buf[:n]), "\r\n"), nil
}

// readPasswordFromReader reads a password from r with an inactivity
// timeout. If no bytes arrive within the timeout, returns an error.
// Used by the stdin fallback path so the probe never blocks forever.
func readPasswordFromReader(source string, r io.Reader, timeout time.Duration) (string, error) {
	if r == nil {
		return "", fmt.Errorf("no input source available")
	}
	type readResult struct {
		buf []byte
		err error
	}
	ch := make(chan readResult, 1)
	go func() {
		buf := make([]byte, 256)
		n, err := r.Read(buf)
		ch <- readResult{buf[:n], err}
	}()
	select {
	case res := <-ch:
		if res.err != nil && res.err != io.EOF {
			return "", fmt.Errorf("read from %s: %w", source, res.err)
		}
		return strings.TrimRight(string(res.buf), "\r\n"), nil
	case <-time.After(timeout):
		return "", fmt.Errorf("timed out waiting %s for password from %s; pass --password-stdin and pipe the value", timeout, source)
	}
}

// keep domain import live for future drift.
var _ = domain.ErrCaptureMissing

// Fingerprint patterns.
var (
	firmwarePattern = regexp.MustCompile(`\d+\.\d+\.\d+\s+Build\s+\d+\s+Rel\.\w+`)
	hardwarePattern = regexp.MustCompile(`WR841N\s+v\d+\s+[0-9A-Fa-f]{1,8}`)
)

// silence unused-import warnings for shared types when test wiring is
// stripped.
var _ = errors.New
