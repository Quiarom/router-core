package domain

import "time"

// Provenance records how a normalized observation was obtained.
type Provenance string

const (
	// ProvenanceObserved: the value was read from a response of the device.
	ProvenanceObserved Provenance = "observed"
	// ProvenanceAbsent: the response did not contain the field. This is NOT
	// the same as a negative value.
	ProvenanceAbsent Provenance = "absent"
	// ProvenanceFixture: the value came from a replayed fixture, not from
	// the physical device.
	ProvenanceFixture Provenance = "fixture"
)

// DeviceInfo identifies the managed device.
type DeviceInfo struct {
	Vendor            string     `json:"vendor"`
	Model             string     `json:"model"`
	HardwareVersion   Untrusted  `json:"hardwareVersion"`
	FirmwareVersion   Untrusted  `json:"firmwareVersion"`
	ManagementAddress string     `json:"managementAddress"`
	Authenticated     Tristate   `json:"authenticated"`
	Provenance        Provenance `json:"provenance"`
}

// RouterStatus is the normalized operational state of the device.
type RouterStatus struct {
	Reachable  Tristate      `json:"reachable"`
	WANStatus  WANStatus     `json:"wanStatus"`
	Uptime     time.Duration `json:"-"`
	UptimeSecs OptInt        `json:"uptimeSeconds"`
	Provenance Provenance    `json:"provenance"`
}

// WANStatus is a vendor-independent WAN connectivity state.
type WANStatus string

const (
	WANUnknown      WANStatus = "unknown"
	WANConnected    WANStatus = "connected"
	WANDisconnected WANStatus = "disconnected"
	WANConnecting   WANStatus = "connecting"
	WANDisabled     WANStatus = "disabled"
)

// Client is one device observed on the LAN.
//
// Every human-readable attribute is untrusted: it is chosen by the client
// device, not by the router and not by the user.
type Client struct {
	Name       Untrusted  `json:"name"`
	IP         string     `json:"ip"`
	MAC        string     `json:"mac"`
	LeaseTime  Untrusted  `json:"leaseTime"`
	Provenance Provenance `json:"provenance"`
}

// SecurityState carries the deterministic, read-only security facts that P0
// supports. Interpretation of these facts (is this network exposed? what should
// be changed?) deliberately lives outside router-core.
type SecurityState struct {
	WPSEnabled              Tristate   `json:"wpsEnabled"`
	DMZEnabled              Tristate   `json:"dmzEnabled"`
	DMZHost                 string     `json:"dmzHost,omitempty"`
	UPnPEnabled             Tristate   `json:"upnpEnabled"`
	ActiveUPnPMappings      OptInt     `json:"activeUpnpMappings"`
	RemoteManagementEnabled Tristate   `json:"remoteManagementEnabled"`
	RemoteManagementPort    OptInt     `json:"remoteManagementPort"`
	ForwardingRules         OptInt     `json:"forwardingRules"`
	Provenance              Provenance `json:"provenance"`

	// Unsupported lists the P0 security fields that this adapter could not
	// observe, with the reason. Consumers must be able to tell "not observed"
	// from "observed as disabled".
	Unsupported map[string]string `json:"unsupported,omitempty"`
}

// MarkUnsupported records that a field could not be observed.
func (s *SecurityState) MarkUnsupported(field, reason string) {
	if s.Unsupported == nil {
		s.Unsupported = map[string]string{}
	}
	s.Unsupported[field] = reason
}
