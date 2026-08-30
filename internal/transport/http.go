package transport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Quiarom/router-core/internal/domain"
)

const (
	defaultTimeout = 5 * time.Second
	maxBodySize    = 2 << 20
)

// Client is a read-only HTTP client for local router management pages.
type Client struct {
	httpClient *http.Client
	timeout    time.Duration
}

// Option configures a Client.
type Option func(*Client)

// WithTimeout changes the per-request timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		if timeout > 0 {
			c.timeout = timeout
		}
	}
}

// New creates a local-only, GET-only HTTP client.
func New(options ...Option) *Client {
	c := &Client{timeout: defaultTimeout}
	for _, option := range options {
		option(c)
	}
	c.httpClient = &http.Client{
		Transport: &http.Transport{Proxy: nil},
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			if req.Response == nil || req.Response.Request == nil {
				return errors.New("router-core: redirect missing source request")
			}
			if !strings.EqualFold(req.URL.Host, req.Response.Request.URL.Host) {
				return errors.New("router-core: cross-host redirect refused")
			}
			return nil
		},
	}
	return c
}

// IsAllowedHost reports whether host is a literal local management host.
// Arbitrary DNS names are intentionally not resolved.
func IsAllowedHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	hostOnly := host
	if h, _, err := net.SplitHostPort(host); err == nil {
		hostOnly = h
	} else {
		hostOnly = strings.Trim(host, "[]")
	}
	ip := net.ParseIP(hostOnly)
	if ip == nil {
		return false
	}
	if ip.IsLoopback() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		return ip4[0] == 10 ||
			(ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) ||
			(ip4[0] == 192 && ip4[1] == 168)
	}
	return false
}

func (c *Client) dispatch(ctx context.Context, method, rawURL string) ([]byte, int, error) {
	if method != http.MethodGet {
		return nil, 0, domain.ErrWriteForbidden
	}
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, 0, fmt.Errorf("router-core: invalid local URL %q", rawURL)
	}
	if !IsAllowedHost(u.Host) {
		return nil, 0, fmt.Errorf("router-core: host %q is not an allowed local address", u.Host)
	}
	requestCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, method, u.String(), nil)
	if err != nil {
		return nil, 0, fmt.Errorf("router-core: create request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %v", domain.ErrUnreachable, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize+1))
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("%w: read response: %v", domain.ErrUnreachable, err)
	}
	if len(body) > maxBodySize {
		return nil, resp.StatusCode, errors.New("router-core: response exceeds the 2 MiB read cap")
	}
	return body, resp.StatusCode, nil
}

// Get fetches a local HTTP URL and returns its body and status code.
func (c *Client) Get(ctx context.Context, rawURL string) ([]byte, int, error) {
	return c.dispatch(ctx, http.MethodGet, rawURL)
}

// Do is the single request dispatch path. Only GET is permitted.
func (c *Client) Do(ctx context.Context, method, rawURL string) ([]byte, int, error) {
	return c.dispatch(ctx, method, rawURL)
}
