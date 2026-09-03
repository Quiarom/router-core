// Package cmd: the inspect subcommand. It runs router-core
// in mock mode, queries its HTTP API, and formats the result
// for human or machine consumption.
//
// The inspect path is intentionally the simplest possible
// user-facing workflow: it does not require an AI credential,
// does not require a physical router, and does not spawn
// router-core-agent. The user gets a deterministic view of
// the current router observations.
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newInspectCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Show the current router observations",
		Long: `Inspect the local router-core runtime and print the current
observations. The runtime is started in mock mode (no network, no
AI credential needed) for the duration of this command.

Output modes:
  human   pretty table (default)
  json    full JSON object to stdout
  jsonl   one JSON object per line (future: events)`,
		Example: `  gvt inspect
  gvt inspect --output json
  gvt inspect --output jsonl`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInspect(cmd.OutOrStdout(), cmd.ErrOrStderr(), output)
		},
	}
	cmd.Flags().StringVar(&output, "output", "human",
		"output format: human, json, jsonl")
	return cmd
}

func runInspect(stdout, stderr io.Writer, output string) error {
	bin, err := findRouterCoreBin()
	if err != nil {
		return err
	}

	// Reserve a loopback IPv4 port explicitly. Using ":0" would
	// bind on IPv6 dual-stack and the sidecar child may fail to
	// reach it on systems where router-core only binds to IPv4.
	addr, err := reserveLoopbackAddr()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sidecar := exec.CommandContext(ctx, bin, "serve", "--mock", "--addr", addr)
	sidecar.Stdout = io.Discard
	sidecar.Stderr = stderr
	if err := sidecar.Start(); err != nil {
		return fmt.Errorf("start router-core: %w", err)
	}
	defer func() {
		if sidecar.Process != nil {
			_ = sidecar.Process.Kill()
		}
		_ = sidecar.Wait()
	}()

	client := &http.Client{Timeout: 2 * time.Second}
	base := "http://" + addr
	if err := waitForReady(ctx, client, base+"/v0/capabilities"); err != nil {
		return fmt.Errorf("router-core never became ready at %s: %w", addr, err)
	}

	caps := map[string]string{}
	if body, err := getJSON(ctx, client, base+"/v0/capabilities"); err == nil {
		// The runtime wraps the map in {"capabilities": {...}}.
		var wrapper struct {
			Capabilities map[string]string `json:"capabilities"`
		}
		if err := json.Unmarshal(body, &wrapper); err == nil {
			caps = wrapper.Capabilities
		}
	}
	device := map[string]interface{}{}
	if body, err := getJSON(ctx, client, base+"/v0/device"); err == nil {
		_ = json.Unmarshal(body, &device)
	}
	status := map[string]interface{}{}
	if body, err := getJSON(ctx, client, base+"/v0/status"); err == nil {
		_ = json.Unmarshal(body, &status)
	}
	clients := []map[string]interface{}{}
	if body, err := getJSON(ctx, client, base+"/v0/clients"); err == nil {
		var c struct {
			Clients []map[string]interface{} `json:"clients"`
		}
		_ = json.Unmarshal(body, &c)
		clients = c.Clients
	}

	switch output {
	case "json":
		return writeJSON(stdout, map[string]any{
			"router":       "fixture (mock mode)",
			"mode":         "mock",
			"capabilities": caps,
			"device":       device,
			"status":       status,
			"clients":      clients,
		})
	case "jsonl":
		if err := writeJSONL(stdout, "device", device); err != nil {
			return err
		}
		if err := writeJSONL(stdout, "status", status); err != nil {
			return err
		}
		if err := writeJSONL(stdout, "clients", clients); err != nil {
			return err
		}
		return writeJSONL(stdout, "capabilities", caps)
	default:
		return renderHuman(stdout, caps, device, status, clients)
	}
}

func findRouterCoreBin() (string, error) {
	exe, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "router-core")
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate, nil
		}
	}
	path, err := exec.LookPath("router-core")
	if err == nil {
		return path, nil
	}
	return "", fmt.Errorf("router-core binary not found next to gavetero or in $PATH; build it with: go build -o bin/router-core ./cmd/router-core")
}

func reserveLoopbackAddr() (string, error) {
	// Bind explicitly to 127.0.0.1:0 so the address is IPv4.
	// The sidecar is launched with the resulting "127.0.0.1:<port>"
	// string, and we want both the parent and the child to agree
	// on the family. Without this, on dual-stack systems, the
	// parent listener might be on [::]:port while the child
	// connects to 127.0.0.1:port (a different socket).
	ln, err := netListenLoopback("127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer ln.Close()
	return ln.Addr().String(), nil
}

func waitForReady(ctx context.Context, client *http.Client, url string) error {
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode/100 == 2 {
				return nil
			}
		}
		time.Sleep(150 * time.Millisecond)
	}
	return fmt.Errorf("timeout")
}

