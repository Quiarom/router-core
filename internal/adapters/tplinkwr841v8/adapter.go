package tplinkwr841v8

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/Quiarom/router-core/internal/domain"
	"github.com/Quiarom/router-core/internal/transport"
)

// session holds the in-memory credential state for an authenticated
// router session. It is NOT persisted to disk. It lives only as long
// as the Adapter that created it. Phase 3 keeps the credential state
// inside one process; Phase 6 introduces OS keyring persistence (ADR 0005).
type session struct {
	user     string
	password string
}

// Adapter wraps the transport and a session. Operations are read-only
// (Phase 2 scope). Mutations are not wired in P3 and remain forbidden
// until Phase 6 introduces the policy + approval gate (ADR 0005).
type Adapter struct {
	host      string
	transport *transport.Client
	session   *session
}

const ModelName = "TL-WR841N/ND"

func New(host string, options ...transport.Option) *Adapter {
	return &Adapter{host: strings.TrimRight(host, "/"), transport: transport.New(options...)}
}

// Login performs the recipe verified against the WR841N v8.4 firmware
// 3.13.33 Build 130506 Rel.48660n on 2026-08-30: GET / with HTTP Basic
// Authorization header (plain text password, NOT md5hex). On success,
// the session is cached in memory and subsequent adapter methods can
// authenticate. The password is never written to disk and never leaves
// the adapter instance.
func (a *Adapter) Login(ctx context.Context, user, password string) error {
	body, status, err := a.transport.GetWithBasicAuth(ctx, normalizeRoot(a.host), user, password)
	if err != nil {
		return err
	}
	switch status {
	case 200:
		// Verify the response body actually looks like a TP-Link
		// dashboard, not an error page served with a permissive 200.
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

// authenticated reports whether the adapter has an active session.
func (a *Adapter) authenticated() bool {
	return a.session != nil
}

// authedFetch performs a GET against the given URL using the session
// credentials via Basic Auth header. Returns ErrCaptureMissing if no
// session is active (caller forgot to call Login).
func (a *Adapter) authedFetch(ctx context.Context, rawURL string) ([]byte, int, error) {
	if a.session == nil {
		return nil, 0, fmt.Errorf("%w: call Adapter.Login(ctx, user, password) first", domain.ErrCaptureMissing)
	}
	return a.transport.GetWithBasicAuth(ctx, rawURL, a.session.user, a.session.password)
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

// statusPath is the authenticated Status endpoint path on the
// WR841N v8.4 firmware. Verified against 3.13.33 Build 130506 Rel.48660n
// on 2026-08-30. Authorization is via the session's Basic Auth header.
const statusPath = "/userRpm/StatusRpm.htm"

func (a *Adapter) Status(ctx context.Context) (domain.RouterStatus, error) {
	if a.session == nil {
		return domain.RouterStatus{}, fmt.Errorf("%w: call Adapter.Login(ctx, user, password) first", domain.ErrCaptureMissing)
	}
	body, _, err := a.authedFetch(ctx, normalizeRoot(a.host)+statusPath)
	if err != nil {
		return domain.RouterStatus{}, err
	}
	return ParseStatus(body)
}

func (a *Adapter) Clients(ctx context.Context) ([]domain.Client, error) {
	body, err := a.fetch(ctx, OpDHCPClients)
	if err != nil {
		return nil, err
	}
	result, err := ParseDHCP(body)
	return result.Clients, err
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
