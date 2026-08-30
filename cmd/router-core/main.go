package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Quiarom/router-core/internal/adapters/fixture"
	"github.com/Quiarom/router-core/internal/adapters/tplinkwr841v8"
	"github.com/Quiarom/router-core/internal/domain"
	"github.com/Quiarom/router-core/internal/transport"
)

const version = "0.1.0"

type options struct {
	host     string
	fixtures string
	json     bool
	timeout  time.Duration
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		usage()
		return nil
	}
	if args[0] == "--version" {
		fmt.Println(version)
		return nil
	}
	if args[0] != "probe" && args[0] != "inspect" {
		return fmt.Errorf("unknown subcommand %q (use -h for usage)", args[0])
	}
	opts := options{host: "192.168.0.1", timeout: 5 * time.Second}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&opts.host, "host", opts.host, "local router address")
	fs.StringVar(&opts.fixtures, "fixtures", "", "fixture directory")
	fs.BoolVar(&opts.json, "json", false, "write JSON")
	fs.DurationVar(&opts.timeout, "timeout", opts.timeout, "per-request timeout")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	var adapter domain.RouterAdapter
	if opts.fixtures != "" {
		adapter = fixture.New(opts.fixtures)
	} else {
		adapter = tplinkwr841v8.New(opts.host, transport.WithTimeout(opts.timeout))
	}
	ctx := context.Background()
	if args[0] == "probe" {
		return probe(ctx, adapter, opts)
	}
	return inspect(ctx, adapter, opts)
}

func probe(ctx context.Context, adapter domain.RouterAdapter, opts options) error {
	info, err := adapter.Identify(ctx)
	if err != nil {
		return actionable(err)
	}
	if opts.json {
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

func inspect(ctx context.Context, adapter domain.RouterAdapter, opts options) error {
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
	if opts.json {
		return printJSON(output)
	}
	fmt.Printf("Reachable: %s\nWAN: %s\nClients: %s\nWPS: %s\nDMZ: %s\nUPnP: %s\nRemote management: %s\n",
		output.Reachable, output.WAN, clientDisplay, output.WPS, output.DMZ,
		output.UPnP, output.RemoteMgmt)
	return nil
}

func untrusted(value domain.Untrusted) string {
	if value.Empty() {
		return "unknown"
	}
	return value.Value()
}

func printJSON(value interface{}) error {
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(encoded))
	return nil
}

func actionable(err error) error {
	if errors.Is(err, domain.ErrCaptureMissing) {
		return fmt.Errorf("blocked: missing capture; see BLOCKED_CAPTURE.md")
	}
	if errors.Is(err, domain.ErrUnverifiedEndpoint) {
		return fmt.Errorf("blocked: endpoint is unverified; see BLOCKED_CAPTURE.md (set ROUTER_ALLOW_UNVERIFIED=1 only for explicit local testing)")
	}
	return err
}

func usage() {
	fmt.Println("Usage: router-core <probe|inspect> [--host HOST] [--fixtures DIR] [--json] [--timeout DURATION]")
	fmt.Println("       router-core --version")
}
