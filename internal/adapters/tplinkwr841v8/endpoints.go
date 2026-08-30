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
	OpUPnP             = "upnp"
	OpRemoteManagement = "remote-management"
)

// These paths come from public sources and are unconfirmed against our hardware.
var Endpoints = map[string]Endpoint{
	OpStatus: {
		Op: OpStatus, Path: "/userRpm/StatusRpm.htm", DashboardPage: "Status",
		Verified:    false,
		CaptureNote: "capture the Status dashboard request and response",
	},
	OpDHCPClients: {
		Op: OpDHCPClients, Path: "/userRpm/AssignedIpAddrListRpm.htm", DashboardPage: "DHCP Clients",
		Verified:    false,
		CaptureNote: "capture the DHCP Clients dashboard request and response",
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
