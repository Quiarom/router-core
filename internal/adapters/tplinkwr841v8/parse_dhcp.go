package tplinkwr841v8

import (
	"net"
	"regexp"
	"strings"

	"github.com/Quiarom/router-core/internal/domain"
)

// These indices are UNVERIFIED until a sanitized DHCP capture is available.
const dhcpGroupSize = 4

type DHCPResult struct {
	Clients []domain.Client
	Skipped int
}

var macPattern = regexp.MustCompile(`^[0-9a-fA-F]{2}([:-][0-9a-fA-F]{2}){5}$`)

func ParseDHCP(html []byte) (DHCPResult, error) {
	tokens, err := arrayOrError(html, "DHCPDynList")
	if err != nil {
		return DHCPResult{}, err
	}
	result := DHCPResult{Clients: make([]domain.Client, 0)}
	for i := 0; i < len(tokens); i += dhcpGroupSize {
		if i+dhcpGroupSize > len(tokens) {
			result.Skipped++
			break
		}
		name, nameOK := Str(tokens, i)
		mac, macOK := Str(tokens, i+1)
		ip, ipOK := Str(tokens, i+2)
		lease, leaseOK := Str(tokens, i+3)
		parsedIP := net.ParseIP(ip)
		if !nameOK || !macOK || !ipOK || !leaseOK || parsedIP == nil || !macPattern.MatchString(mac) {
			result.Skipped++
			continue
		}
		result.Clients = append(result.Clients, domain.Client{
			Name: domain.NewUntrusted(name, "router:dhcp-client-list"),
			IP:   parsedIP.String(), MAC: normalizeMAC(mac),
			LeaseTime:  domain.NewUntrusted(lease, "router:dhcp-client-list"),
			Provenance: domain.ProvenanceObserved,
		})
	}
	return result, nil
}

func normalizeMAC(mac string) string {
	parts := strings.FieldsFunc(strings.ToLower(mac), func(r rune) bool { return r == ':' || r == '-' })
	return strings.Join(parts, ":")
}
