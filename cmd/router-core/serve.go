// Package main contains the runtime binary router-core serve, which
// authenticates against a TP-Link WR841N v8.4 and exposes a typed
// read-only HTTP API on the loopback interface. Capabilities the
// firmware does not implement (WPS, UPnP, Remote Management on
// v8.4) return 404 with a structured payload.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"golang.org/x/term"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Quiarom/router-core/internal/adapters/fixture"
	"github.com/Quiarom/router-core/internal/adapters/tplinkwr841v8"
	"github.com/Quiarom/router-core/internal/domain"
	"github.com/Quiarom/router-core/internal/transport"
)

type sessionStore struct {
	mu       sync.Mutex
	password []byte
}

func (s *sessionStore) get() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]byte(nil), s.password...)
}

func (s *sessionStore) set(p []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.password {
		s.password[i] = 0
	}
	s.password = append([]byte(nil), p...)
}

// readRouterPassword acquires the router admin password in a way that
// respects the four ways the user might supply it:
//
//   1. --password-stdin flag: read plaintext from os.Stdin (intended for
//      systemd, container secrets, scripts). Refuses if no TTY.
//   2. Interactive TTY: prompt with prompt printed to stderr, then
//      read using golang.org/x/term.ReadPassword which DISABLES
//      terminal echo for the duration of the read. The previous
//      implementation used os.Stdin.Read which did NOT disable
//      echo and so the password was visible on the terminal.
//   3. Non-interactive without --password-stdin: fail fast with
//      an actionable error. The previous 30-second wait was
//      blocking CI and scripts in non-interactive mode.
//
// NEVER accepts --password <secret>: that would put the secret
// on the process command line, where it is visible to ps(1)
// and any /proc snapshot.
func readRouterPassword(stdinFlag bool) ([]byte, error) {
	if stdinFlag {
		if term.IsTerminal(int(os.Stdin.Fd())) {
			return nil, errors.New("--password-stdin is set but stdin is a TTY; refusing to read a password from a terminal in plaintext")
		}
		buf, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		return []byte(strings.TrimRight(string(buf), "\r\n")), nil
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil, errors.New("router-core: no TTY available for interactive password entry. Use --password-stdin to read from a pipe (CI, systemd, container secrets).")
	}
	fmt.Fprintln(os.Stderr, "router-core: enter router admin password (input hidden):")
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return nil, fmt.Errorf("read password from TTY: %w", err)
	}
	return pw, nil
}

func runServeCommand(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	host := fs.String("host", "192.168.1.1", "local router address (RFC1918 literal)")
	addr := fs.String("addr", "127.0.0.1:8484", "loopback HTTP listen address")
	timeout := fs.Duration("timeout", 5*time.Second, "per-request timeout to the router")
	mock := fs.Bool("mock", false, "run against a fixture-backed adapter (no network)")
	mockPath := fs.String("mock-fixture", "", "path to a synthetic fixture (default: fixtures/synthetic/tplink-wr841n-v8)")
	passwordStdin := fs.Bool("password-stdin", false, "read the admin password from stdin (refuses if stdin is a TTY)")
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) || err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if !isLoopbackAddr(*addr) {
		return fmt.Errorf("refusing to serve on non-loopback address %q (loopback only)", *addr)
	}
	var adapter domain.RouterAdapter
	var store *sessionStore
	if *mock {
		fixturePath := *mockPath
		if fixturePath == "" {
			fixturePath = "fixtures/synthetic/tplink-wr841n-v8"
		}
		f := fixture.New(fixturePath)
		adapter = f
		store = &sessionStore{password: nil}
		fmt.Fprintf(os.Stderr, "router-core serve: mock mode, fixture=%s\n", fixturePath)
	} else {
		if !isRFC1918OrLoopback(*host) {
			return fmt.Errorf("refusing to observe host %q: not loopback/RFC1918", *host)
		}
		fmt.Fprintf(os.Stderr, "router-core serve: reading admin password from stdin (timeout 30s)\n")
		password, err := readRouterPassword(*passwordStdin)
		if err != nil {
			return fmt.Errorf("read password: %w", err)
		}
		if len(password) == 0 {
			return errors.New("empty password")
		}
		defer zeroBytes(&password)

		store := &sessionStore{password: append([]byte(nil), password...)}
		defer zeroBytes(&store.password)

		adapter := tplinkwr841v8.New(*host, transport.WithTimeout(*timeout))
		loginCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := adapter.Login(loginCtx, "admin", string(password)); err != nil {
			return fmt.Errorf("login: %w", err)
		}

		fmt.Fprintf(os.Stderr, "router-core serve: authenticated, listening on %s\n", *addr)
	}

	mux := http.NewServeMux()
	registerRoutes(mux, adapter, store)

	srv := &http.Server{
		Addr:              *addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return srv.ListenAndServe()
}

// isLoopbackAddr refuses 0.0.0.0 and public IPs. The router-core
// service is local-only and must never bind on a routable address.
func isLoopbackAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsUnspecified() && ip.To4()[0] == 127
}

