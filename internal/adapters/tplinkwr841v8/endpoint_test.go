package tplinkwr841v8

import (
	"errors"
	"os"
	"testing"

	"github.com/Quiarom/router-core/internal/domain"
)

func TestUnverifiedEndpointRefused(t *testing.T) {
	t.Setenv("ROUTER_ALLOW_UNVERIFIED", "")
	// Use OpDHCPClients which remains Verified: false until we have
	// physical capture for that endpoint (per Phase 2 plan).
	if err := dispatchAllowed(Endpoints[OpDHCPClients]); !errors.Is(err, domain.ErrUnverifiedEndpoint) {
		t.Fatalf("got %v", err)
	}
	t.Setenv("ROUTER_ALLOW_UNVERIFIED", "1")
	if err := dispatchAllowed(Endpoints[OpDHCPClients]); err != nil {
		t.Fatal(err)
	}
	_ = os.Getenv
}

// TestStatusEndpointIsVerified pins the Verified flag on OpStatus so
// future regressions (e.g. accidentally flipping back to false) are
// caught at test time. ADR 0005 documents the evidence behind this
// verification.
func TestStatusEndpointIsVerified(t *testing.T) {
	if !Endpoints[OpStatus].Verified {
		t.Fatalf("OpStatus must be Verified: true after ADR 0005; got false")
	}
}
