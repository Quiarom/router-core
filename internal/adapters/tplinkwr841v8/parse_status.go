package tplinkwr841v8

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Quiarom/router-core/internal/domain"
)

var errUnexpectedShape = errors.New("router-core: unexpected response shape")

// firmwarePattern matches the firmware string emitted by the WR841N
// v8.4 firmware family in the `var statusPara = new Array(...)` block.
// E.g. "3.15.9 Build 140724 Rel.63227n". The shape is stable across
// minor builds; the position inside the array is not.
var firmwarePattern = regexp.MustCompile(`(\d+\.\d+\.\d+ Build \d+ Rel\.\d+[a-z]*\s*)`)

// hardwarePattern matches the hardware string. E.g.
// "WR841N v8 00000000". Stable across the v8.x family.
var hardwarePattern = regexp.MustCompile(`(WR\d+N v\d+ \d+)`)

// These indices are kept for ParseStatus (uptime, WAN link) where
// the field is positional. Firmware and hardware are now extracted
// by pattern because their positions changed between firmware
// builds (3.13.33 vs 3.15.9).
const (
	statusUptimeIndex = 3
	statusWANIndex    = 9
)

func ParseIdentity(html []byte) (firmware, hardware string, err error) {
	body := string(html)
	if m := firmwarePattern.FindStringSubmatch(body); len(m) >= 2 {
		firmware = strings.TrimSpace(m[1])
	}
	if m := hardwarePattern.FindStringSubmatch(body); len(m) >= 2 {
		hardware = strings.TrimSpace(m[1])
	}
	if firmware == "" && hardware == "" {
		return "", "", errUnexpectedShape
	}
	return firmware, hardware, nil
}

func ParseStatus(html []byte) (domain.RouterStatus, error) {
	tokens, err := arrayOrError(html, "statusPara")
	if err == nil {
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
	if _, _, perr := ParseIdentity(html); perr == nil {
		return domain.RouterStatus{Reachable: domain.True, WANStatus: domain.WANUnknown, Provenance: domain.ProvenanceObserved}, nil
	}
	return domain.RouterStatus{}, errUnexpectedShape
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
