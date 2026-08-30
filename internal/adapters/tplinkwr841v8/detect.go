package tplinkwr841v8

import (
	"strings"

	"github.com/Quiarom/router-core/internal/domain"
)

func IsLoginPage(html []byte) bool {
	s := strings.ToLower(string(html))
	for _, name := range knownArrays {
		if strings.Contains(s, strings.ToLower(name)) {
			return false
		}
	}
	return strings.Contains(s, "loginpassword") ||
		strings.Contains(s, "loginpwd") ||
		strings.Contains(s, "name=\"login\"") ||
		(strings.Contains(s, "password") && strings.Contains(s, "login"))
}

var knownArrays = []string{
	"statusPara", "DHCPDynList", "wpsPara", "dmzPara",
	"upnpPara", "remotePara", "virtualServerPara",
}

// Classify rejects authentication pages and responses with no recognizable HTML.
func Classify(html []byte) error {
	if IsLoginPage(html) {
		return domain.ErrUnauthenticated
	}
	trimmed := strings.TrimSpace(string(html))
	if trimmed == "" || !strings.Contains(strings.ToLower(trimmed), "<") {
		return domain.ErrUnexpectedResponse
	}
	for _, name := range knownArrays {
		if strings.Contains(trimmed, name) {
			return nil
		}
	}
	return domain.ErrUnexpectedResponse
}

func arrayOrError(html []byte, name string) ([]Token, error) {
	if err := Classify(html); err != nil {
		return nil, err
	}
	tokens, ok := ExtractArray(html, name)
	if !ok {
		return nil, domain.ErrUnexpectedResponse
	}
	return tokens, nil
}
