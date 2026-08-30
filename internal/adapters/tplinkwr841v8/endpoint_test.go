package tplinkwr841v8

import (
	"errors"
	"os"
	"testing"

	"github.com/Quiarom/router-core/internal/domain"
)

func TestUnverifiedEndpointRefused(t *testing.T) {
	t.Setenv("ROUTER_ALLOW_UNVERIFIED", "")
	if err := dispatchAllowed(Endpoints[OpStatus]); !errors.Is(err, domain.ErrUnverifiedEndpoint) {
		t.Fatalf("got %v", err)
	}
	t.Setenv("ROUTER_ALLOW_UNVERIFIED", "1")
	if err := dispatchAllowed(Endpoints[OpStatus]); err != nil {
		t.Fatal(err)
	}
	_ = os.Getenv
}
