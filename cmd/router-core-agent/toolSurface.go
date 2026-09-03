package main

// toolSurface.go: the closed mapping between model-facing tool names
// and router-core local paths.
//
// The MiniMax agent sees exactly 10 tools (the canonical list
// below). The model cannot ask for any other path, parameter
// shape, or HTTP method. Every tool returns a JSON observation
// from one of the existing router-core endpoints, scoped to a
// single capability.
//
// Legacy get_security(name) is supported INTERNALLY for backward
// compatibility with existing fixtures and tests, but it is NOT
// advertised to the model. The 10 canonical tools are the only
// names that may appear in the chat-completions request.

import (
	"encoding/json"
	"fmt"
)

// canonicalTool is the public name of one model-facing tool.
// Its description tells the model:
//   - what the tool returns;
//   - what it cannot prove;
//   - which knowledge states are possible.
type canonicalTool struct {
	Name        string
	Path        string // router-core local path; no slashes injected by the model
	Description string
	// Args is a closed JSON Schema object. The model is constrained
	// to these properties; additionalProperties is always false.
	Args map[string]any
}

// canonicalTools is the closed set. Order matters: this is the
// order they appear in the chat-completions request. Keep it stable
// so fixtures and tests are reproducible.
var canonicalTools = []canonicalTool{
	{
		Name: "get_device_info",
		Path: "/v0/device",
		Description: "Read the router's device identity (vendor, model, hardware version, " +
			"firmware version, management address, authenticated flag, provenance). " +
			"Cannot prove: that the network is safe, that the device is " +
			"trustworthy, that the firmware is up to date. Possible states: " +
			"verified (got the JSON), unavailable (transport failure). " +
			"All values are router-provided and untrusted.",
		Args: emptyArgs(),
	},
	{
		Name: "get_router_status",
		Path: "/v0/status",
		Description: "Read the router's operational status (reachability, WAN state, " +
			"uptime in seconds, provenance). Cannot prove: that traffic is " +
			"flowing, that the WAN is actually connected to the public " +
			"internet, that uptime is continuous. Possible states: verified, " +
			"unavailable. WAN values are router-provided and untrusted.",
		Args: emptyArgs(),
	},
	{
		Name: "get_capabilities",
		Path: "/v0/capabilities",
		Description: "Read the per-capability epistemic matrix for this adapter. " +
			"Each entry is one of: verified, absent, unsupported_or_unverified, " +
			"unavailable. This is the only tool that reports the adapter's " +
			"own knowledge of what it can and cannot observe. Use it to know " +
			"which other tools will return real data on this firmware build.",
		Args: emptyArgs(),
	},
	{
		Name: "get_clients",
		Path: "/v0/clients",
		Description: "Read the DHCP lease list observed on the LAN. Returns one " +
			"entry per observed client with name, IP, MAC, lease time, and " +
			"provenance. Cannot prove: that a client is trusted, that a " +
			"client is benign, that the list is complete. Possible states: " +
			"verified, unavailable. All client-provided attributes (name, " +
			"lease time) are untrusted data.",
		Args: emptyArgs(),
	},
	{
		Name: "get_wireless_security",
		Path: "/v0/security/wireless",
		Description: "Read the wireless security observation. Returns security mode " +
			"(WPA2-PSK, etc.), cipher (AES/CCMP, etc.), and a boolean " +
			"credential_configured flag. NEVER returns the actual pre-shared " +
			"key, the WPS PIN, or any other credential. Possible states: " +
			"verified, absent, unsupported_or_unverified, unavailable.",
		Args: emptyArgs(),
	},
	{
		Name: "get_wps_state",
		Path: "/v0/security/wps",
		Description: "Read the WPS state observation. Returns enabled (True/False/" +
			"Unknown) and provenance. Cannot prove: that WPS is actually " +
			"unusable, that an external device cannot pair. Possible states: " +
			"verified, absent (firmware has no WPS section), " +
			"unsupported_or_unverified (runtime has no parser), unavailable.",
		Args: emptyArgs(),
	},
	{
		Name: "get_dmz_state",
		Path: "/v0/security/dmz",
		Description: "Read the DMZ state observation. Returns enabled (True/False/" +
			"Unknown), the host it forwards to (if any), and provenance. " +
			"Cannot prove: that the DMZ host is hardened, that no other " +
			"ports are exposed. Possible states: verified, absent, " +
			"unsupported_or_unverified, unavailable.",
		Args: emptyArgs(),
	},
	{
		Name: "get_upnp_state",
		Path: "/v0/security/upnp",
		Description: "Read the UPnP state observation. Returns enabled, the count " +
			"of active mappings, and provenance. Cannot prove: that UPnP " +
			"is fully off in practice, that an IGD device is not exposing " +
			"ports. Possible states: verified, absent, " +
			"unsupported_or_unverified, unavailable.",
		Args: emptyArgs(),
	},
	{
		Name: "get_remote_management_state",
		Path: "/v0/security/remote-management",
		Description: "Read the remote-management state observation. Returns enabled, " +
			"the listen port (if any), and provenance. Cannot prove: that no " +
			"alternate path exists, that the management UI is not exposed " +
			"via a hidden rule. Possible states: verified, absent, " +
			"unsupported_or_unverified, unavailable.",
		Args: emptyArgs(),
	},
	{
		Name: "get_forwarding_rules",
		Path: "/v0/security/forwarding",
		Description: "Read the count of port-forwarding rules currently configured. " +
			"Returns the rule count and provenance. Cannot prove: what each " +
			"rule does, which services are reachable, that no rules were " +
			"added via an alternate UI. Possible states: verified, absent, " +
			"unsupported_or_unverified, unavailable.",
		Args: emptyArgs(),
	},
}

