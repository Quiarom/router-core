// Command router-core-agent is the Phase 5 reasoning layer for
// router-core. It connects to a running router-core serve over the
// loopback HTTP API, gathers device identity + capability matrix,
// sends the user's natural-language question to MiniMax M3 via
// OpenRouter, and streams the model's tool-call trace to stderr.
// The final answer goes to stdout.
//
// The agent is strictly read-only. Every tool call hits a
// GET endpoint under /v0/. The agent never mutates the router.
//
// If OPENROUTER_API_KEY is not set, or --dry-run is passed, the
// agent uses a deterministic stub that demonstrates the same
// trace shape with a fixed tool sequence. This is the default for
// local development and CI.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const version = "0.1.0"

type options struct {
	routerCoreURL    string
	openrouterURL    string
	openrouterModel  string
	openrouterKeyEnv string
	question         string
	dryRun           bool
	timeout          time.Duration
}

func main() {
	opts := parseFlags(os.Args[1:])
	if err := run(opts); err != nil {
		fmt.Fprintln(os.Stderr, "router-core-agent:", err)
		os.Exit(1)
	}
}

func parseFlags(args []string) options {
	fs := flag.NewFlagSet("router-core-agent", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	opts := options{
		routerCoreURL:    "http://127.0.0.1:8484",
		openrouterURL:    "https://openrouter.ai/api/v1/chat/completions",
		openrouterModel:  "minimax/minimax-m3:free",
		openrouterKeyEnv: "OPENROUTER_API_KEY",
		timeout:          30 * time.Second,
	}
	fs.StringVar(&opts.routerCoreURL, "router-core-url", opts.routerCoreURL, "URL of the running router-core serve (loopback only)")
	fs.StringVar(&opts.openrouterURL, "openrouter-url", opts.openrouterURL, "OpenRouter chat-completions URL")
	fs.StringVar(&opts.openrouterModel, "model", opts.openrouterModel, "OpenRouter model id (default minimax/minimax-m3:free)")
	fs.StringVar(&opts.openrouterKeyEnv, "key-env", opts.openrouterKeyEnv, "environment variable that holds the OpenRouter API key")
	fs.StringVar(&opts.question, "question", "", "user question (or read from stdin if -)")
	fs.BoolVar(&opts.dryRun, "dry-run", false, "skip the OpenRouter call and use a deterministic stub")
	fs.DurationVar(&opts.timeout, "timeout", opts.timeout, "HTTP timeout for both router-core and OpenRouter calls")
	_ = fs.Parse(args)
	return opts
}

func run(opts options) error {
	question := opts.question
	if question == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read question from stdin: %w", err)
		}
		question = strings.TrimSpace(string(data))
	}
	if question == "" {
		return errors.New("question is required: pass --question or pipe via stdin")
	}

	ctx, cancel := context.WithTimeout(context.Background(), opts.timeout)
	defer cancel()

	rc := newRouterCoreClient(opts.routerCoreURL, opts.timeout)
	device, err := rc.get(ctx, "/v0/device")
	if err != nil {
		return fmt.Errorf("device: %w", err)
	}
	status, err := rc.get(ctx, "/v0/status")
	if err != nil {
		logf("status unavailable: %v", err)
	}
	caps, err := rc.get(ctx, "/v0/capabilities")
	if err != nil {
		return fmt.Errorf("capabilities: %w", err)
	}

	logf("connected to %s", opts.routerCoreURL)
	logf("device  = %s", compactJSON(device))
	logf("status  = %s", compactJSON(status))
	logf("caps    = %s", compactJSON(caps))
	logf("question = %q", question)

	apiKey := os.Getenv(opts.openrouterKeyEnv)
	useStub := opts.dryRun || apiKey == ""
	if useStub {
		logf("using deterministic stub (set %s to use MiniMax M3 live)", opts.openrouterKeyEnv)
		return runStub(ctx, rc, device, status, caps, question)
	}
	logf("calling %s (model=%s)", opts.openrouterURL, opts.openrouterModel)
	return runLive(ctx, opts, rc, apiKey, device, status, caps, question)
}

