package tplinkwr841v8

import (
	"os"

	"github.com/Quiarom/router-core/internal/domain"
)

type Endpoint struct {
	Op            string
	Path          string
	DashboardPage string
	Verified      bool
	CaptureNote   string
}

const (
	OpStatus           = "status"
	OpDHCPClients      = "dhcp-clients"
	OpWPS              = "wps"
	OpForwarding       = "forwarding"
	OpDMZ              = "dmz"
	OpUPnP             = "upnp"
	OpRemoteManagement = "remote-management"
)

// These paths come from public sources and are unconfirmed against our hardware.
var Endpoints = map[string]Endpoint{
	OpStatus: {
		Op: OpStatus, Path: "/userRpm/StatusRpm.htm", DashboardPage: "Status",
		Verified:    true,
		CaptureNote: "verified 2026-08-30 against physical lab unit (3.13.33 Build 130506 Rel.48660n); auth via HTTP Basic header (plaintext password, NOT md5hex) per ADR 0005",
	},
	OpDHCPClients: {
		Op: OpDHCPClients, Path: "/userRpm/AssignedIpAddrListRpm.htm", DashboardPage: "DHCP Clients",
		Verified:    true,
		CaptureNote: "verified 2026-08-31 against physical lab unit (3.13.33 Build 130506 Rel.48660n); auth via HTTP Basic header (plaintext password, NOT md5hex) per ADR 0005",
	},
	OpWPS: {
		Op: OpWPS, Path: "/userRpm/WpsRpm.htm", DashboardPage: "WPS",
		Verified:    false,
		CaptureNote: "capture the WPS dashboard request and response",
	},
	OpForwarding: {
		Op: OpForwarding, Path: "/userRpm/VirtualServerRpm.htm", DashboardPage: "Forwarding",
		Verified:    false,
		CaptureNote: "capture the DMZ/Virtual Servers dashboard request and response",
	},
	OpDMZ: {
		Op: OpDMZ, Path: "/userRpm/DMZRpm.htm", DashboardPage: "DMZ",
		Verified:    false,
		CaptureNote: "capture the separate DMZ page; this page split is unconfirmed and DMZ may live on the forwarding page on this build",
	},
	OpUPnP: {
		Op: OpUPnP, Path: "/userRpm/UpnpRpm.htm", DashboardPage: "UPnP",
		Verified:    false,
		CaptureNote: "capture the UPnP dashboard request and response",
	},
	OpRemoteManagement: {
		Op: OpRemoteManagement, Path: "/userRpm/AccessCtrlRpm.htm", DashboardPage: "Remote Management",
		Verified:    false,
		CaptureNote: "capture the remote-management dashboard request and response",
	},
}

func dispatchAllowed(endpoint Endpoint) error {
	if endpoint.Verified || os.Getenv("ROUTER_ALLOW_UNVERIFIED") == "1" {
		return nil
	}
	return domain.ErrUnverifiedEndpoint
}
