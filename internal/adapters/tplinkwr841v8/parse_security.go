package tplinkwr841v8

import (
	"encoding/json"
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

// WirelessSecurity is the normalized observation of the
// /userRpm/WlanSecurityRpm.htm page. The v8.4 firmware on the
// lab unit emits `var wlanPara = new Array(...)` plus a
// companion `var ssidList = new Array(...)`. The positions of
// the fields inside wlanPara shift between firmware builds;
// we return the typed fields we are confident about and the
// raw array under Raw for callers that need more.
type WirelessSecurity struct {
	// Enabled reports whether the wireless radio is on. The
	// firmware uses 8 (= radio + some sub-flag) for enabled and
	// 0 for disabled.
	Enabled bool
	// SSID is the broadcast SSID. The lab operator's SSID is
	// `TP-LINK_CBEC16`; the fixture used in tests preserves the
	// SSID because it is a public form value, not a secret.
	SSID string
	// SecurityTypeRaw is the firmware's secType integer. The
	// 1-indexed mapping on the v8.4 build is: 1 = open, 2 =
	// WPA-PSK (auto), 3 = WPA2-PSK (auto), 4 = WPA/WPA2 mixed.
	// We do not assert a mapping; we return the raw integer so
	// the agent and frontend can interpret it.
	SecurityTypeRaw int
	// SecurityType is the operator-friendly string the agent
	// and the frontend render. It uses the 1-indexed mapping
	// documented in SecurityTypeRaw.
	SecurityType string
	// Cipher reports the cipher selection as a string. The
	// firmware encodes combinations as a string of 1/0 flags
	// ("332" = AES+TKIP, "33" = AES, "2" = TKIP, "0" = none).
	// We return it as-is.
	Cipher string
	// KeyRenewalSecs is the group key update period in seconds
	// (default 1812 = ~30.2 minutes). 0 means "not exposed".
	KeyRenewalSecs int
	// HasPreSharedKey reports whether a non-empty PSK is set.
	// The lab unit's PMK is replaced with a placeholder by the
	// sanitize package, so this field is a true boolean signal,
	// not a copy of the key.
	HasPreSharedKey bool
	// Raw is the verbatim wlanPara array. Use it when you need a
	// field the typed struct does not expose.
	Raw json.RawMessage
}

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
	parsers := map[string]func([]byte) (domain.SecurityState, error){
		"wpsPara":           ParseWPS,
		"dmzPara":           ParseDMZ,
		"upnpPara":          ParseUPnP,
		"remotePara":        ParseRemoteManagement,
		"virtualServerPara": ParseForwarding,
	}
	found := false
	for _, name := range knownArrays {
		parse, ok := parsers[name]
		if !ok || !strings.Contains(string(html), name) {
			continue
		}
		found = true
		parsed, err := parse(html)
		if err != nil {
			return domain.SecurityState{}, err
		}
		state.Merge(parsed)
	}
	if !found {
		state.MarkUnsupported("security", "no capture for security page")
	}
	return state, nil
}

// wirelessSecurityIndexMap documents the v8.4 wlanPara positions
// we read. The numbers shift between firmware builds; if the
// live firmware's layout changes, update this map AND the
// `wirelessSecuritySecurityType` translation below.
//
// Verified against the lab unit on 2026-08-31 (3.15.9 Build
// 140724 Rel.63227n).
const (
	wirelessWlanEnabledIndex   = 0
	wirelessSsidIndexIndex     = 1
	wirelessSecurityTypeIndex  = 2
	wirelessCipherTypeIndex    = 3
	wirelessKeyTypeIndex       = 4 // 0 = passphrase, 1 = hex
	wirelessPassphraseIndex    = 5 // 0 = disabled
	wirelessKeyRenewalIndex    = 7 // seconds
	wirelessRadiusEnabledIndex = 8
	wirelessPmkIndex           = 9
)

// wirelessSecuritySecurityType maps the firmware's secType
// integer to an operator-friendly string. v8.4 1-indexed:
//
//	1 = open
//	2 = WPA-PSK (auto)
//	3 = WPA2-PSK (auto)
//	4 = WPA/WPA2 mixed
func wirelessSecuritySecurityType(raw int) string {
	switch raw {
	case 1:
		return "open"
	case 2:
		return "wpa-psk"
	case 3:
		return "wpa2-psk"
	case 4:
		return "wpa/wpa2-mixed"
	default:
		return "unknown"
	}
}

// ParseWirelessSecurity extracts the typed fields we are
// confident about from the v8.4 wireless-security page. The
// raw wlanPara array is preserved under Raw for callers that
// need fields the typed struct does not expose.
func ParseWirelessSecurity(html []byte) (WirelessSecurity, error) {
	if IsLoginPage(html) {
		return WirelessSecurity{}, domain.ErrUnauthenticated
	}
	tokens, ok := ExtractArray(html, "wlanPara")
	if !ok {
		return WirelessSecurity{}, domain.ErrUnexpectedResponse
	}
	raw, _ := json.Marshal(tokens)

	// SSID lives in the companion `ssidList` array, index 0. We
	// use the same ExtractArray primitive on that array.
	var ssid string
	if ssidTokens, ok := ExtractArray(html, "ssidList"); ok && len(ssidTokens) > 0 {
		if len(ssidTokens) > 0 && ssidTokens[0].Kind == TokenString {
			ssid = ssidTokens[0].Literal
		}
	}

	enabled := false
	if n, ok := Int(tokens, wirelessWlanEnabledIndex); ok {
		enabled = n != 0
	}
	secType := 0
	if n, ok := Int(tokens, wirelessSecurityTypeIndex); ok {
		secType = n
	}
	cipher := ""
	if s, ok := Str(tokens, wirelessCipherTypeIndex); ok {
		cipher = strings.TrimSpace(s)
	}
	renewal := 0
	if n, ok := Int(tokens, wirelessKeyRenewalIndex); ok {
		renewal = n
	}
	hasPSK := false
	if s, ok := Str(tokens, wirelessPmkIndex); ok {
		// After sanitization the PMK is replaced with a
		// placeholder. Treat any non-empty placeholder as "set".
		hasPSK = strings.TrimSpace(s) != ""
	}

	return WirelessSecurity{
		Enabled:         enabled,
		SSID:            ssid,
		SecurityTypeRaw: secType,
		SecurityType:    wirelessSecuritySecurityType(secType),
		Cipher:          cipher,
		KeyRenewalSecs:  renewal,
		HasPreSharedKey: hasPSK,
		Raw:             raw,
	}, nil
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
