package tplinkwr841v8

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/Quiarom/router-core/internal/domain"
	"github.com/Quiarom/router-core/internal/transport"
)

type Adapter struct {
	host      string
	transport *transport.Client
}

const ModelName = "TL-WR841N/ND"

func New(host string, options ...transport.Option) *Adapter {
	return &Adapter{host: strings.TrimRight(host, "/"), transport: transport.New(options...)}
}

func (a *Adapter) authenticate(context.Context) error {
	return fmt.Errorf("%w: see BLOCKED_CAPTURE.md for the login request and response capture", domain.ErrCaptureMissing)
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

func (a *Adapter) Status(ctx context.Context) (domain.RouterStatus, error) {
	body, err := a.fetch(ctx, OpStatus)
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