func getJSON(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 2<<20))
}

func writeJSON(w io.Writer, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(w, string(b))
	return err
}

func writeJSONL(w io.Writer, kind string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "{\"kind\":%q,\"data\":%s}\n", kind, string(b))
	return err
}

func renderHuman(w io.Writer, caps map[string]string, device, status map[string]interface{}, clients []map[string]interface{}) error {
	// Flatten untrusted values: an Untrusted JSON object looks
	// like {"source": "...", "trust": "untrusted", "value": "..."}.
	// The user-facing rendering shows just the value, with a
	// small marker when the value was untrusted.
	device = flattenMap(device)
	status = flattenMap(status)

	fmt.Fprintln(w, "Gavetero Inspect")
	fmt.Fprintln(w, "================")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Mode:  mock (fixture-backed, no network)")
	fmt.Fprintln(w)
	if len(device) > 0 {
		fmt.Fprintln(w, "Device")
		fmt.Fprintln(w, "------")
		for _, k := range []string{"vendor", "model", "hardwareVersion", "firmwareVersion", "managementAddress", "authenticated", "provenance"} {
			if v, ok := device[k]; ok {
				fmt.Fprintf(w, "  %-18s %v\n", k, v)
			}
		}
		fmt.Fprintln(w)
	}
	if len(status) > 0 {
		fmt.Fprintln(w, "Status")
		fmt.Fprintln(w, "------")
		for _, k := range []string{"reachable", "wanStatus", "uptimeSeconds", "provenance"} {
			if v, ok := status[k]; ok {
				fmt.Fprintf(w, "  %-18s %v\n", k, v)
			}
		}
		fmt.Fprintln(w)
	}
	if len(caps) > 0 {
		fmt.Fprintln(w, "Capabilities")
		fmt.Fprintln(w, "------------")
		order := []string{
			"device", "status", "clients",
			"wireless", "wireless_security",
			"wps", "wps_state",
			"dmz", "dmz_state",
			"upnp", "upnp_state",
			"remote_management",
			"forwarding", "forwarding_rules",
		}
		seen := map[string]bool{}
		for _, k := range order {
			if v, ok := caps[k]; ok {
				fmt.Fprintf(w, "  %-22s %s\n", k, v)
				seen[k] = true
			}
		}
		// Any extras that the runtime returned.
		extras := []string{}
		for k := range caps {
			if !seen[k] {
				extras = append(extras, k)
			}
		}
		if len(extras) > 0 {
			for _, k := range extras {
				fmt.Fprintf(w, "  %-22s %s\n", k, caps[k])
			}
		}
		fmt.Fprintln(w)
	}
	if clients != nil {
		fmt.Fprintf(w, "Clients (%d observed)\n", len(clients))
		fmt.Fprintln(w, "---------------------")
		for _, c := range clients {
			flat := flattenMap(c)
			ip := toString(flat["ip"])
			mac := toString(flat["mac"])
			name := toString(flat["name"])
			fmt.Fprintf(w, "  %-18s %-20s %s\n", ip, mac, name)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w, "Legend:")
	fmt.Fprintln(w, "  verified                     adapter read the value")
	fmt.Fprintln(w, "  absent                       firmware does not implement")
	fmt.Fprintln(w, "  unsupported_or_unverified    runtime has no parser")
	fmt.Fprintln(w, "  unavailable                  transport failure")
	return nil
}

// flattenMap returns m with every Untrusted-shaped value
// replaced by its inner string. The shape of an Untrusted JSON
// object is {"source": "...", "trust": "untrusted", "value": "..."}.
// When the value field is present we keep just the value; when
// the trust field is "untrusted" we prefix the rendered line
// with "~ " to mark it as router-supplied data.
func flattenMap(m map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = flattenValue(v)
	}
	return out
}

func flattenValue(v interface{}) interface{} {
	if v == nil {
		return nil
	}
	switch x := v.(type) {
	case map[string]interface{}:
		// Untrusted object?
		if val, ok := x["value"]; ok {
			if trust, ok2 := x["trust"].(string); ok2 && trust == "untrusted" {
				return "~ " + toString(val)
			}
			return val
		}
		return x
	case []interface{}:
		out := make([]interface{}, len(x))
		for i, item := range x {
			out[i] = flattenValue(item)
		}
		return out
	default:
		return v
	}
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// avoid unused import
var _ = strings.TrimSpace

// netListenLoopback is a package-level seam for tests. The
// net.go / net_windows.go files rebind it via init().
var netListenLoopback = func(addr string) (net.Listener, error) {
	return nil, fmt.Errorf("netListenLoopback not bound (build tag issue?)")
}
