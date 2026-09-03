package main

import (
	"encoding/json"
	"errors"
	"testing"
)

// TestModelToolCount is the canonical assertion of §6: the
// model-facing tool list must contain exactly 10 tools.
func TestModelToolCount(t *testing.T) {
	list := modelToolList()
	if len(list) != modelToolCount {
		t.Fatalf("model tool list must contain exactly %d tools; got %d", modelToolCount, len(list))
	}
	if len(canonicalTools) != modelToolCount {
		t.Fatalf("canonicalTools must contain exactly %d entries; got %d", modelToolCount, len(canonicalTools))
	}
}

// TestCanonicalToolNames pins the exact set of advertised names.
// If a future commit adds or removes a tool, this test forces
// the author to update the canonical list consciously.
func TestCanonicalToolNames(t *testing.T) {
	want := []string{
		"get_device_info",
		"get_router_status",
		"get_capabilities",
		"get_clients",
		"get_wireless_security",
		"get_wps_state",
		"get_dmz_state",
		"get_upnp_state",
		"get_remote_management_state",
		"get_forwarding_rules",
	}
	if len(canonicalTools) != len(want) {
		t.Fatalf("canonical count changed: got %d, want %d", len(canonicalTools), len(want))
	}
	for i, name := range want {
		if canonicalTools[i].Name != name {
			t.Errorf("canonicalTools[%d].Name = %q, want %q", i, canonicalTools[i].Name, name)
		}
	}
}

// TestEveryToolHasExactlyOnePath pins §6 "closed mapping":
// every advertised tool has exactly one local router-core path.
func TestEveryToolHasExactlyOnePath(t *testing.T) {
	seen := map[string]string{}
	for _, t1 := range canonicalTools {
		if t1.Path == "" {
			t.Errorf("%s: empty path", t1.Name)
		}
		if !startsWithSlash(t1.Path) {
			t.Errorf("%s: path %q does not start with /", t1.Name, t1.Path)
		}
		if other, ok := seen[t1.Path]; ok {
			t.Errorf("path %q is shared by %q and %q (must be unique)", t1.Path, other, t1.Name)
		}
		seen[t1.Path] = t1.Name
	}
}

// TestResolveToolPathKnownNames returns the local path for every
// canonical name. The model uses this to dispatch.
func TestResolveToolPathKnownNames(t *testing.T) {
	for _, t1 := range canonicalTools {
		path, err := resolveToolPath(t1.Name)
		if err != nil {
			t.Errorf("%s: resolve failed: %v", t1.Name, err)
			continue
		}
		if path != t1.Path {
			t.Errorf("%s: resolve returned %q, want %q", t1.Name, path, t1.Path)
		}
	}
}

// TestResolveToolPathRejectsUnknownNames is the negative side of
// §6: an unknown tool name is rejected with ErrUnknownTool.
func TestResolveToolPathRejectsUnknownNames(t *testing.T) {
	bad := []string{
		"",
		"get_admin_password",
		"get_psk",
		"get_security",     // legacy generic, NOT advertised
		"get_wireless_key", // looks like wireless but is not in the canonical set
		"GET /v0/device",   // path injection attempt
		"../v0/device",
	}
	for _, name := range bad {
		_, err := resolveToolPath(name)
		if err == nil {
			t.Errorf("resolveToolPath(%q) must fail", name)
			continue
		}
		if !errors.Is(err, ErrUnknownTool) {
			t.Errorf("resolveToolPath(%q) returned %v, want ErrUnknownTool", name, err)
		}
	}
}

// TestNoToolExposesSecretNames is a defense-in-depth check that
// the canonical tool descriptions do not contain secret-shaped
// identifiers like "psk=", "admin password", or HTTP credentials.
// The runtime secret boundary is enforced in commit 4; this
// test catches accidental leakage in tool descriptions.
func TestNoToolExposesSecretNames(t *testing.T) {
	forbidden := []string{
		"psk=",
		"wpaPassphrase",
		"adminPassword",
		"wpsPin",
		"sessionToken",
		"Authorization: Basic",
		"Authorization: Bearer",
	}
	for _, t1 := range canonicalTools {
		for _, n := range forbidden {
			if contains(t1.Description, n) {
				t.Errorf("%s: description contains forbidden substring %q", t1.Name, n)
			}
		}
	}
}

// TestModelToolListJSONShape pins the JSON shape of the tools
// array sent to the model. It must be a JSON array of objects
// each with type=function and a function.name.
func TestModelToolListJSONShape(t *testing.T) {
	b := MustEncodeModelToolList()
	var arr []map[string]any
	if err := json.Unmarshal(b, &arr); err != nil {
		t.Fatalf("tool list is not a JSON array of objects: %v", err)
	}
	if len(arr) != modelToolCount {
		t.Fatalf("JSON tool list has %d entries, want %d", len(arr), modelToolCount)
	}
	for i, item := range arr {
		if item["type"] != "function" {
			t.Errorf("tools[%d].type = %v, want \"function\"", i, item["type"])
		}
		fn, ok := item["function"].(map[string]any)
		if !ok {
			t.Errorf("tools[%d].function is not an object", i)
			continue
		}
		if _, ok := fn["name"]; !ok {
			t.Errorf("tools[%d].function.name missing", i)
		}
		if _, ok := fn["description"]; !ok {
			t.Errorf("tools[%d].function.description missing", i)
		}
	}
}

func startsWithSlash(s string) bool { return len(s) > 0 && s[0] == '/' }

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
