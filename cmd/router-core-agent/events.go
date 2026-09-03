// events.go: structured investigation events for the live
// reasoning path.
//
// Per commit 8 of the pre-submission rules:
//   - expose observable events only;
//   - do NOT expose hidden chain-of-thought;
//   - do NOT fabricate internal reasoning states;
//   - in particular, do not emit "evidence_sufficient" unless
//     the model explicitly produces that as a structured signal.
//
// Event concepts allowed here:
//
//   tool_call     - the model requested a tool.
//   observation   - the runtime returned a typed observation.
//   completed     - the model produced the final answer.
//
// An observation carries:
//   tool            - canonical tool name
//   path            - local router-core path
//   http_status     - HTTP status the runtime returned
//   state           - one of verified, absent, unsupported_or_unverified, unavailable
//   note            - concise factual note (no interpretation, no advice)
//
// The secret boundary from commit 4 still applies: no event
// may contain admin password, Wi-Fi PSK, WPS PIN, session
// token, or Authorization material.

package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// EventKind is the discriminator for the additive Events slice.
type EventKind string

const (
	EventToolCall   EventKind = "tool_call"
	EventObservation EventKind = "observation"
	EventCompleted  EventKind = "completed"
)

// Event is a single observable moment in the investigation.
// Fields are pointers (with omitempty) so the JSON shape is
// additive: a tool_call does not have a state, an observation
// always has one. The frontend can render a vertical trace
// from this slice.
type Event struct {
	Kind       EventKind      `json:"kind"`
	Tool       string         `json:"tool,omitempty"`
	Path       string         `json:"path,omitempty"`
	HTTPStatus int            `json:"http_status,omitempty"`
	State      string         `json:"state,omitempty"`
	Note       string         `json:"note,omitempty"`
	At         string         `json:"at,omitempty"` // RFC3339 timestamp
}

// NewToolCallEvent records the moment the model requests a tool.
// The runtime has not yet fetched the observation; that comes
// in a follow-up observation event.
func NewToolCallEvent(tool string) Event {
	return Event{Kind: EventToolCall, Tool: tool, At: nowRFC3339()}
}

// NewObservationEvent records what the runtime returned for a
// tool call. State is one of the four documented knowledge
// states. Note is a short factual sentence describing what was
// or was not learned; it MUST NOT contain credentials, advice,
// or interpretation.
func NewObservationEvent(tool, path string, httpStatus int, state, note string) Event {
	return Event{
		Kind:       EventObservation,
		Tool:       tool,
		Path:       path,
		HTTPStatus: httpStatus,
		State:      state,
		Note:       note,
		At:         nowRFC3339(),
	}
}

// NewCompletedEvent records the moment the model produced a
// final answer. The note is the model-provided answer, or
// an empty string if the answer is not a note.
func NewCompletedEvent() Event {
	return Event{Kind: EventCompleted, At: nowRFC3339()}
}

// forbiddenEventSubstrings mirrors the agent-faces secret
// boundary from commit 4. Events must not contain any of these.
// The check is case-insensitive to catch "PSK=", "AdminPassword=",
// etc. The check applies to all free-text fields of an event:
// Tool, Path, State, Note. It does not apply to the At field
// (a timestamp).
var forbiddenEventSubstrings = []string{
	"psk=",
	"psk:",
	"wpaPassphrase=",
	"wpaPassphrase:",
	"wpaPsk=",
	"wpaPsk:",
	"wirelessPassword=",
	"adminPassword=",
	"wpsPin=",
	"sessionToken=",
	"Cookie:",
	"Authorization: Basic",
	"Authorization: Bearer",
}

// forbiddenEventWords is a closed set of credential-shaped
// words that must never appear in a tool name, path, state,
// or note, in any case. A tool named "get_psk" or a note
// that says "wifiPassword" is rejected just as if it said
// "psk=" or "wifiPassword=".
var forbiddenEventWords = []string{
	"psk",
	"password",
	"passphrase",
	"wpspin",
	"sessiontoken",
	"wirelesspassword",
	"adminpassword",
	"authorization",
}

func init() {
	// Normalize to lowercase for case-insensitive comparison.
	for i, s := range forbiddenEventSubstrings {
		forbiddenEventSubstrings[i] = strings.ToLower(s)
	}
	for i, w := range forbiddenEventWords {
		forbiddenEventWords[i] = strings.ToLower(w)
	}
}

// Validate returns an error if e contains forbidden credential
// substrings in any of its free-text fields. Two checks:
//   - exact substring match (psk=, adminPassword=, etc.)
//   - standalone credential word (psk, password, etc.)
// Both are case-insensitive.
func (e Event) Validate() error {
	combined := strings.ToLower(e.Tool + " " + e.Path + " " + e.State + " " + e.Note)
	for _, sub := range forbiddenEventSubstrings {
		if strings.Contains(combined, sub) {
			return fmt.Errorf("event contains forbidden substring %q", sub)
		}
	}
	for _, w := range forbiddenEventWords {
		// Match the word with word boundaries. The simplest
		// word-boundary check on free text: the word must
		// appear surrounded by non-letter/digit characters
		// OR be the entire string. We approximate by
		// checking for occurrences followed or preceded by
		// a non-letter character.
		for i := 0; i+len(w) <= len(combined); i++ {
			if combined[i:i+len(w)] != w {
				continue
			}
			pre := byte(0)
			if i > 0 {
				pre = combined[i-1]
			}
			post := byte(0)
			if i+len(w) < len(combined) {
				post = combined[i+len(w)]
			}
			leftBoundary := i == 0 || !isWordChar(pre)
			rightBoundary := i+len(w) == len(combined) || !isWordChar(post)
			if leftBoundary && rightBoundary {
				return fmt.Errorf("event contains forbidden credential word %q", w)
			}
		}
	}
	return nil
}

// isWordChar reports whether b is a word character: letter, digit.
// Underscore is intentionally NOT a word character here: in
// "get_psk", the underscore separates two words, and we want
// the word boundary at the underscore so that "psk" is detected
// as a standalone credential word.
func isWordChar(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

// MustEncodeEvent returns the JSON the agent emits for e.
// Used by tests.
func MustEncodeEvent(e Event) json.RawMessage {
	b, err := json.Marshal(e)
	if err != nil {
		panic(err)
	}
	return b
}
