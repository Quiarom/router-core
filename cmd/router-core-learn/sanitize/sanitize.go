// Package sanitize redacts secret material from strings before they are
// persisted as physical capture evidence. Fail-closed: if a candidate
// value cannot be classified with confidence, the value is omitted from
// the output rather than passed through unredacted.
//
// Sanitization policy is defined in docs/PRIOR_ART_PROTOCOL.md and the
// Phase 2 plan. The redact() function returns "" when it cannot classify
// a value, so the caller decides whether to drop the field or the whole
// record.
package sanitize

import (
	"regexp"
	"strings"
)

// Category identifies what kind of placeholder to use.
type Category string

const (
	PlaceholderAuthSecret     = "<AUTH_SECRET>"
	PlaceholderSessionToken   = "<SESSION_TOKEN>"
	PlaceholderRouterMAC      = "<ROUTER_MAC>"
	PlaceholderClientMAC      = "<CLIENT_MAC>"
	PlaceholderClientHostname = "<CLIENT_HOSTNAME>"
	PlaceholderSSID           = "<SSID>"
	PlaceholderWiFiPassword   = "<WIFI_PASSWORD>"
	PlaceholderSecret         = "<SECRET>"
)

// Structural session-token redirect shape observed in the prior art.
// Matches both the redirect script (window.parent.location.href = "...")
// and any embedded /<TOKEN>/userRpm/ path. The token capture group is
// group 1.
var (
	sessionTokenRedirectScript = regexp.MustCompile(
		`window\.parent\.location\.href\s*=\s*["']http://[0-9A-Za-z.\-]+/([A-Za-z0-9]{16})/userRpm/Index\.htm["']`,
	)
	sessionTokenPathAnywhere = regexp.MustCompile(
		`/[A-Za-z0-9]{16}/userRpm/`,
	)
	sessionTokenBare = regexp.MustCompile(
		`\b[A-Za-z0-9]{16}\b`,
	)
	// Quoted bare token: catches assignments like debug="ABCDEFGHIJKLMNOP"
	// or debug='ABCDEFGHIJKLMNOP'. Go's RE2 has no backreferences so we
	// compile two narrow alternations.
	sessionTokenDoubleQuoted = regexp.MustCompile(`"[A-Za-z0-9]{16}"`)
	sessionTokenSingleQuoted = regexp.MustCompile(`'[A-Za-z0-9]{16}'`)

	// Basic <base64> Authorization value: a long base64 blob preceded by
	// (optionally URL-encoded) "Basic ".
	basicAuthValue = regexp.MustCompile(
		`(Basic(?:%20|\s)+)?[A-Za-z0-9+/=]{20,}`,
	)

	// MAC addresses.
	macAny = regexp.MustCompile(
		`\b([0-9A-Fa-f]{2}[:-]){5}[0-9A-Fa-f]{2}\b`,
	)

	// Password-style form fields: password=...& or "password":"...".
	formPasswordField = regexp.MustCompile(
		`(?i)(password|pwd|passwd|loginPwd|loginPassword|Authorization)=([^&\s"]*)`,
	)
	jsonPasswordField = regexp.MustCompile(
		`(?i)("(?:password|pwd|passwd|loginPwd|loginPassword)")\s*:\s*"([^"]*)"`,
	)

	// SSID / wireless key / Wi-Fi PSK fields.
	ssidField = regexp.MustCompile(
		`(?i)(ssid|wirelessPara(?:Name|Ssid)?|wlanSsid)\s*[=:]\s*"?([^";,\s&]+)"?`,
	)
	wifiPskField = regexp.MustCompile(
		`(?i)(psk|wpaPassphrase|wifiPassword|wirelessPassword|wpaPsk)\s*[=:]\s*"?([^";,\s&]+)"?`,
	)

	// Client identifier names in DHCP-shaped pages (heuristic).
	clientHostnameField = regexp.MustCompile(
		`(?i)(?:hostName|hostname|clientName)\s*[=:]\s*"?([^";,\s&]+)"?`,
	)
)

// Options control which categories to redact. Default zero value applies
// the full policy.
type Options struct {
	// PreserveFingerprint keeps router model / hardware / firmware
	// strings verbatim. These are intentionally part of the adapter
	// evidence and must not be redacted.
	PreserveFingerprint bool

	// RouterMACOnly treats MAC matches inside known router-fingerprint
	// contexts as the router's own MAC. Without this flag, MAC matches
	// are treated as CLIENT MAC.
	RouterMACOnly bool

	// AllowEmptyTokenMatch, when true, lets sessionTokenPathAnywhere
	// accept the empty string (no match). Default false: any match is
	// redacted.
	AllowEmptyTokenMatch bool
}

// Default returns the policy used for sanitizing TP-Link capture
// evidence: full redaction of secrets, MACs, SSIDs, and session
// material; fingerprints preserved.
func Default() Options {
	return Options{PreserveFingerprint: true}
}

