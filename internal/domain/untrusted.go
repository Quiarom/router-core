package domain

import (
	"encoding/json"
	"strings"
	"unicode"
)

// MaxUntrustedLen caps the length of any single value that originated from the
// router or from other devices on the network.
const MaxUntrustedLen = 256

// Untrusted wraps text that came from the network: hostnames, SSIDs, DHCP
// client names, UPnP descriptions, log lines.
//
// Such text is DATA. It is never interpreted as an instruction by any layer of
// router-core, and the trust marker travels with the value so that a later
// reasoning layer can keep the same distinction.
type Untrusted struct {
	value    string
	Source   string // e.g. "router:dhcp-client-list"
	Modified bool   // true when sanitisation changed the raw bytes
}

// NewUntrusted sanitises raw network text for safe transport and display.
//
// Sanitisation only removes the ability of the value to forge structure in a
// log line, a terminal or a prompt (control characters, newlines, excessive
// length). It never rewrites the semantic content: an adversarial hostname
// stays readable so a human or a model can notice it.
func NewUntrusted(raw, source string) Untrusted {
	var b strings.Builder
	b.Grow(len(raw))
	modified := false
	for _, r := range raw {
		switch {
		case r == '\uFFFD':
			modified = true
		case r == '\t' || r == '\n' || r == '\r':
			b.WriteByte(' ')
			modified = true
		case unicode.IsControl(r), !unicode.IsPrint(r):
			modified = true
		default:
			b.WriteRune(r)
		}
	}
	out := strings.TrimSpace(b.String())
	if out != strings.TrimSpace(raw) {
		modified = true
	}
	if len(out) > MaxUntrustedLen {
		out = out[:MaxUntrustedLen]
		modified = true
	}
	return Untrusted{value: out, Source: source, Modified: modified}
}

// Value returns the sanitised text. Callers must treat it as data.
func (u Untrusted) Value() string { return u.value }

func (u Untrusted) Empty() bool { return u.value == "" }

func (u Untrusted) String() string { return u.value }

func (u Untrusted) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Value    string `json:"value"`
		Trust    string `json:"trust"`
		Source   string `json:"source,omitempty"`
		Modified bool   `json:"sanitized,omitempty"`
	}{u.value, "untrusted", u.Source, u.Modified})
}