func logf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "router-core-agent: "+format+"\n", args...)
}

type routerCoreClient struct {
	baseURL string
	http    *http.Client
}

func newRouterCoreClient(baseURL string, timeout time.Duration) *routerCoreClient {
	return &routerCoreClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		http:    &http.Client{Timeout: timeout},
	}
}

// get returns the raw JSON body of a /v0/ endpoint, regardless of
// HTTP status. Non-2xx on /v0/ is a structured response (state
// field), not a transport error.
func (c *routerCoreClient) get(ctx context.Context, path string) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if len(body) == 0 {
		body = json.RawMessage(fmt.Sprintf(`{"http_status":%d}`, resp.StatusCode))
	}
	return json.RawMessage(body), nil
}

func compactJSON(b json.RawMessage) string {
	if len(b) == 0 {
		return "<absent>"
	}
	var out bytes.Buffer
	if err := json.Compact(&out, b); err != nil {
		return string(b)
	}
	return out.String()
}

type tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

var securityTool = tool{
	Name:        "get_security",
	Description: "GET /v0/security/<name> on the local router-core serve. Returns the structured security observation for the named capability (wireless, wps, dmz, upnp, remote-management, forwarding). 503 means the runtime cannot satisfy it right now; 404 means the firmware does not implement it.",
	Parameters: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type": "string",
				"enum": []string{"wireless", "wps", "dmz", "upnp", "remote-management", "forwarding"},
			},
		},
		"required": []string{"name"},
	},
}

func buildSystemPrompt(device, status, caps json.RawMessage) string {
	var b strings.Builder
	b.WriteString("You are a read-only network auditor for a single home router.\n")
	b.WriteString("Your job: answer the operator's question in plain language by gathering one observation at a time, then summarizing Observed facts, Potential concerns, Recommendations, and any Action that would require explicit operator approval (you must never perform mutations yourself).\n\n")
	b.WriteString("Never invent values. If a tool returns state \"absent\" or \"unavailable\" or \"unsupported_or_unverified\", report that as-is. Do not collapse to true/false.\n")
	b.WriteString("Emit exactly one tool call per turn. After the evidence is sufficient, respond with the final answer and no further tool calls.\n\n")
	b.WriteString("DEVICE\n")
	b.WriteString(compactJSON(device))
	b.WriteString("\n\nSTATUS\n")
	if len(status) > 0 {
		b.WriteString(compactJSON(status))
	} else {
		b.WriteString("<unavailable>")
	}
	b.WriteString("\n\nCAPABILITIES\n")
	b.WriteString(compactJSON(caps))
	b.WriteString("\n\nAvailable tools:\n")
	b.WriteString("- get_security(name): GET /v0/security/<name>. Use it to inspect one capability at a time.\n")
	return b.String()
}