// Apply redacts secrets in s and returns the result. When the redaction
// step cannot classify a value with confidence, the corresponding match
// is dropped (replaced with empty string) and the caller decides
// whether to keep the surrounding structure.
//
// s may be any UTF-8 string (URL, header value, HTML body, JSON-ish
// payload).
func Apply(s string, opts Options) string {
	if s == "" {
		return s
	}
	// 1. Session token: structural redirect first (matches the exact
	// shape we know the firmware emits), then any /<TOKEN>/userRpm/
	// path, then bare 16-char tokens.
	s = sessionTokenRedirectScript.ReplaceAllStringFunc(s, func(m string) string {
		return strings.Replace(m, sessionTokenRedirectScript.FindStringSubmatch(m)[1], PlaceholderSessionToken, 1)
	})
	s = sessionTokenPathAnywhere.ReplaceAllStringFunc(s, func(m string) string {
		// Extract the token between the leading / and the trailing /userRpm/.
		start := strings.Index(m, "/") + 1
		end := strings.Index(m, "/userRpm/")
		if start <= 0 || end <= start {
			return m
		}
		return "/" + PlaceholderSessionToken + m[end:]
	})
	// Quoted 16-char tokens (debug="..." or '...' values). Safe to
	// redact regardless of PreserveFingerprint because fingerprints are
	// never emitted as bare quoted strings.
	sessionTokenReplace := func(quote byte) {
		re := sessionTokenDoubleQuoted
		placeholder := `"`
		if quote == '\'' {
			re = sessionTokenSingleQuoted
			placeholder = `'`
		}
		s = re.ReplaceAllStringFunc(s, func(m string) string {
			return placeholder + PlaceholderSessionToken + placeholder
		})
	}
	sessionTokenReplace('"')
	sessionTokenReplace('\'')
	// Bare 16-char tokens only when PreserveFingerprint is false, to
	// avoid redacting fingerprint fragments like the hardware serial.
	if !opts.PreserveFingerprint {
		s = sessionTokenBare.ReplaceAllString(s, PlaceholderSessionToken)
	}

	// 2. Authorization material.
	s = basicAuthValue.ReplaceAllString(s, PlaceholderAuthSecret)

	// 3. MAC addresses. Router MAC vs client MAC distinction is
	// context-dependent; we apply RouterMAC placeholder when the
	// surrounding text contains "wan" or "routerMac" hints.
	s = redactMACs(s, opts.RouterMACOnly)

	// 4. Password form/JSON fields.
	s = formPasswordField.ReplaceAllStringFunc(s, func(m string) string {
		sm := formPasswordField.FindStringSubmatch(m)
		if len(sm) < 3 {
			return ""
		}
		return sm[1] + "=" + PlaceholderSecret
	})
	s = jsonPasswordField.ReplaceAllStringFunc(s, func(m string) string {
		sm := jsonPasswordField.FindStringSubmatch(m)
		if len(sm) < 3 {
			return ""
		}
		return sm[1] + `:"` + PlaceholderSecret + `"`
	})

	// 5. SSID / Wi-Fi key.
	s = ssidField.ReplaceAllStringFunc(s, func(m string) string {
		sm := ssidField.FindStringSubmatch(m)
		if len(sm) < 3 {
			return ""
		}
		return sm[1] + "=" + PlaceholderSSID
	})
	s = wifiPskField.ReplaceAllStringFunc(s, func(m string) string {
		sm := wifiPskField.FindStringSubmatch(m)
		if len(sm) < 3 {
			return ""
		}
		return sm[1] + "=" + PlaceholderWiFiPassword
	})

	// 6. Client hostnames (heuristic, only when not in preserve mode).
	if !opts.PreserveFingerprint {
		s = clientHostnameField.ReplaceAllStringFunc(s, func(m string) string {
			sm := clientHostnameField.FindStringSubmatch(m)
			if len(sm) < 2 {
				return ""
			}
			return "hostName=" + PlaceholderClientHostname
		})
	}

	return s
}

// redactMACs replaces MAC addresses with the appropriate placeholder.
// When routerOnly is true, every match is treated as the router's MAC
// (e.g. WAN MAC pages). Otherwise every match is treated as a client
// MAC.
func redactMACs(s string, routerOnly bool) string {
	placeholder := PlaceholderClientMAC
	if routerOnly {
		placeholder = PlaceholderRouterMAC
	}
	return macAny.ReplaceAllString(s, placeholder)
}

// IsSessionTokenMatch reports whether the input contains the
// structural redirect shape emitted by legacy WR841N login responses.
// Used by the probe to validate that a candidate recipe actually
// produced the expected response shape (independent of regex greedy
// matches).
func IsSessionTokenMatch(s string) bool {
	if s == "" {
		return false
	}
	return sessionTokenRedirectScript.MatchString(s) ||
		sessionTokenPathAnywhere.MatchString(s)
}

// ExtractSessionToken returns the first structural session token found
// in s. It is used only in-memory by the probe to build the Status URL
// and is never persisted or logged. Returns "" if none found.
func ExtractSessionToken(s string) string {
	if m := sessionTokenRedirectScript.FindStringSubmatch(s); len(m) >= 2 {
		return m[1]
	}
	if loc := sessionTokenPathAnywhere.FindStringIndex(s); loc != nil {
		// Extract the token between leading / and trailing /userRpm/.
		start := loc[0] + 1
		end := start
		for end < loc[1] {
			if end+len("/userRpm/") <= loc[1] && s[end:end+len("/userRpm/")] == "/userRpm/" {
				return s[start:end]
			}
			end++
		}
	}
	return ""
}

// MaskSessionToken replaces real session-token occurrences in any
// captured response body with the canonical placeholder, so the
// persisted file is safe to read.
func MaskSessionToken(s string) string {
	return Apply(s, Default())
}
