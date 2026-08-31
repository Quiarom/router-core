package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/Quiarom/router-core/cmd/router-core-learn/sanitize"
)

// newProbeLoginPageCmd builds the `probe-login-page` subcommand. It
// fetches the unauthenticated root URL of the WR841N and persists the
// sanitized response to the capture directory. No credentials are
// sent — this is purely a discovery probe to see the shape of the
// login page itself.
//
// Usage:
//
//	router-core-learn probe-login-page --host 192.168.1.1
//
// Output:
//
//	fixtures/captured/tplink-wr841n-v8/login-page.html (sanitized)
func newProbeLoginPageCmd() *cobra.Command {
	host := ""
	outDir := ""

	cmd := &cobra.Command{
		Use:   "probe-login-page",
		Short: "Fetch the unauthenticated login page and persist it sanitized",
		Long: `probe-login-page fetches http://<host>/ without any credentials
and writes the sanitized response body to <out>/login-page.html.

This is a discovery probe: no authentication is attempted. It exists
so the operator can inspect the actual shape of the login page (form
action, input field names, error states) without DevTools.

The body is sanitized in memory before persistence: any session
tokens, Authorization material, MAC addresses, SSIDs, or password
fields are replaced with placeholders. The HTML structure and any
visible form field NAMES are preserved so the operator can build the
correct authentication recipe from the captured page.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProbeLoginPage(cmd, host, outDir)
		},
	}

	cmd.Flags().StringVar(&host, "host", "192.168.0.1",
		"local router address (RFC1918 literal)")
	cmd.Flags().StringVar(&outDir, "out", "",
		"output directory (default: ./fixtures/captured/tplink-wr841n-v8)")
	cmd.Flags().Duration("timeout", 5*time.Second, "per-request timeout")
	_ = cmd.Flags().Lookup("timeout")

	return cmd
}

func runProbeLoginPage(cmd *cobra.Command, host, outDir string) error {
	if !isRFC1918OrLoopback(host) {
		return exitCodeError(5,
			fmt.Sprintf("refusing to probe host %q: not loopback/RFC1918", host),
			nil)
	}

	resolvedOut := outDir
	if resolvedOut == "" {
		resolvedOut = filepath.Join("fixtures", "captured", "tplink-wr841n-v8")
	}

	timeout, err := cmd.Flags().GetDuration("timeout")
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()

	fmt.Fprintf(out, "\nrouter-core-learn %s\n", probeVersion)
	fmt.Fprintf(out, "target: TP-Link TL-WR841N v8.4, firmware %s\n", expectedFirmware)
	fmt.Fprintf(out, "host:   %s\n", host)
	fmt.Fprintf(out, "out:    %s\n\n", resolvedOut)

	fmt.Fprintf(out, "[1/1] Fetching unauthenticated login page\n")
	fmt.Fprintf(out, "  GET http://%s/ …\n", host)

	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	u, err := url.Parse("http://" + host + "/")
	if err != nil {
		return exitCodeError(1, "parse URL", err)
	}
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return exitCodeError(1, "build request", err)
	}
	req.Header.Set("User-Agent", probeUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintln(out, "  → FAIL")
		return exitCodeError(3, "host unreachable", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return exitCodeError(1, "read body", err)
	}

	// Structural summary: parse the WWW-Authenticate header if present,
	// detect <form> elements by name, and report body size.
	wwwAuth := resp.Header.Get("WWW-Authenticate")
	bodyStr := string(body)

	fmt.Fprintf(out, "  HTTP %d, %d bytes\n", resp.StatusCode, len(body))
	if wwwAuth != "" {
		fmt.Fprintf(out, "  WWW-Authenticate: %s\n", sanitize.Apply(wwwAuth, sanitize.Default()))
	} else {
		fmt.Fprintln(out, "  WWW-Authenticate: (absent)")
	}
	formCount := strings.Count(bodyStr, "<form")
	inputCount := strings.Count(bodyStr, "<input")
	fmt.Fprintf(out, "  <form> elements:  %d\n", formCount)
	fmt.Fprintf(out, "  <input> elements: %d\n", inputCount)
	if titleStart := strings.Index(bodyStr, "<title>"); titleStart >= 0 {
		titleEnd := strings.Index(bodyStr[titleStart:], "</title>")
		if titleEnd > 0 {
			title := bodyStr[titleStart+len("<title>") : titleStart+titleEnd]
			fmt.Fprintf(out, "  <title>: %s\n", strings.TrimSpace(title))
		}
	}

	// Persist sanitized body.
	if err := os.MkdirAll(resolvedOut, 0o755); err != nil {
		return exitCodeError(1, "create out dir", err)
	}
	saniBody := sanitize.Apply(bodyStr, sanitize.Default())
	outFile := filepath.Join(resolvedOut, "login-page.html")
	if err := os.WriteFile(outFile, []byte(saniBody), 0o644); err != nil {
		return exitCodeError(1, "write file", err)
	}
	fmt.Fprintf(out, "  sanitized body written to: %s\n", outFile)
	fmt.Fprintln(out, "  → OK")
	fmt.Fprintln(out, "\nNo credentials sent. No mutation performed. Sanitized.")
	return nil
}
