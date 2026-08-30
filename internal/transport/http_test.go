package transport

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Quiarom/router-core/internal/domain"
)

func TestAllowedHost(t *testing.T) {
	for _, host := range []string{"127.0.0.1", "localhost", "10.0.0.1", "172.16.0.1", "192.168.1.1"} {
		if !IsAllowedHost(host) {
			t.Errorf("%s rejected", host)
		}
	}
	for _, host := range []string{"8.8.8.8", "router.example", "172.15.0.1", "192.169.0.1"} {
		if IsAllowedHost(host) {
			t.Errorf("%s accepted", host)
		}
	}
}

func TestDispatchRejectsWrites(t *testing.T) {
	client := New()
	_, _, err := client.dispatch(context.Background(), http.MethodPost, "http://127.0.0.1/")
	if !errors.Is(err, domain.ErrWriteForbidden) {
		t.Fatalf("got %v", err)
	}
}

func TestTimeoutIsUnreachable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer server.Close()
	client := New(WithTimeout(10 * time.Millisecond))
	_, _, err := client.Get(context.Background(), server.URL)
	if !errors.Is(err, domain.ErrUnreachable) {
		t.Fatalf("got %v", err)
	}
}
