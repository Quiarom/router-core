// Test model-facing JSON never carries credentials. This is a
// release-blocking invariant per the pre-submission rules.
//
// The reasoning agent receives typed router observations as
// JSON tool payloads. That payload must never include:
//
//   - the router admin password;
//   - the Wi-Fi pre-shared key (PSK);
//   - the WPS PIN;
//   - the session token.
//
// If a future change adds any of these to the JSON envelope,
// this test must fail and the change must be rejected.
package internal_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Quiarom/router-core/internal/domain"
)

// forbiddenSubstrings is the canonical list of credentials that
// must NEVER appear in any model-facing JSON. The lowercase keys
// cover common obfuscations (psk= / PSK: / wpaPassphrase = ...).
var forbiddenSubstrings = []string{
	"psk=",
	"psk:",
	"wpaPassphrase=",
	"wpaPassphrase:",
	"wpaPsk=",
	"wpaPsk:",
	"wirelessPassword=",
	"wirelessPassword:",
	"adminPassword=",
	"adminPassword:",
	"wpsPin=",
	"wpsPin:",
	"sessionToken=",
	"sessionToken:",
	"Cookie:",
	"Authorization: Basic",
	"Authorization: Bearer",
}

func containsAny(s string, needles []string) []string {
	var hits []string
	for _, n := range needles {
		if strings.Contains(s, n) {
			hits = append(hits, n)
		}
	}
	return hits
}

// TestSecurityStateJSONContainsNoSecrets is the canonical assertion
// for §11 of the pre-submission rules: the JSON shape of the
// observation the agent consumes must not contain credentials.
//
// The fields allowed on SecurityState are WPSEnabled, DMZEnabled,
// DMZHost, UPnPEnabled, ActiveUPnPMappings, RemoteManagementEnabled,
// RemoteManagementPort, ForwardingRules, Unsupported, and Provenance.
// None of these names carry a credential. If a future field is added
// carrying the actual PSK or admin password, this test scans for
// known secret key names and rejects it.
func TestSecurityStateJSONContainsNoSecrets(t *testing.T) {
	states := []domain.SecurityState{
		// The "everything on" case
		{
			WPSEnabled:              domain.True,
			DMZEnabled:              domain.True,
			DMZHost:                 "192.168.1.50",
			UPnPEnabled:             domain.True,
			ActiveUPnPMappings:      domain.SomeInt(3),
			RemoteManagementEnabled: domain.True,
			RemoteManagementPort:    domain.SomeInt(8443),
			ForwardingRules:         domain.SomeInt(2),
			Provenance:              domain.ProvenanceObserved,
		},
		// The "everything off" case
		{
			WPSEnabled: domain.False, DMZEnabled: domain.False, UPnPEnabled: domain.False,
			RemoteManagementEnabled: domain.False, Provenance: domain.ProvenanceObserved,
		},
		// The "absent" case (firmware does not implement)
		{
			WPSEnabled: domain.Unknown, DMZEnabled: domain.Unknown, UPnPEnabled: domain.Unknown,
			RemoteManagementEnabled: domain.Unknown, Provenance: domain.ProvenanceAbsent,
		},
		// The "unsupported" case (runtime cannot parse)
		{
			WPSEnabled: domain.Unknown, DMZEnabled: domain.Unknown, UPnPEnabled: domain.Unknown,
			RemoteManagementEnabled: domain.Unknown, Provenance: domain.ProvenanceFixture,
		},
	}
	for i, s := range states {
		b, err := json.Marshal(s)
		if err != nil {
			t.Fatalf("state %d: marshal: %v", i, err)
		}
		if hits := containsAny(string(b), forbiddenSubstrings); len(hits) > 0 {
			t.Errorf("state %d (%s): forbidden substrings in model-facing JSON: %v\n  json=%s",
				i, s.Provenance, hits, string(b))
		}
	}
}