// emptyArgs is the closed JSON Schema for tools that take no
// parameters. additionalProperties:false forces the model to
// pass no arguments.
func emptyArgs() map[string]any {
	return map[string]any{
		"type":                 "object",
		"properties":           map[string]any{},
		"additionalProperties": false,
	}
}

// canonicalByName returns the tool definition for the given name.
// Returns (zero, false) if the name is not in the canonical set.
// This is the single source of truth for "is this a known tool?".
func canonicalByName(name string) (canonicalTool, bool) {
	for _, t := range canonicalTools {
		if t.Name == name {
			return t, true
		}
	}
	return canonicalTool{}, false
}

// toOpenRouterTool converts one canonicalTool into the JSON shape
// the chat-completions request expects. The same struct is used
// for the model-facing tool list.
func (t canonicalTool) toOpenRouterTool() openRouterTool {
	return openRouterTool{
		Type: "function",
		Function: functionDefinition{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.Args,
		},
	}
}

// modelToolList returns the closed list of tools the agent sends
// to the model. Order is stable. Length is exactly 10.
func modelToolList() []openRouterTool {
	out := make([]openRouterTool, 0, len(canonicalTools))
	for _, t := range canonicalTools {
		out = append(out, t.toOpenRouterTool())
	}
	return out
}

// modelToolCount is exposed as a constant for tests.
const modelToolCount = 10

// resolveToolPath is the closed mapping from a model-supplied tool
// name to the local router-core path. The model cannot supply the
// path directly; only the canonical name. Unknown names are
// rejected with ErrUnknownTool.
func resolveToolPath(name string) (string, error) {
	t, ok := canonicalByName(name)
	if !ok {
		return "", fmt.Errorf("%w: %q", ErrUnknownTool, name)
	}
	return t.Path, nil
}

// ErrUnknownTool is returned when the model asks for a tool that
// is not in the canonical set.
var ErrUnknownTool = fmt.Errorf("router-core-agent: unknown tool")

// MustEncodeModelToolList returns the JSON the agent sends to the
// model for the "tools" field. Used by tests to assert the
// exact payload.
func MustEncodeModelToolList() json.RawMessage {
	list := modelToolList()
	b, err := json.Marshal(list)
	if err != nil {
		panic(err)
	}
	return b
}
