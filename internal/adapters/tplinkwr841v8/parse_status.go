package tplinkwr841v8

import (
	"strconv"
	"strings"
	"time"

	"github.com/Quiarom/router-core/internal/domain"
)

// These indices are UNVERIFIED until a sanitized Status capture is available.
const (
	statusFirmwareIndex = 6
	statusHardwareIndex = 7
	statusUptimeIndex   = 8
	statusWANIndex      = 9
)

func ParseIdentity(html []byte) (firmware, hardware string, err error) {
	tokens, err := arrayOrError(html, "statusPara")
	if err != nil {
		return "", "", err
	}
	firmware, _ = Str(tokens, statusFirmwareIndex)
	hardware, _ = Str(tokens, statusHardwareIndex)
	return firmware, hardware, nil
}

func ParseStatus(html []byte) (domain.RouterStatus, error) {
	tokens, err := arrayOrError(html, "statusPara")
	if err != nil {
		return domain.RouterStatus{}, err
	}
	out := domain.RouterStatus{Reachable: domain.True, WANStatus: domain.WANUnknown, Provenance: domain.ProvenanceObserved}
	if n, ok := Int(tokens, statusUptimeIndex); ok && n >= 0 {
		out.UptimeSecs = domain.SomeInt(n)
		out.Uptime = time.Duration(n) * time.Second
	}
	if value, ok := Str(tokens, statusWANIndex); ok {
		out.WANStatus = normalizeWAN(value)
	} else if n, ok := Int(tokens, statusWANIndex); ok {
		out.WANStatus = normalizeWAN(strconv.Itoa(n))
	}
	return out, nil
}

func normalizeWAN(value string) domain.WANStatus {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "connected", "up", "online", "pppoe connected":
		return domain.WANConnected
	case "0", "disconnected", "down", "offline":
		return domain.WANDisconnected
	case "2", "connecting", "dialing":
		return domain.WANConnecting
	case "disabled", "disable":
		return domain.WANDisabled
	default:
		return domain.WANUnknown
	}
}