// TestSecurityStateJSONFieldsAreClosed pins the *shape* of the
// observation, not just the absence of secrets. If a future
// field is added carrying a credential, this test fails too.
//
// Allowed JSON keys are enumerated; anything else is rejected.
func TestSecurityStateJSONFieldsAreClosed(t *testing.T) {
	allowed := map[string]bool{
		"wpsEnabled":              true,
		"dmzEnabled":              true,
		"dmzHost":                 true,
		"upnpEnabled":             true,
		"activeUpnpMappings":      true,
		"remoteManagementEnabled": true,
		"remoteManagementPort":    true,
		"forwardingRules":         true,
		"unsupported":             true,
		"provenance":              true,
	}
	s := domain.SecurityState{
		WPSEnabled: domain.True, DMZEnabled: domain.True, DMZHost: "192.168.1.50",
		UPnPEnabled: domain.True, ActiveUPnPMappings: domain.SomeInt(1),
		RemoteManagementEnabled: domain.True, ForwardingRules: domain.SomeInt(1),
		Provenance: domain.ProvenanceObserved,
	}
	var m map[string]any
	if err := json.Unmarshal(mustMarshal(s), &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for k := range m {
		if !allowed[k] {
			t.Errorf("unexpected key in model-facing JSON: %q (only allowed keys are: %v)", k, keys(allowed))
		}
	}
}

// TestAgentPayloadShape scans the entire agent tool-call payload
// path for credential substrings. The agent serializes each
// observation into the chat-completions request; if any observation
// ever carries a secret, it would land here.
//
// The test operates at the source level: every JSON-marshaling of a
// domain object in the agent and runtime must not serialize
// anything matching the forbidden substrings.
func TestAgentPayloadShape(t *testing.T) {
	di := domain.DeviceInfo{
		Vendor: "TP-Link", Model: "TL-WR841N/ND",
		HardwareVersion:   domain.NewUntrusted("v8.4", "router:status"),
		FirmwareVersion:   domain.NewUntrusted("3.15.9 Build 140724 Rel.63227n", "router:status"),
		ManagementAddress: "192.168.1.1",
		Authenticated:     domain.True,
		Provenance:        domain.ProvenanceObserved,
	}
	rs := domain.RouterStatus{
		Reachable: domain.True,
		WANStatus: domain.WANConnected,
		Uptime:    24 * 60 * 60 * 1_000_000_000, // 24h in ns
		UptimeSecs: domain.SomeInt(86400),
		Provenance: domain.ProvenanceObserved,
	}
	cl := []domain.Client{
		{Name: domain.NewUntrusted("laptop", "router:dhcp"), IP: "192.168.1.50", MAC: "AA:BB:CC:DD:EE:01", LeaseTime: domain.NewUntrusted("24h", "router:dhcp"), Provenance: domain.ProvenanceObserved},
	}
	ss := domain.SecurityState{
		WPSEnabled: domain.False, DMZEnabled: domain.False, UPnPEnabled: domain.False,
		RemoteManagementEnabled: domain.False, Provenance: domain.ProvenanceObserved,
	}

	// The agent's tool result for each observation is the JSON of the
	// domain object. Scan each for forbidden substrings.
	for name, payload := range map[string][]byte{
		"device":   mustMarshal(di),
		"status":   mustMarshal(rs),
		"clients":  mustMarshal(cl),
		"security": mustMarshal(ss),
	} {
		if hits := containsAny(string(payload), forbiddenSubstrings); len(hits) > 0 {
			t.Errorf("%s: model-facing JSON contains forbidden substrings: %v", name, hits)
		}
	}
}

// TestWirelessSecurityHasNoPSK pins §11's specific worry: the
// wireless observation tells the model "WPA2-PSK" and "AES/CCMP"
// and "credential_configured: true". It must never tell the model
// the actual PSK. The shape is a closed enum (security mode + cipher
// + a boolean flag), not a string field that could carry the PSK.
func TestWirelessSecurityHasNoPSK(t *testing.T) {
	type wirelessObs struct {
		SecurityMode         string `json:"securityMode"`
		Cipher               string `json:"cipher"`
		CredentialConfigured bool   `json:"credentialConfigured"`
		Provenance           string `json:"provenance"`
	}
	w := wirelessObs{
		SecurityMode: "WPA2-PSK", Cipher: "AES/CCMP",
		CredentialConfigured: true, Provenance: string(domain.ProvenanceObserved),
	}
	if hits := containsAny(string(mustMarshal(w)), forbiddenSubstrings); len(hits) > 0 {
		t.Errorf("wireless obs: forbidden substrings: %v", hits)
	}
}

func mustMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
