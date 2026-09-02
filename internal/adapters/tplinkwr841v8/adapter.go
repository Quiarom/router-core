// Package tplinkwr841v8 implements the read-only RouterAdapter
// contract for the TP-Link TL-WR841N/ND v8.4 stock firmware.
// The verified authentication recipe is HTTP Basic Authorization
// with the plaintext password (NOT md5hex) sent as the
// Authorization header against /. Subsequent /userRpm/<path>
// requests need a Referer header pointing to the parent frameset
// page or the firmware responds with the 68-byte "no authority"
// rejection. Verified live 2026-08-31 against the physical
// lab unit at 192.168.1.1.
package tplinkwr841v8

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/Quiarom/router-core/internal/domain"
	"github.com/Quiarom/router-core/internal/transport"
)

type session struct {
	user     string
	password string
	// referer is the URL used as the Referer header for
	// /userRpm/<path> requests so the firmware returns the real
	// dashboard body instead of the 68-byte "no authority" rejection.
	referer string
}

// dashboardSignal marks an authenticated dashboard body. Its
// absence on a 200 OK indicates the "no authority" rejection.
const dashboardSignal = "var statusPara = new Array"

// statusPath is the Status endpoint path. Verified 2026-08-31.
const statusPath = "/userRpm/StatusRpm.htm"

// Adapter wraps the transport and a session. Read-only.
type Adapter struct {
	host      string
	transport *transport.Client
	mu        sync.Mutex
	session   *session
}

const ModelName = "TL-WR841N/ND"

func New(host string, options ...transport.Option) *Adapter {
	return &Adapter{host: strings.TrimRight(host, "/"), transport: transport.New(options...)}
}

// Login authenticates against the WR841N v8.4 with the recipe
// verified 2026-08-31: HTTP Basic Authorization header with
// plaintext password (NOT md5hex) against /. The session caches
// user, password, and the parent Referer so subsequent
// /userRpm/<path> requests can be made with the right shape.
func (a *Adapter) Login(ctx context.Context, user, password string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	body, status, err := a.transport.GetWithBasicAuth(ctx, normalizeRoot(a.host), user, password)
	if err != nil {
		return err
	}
	switch status {
	case 200:
		if bytes.Contains(body, []byte("Login Incorrect")) {
			return domain.ErrUnauthenticated
		}
		a.session = &session{
			user:     user,
			password: password,
			referer:  normalizeRoot(a.host),
		}
		return nil
	case 401:
		return domain.ErrUnauthenticated
	default:
		return fmt.Errorf("tplinkwr841v8: login got HTTP %d, expected 200 or 401", status)
	}
}

func (a *Adapter) authenticated() bool {
	return a.session != nil
}

// SessionRefererForTest sets the Referer on the active session.
// Tests use this to verify both branches; production code paths
// set the Referer inside Login.
func (a *Adapter) SessionRefererForTest(referer string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.session == nil {
		return
	}
	a.session.referer = referer
}

// authedFetch GETs the given URL with the session credentials and
// the cached Referer. Returns ErrCaptureMissing if Login was not
// called first.
func (a *Adapter) authedFetch(ctx context.Context, rawURL string) ([]byte, int, error) {
	if a.session == nil {
		return nil, 0, fmt.Errorf("%w: call Adapter.Login(ctx, user, password) first", domain.ErrCaptureMissing)
	}
	return a.transport.GetWithBasicAuthAndReferer(ctx, rawURL, a.session.referer, a.session.user, a.session.password)
}

// authedFetchWithFallback GETs the given path with Basic Auth and
// the cached Referer. The v8.4 firmware returns the real
// dashboard body when the Referer points to the parent frameset
// page; the fallback is kept for the older 68-byte rejection
// case.
func (a *Adapter) authedFetchWithFallback(ctx context.Context, path string) ([]byte, int, error) {
	if a.session == nil {
		return nil, 0, fmt.Errorf("%w: call Adapter.Login(ctx, user, password) first", domain.ErrCaptureMissing)
	}
	host := a.host
	url := strings.TrimRight(normalizeRoot(host), "/") + path

	body, status, err := a.transport.GetWithBasicAuthAndReferer(ctx, url, a.session.referer, a.session.user, a.session.password)
	if err != nil {
		return body, status, err
	}
	if isNoAuthBody(body, status) {
		altBody, altStatus, altErr := a.transport.GetWithBasicAuth(ctx, url, a.session.user, a.session.password)
		if altErr == nil && !isNoAuthBody(altBody, altStatus) {
			return altBody, altStatus, nil
		}
	}
	return body, status, nil
}

// isNoAuthBody reports whether the body is the "no authority"
// rejection. Body size alone is unreliable because small
// dashboards can be under any arbitrary byte threshold.
func isNoAuthBody(body []byte, status int) bool {
	return status == 200 && !bytes.Contains(body, []byte(dashboardSignal))
}

func (a *Adapter) endpointURL(op string) (string, error) {
	endpoint, ok := Endpoints[op]
	if !ok {
		return "", fmt.Errorf("router-core: unknown endpoint %q", op)
	}
	if err := dispatchAllowed(endpoint); err != nil {
		return "", fmt.Errorf("%w: %s", err, endpoint.CaptureNote)
	}
	return strings.TrimRight(a.host, "/") + endpoint.Path, nil
}

