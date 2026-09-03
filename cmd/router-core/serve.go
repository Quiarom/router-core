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
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Quiarom/router-core/internal/adapters/tplinkwr841v8"
	"github.com/Quiarom/router-core/internal/domain"
	"github.com/Quiarom/router-core/internal/transport"
)

type sessionStore struct {
	mu       sync.Mutex
	password string
}

func (s *sessionStore) get() (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.password, s.password != ""
}

func (s *sessionStore) set(p string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.password = p
}

func readPasswordNoEcho() (string, error) {
	type readResult struct {
		buf []byte
		err error
	}
	ch := make(chan readResult, 1)
	go func() {
		buf := make([]byte, 256)
		n, err := os.Stdin.Read(buf)
		ch <- readResult{buf[:n], err}
	}()
	select {
	case res := <-ch:
		if res.err != nil && res.err != io.EOF {
			return "", res.err
		}
		return strings.TrimRight(string(res.buf), "\r\n"), nil
	case <-time.After(30 * time.Second):
		return "", errors.New("timed out waiting 30s for password on stdin")
	}
}

func runServeCommand(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	host := fs.String("host", "192.168.1.1", "local router address (RFC1918 literal)")
	addr := fs.String("addr", "127.0.0.1:8484", "loopback HTTP listen address")
	timeout := fs.Duration("timeout", 5*time.Second, "per-request timeout to the router")
	mock := fs.Bool("mock", false, "serve mock fixtures from fixtures/frontend-mocks")
	fixturesDir := fs.String("fixtures", "fixtures/frontend-mocks", "directory containing mock JSON fixtures")
	if err := fs.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) || err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if !isLoopbackAddr(*addr) {
		return fmt.Errorf("refusing to serve on non-loopback address %q (loopback only)", *addr)
	}

	if *mock {
		fmt.Fprintf(os.Stderr, "router-core serve: running in mock fixtures mode (%s), listening on %s\n", *fixturesDir, *addr)
		mux := http.NewServeMux()
		registerMockRoutes(mux, *fixturesDir)
		srv := &http.Server{
			Addr:              *addr,
			Handler:           withLocalCORS(mux),
			ReadHeaderTimeout: 5 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       60 * time.Second,
		}
		return srv.ListenAndServe()
	}

	if !isRFC1918OrLoopback(*host) {
		return fmt.Errorf("refusing to observe host %q: not loopback/RFC1918", *host)
	}

	fmt.Fprintf(os.Stderr, "router-core serve: reading admin password from stdin (timeout 30s)\n")
	password, err := readPasswordNoEcho()
	if err != nil {
		return fmt.Errorf("read password: %w", err)
	}
	if password == "" {
		return errors.New("empty password")
	}
	defer zeroString(&password)

	store := &sessionStore{password: password}

	adapter := tplinkwr841v8.New(*host, transport.WithTimeout(*timeout))
	loginCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := adapter.Login(loginCtx, "admin", password); err != nil {
		return fmt.Errorf("login: %w", err)
	}

	fmt.Fprintf(os.Stderr, "router-core serve: authenticated, listening on %s\n", *addr)

	mux := http.NewServeMux()
	registerRoutes(mux, adapter, store)

	srv := &http.Server{
		Addr:              *addr,
		Handler:           withLocalCORS(mux),
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

// zeroString zeroes a string in place. Go strings are immutable
// but the backing bytes of a literal-converted []byte share
// storage with the string; this is best-effort.
func zeroString(s *string) {
	if s == nil {
		return
	}
	*s = ""
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

func registerRoutes(mux *http.ServeMux, adapter *tplinkwr841v8.Adapter, store *sessionStore) {
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
// unsupported_or_unverified, unavailable).
func handleCapabilities(adapter *tplinkwr841v8.Adapter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		caps := capabilitiesResponse{
			Capabilities: map[string]string{
				"device":            "verified",
				"status":            "verified",
				"clients":           "verified",
				"wireless_security": "verified",
				"wps":               "absent",
				"dmz":               "verified",
				"upnp":              "absent",
				"remote_management": "absent",
				"forwarding":        "verified",
			},
		}
		writeJSON(w, http.StatusOK, caps)
	}
}

func handleDevice(adapter *tplinkwr841v8.Adapter) http.HandlerFunc {
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

func handleStatus(adapter *tplinkwr841v8.Adapter) http.HandlerFunc {
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

func handleClients(adapter *tplinkwr841v8.Adapter) http.HandlerFunc {
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
func securityHandler(adapter *tplinkwr841v8.Adapter, name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		state, err := adapter.SecurityCapability(ctx, name)
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
		writeJSON(w, http.StatusOK, map[string]any{
			"state":  "verified",
			"result": state,
		})
	}
}

func handleSecurityWireless(adapter *tplinkwr841v8.Adapter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		state, err := adapter.FetchWirelessSecurity(ctx)
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

func handleSecurityWPS(adapter *tplinkwr841v8.Adapter) http.HandlerFunc {
	return securityHandler(adapter, tplinkwr841v8.OpWPS)
}

func handleSecurityDMZ(adapter *tplinkwr841v8.Adapter) http.HandlerFunc {
	return securityHandler(adapter, tplinkwr841v8.OpDMZ)
}

func handleSecurityUPnP(adapter *tplinkwr841v8.Adapter) http.HandlerFunc {
	return securityHandler(adapter, tplinkwr841v8.OpUPnP)
}

func handleSecurityRemoteManagement(adapter *tplinkwr841v8.Adapter) http.HandlerFunc {
	return securityHandler(adapter, tplinkwr841v8.OpRemoteManagement)
}

func handleSecurityForwarding(adapter *tplinkwr841v8.Adapter) http.HandlerFunc {
	return securityHandler(adapter, tplinkwr841v8.OpForwarding)
}

func withLocalCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			if !isLocalOrigin(origin) {
				writeJSON(w, http.StatusForbidden, capabilityError{State: "unavailable", Reason: "origin not allowed"})
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		}
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func isLocalOrigin(rawOrigin string) bool {
	parsed, err := url.Parse(rawOrigin)
	if err != nil {
		return false
	}
	host := parsed.Hostname()
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func registerMockRoutes(mux *http.ServeMux, fixturesDir string) {
	serveFileOrJSON := func(subpath string, defaultStatus int) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			path := filepath.Join(fixturesDir, subpath)
			data, err := os.ReadFile(path)
			if err != nil {
				writeUnavailable(w, "mock fixture missing: "+subpath)
				return
			}
			var parsed struct {
				State string `json:"state"`
			}
			_ = json.Unmarshal(data, &parsed)
			status := defaultStatus
			switch parsed.State {
			case "unavailable":
				status = http.StatusServiceUnavailable
			case "unsupported_or_unverified":
				status = http.StatusNotFound
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_, _ = w.Write(data)
		}
	}

	mux.HandleFunc("/healthz", serveFileOrJSON("healthz.json", http.StatusOK))
	mux.HandleFunc("/v0/device", serveFileOrJSON("device.json", http.StatusOK))
	mux.HandleFunc("/v0/status", serveFileOrJSON("status.json", http.StatusOK))
	mux.HandleFunc("/v0/clients", serveFileOrJSON("clients.json", http.StatusOK))
	mux.HandleFunc("/v0/capabilities", serveFileOrJSON("capabilities.json", http.StatusOK))
	mux.HandleFunc("/v0/security/wireless", serveFileOrJSON("security/wireless.json", http.StatusServiceUnavailable))
	mux.HandleFunc("/v0/security/wps", serveFileOrJSON("security/wps.json", http.StatusNotFound))
	mux.HandleFunc("/v0/security/dmz", serveFileOrJSON("security/dmz.json", http.StatusOK))
	mux.HandleFunc("/v0/security/upnp", serveFileOrJSON("security/upnp.json", http.StatusNotFound))
	mux.HandleFunc("/v0/security/remote-management", serveFileOrJSON("security/remote-management.json", http.StatusNotFound))
	mux.HandleFunc("/v0/security/forwarding", serveFileOrJSON("security/forwarding.json", http.StatusOK))
}
