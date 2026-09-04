// Package cmd: the application layer for the router-core binary.
//
// Each RunE in the Cobra subcommand files delegates to one of
// the run* functions defined here. These functions take a
// typed options struct (see options.go) and do not know about
// Cobra. The runtime logic (probe, inspect, serve) is called
// here without going through package main's flag-based
// dispatcher.
//
// This separation keeps Cobra concerns (flag parsing, help
// rendering, error routing) at the command layer, and keeps
// the application logic testable without instantiating a
// command tree.
package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Quiarom/router-core/internal/adapters/fixture"
	"github.com/Quiarom/router-core/internal/adapters/tplinkwr841v8"
	"github.com/Quiarom/router-core/internal/domain"
	"github.com/Quiarom/router-core/internal/transport"
)

// runProbe is the application function behind `router-core probe`.
// It builds the adapter (fixture-backed if --fixtures is set,
// otherwise a real TP-Link WR841N adapter), then prints the
// device identity.
func runProbe(opts probeOptions) error {
	adapter, err := buildReadAdapter(opts.Host, opts.Fixtures, opts.Timeout)
	if err != nil {
		return err
	}
	return probeApp(context.Background(), adapter, opts.JSON)
}

// runInspect is the application function behind `router-core inspect`.
func runInspect(opts inspectOptions) error {
	adapter, err := buildReadAdapter(opts.Host, opts.Fixtures, opts.Timeout)
	if err != nil {
		return err
	}
	return inspectApp(context.Background(), adapter, opts.JSON)
}

// buildReadAdapter builds a router adapter for the read commands
// (probe, inspect). The fixture adapter is used when --fixtures
// is set; otherwise the real TP-Link adapter is constructed.
func buildReadAdapter(host, fixtures string, timeout time.Duration) (domain.RouterAdapter, error) {
	if fixtures != "" {
		return fixture.New(fixtures), nil
	}
	return tplinkwr841v8.New(host, transport.WithTimeout(timeout)), nil
}

// probeApp prints the device identity returned by the adapter.
// text by default, JSON with opts.JSON.
func probeApp(ctx context.Context, adapter domain.RouterAdapter, jsonOut bool) error {
	info, err := adapter.Identify(ctx)
	if err != nil {
		return actionable(err)
	}
	if jsonOut {
		return printJSON(info)
	}
	auth := "blocked: missing capture (see BLOCKED_CAPTURE.md)"
	if info.Provenance == domain.ProvenanceFixture {
		auth = "n/a (fixture replay)"
	} else if info.Authenticated == domain.True {
		auth = "success"
	} else if info.Authenticated == domain.False {
		auth = "failed: authentication rejected"
	}
	fmt.Printf("%s %s\nHardware: %s\nFirmware: %s\nHost: %s\nAuthentication: %s\n",
		info.Vendor, info.Model, untrusted(info.HardwareVersion), untrusted(info.FirmwareVersion),
		info.ManagementAddress, auth)
	return nil
}

// inspectApp prints status, clients, and security observations.
func inspectApp(ctx context.Context, adapter domain.RouterAdapter, jsonOut bool) error {
	status, err := adapter.Status(ctx)
	if err != nil {
		return actionable(err)
	}
	clients, err := adapter.Clients(ctx)
	clientsAbsent := errors.Is(err, domain.ErrObservationAbsent)
	if err != nil && !clientsAbsent {
		return actionable(err)
	}
	security, err := adapter.Security(ctx)
	if err != nil {
		return actionable(err)
	}
	var clientCount interface{} = len(clients)
	clientDisplay := fmt.Sprintf("%d", len(clients))
	if clientsAbsent {
		clientCount = nil
		clientDisplay = "unknown"
	}
	output := struct {
		Reachable  domain.Tristate  `json:"reachable"`
		WAN        domain.WANStatus `json:"wan"`
		Clients    interface{}      `json:"clients"`
		WPS        domain.Tristate  `json:"wps"`
		DMZ        domain.Tristate  `json:"dmz"`
		UPnP       domain.Tristate  `json:"upnp"`
		RemoteMgmt domain.Tristate  `json:"remoteManagement"`
	}{
		status.Reachable, status.WANStatus, clientCount, security.WPSEnabled,
		security.DMZEnabled, security.UPnPEnabled, security.RemoteManagementEnabled,
	}
	if jsonOut {
		return printJSON(output)
	}
	fmt.Printf("Reachable: %s\nWAN: %s\nClients: %s\nWPS: %s\nDMZ: %s\nUPnP: %s\nRemote management: %s\n",
		output.Reachable, output.WAN, clientDisplay, output.WPS, output.DMZ,
		output.UPnP, output.RemoteMgmt)
	return nil
}

// untrusted returns the string value of a domain.Untrusted, or
// "unknown" if empty.
func untrusted(value domain.Untrusted) string {
	if value.Empty() {
		return "unknown"
	}
	return value.Value()
}

// printJSON marshals v to indented JSON and writes to stdout.
func printJSON(value interface{}) error {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(encoded))
	return nil
}

// actionable converts a domain error into a human-friendly
// string with a "what to do next" hint.
func actionable(err error) error {
	if errors.Is(err, domain.ErrCaptureMissing) {
		return fmt.Errorf("blocked: missing capture; see BLOCKED_CAPTURE.md")
	}
	if errors.Is(err, domain.ErrUnverifiedEndpoint) {
		return fmt.Errorf("blocked: endpoint is unverified; see BLOCKED_CAPTURE.md (set ROUTER_ALLOW_UNVERIFIED=1 only for explicit local testing)")
	}
	return err
}
