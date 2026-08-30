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
		Vendor: "TP-Link", Model: "TL-WR841N/ND", ManagementAddress: a.host,
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
	var state domain.SecurityState
	for _, item := range []struct {
		op    string
		parse func([]byte) (domain.SecurityState, error)
	}{
		{OpWPS, ParseWPS}, {OpForwarding, ParseForwarding}, {OpUPnP, ParseUPnP},
		{OpRemoteManagement, ParseRemoteManagement},
	} {
		body, err := a.fetch(ctx, item.op)
		if err != nil {
			return state, err
		}
		part, err := item.parse(body)
		if err != nil {
			return state, err
		}
		mergeSecurity(&state, part)
	}
	dmzBody, err := a.fetch(ctx, OpForwarding)
	if err != nil {
		return state, err
	}
	part, parseErr := ParseDMZ(dmzBody)
	if parseErr != nil {
		return state, parseErr
	}
	mergeSecurity(&state, part)
	part, parseErr = ParseForwarding(dmzBody)
	if parseErr != nil {
		return state, parseErr
	}
	mergeSecurity(&state, part)
	return state, nil
}

func normalizeRoot(host string) string {
	if parsed, err := url.Parse(host); err == nil && parsed.Scheme != "" {
		return strings.TrimRight(host, "/") + "/"
	}
	return "http://" + strings.TrimRight(host, "/") + "/"
}

func mergeSecurity(dst *domain.SecurityState, src domain.SecurityState) {
	if src.WPSEnabled != domain.Unknown {
		dst.WPSEnabled = src.WPSEnabled
	}
	if src.DMZEnabled != domain.Unknown {
		dst.DMZEnabled = src.DMZEnabled
	}
	if src.DMZHost != "" {
		dst.DMZHost = src.DMZHost
	}
	if src.UPnPEnabled != domain.Unknown {
		dst.UPnPEnabled = src.UPnPEnabled
	}
	if src.ActiveUPnPMappings.Valid {
		dst.ActiveUPnPMappings = src.ActiveUPnPMappings
	}
	if src.RemoteManagementEnabled != domain.Unknown {
		dst.RemoteManagementEnabled = src.RemoteManagementEnabled
	}
	if src.RemoteManagementPort.Valid {
		dst.RemoteManagementPort = src.RemoteManagementPort
	}
	if src.ForwardingRules.Valid {
		dst.ForwardingRules = src.ForwardingRules
	}
	for field, reason := range src.Unsupported {
		dst.MarkUnsupported(field, reason)
	}
	dst.Provenance = src.Provenance
}