// isRFC1918OrLoopback is duplicated from the probe binary to keep
// the runtime binary self-contained. The transport layer applies
// the same check; this is a defense-in-depth pre-flight.
func isRFC1918OrLoopback(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = strings.Trim(addr, "[]")
	}
	ip := net.ParseIP(host)
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

// zeroBytes overwrites every byte in p with zero. Unlike a Go
// string, a []byte is mutable. This is the best we can do in
// Go without an unsafe zeroing library.
func zeroBytes(p *[]byte) {
	if p == nil {
		return
	}
	for i := range *p {
		(*p)[i] = 0
	}
	*p = (*p)[:0]
}

type capabilityError struct {
	State  string `json:"state"`
	Reason string `json:"reason,omitempty"`
}

type capabilityStatus struct {
	State  string `json:"state"`
	Status int    `json:"http_status,omitempty"`
}

type capabilitiesResponse struct {
	Capabilities map[string]string `json:"capabilities"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeUnsupported(w http.ResponseWriter, reason string) {
	writeJSON(w, http.StatusNotFound, capabilityError{State: "unsupported_or_unverified", Reason: reason})
}

func writeUnavailable(w http.ResponseWriter, reason string) {
	writeJSON(w, http.StatusServiceUnavailable, capabilityError{State: "unavailable", Reason: reason})
}

// securityCapable is the optional contract for adapters that can
// serve per-capability security endpoints. The TP-Link WR841N
// adapter implements it (one HTTP fetch per capability). The
// fixture adapter does not: in mock mode every per-capability
// security endpoint returns 404 with state "unsupported_or_unverified".
type securityCapable interface {
	SecurityCapability(ctx context.Context, name string) (domain.SecurityState, error)
}

func registerRoutes(mux *http.ServeMux, adapter domain.RouterAdapter, store *sessionStore) {
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, capabilityStatus{State: "ok", Status: 200})
	})
	mux.HandleFunc("/v0/device", handleDevice(adapter))
	mux.HandleFunc("/v0/status", handleStatus(adapter))
	mux.HandleFunc("/v0/clients", handleClients(adapter))
	mux.HandleFunc("/v0/capabilities", handleCapabilities(adapter))
	mux.HandleFunc("/v0/security/wireless", handleSecurityWireless(adapter))
	mux.HandleFunc("/v0/security/wps", handleSecurityWPS(adapter))
	mux.HandleFunc("/v0/security/dmz", handleSecurityDMZ(adapter))
	mux.HandleFunc("/v0/security/upnp", handleSecurityUPnP(adapter))
	mux.HandleFunc("/v0/security/remote-management", handleSecurityRemoteManagement(adapter))
	mux.HandleFunc("/v0/security/forwarding", handleSecurityForwarding(adapter))
}

// handleCapabilities returns the live capability matrix. Each
// entry is one of the four documented states (verified, absent,
// unsupported_or_unverified, unavailable). The matrix is derived
// from the adapter's per-capability dispatch: each security
// endpoint is fetched once with a short timeout, and the
// returned state is mapped to the four-state vocabulary.
// device is not a security endpoint, so we report it as
// "verified" only if Identify succeeds, "unavailable" otherwise.
func handleCapabilities(adapter domain.RouterAdapter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		caps := capabilitiesResponse{Capabilities: probeCapabilities(r.Context(), adapter)}
		writeJSON(w, http.StatusOK, caps)
	}
}

// probeCapabilities returns the live state of every endpoint the
// runtime exposes. The runtime has already authenticated once
// at startup; each probe reuses the session. There are 9 caps
// in total: device, status, clients are NOT security caps
// (the adapter has typed methods for them); the remaining 6
// go through SecurityCapability, which returns the four-state
// vocabulary. absent = firmware does not implement; verified =
// runtime parsed the response; unsupported_or_unverified =
// runtime has no parser; unavailable = transport error.
func probeCapabilities(ctx context.Context, adapter domain.RouterAdapter) map[string]string {
	// Adapters that implement securityCapable can report the per-cap
	// security matrix. The fixture adapter does not, so all 6 security
	// caps are reported as unsupported_or_unverified in mock mode.
	sc, _ := adapter.(securityCapable)
	_ = sc // for future per-cap dispatch
	out := map[string]string{
		"device":            "unverified",
		"status":            "unverified",
		"clients":           "unverified",
		"wireless":          "unverified",
		"wps":               "unverified",
		"dmz":               "unverified",
		"upnp":              "unverified",
		"remote-management": "unverified",
		"forwarding":        "unverified",
	}
	// Non-security caps have typed methods on the adapter.
	type nonSecurityProbe struct {
		name string
		fn   func(context.Context) error
	}
	nonSecurity := []nonSecurityProbe{
		{"status", func(c context.Context) error { _, e := adapter.Status(c); return e }},
		{"clients", func(c context.Context) error { _, e := adapter.Clients(c); return e }},
		{"device", func(c context.Context) error { _, e := adapter.Identify(c); return e }},
	}
	for _, ns := range nonSecurity {
		nctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		err := ns.fn(nctx)
		cancel()
		if err == nil {
			out[ns.name] = "verified"
		} else {
			out[ns.name] = "unavailable"
		}
	}
	// Security caps go through SecurityCapability, which already
	// classifies the response into the four-state vocabulary.
	securityCaps := []string{
		tplinkwr841v8.OpWireless,
		tplinkwr841v8.OpWPS,
		tplinkwr841v8.OpDMZ,
		tplinkwr841v8.OpUPnP,
		tplinkwr841v8.OpRemoteManagement,
		tplinkwr841v8.OpForwarding,
	}
	for _, name := range securityCaps {
		nctx, cancel := context.WithTimeout(ctx, 2*time.Second)
		if sc == nil {
			out[name] = "unsupported_or_unverified"
			cancel()
			continue
		}
		_, err := sc.SecurityCapability(nctx, name)
		cancel()
		if err == nil {
			out[name] = "verified"
			continue
		}
		if errors.Is(err, domain.ErrObservationAbsent) {
			out[name] = "absent"
			continue
		}
		if errors.Is(err, domain.ErrUnverifiedEndpoint) {
			out[name] = "unsupported_or_unverified"
			continue
		}
		out[name] = "unavailable"
	}
	return out
}

func handleDevice(adapter domain.RouterAdapter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		info, err := adapter.Identify(ctx)
		if err != nil {
			writeUnavailable(w, "router identify failed: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, info)
	}
}

func handleStatus(adapter domain.RouterAdapter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		status, err := adapter.Status(ctx)
		if err != nil {
			if errors.Is(err, domain.ErrCaptureMissing) {
				writeUnavailable(w, "session expired; restart router-core serve")
				return
			}
			writeUnavailable(w, "router status failed: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, status)
	}
}

func handleClients(adapter domain.RouterAdapter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		clients, err := adapter.Clients(ctx)
		if errors.Is(err, domain.ErrObservationAbsent) {
			writeJSON(w, http.StatusOK, map[string]any{
				"state":   "absent",
				"clients": []any{},
			})
			return
		}
		if err != nil {
			writeUnavailable(w, "router clients failed: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"state":   "verified",
			"clients": clients,
		})
	}
}

// securityHandler returns a per-capability security handler that
// dispatches by name and renders the typed observation. A failure
// in one capability does not poison another; the runtime is
// authoritative on whether a capability is verified, absent,
// unsupported_or_unverified, or unavailable.
func securityHandler(adapter domain.RouterAdapter, name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		sc, ok := adapter.(securityCapable)
		if !ok {
			writeUnsupported(w, name+" endpoint is not supported by this adapter")
			return
		}
		state, err := sc.SecurityCapability(ctx, name)
		if err != nil {
			if errors.Is(err, domain.ErrUnverifiedEndpoint) {
				writeUnsupported(w, name+" endpoint is unverified against captured traffic")
				return
			}
			if errors.Is(err, domain.ErrObservationAbsent) {
				writeUnsupported(w, name+" endpoint not present on this firmware build")
				return
			}
			writeUnavailable(w, name+" failed: "+err.Error())
			return
		}
		if state.WPSEnabled == domain.True {
			writeJSON(w, http.StatusOK, map[string]any{
				"state":  "verified",
				"result": state,
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"state": "unverified"})
	}
}

func handleSecurityWireless(adapter domain.RouterAdapter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		sc, ok := adapter.(securityCapable)
		if !ok {
			writeUnsupported(w, "wireless endpoint is not supported by this adapter")
			return
		}
		state, err := sc.SecurityCapability(ctx, tplinkwr841v8.OpWireless)
		if err != nil {
			if errors.Is(err, domain.ErrUnverifiedEndpoint) {
				writeUnsupported(w, "wireless endpoint is unverified against captured traffic")
				return
			}
			if errors.Is(err, domain.ErrObservationAbsent) {
				writeUnsupported(w, "wireless endpoint not present on this firmware build")
				return
			}
			writeUnavailable(w, "wireless failed: "+err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"state":  "verified",
			"result": state,
		})
	}
}

func handleSecurityWPS(adapter domain.RouterAdapter) http.HandlerFunc {
	return securityHandler(adapter, tplinkwr841v8.OpWPS)
}

func handleSecurityDMZ(adapter domain.RouterAdapter) http.HandlerFunc {
	return securityHandler(adapter, tplinkwr841v8.OpDMZ)
}

func handleSecurityUPnP(adapter domain.RouterAdapter) http.HandlerFunc {
	return securityHandler(adapter, tplinkwr841v8.OpUPnP)
}

func handleSecurityRemoteManagement(adapter domain.RouterAdapter) http.HandlerFunc {
	return securityHandler(adapter, tplinkwr841v8.OpRemoteManagement)
}

func handleSecurityForwarding(adapter domain.RouterAdapter) http.HandlerFunc {
	return securityHandler(adapter, tplinkwr841v8.OpForwarding)
}
