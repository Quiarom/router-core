package fixture

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Quiarom/router-core/internal/adapters/tplinkwr841v8"
	"github.com/Quiarom/router-core/internal/domain"
)

type Adapter struct{ dir string }

func New(dir string) *Adapter { return &Adapter{dir: dir} }

func (a *Adapter) read(name string) ([]byte, bool, error) {
	body, err := os.ReadFile(filepath.Join(a.dir, name))
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	return body, err == nil, err
}

func (a *Adapter) Identify(context.Context) (domain.DeviceInfo, error) {
	info := domain.DeviceInfo{
		Vendor: "TP-Link", Model: tplinkwr841v8.ModelName, ManagementAddress: "fixture",
		Authenticated: domain.Unknown, Provenance: domain.ProvenanceFixture,
	}
	if body, exists, err := a.read("status.html"); err != nil {
		return info, err
	} else if exists {
		firmware, hardware, parseErr := tplinkwr841v8.ParseIdentity(body)
		if parseErr != nil {
			return info, parseErr
		}
		info.FirmwareVersion = domain.NewUntrusted(firmware, "router:status")
		info.HardwareVersion = domain.NewUntrusted(hardware, "router:status")
	}
	return info, nil
}

func (a *Adapter) Status(context.Context) (domain.RouterStatus, error) {
	body, exists, err := a.read("status.html")
	if err != nil {
		return domain.RouterStatus{}, err
	}
	if !exists {
		return missingStatus(), nil
	}
	status, err := tplinkwr841v8.ParseStatus(body)
	status.Provenance = domain.ProvenanceFixture
	return status, err
}

func (a *Adapter) Clients(context.Context) ([]domain.Client, error) {
	body, exists, err := a.read("dhcp.html")
	if err != nil || !exists {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%w: dhcp.html", domain.ErrObservationAbsent)
	}
	result, err := tplinkwr841v8.ParseDHCP(body)
	for i := range result.Clients {
		result.Clients[i].Provenance = domain.ProvenanceFixture
	}
	return result.Clients, err
}

func (a *Adapter) Security(context.Context) (domain.SecurityState, error) {
	state := domain.SecurityState{Provenance: domain.ProvenanceFixture}
	pages := []struct {
		file  string
		parse func([]byte) (domain.SecurityState, error)
		field string
	}{
		{"wps.html", tplinkwr841v8.ParseWPS, "wps"},
		{"dmz.html", tplinkwr841v8.ParseDMZ, "dmz"},
		{"upnp.html", tplinkwr841v8.ParseUPnP, "upnp"},
		{"remote_management.html", tplinkwr841v8.ParseRemoteManagement, "remoteManagement"},
	}
	for _, page := range pages {
		body, exists, err := a.read(page.file)
		if err != nil {
			return state, err
		}
		if !exists {
			state.MarkUnsupported(page.field, "no capture for "+page.field)
			continue
		}
		part, err := page.parse(body)
		if err != nil {
			return state, err
		}
		state.Merge(part)
	}
	if body, exists, err := a.read("dmz.html"); err != nil {
		return state, err
	} else if exists {
		part, err := tplinkwr841v8.ParseForwarding(body)
		if err != nil {
			return state, err
		}
		state.Merge(part)
	} else {
		state.MarkUnsupported("forwardingRules", "no capture for forwardingRules")
	}
	return state, nil
}

func missingStatus() domain.RouterStatus {
	return domain.RouterStatus{Reachable: domain.Unknown, WANStatus: domain.WANUnknown, Provenance: domain.ProvenanceFixture}
}