// Identify authenticates if no session is active, then reads the
// authenticated Status dashboard and parses the firmware and
// hardware fingerprints out of its `var statusPara` block.
func (a *Adapter) Identify(ctx context.Context) (domain.DeviceInfo, error) {
	info := domain.DeviceInfo{
		Vendor: "TP-Link", Model: ModelName, ManagementAddress: a.host,
		Authenticated: domain.Unknown, Provenance: domain.ProvenanceObserved,
	}
	a.mu.Lock()
	sess := a.session
	a.mu.Unlock()
	if sess == nil {
		info.Provenance = domain.ProvenanceAbsent
		return info, fmt.Errorf("%w: call Adapter.Login(ctx, user, password) first", domain.ErrCaptureMissing)
	}
	body, _, err := a.authedFetchWithFallback(ctx, statusPath)
	if err != nil {
		info.Provenance = domain.ProvenanceAbsent
		return info, err
	}
	if firmware, hardware, parseErr := ParseIdentity(body); parseErr == nil {
		info.FirmwareVersion = domain.NewUntrusted(firmware, "router:status")
		info.HardwareVersion = domain.NewUntrusted(hardware, "router:status")
		info.Authenticated = domain.True
	}
	return info, nil
}

func (a *Adapter) Status(ctx context.Context) (domain.RouterStatus, error) {
	if a.session == nil {
		return domain.RouterStatus{}, fmt.Errorf("%w: call Adapter.Login(ctx, user, password) first", domain.ErrCaptureMissing)
	}
	body, _, err := a.authedFetchWithFallback(ctx, statusPath)
	if err != nil {
		return domain.RouterStatus{}, err
	}
	return ParseStatus(body)
}

func (a *Adapter) Clients(ctx context.Context) ([]domain.Client, error) {
	if a.session == nil {
		return nil, fmt.Errorf("%w: call Adapter.Login(ctx, user, password) first", domain.ErrCaptureMissing)
	}
	body, err := a.fetchWithAuth(ctx, OpDHCPClients)
	if err != nil {
		return nil, err
	}
	result, err := ParseDHCP(body)
	return result.Clients, err
}

func (a *Adapter) fetchWithAuth(ctx context.Context, op string) ([]byte, error) {
	rawURL, err := a.endpointURL(op)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "http://" + rawURL
	}
	body, _, err := a.transport.GetWithBasicAuthAndReferer(ctx, rawURL, a.session.referer, a.session.user, a.session.password)
	return body, err
}

// FetchWirelessSecurity fetches the wireless-security page and
// returns the parsed observation. Exported so the serve
// handler can call it directly; the typed return value avoids
// the SecurityState aggregation that does not fit this
// capability's data shape (SSID, cipher, key renewal, etc.).
func (a *Adapter) FetchWirelessSecurity(ctx context.Context) (WirelessSecurity, error) {
	if a.session == nil {
		return WirelessSecurity{}, fmt.Errorf("%w: call Adapter.Login(ctx, user, password) first", domain.ErrCaptureMissing)
	}
	if err := dispatchAllowed(Endpoints[OpWireless]); err != nil {
		return WirelessSecurity{}, err
	}
	body, err := a.fetchWithAuth(ctx, OpWireless)
	if err != nil {
		return WirelessSecurity{}, err
	}
	return ParseWirelessSecurity(body)
}

// SecurityCapability fetches a single security capability and
// returns the parsed state plus the underlying error, if any.
// Used by the per-capability /v0/security/<name> handlers so a
// failure in one capability does not poison the others.
func (a *Adapter) SecurityCapability(ctx context.Context, name string) (domain.SecurityState, error) {
	if a.session == nil {
		return domain.SecurityState{}, fmt.Errorf("%w: call Adapter.Login(ctx, user, password) first", domain.ErrCaptureMissing)
	}
	if err := dispatchAllowed(Endpoints[name]); err != nil {
		return domain.SecurityState{}, err
	}
	body, err := a.fetchWithAuth(ctx, name)
	if err != nil {
		return domain.SecurityState{}, err
	}
	return parseSecurityCapability(name, body)
}

// parseSecurityCapability dispatches by name to the matching
// parser. Returns a populated SecurityState on success; returns
// the parser's error otherwise.
func parseSecurityCapability(name string, body []byte) (domain.SecurityState, error) {
	switch name {
	case OpWPS:
		return ParseWPS(body)
	case OpDMZ:
		return ParseDMZ(body)
	case OpUPnP:
		return ParseUPnP(body)
	case OpRemoteManagement:
		return ParseRemoteManagement(body)
	case OpForwarding:
		return ParseForwarding(body)
	default:
		return domain.SecurityState{}, fmt.Errorf("router-core: unknown security capability %q", name)
	}
}

// Security aggregates the per-capability observations. Each
// capability is fetched and parsed independently; a failure in
// one does not short-circuit the others. The aggregated
// SecurityState's fields reflect the union of successful
// observations.
func (a *Adapter) Security(ctx context.Context) (domain.SecurityState, error) {
	state := domain.SecurityState{Provenance: domain.ProvenanceObserved}
	for _, name := range []string{OpWPS, OpDMZ, OpUPnP, OpRemoteManagement, OpForwarding} {
		part, err := a.SecurityCapability(ctx, name)
		if err != nil {
			continue
		}
		state.Merge(part)
	}
	return state, nil
}

func normalizeRoot(host string) string {
	if parsed, err := url.Parse(host); err == nil && parsed.Scheme != "" {
		return strings.TrimRight(host, "/") + "/"
	}
	return "http://" + strings.TrimRight(host, "/") + "/"
}
