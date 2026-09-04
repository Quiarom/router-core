// Package cmd: the ask subcommand. This is the user-facing
// entry point for MiniMax M3 reasoning over Gavetero's
// router observations.
//
// ask is a thin orchestrator. It does not reimplement the
// agent. It spawns the existing router-core runtime as a
// sidecar, then spawns the existing router-core-agent as a
// second sidecar, captures the answer, and prints it.
//
// The user does not see:
//
//   - which ports the sidecars bind to
//   - the router-core-url the agent connects to
//   - the model identifier (default MiniMaxAI/MiniMax-M3)
//   - the fallback model (default MiniMaxAI/MiniMax-M2.7)
//   - the timeout (default 60s)
//
// The user can override any of those for debugging with flags.
package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	defaultModel         = "MiniMaxAI/MiniMax-M3"
	defaultFallbackModel = "MiniMaxAI/MiniMax-M2.7"
	defaultTimeoutSec    = 60
)

func newAskCmd() *cobra.Command {
	var (
		output        string
		dryRun        bool
		mockMode      bool
		noFallback    bool
		model         string
		fallbackModel string
		timeoutSec    int
		verboseAgent  bool
		noMock        bool
	)

	cmd := &cobra.Command{
		Use:   "ask <question>",
		Short: "Investigate a network question with MiniMax M3",
		Long: `ask sends a question to the local MiniMax M3 agent and prints
the answer. The local runtime (router-core) is started
automatically in mock mode (no network, no physical router
required) for the duration of this command.

Output modes:

  human   the answer and a short summary of the tool calls
  json    the full JSON returned by the agent
  jsonl   one event per line: tool_call, observation, completed

The default model is MiniMaxAI/MiniMax-M3, served by GMI
Cloud. The fallback model is MiniMaxAI/MiniMax-M2.7 when
the primary fails with a transient error.

Resolution order for the API key (GMI):

  1. GMI_SERVING_API_KEY environment variable
  2. OS credential store entry (set by gvt setup)
  3. none: ask returns a clean actionable error`,
		Example: `  gvt ask "Is my network exposed?"
  gvt ask "What is my Wi-Fi state?" --output json
  gvt ask "List connected devices" --output jsonl
  gvt ask "..." --dry-run         # does not contact GMI`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			question := strings.Join(args, " ")
			opts := askOptions{
				Question:      question,
				Output:        output,
				DryRun:        dryRun,
				MockMode:      mockMode && !noMock,
				NoFallback:    noFallback,
				Model:         model,
				FallbackModel: fallbackModel,
				Timeout:       time.Duration(timeoutSec) * time.Second,
				VerboseAgent:  verboseAgent,
			}
			return runAsk(cmd.OutOrStdout(), cmd.ErrOrStderr(), opts)
		},
	}
	cmd.Flags().StringVar(&output, "output", "human",
		"output format: human, json, jsonl")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false,
		"use the deterministic offline agent (does not call GMI)")
	cmd.Flags().BoolVar(&mockMode, "mock", true,
		"start router-core in mock mode (default)")
	cmd.Flags().BoolVar(&noMock, "no-mock", false,
		"talk to a real router instead of the mock fixture")
	cmd.Flags().BoolVar(&noFallback, "no-fallback", false,
		"disable the M2.7 fallback; fail on the primary model only")
	cmd.Flags().StringVar(&model, "model", defaultModel,
		"primary model identifier")
	cmd.Flags().StringVar(&fallbackModel, "fallback-model", defaultFallbackModel,
		"fallback model identifier (used when primary fails with a transient error)")
	cmd.Flags().IntVar(&timeoutSec, "timeout", defaultTimeoutSec,
		"overall timeout in seconds for the ask operation")
	cmd.Flags().BoolVar(&verboseAgent, "verbose-agent", false,
		"print the agent's stderr (tool call trace) to the gavetero stderr")
	return cmd
}

type askOptions struct {
	Question      string
	Output        string
	DryRun        bool
	MockMode      bool
	NoFallback    bool
	Model         string
	FallbackModel string
	Timeout       time.Duration
	VerboseAgent  bool
}

