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

// session is the in-memory credential state for one authenticated
// router connection. Never persisted to disk. sessionToken is the
// URL-embedded token some legacy WR841N firmware requires for
// protected endpoints; the v8.4 build verified 2026-08-30 does not
// return one from /, so this field is typically empty.
type session struct {
	user         string
	password     string
	sessionToken string
}

// dashboardSignal is the substring emitted by the WR841N v8.4 firmware
// at the top of every authenticated dashboard body. Its presence in
// a 200 OK response distinguishes a real page from the
// "You have no authority" rejection.
const dashboardSignal = "var statusPara = new Array"

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

// Login authenticates with the recipe verified 2026-08-30 against
// the WR841N v8.4 firmware: GET / with HTTP Basic Authorization
// header (plaintext password, NOT md5hex). On success, the session
// is cached in memory.
func (a *Adapter) Login(ctx context.Context, user, password string) error {
	body, status, err := a.transport.GetWithBasicAuth(ctx, normalizeRoot(a.host), user, password)
	if err != nil {
		return err
	}
	switch status {
	case 200:
		if bytes.Contains(body, []byte("Login Incorrect")) {
			return domain.ErrUnauthenticated
		}
		a.session = &session{user: user, password: password}
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

// SessionTokenForTest sets the sessionToken on the active session.
// Used by tests that emulate a firmware requiring a URL-embedded
// token; production code paths obtain the token via the firmware's
// /LoginRpm.htm?Save=Save response or the operator.
func (a *Adapter) SessionTokenForTest(token string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.session == nil {
		return
	}
	a.session.sessionToken = token
}

// authedFetch performs a GET against the given URL using the session
// credentials via Basic Auth header. Returns ErrCaptureMissing if
// Login was not called first.
func (a *Adapter) authedFetch(ctx context.Context, rawURL string) ([]byte, int, error) {
	if a.session == nil {
		return nil, 0, fmt.Errorf("%w: call Adapter.Login(ctx, user, password) first", domain.ErrCaptureMissing)
	}
	return a.transport.GetWithBasicAuth(ctx, rawURL, a.session.user, a.session.password)
}

// authedFetchWithFallback GETs the given path with Basic Auth header,
// retrying with a /<token>/<path> URL prefix when the firmware returns
// the "no authority" body. The v8.4 firmware requires the prefix for
// most /userRpm/<path> endpoints; the bare path returns 68 bytes
// of "no authority" instead.
func (a *Adapter) authedFetchWithFallback(ctx context.Context, path string) ([]byte, int, error) {
	if a.session == nil {
		return nil, 0, fmt.Errorf("%w: call Adapter.Login(ctx, user, password) first", domain.ErrCaptureMissing)
	}
	host := a.host
	baseURL := normalizeRoot(host) + path

	body, status, err := a.transport.GetWithBasicAuth(ctx, baseURL, a.session.user, a.session.password)
	if err != nil {
		return body, status, err
	}
	if a.session.sessionToken != "" && isNoAuthBody(body, status) {
		tokenURL := normalizeRoot(host) + "/" + a.session.sessionToken + path
		tokBody, tokStatus, tokErr := a.transport.GetWithBasicAuth(ctx, tokenURL, a.session.user, a.session.password)
		if tokErr == nil && !isNoAuthBody(tokBody, tokStatus) {
			return tokBody, tokStatus, nil
		}
	}
	return body, status, nil
}

// isNoAuthBody reports whether the body is the "no authority"
// rejection. Body size alone is unreliable because small dashboards
// (e.g. Status with a sparse statusPara) can be under any
// arbitrary byte threshold.
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

func (a *Adapter) fetch(ctx context.Context, op string) ([]byte, error) {
	rawURL, err := a.endpointURL(op)
	if err != nil {
		return nil, err
	}
	body, _, err := a.transport.Get(ctx, rawURL)
	return body, err
}

func (a *Adapter) Identify(ctx context.Context) (domain.DeviceInfo, error) {
	info := domain.DeviceInfo{
		Vendor: "TP-Link", Model: ModelName, ManagementAddress: a.host,
		Authenticated: domain.Unknown, Provenance: domain.ProvenanceObserved,
	}
	body, _, err := a.transport.Get(ctx, normalizeRoot(a.host))
	if err != nil {
		info.Provenance = domain.ProvenanceAbsent
		return info, err
	}
	if firmware, hardware, parseErr := ParseIdentity(body); parseErr == nil {
		info.FirmwareVersion = domain.NewUntrusted(firmware, "router:status")
		info.HardwareVersion = domain.NewUntrusted(hardware, "router:status")
	}
	return info, nil
}

// statusPath is the Status endpoint path. Verified 2026-08-30.
const statusPath = "/userRpm/StatusRpm.htm"

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

// fetchWithAuth is the authenticated counterpart of fetch.
func (a *Adapter) fetchWithAuth(ctx context.Context, op string) ([]byte, error) {
	rawURL, err := a.endpointURL(op)
	if err != nil {
		return nil, err
	}
	// endpointURL returns host + path; prepend scheme if missing.
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "http://" + rawURL
	}
	body, _, err := a.transport.GetWithBasicAuth(ctx, rawURL, a.session.user, a.session.password)
	return body, err
}

func (a *Adapter) Security(ctx context.Context) (domain.SecurityState, error) {
	state := domain.SecurityState{Provenance: domain.ProvenanceObserved}
	for _, item := range []struct {
		op    string
		parse func([]byte) (domain.SecurityState, error)
	}{
		{OpWPS, ParseWPS}, {OpDMZ, ParseDMZ}, {OpUPnP, ParseUPnP},
		{OpRemoteManagement, ParseRemoteManagement},
		{OpForwarding, ParseForwarding},
	} {
		body, err := a.fetch(ctx, item.op)
		if err != nil {
			return state, err
		}
		part, err := item.parse(body)
		if err != nil {
			return state, err
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
