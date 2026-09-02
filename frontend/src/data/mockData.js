export const mockDevice = {
  vendor: "TP-Link",
  model: "TL-WR841N/ND",
  hardwareVersion: {
    value: "WR841N v8 00000000",
    trust: "untrusted",
    source: "router:status"
  },
  firmwareVersion: {
    value: "3.15.9 Build 140724 Rel.63227n",
    trust: "untrusted",
    source: "router:status"
  },
  managementAddress: "192.168.1.1",
  authenticated: "true",
  provenance: "observed"
};

export const mockStatus = {
  reachable: "true",
  wanStatus: "connected",
  uptimeSeconds: 20000,
  uptime: "5h33m20s",
  provenance: "observed"
};

export const mockClients = {
  state: "verified",
  clients: [
    {
      name: "omarchy-laptop",
      mac: "00:11:22:33:44:55",
      ip: "192.168.1.100",
      lease: "01:24:35"
    },
    {
      name: "pixel-phone",
      mac: "AA:BB:CC:11:22:33",
      ip: "192.168.1.101",
      lease: "01:58:12"
    },
    {
      name: "smart-tv-livingroom",
      mac: "10:20:30:40:50:60",
      ip: "192.168.1.105",
      lease: "Permanent"
    }
  ]
};

export const mockCapabilities = {
  capabilities: {
    device: "verified",
    status: "verified",
    clients: "verified",
    wireless_security: "unverified",
    wps: "absent",
    dmz: "verified",
    upnp: "absent",
    remote_management: "absent",
    forwarding: "verified"
  }
};

export const mockSecurity = {
  dmz: {
    state: "verified",
    dmz_enabled: "false",
    dmz_host: "",
    description: "Demilitarized Zone forwarding is disabled. No internal host is fully exposed to WAN."
  },
  forwarding: {
    state: "verified",
    forwarding_rules: [],
    description: "Virtual server and port forwarding rules. Currently 0 active forwarded ports."
  },
  wireless: {
    state: "unavailable",
    reason: "router-core: endpoint is unverified against captured traffic",
    description: "WPA2-PSK status extraction pending lab verification."
  },
  wps: {
    state: "unsupported_or_unverified",
    reason: "WPS endpoint not present on this firmware build (HTTP 501)",
    description: "Wi-Fi Protected Setup is absent or unsupported in hardware/firmware."
  },
  upnp: {
    state: "unsupported_or_unverified",
    reason: "UPnP endpoint not present on this firmware build (HTTP 501)",
    description: "Universal Plug and Play is absent or unsupported in hardware/firmware."
  },
  remote_management: {
    state: "unsupported_or_unverified",
    reason: "Remote Management endpoint not present on this firmware build (HTTP 501)",
    description: "Remote WAN management interface is disabled/absent."
  }
};