func runAsk(stdout, stderr io.Writer, opts askOptions) error {
	// 1. Resolve the API key. Precedence: env > keyring > error.
	apiKey, err := resolveAPIKey()
	if err != nil && !opts.DryRun {
		return err
	}

	// 2. Find the binaries.
	routerBin, err := findRouterCoreBin()
	if err != nil {
		return err
	}
	agentBin, err := findRouterCoreAgentBin()
	if err != nil {
		return err
	}

	// 3. Reserve loopback ports for both sidecars.
	routerAddr, err := reserveLoopbackAddr()
	if err != nil {
		return fmt.Errorf("reserve router port: %w", err)
	}
	agentAddr, err := reserveLoopbackAddr()
	if err != nil {
		return fmt.Errorf("reserve agent port: %w", err)
	}

	// 4. Start the router-core sidecar.
	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	routerCmd := exec.CommandContext(ctx, routerBin, "serve")
	routerCmd.Args = append(routerCmd.Args, buildRouterServeArgs(opts, routerAddr)...)
	if opts.VerboseAgent {
		routerCmd.Stderr = stderr
	} else {
		routerCmd.Stderr = io.Discard
	}
	routerCmd.Stderr = stderr // surface any startup error
	routerCmd.Stdout = io.Discard
	if err := routerCmd.Start(); err != nil {
		return fmt.Errorf("start router-core: %w", err)
	}
	defer func() {
		if routerCmd.Process != nil {
			_ = routerCmd.Process.Kill()
		}
		_ = routerCmd.Wait()
	}()

	// Wait for /v0/capabilities.
	routerClient := &http.Client{Timeout: 2 * time.Second}
	routerURL := "http://" + routerAddr
	if err := waitForReady(ctx, routerClient, routerURL+"/v0/capabilities"); err != nil {
		return fmt.Errorf("router-core never became ready at %s: %w", routerURL, err)
	}

	// 5. Start the router-core-agent sidecar.
	agentCmd := exec.CommandContext(ctx, agentBin)
	agentCmd.Args = append(agentCmd.Args, buildAgentServeArgs(opts, agentAddr, routerURL)...)
	if opts.VerboseAgent {
		agentCmd.Stderr = stderr
	} else {
		agentCmd.Stderr = io.Discard
	}
	agentCmd.Stdout = io.Discard
	// Inject the API key as an env var, not as a flag. The agent
	// picks it up via GMI_SERVING_API_KEY automatically.
	if apiKey != "" {
		agentCmd.Env = append(os.Environ(), "GMI_SERVING_API_KEY="+apiKey)
	}
	if err := agentCmd.Start(); err != nil {
		return fmt.Errorf("start router-core-agent: %w", err)
	}
	defer func() {
		if agentCmd.Process != nil {
			_ = agentCmd.Process.Kill()
		}
		_ = agentCmd.Wait()
	}()

	// Wait for /healthz.
	agentClient := &http.Client{Timeout: 2 * time.Second}
	agentURL := "http://" + agentAddr
	if err := waitForReady(ctx, agentClient, agentURL+"/healthz"); err != nil {
		return fmt.Errorf("router-core-agent never became ready at %s: %w", agentURL, err)
	}

	// 6. POST the question to /v0/chat.
	payload, _ := json.Marshal(map[string]any{
		"question": opts.Question,
	})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, agentURL+"/v0/chat", strings.NewReader(string(payload)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := agentClient.Do(req)
	if err != nil {
		return fmt.Errorf("ask: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		return fmt.Errorf("agent returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return fmt.Errorf("read agent response: %w", err)
	}

	// 7. Render the answer.
	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parse agent response: %w", err)
	}
	return renderAsk(stdout, opts, result, body)
}

func buildRouterServeArgs(opts askOptions, addr string) []string {
	args := []string{
		"serve",
		"--addr", addr,
	}
	if opts.MockMode {
		args = append(args, "--mock")
	}
	return args
}

func buildAgentServeArgs(opts askOptions, addr, routerURL string) []string {
	args := []string{
		"--serve", addr,
		"--router-core-url", routerURL,
		"--question", opts.Question,
		"--model", opts.Model,
		"--timeout", fmt.Sprintf("%ds", int(opts.Timeout.Seconds())),
	}
	if opts.DryRun {
		args = append(args, "--dry-run")
	}
	if opts.NoFallback {
		args = append(args, "--model-fallback", "")
	} else {
		args = append(args, "--model-fallback", opts.FallbackModel)
	}
	return args
}

func renderAsk(stdout io.Writer, opts askOptions, result map[string]any, raw []byte) error {
	switch opts.Output {
	case "json":
		_, err := fmt.Fprintln(stdout, string(raw))
		return err
	case "jsonl":
		return renderAskJSONL(stdout, result)
	default:
		return renderAskHuman(stdout, result)
	}
}

func renderAskHuman(stdout io.Writer, result map[string]any) error {
	answer, _ := result["answer"].(string)
	model, _ := result["model"].(string)
	mode, _ := result["mode"].(string)
	fmt.Fprintln(stdout, "Gavetero answer")
	fmt.Fprintln(stdout, "==============")
	fmt.Fprintln(stdout, "")
	if answer != "" {
		fmt.Fprintln(stdout, answer)
		fmt.Fprintln(stdout, "")
	}
	if steps, ok := result["steps"].([]any); ok && len(steps) > 0 {
		fmt.Fprintf(stdout, "Tools called: %d\n", len(steps))
		for i, s := range steps {
			step, _ := s.(map[string]any)
			if step == nil {
				continue
			}
			tool, _ := step["tool"].(string)
			path, _ := step["path"].(string)
			status, _ := step["http_status"].(float64)
			fmt.Fprintf(stdout, "  %d. %s %s -> %d\n", i+1, tool, path, int(status))
		}
		fmt.Fprintln(stdout, "")
	}
	if model != "" {
		fmt.Fprintf(stdout, "Model: %s\n", model)
	}
	if mode != "" {
		fmt.Fprintf(stdout, "Mode:  %s\n", mode)
	}
	return nil
}

func renderAskJSONL(stdout io.Writer, result map[string]any) error {
	// One event per line. The agent in the integration branch
	// emits Events alongside Steps. The current response
	// includes steps[] but not events[] (the model-driven path
	// uses the stub agent when --dry-run is set, which has
	// neither). We derive events from steps to keep the jsonl
	// contract consistent.
	enc := json.NewEncoder(stdout)
	enc.SetEscapeHTML(false)

	if steps, ok := result["steps"].([]any); ok {
		for i, s := range steps {
			step, _ := s.(map[string]any)
			if step == nil {
				continue
			}
			tool, _ := step["tool"].(string)
			path, _ := step["path"].(string)
			status, _ := step["http_status"].(float64)
			if tool != "" {
				_ = enc.Encode(map[string]any{
					"kind":  "tool_call",
					"tool":  tool,
					"path":  path,
					"index": i + 1,
				})
			}
			if tool != "" {
				_ = enc.Encode(map[string]any{
					"kind":        "observation",
					"tool":        tool,
					"path":        path,
					"http_status": int(status),
					"index":       i + 1,
				})
			}
		}
	}
	if answer, _ := result["answer"].(string); answer != "" {
		_ = enc.Encode(map[string]any{
			"kind":   "completed",
			"answer": answer,
		})
	}
	return nil
}

func resolveAPIKey() (string, error) {
	if env := os.Getenv("GMI_SERVING_API_KEY"); env != "" {
		return env, nil
	}
	if v, err := keyringGet("gavetero", "gmi-cloud-api-key"); err == nil && v != "" {
		return v, nil
	}
	return "", fmt.Errorf(
		"GMI API key not found.\n" +
			"  - export GMI_SERVING_API_KEY=YOUR_KEY\n" +
			"  - or run `gvt setup` to store the key in the OS credential store",
	)
}
