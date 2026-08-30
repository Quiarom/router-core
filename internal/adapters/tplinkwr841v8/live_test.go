package tplinkwr841v8

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Quiarom/router-core/internal/transport"
)

func TestLiveOptIn(t *testing.T) {
	if os.Getenv("ROUTER_LIVE_TESTS") != "1" {
		t.Skip("set ROUTER_LIVE_TESTS=1 to opt into local read-only integration tests")
	}
	host := os.Getenv("ROUTER_LIVE_HOST")
	if host == "" {
		host = "192.168.0.1"
	}
	client := transport.New(transport.WithTimeout(2 * time.Second))
	if _, _, err := client.Get(context.Background(), normalizeRoot(host)); err != nil {
		t.Logf("local router probe: %v", err)
	}
}
