package fixture

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Quiarom/router-core/internal/adapters/tplinkwr841v8"
	"github.com/Quiarom/router-core/internal/domain"
)

func TestSyntheticFixtureAdapter(t *testing.T) {
	adapter := New("../../../fixtures/synthetic/tplink-wr841n-v8")
	info, err := adapter.Identify(context.Background())
	if err != nil || info.Provenance != domain.ProvenanceFixture || info.Authenticated != domain.Unknown || info.FirmwareVersion.Empty() {
		t.Fatalf("info=%+v err=%v", info, err)
	}
	status, err := adapter.Status(context.Background())
	if err != nil || status.WANStatus != domain.WANConnected {
		t.Fatalf("status=%+v err=%v", status, err)
	}
	clients, err := adapter.Clients(context.Background())
	if err != nil || len(clients) != 2 {
		t.Fatalf("clients=%+v err=%v", clients, err)
	}
	security, err := adapter.Security(context.Background())
	if err != nil || security.WPSEnabled != domain.True || !security.ForwardingRules.Valid {
		t.Fatalf("security=%+v err=%v", security, err)
	}
}

func TestMissingAndEmptyDHCPObservations(t *testing.T) {
	missing := New(t.TempDir())
	clients, err := missing.Clients(context.Background())
	if clients != nil || !errors.Is(err, domain.ErrObservationAbsent) {
		t.Fatalf("missing clients=%v err=%v", clients, err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "dhcp.html"), []byte(`<html><script>var DHCPDynList = new Array();</script></html>`), 0600); err != nil {
		t.Fatal(err)
	}
	clients, err = New(dir).Clients(context.Background())
	if err != nil || clients == nil || len(clients) != 0 {
		t.Fatalf("empty clients=%v err=%v", clients, err)
	}
}

func TestCapturedFixtureDirectorySkipsWhenEmpty(t *testing.T) {
	files, err := filepath.Glob("../../../fixtures/captured/*.html")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Skip("captured fixtures are optional; sanitized hardware captures are not present")
	}
	for _, file := range files {
		if _, err := os.Stat(file); err != nil {
			t.Fatal(err)
		}
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		switch filepath.Base(file) {
		case "status.html":
			_, err = tplinkwr841v8.ParseStatus(body)
		case "dhcp.html":
			_, err = tplinkwr841v8.ParseDHCP(body)
		case "wps.html":
			_, err = tplinkwr841v8.ParseWPS(body)
		case "dmz.html":
			_, err = tplinkwr841v8.ParseDMZ(body)
		case "upnp.html":
			_, err = tplinkwr841v8.ParseUPnP(body)
		case "remote_management.html":
			_, err = tplinkwr841v8.ParseRemoteManagement(body)
		}
		if err != nil {
			t.Fatalf("%s: %v", file, err)
		}
	}
}