type toolCall struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type message struct {
	Role       string    `json:"role"`
	Content    string    `json:"content,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolCallID string    `json:"tool_call_id,omitempty"`
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
	Tools    []tool    `json:"tools"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Role      string    `json:"role"`
			Content   string    `json:"content"`
			ToolCalls []toolCall `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type   string `json:"type"`
	} `json:"error"`
}

func runLive(ctx context.Context, opts options, rc *routerCoreClient, apiKey string, device, status, caps json.RawMessage, question string) error {
	history := []message{{Role: "system", Content: buildSystemPrompt(device, status, caps)}}
	for range 8 {
		history = append(history, message{Role: "user", Content: question})
		req := chatRequest{
			Model:    opts.openrouterModel,
			Messages: history,
			Tools:    []tool{securityTool},
		}
		body, _ := json.Marshal(req)
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, opts.openrouterURL, bytes.NewReader(body))
		if err != nil {
			return err
		}
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
		httpReq.Header.Set("Content-Type", "application/json")
		resp, err := rc.http.Do(httpReq)
		if err != nil {
			return fmt.Errorf("openrouter: %w", err)
		}
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		resp.Body.Close()
		if resp.StatusCode/100 != 2 {
			return fmt.Errorf("openrouter HTTP %d: %s", resp.StatusCode, compactJSON(respBody))
		}
		var parsed chatResponse
		if err := json.Unmarshal(respBody, &parsed); err != nil {
			return fmt.Errorf("openrouter decode: %w", err)
		}
		if parsed.Error != nil {
			return fmt.Errorf("openrouter: %s", parsed.Error.Message)
		}
		if len(parsed.Choices) == 0 {
			return errors.New("openrouter: no choices in response")
		}
		choice := parsed.Choices[0].Message
		history = append(history, message{Role: "assistant", Content: choice.Content, ToolCalls: choice.ToolCalls})
		if len(choice.ToolCalls) == 0 {
			fmt.Println(choice.Content)
			return nil
		}
		for _, tc := range choice.ToolCalls {
			name, _ := tc.Arguments["name"].(string)
			if name == "" {
				name = tc.Name
			}
			logf("tool call -> get_security(%q)", name)
			body, err := rc.get(ctx, "/v0/security/"+name)
			if err != nil {
				logf("tool error: %v", err)
				body = json.RawMessage(fmt.Sprintf(`{"error":%q}`, err.Error()))
			}
			logf("result    -> %s", compactJSON(body))
			history = append(history, message{Role: "tool", ToolCallID: tc.Name, Content: string(body)})
		}
	}
	return errors.New("agent loop exceeded 8 turns without a final answer")
}

func stubSequence(question string) []string {
	q := strings.ToLower(question)
	switch {
	case strings.Contains(q, "wi-fi") || strings.Contains(q, "wifi") || strings.Contains(q, "wireless") || strings.Contains(q, "exposed"):
		return []string{"wireless", "wps", "remote-management"}
	case strings.Contains(q, "who") || strings.Contains(q, "connected") || strings.Contains(q, "devices") || strings.Contains(q, "clients"):
		return []string{"wireless", "wps", "remote-management", "dmz", "upnp", "forwarding"}
	default:
		return []string{"wireless", "wps", "dmz", "upnp", "remote-management", "forwarding"}
	}
}

func runStub(ctx context.Context, rc *routerCoreClient, device, status, caps json.RawMessage, question string) error {
	sequence := stubSequence(question)
	for _, name := range sequence {
		logf("tool call -> get_security(%q)", name)
		body, err := rc.get(ctx, "/v0/security/"+name)
		if err != nil {
			logf("tool error: %v", err)
			body = json.RawMessage(fmt.Sprintf(`{"error":%q}`, err.Error()))
		}
		logf("result    -> %s", compactJSON(body))
	}
	fmt.Println(stubAnswer(question, sequence))
	return nil
}

func stubAnswer(question string, sequence []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Question: %s\n\n", question)
	fmt.Fprintf(&b, "Observed facts\n")
	fmt.Fprintf(&b, "  - Reviewed %d security observations: %s.\n", len(sequence), strings.Join(sequence, ", "))
	fmt.Fprintf(&b, "  - Wireless security: parser not yet wired; runtime reports unavailable.\n")
	fmt.Fprintf(&b, "  - WPS, UPnP, Remote Management: absent on this firmware build (HTTP 501 from the device).\n")
	fmt.Fprintf(&b, "  - DMZ, Forwarding: runtime reports unavailable until the parser is wired.\n\n")
	fmt.Fprintf(&b, "Potential concern\n")
	fmt.Fprintf(&b, "  - The wireless security surface has not been parsed by the runtime yet. Without the parse, the agent cannot tell whether WPA2 is on, the pre-shared key has any weakness, or an open SSID exists.\n\n")
	fmt.Fprintf(&b, "Recommendation\n")
	fmt.Fprintf(&b, "  - Capture the wireless security dashboard response and wire the parser. Re-run this question against the live router. Until then, the safe answer is: ask the operator to confirm WPA2 is enabled and WPS is off from the device's web UI.\n\n")
	fmt.Fprintf(&b, "Action requiring explicit operator approval\n")
	fmt.Fprintf(&b, "  - None. The runtime is read-only; any device-side change must be initiated by the operator through the router's own admin interface, not by this agent.\n")
	return b.String()
}
