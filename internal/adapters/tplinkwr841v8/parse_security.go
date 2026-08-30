package tplinkwr841v8

import (
	"net"
	"strings"

	"github.com/Quiarom/router-core/internal/domain"
)

// All page indices below are UNVERIFIED until sanitized captures are available.
const (
	wpsEnabledIndex      = 0
	dmzEnabledIndex      = 0
	dmzHostIndex         = 1
	upnpEnabledIndex     = 0
	upnpMappingsIndex    = 1
	remoteEnabledIndex   = 0
	remotePortIndex      = 1
	forwardingCountIndex = 0
)

func ParseWPS(html []byte) (domain.SecurityState, error) {
	tokens, present, err := securityArray(html, "wpsPara", "WPS")
	if err != nil {
		return domain.SecurityState{}, err
	}
	state := domain.SecurityState{Provenance: domain.ProvenanceObserved}
	if !present {
		state.MarkUnsupported("wps", "no capture for WPS")
		return state, nil
	}
	state.WPSEnabled = tristate(tokens, wpsEnabledIndex)
	if state.WPSEnabled == domain.Unknown {
		state.MarkUnsupported("wps", "missing WPS state field")
	}
	return state, nil
}

func ParseDMZ(html []byte) (domain.SecurityState, error) {
	tokens, present, err := securityArray(html, "dmzPara", "DMZ")
	if err != nil {
		return domain.SecurityState{}, err
	}
	state := domain.SecurityState{Provenance: domain.ProvenanceObserved}
	if !present {
		state.MarkUnsupported("dmz", "no capture for DMZ")
		return state, nil
	}
	state.DMZEnabled = tristate(tokens, dmzEnabledIndex)
	if state.DMZEnabled == domain.Unknown {
		state.MarkUnsupported("dmz", "missing DMZ state field")
	}
	if host, ok := Str(tokens, dmzHostIndex); ok && net.ParseIP(host) != nil {
		state.DMZHost = host
	} else {
		state.MarkUnsupported("dmzHost", "missing DMZ host field")
	}
	return state, nil
}

func ParseUPnP(html []byte) (domain.SecurityState, error) {
	tokens, present, err := securityArray(html, "upnpPara", "UPnP")
	if err != nil {
		return domain.SecurityState{}, err
	}
	state := domain.SecurityState{Provenance: domain.ProvenanceObserved}
	if !present {
		state.MarkUnsupported("upnp", "no capture for UPnP")
		return state, nil
	}
	state.UPnPEnabled = tristate(tokens, upnpEnabledIndex)
	if state.UPnPEnabled == domain.Unknown {
		state.MarkUnsupported("upnp", "missing UPnP state field")
	}
	if n, ok := Int(tokens, upnpMappingsIndex); ok && n >= 0 {
		state.ActiveUPnPMappings = domain.SomeInt(n)
	} else {
		state.MarkUnsupported("activeUpnpMappings", "missing UPnP mapping count")
	}
	return state, nil
}

func ParseRemoteManagement(html []byte) (domain.SecurityState, error) {
	tokens, present, err := securityArray(html, "remotePara", "remote management")
	if err != nil {
		return domain.SecurityState{}, err
	}
	state := domain.SecurityState{Provenance: domain.ProvenanceObserved}
	if !present {
		state.MarkUnsupported("remoteManagement", "no capture for remote management")
		return state, nil
	}
	state.RemoteManagementEnabled = tristate(tokens, remoteEnabledIndex)
	if state.RemoteManagementEnabled == domain.Unknown {
		state.MarkUnsupported("remoteManagement", "missing remote-management state field")
	}
	if n, ok := Int(tokens, remotePortIndex); ok && n > 0 && n <= 65535 {
		state.RemoteManagementPort = domain.SomeInt(n)
	} else {
		state.MarkUnsupported("remoteManagementPort", "missing remote-management port")
	}
	return state, nil
}

func ParseForwarding(html []byte) (domain.SecurityState, error) {
	tokens, present, err := securityArray(html, "virtualServerPara", "forwarding")
	if err != nil {
		return domain.SecurityState{}, err
	}
	state := domain.SecurityState{Provenance: domain.ProvenanceObserved}
	if !present {
		state.MarkUnsupported("forwardingRules", "no capture for forwarding")
		return state, nil
	}
	if n, ok := Int(tokens, forwardingCountIndex); ok && n >= 0 {
		state.ForwardingRules = domain.SomeInt(n)
	} else {
		state.MarkUnsupported("forwardingRules", "missing forwarding rule count")
	}
	return state, nil
}

func ParseSecurity(html []byte) (domain.SecurityState, error) {
	if err := Classify(html); err != nil {
		return domain.SecurityState{}, err
	}
	state := domain.SecurityState{Provenance: domain.ProvenanceObserved}
	parts := []struct {
		name  string
		parse func([]byte) (domain.SecurityState, error)
	}{
		{"wpsPara", ParseWPS},
		{"dmzPara", ParseDMZ},
		{"upnpPara", ParseUPnP},
		{"remotePara", ParseRemoteManagement},
		{"virtualServerPara", ParseForwarding},
	}
	found := false
	for _, part := range parts {
		if !strings.Contains(string(html), part.name) {
			continue
		}
		found = true
		parsed, err := part.parse(html)
		if err != nil {
			return domain.SecurityState{}, err
		}
		mergeSecurity(&state, parsed)
	}
	if !found {
		state.MarkUnsupported("security", "no capture for security page")
	}
	return state, nil
}

func securityArray(html []byte, name, page string) ([]Token, bool, error) {
	if IsLoginPage(html) {
		return nil, false, domain.ErrUnauthenticated
	}
	trimmed := strings.TrimSpace(string(html))
	if trimmed == "" || !strings.Contains(strings.ToLower(trimmed), "<") {
		return nil, false, domain.ErrUnexpectedResponse
	}
	tokens, ok := ExtractArray(html, name)
	if ok {
		return tokens, true, nil
	}
	if strings.Contains(strings.ToLower(string(html)), strings.ToLower(name)) {
		return nil, false, domain.ErrUnexpectedResponse
	}
	return nil, false, nil
}

func tristate(tokens []Token, i int) domain.Tristate {
	if n, ok := Int(tokens, i); ok {
		if n == 0 {
			return domain.False
		}
		if n == 1 {
			return domain.True
		}
	}
	if s, ok := Str(tokens, i); ok {
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "1", "true", "on", "enabled", "enable", "yes":
			return domain.True
		case "0", "false", "off", "disabled", "disable", "no":
			return domain.False
		}
	}
	return domain.Unknown
}
