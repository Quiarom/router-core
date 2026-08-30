package domain

import (
	"context"
	"errors"
)

// RouterAdapter is the single contract between router-core and any concrete
// device implementation, whether hand-written (P0) or generated later.
//
// Every method is READ-ONLY. A conforming adapter must never mutate device
// configuration, and must never expose a method that could.
type RouterAdapter interface {
	Identify(ctx context.Context) (DeviceInfo, error)
	Status(ctx context.Context) (RouterStatus, error)
	Clients(ctx context.Context) ([]Client, error)
	Security(ctx context.Context) (SecurityState, error)
}

var (
	// ErrUnauthenticated: the device answered with a login/auth page instead
	// of the requested page.
	ErrUnauthenticated = errors.New("router-core: not authenticated")
	// ErrSessionConflict: the device reports another administrative session.
	ErrSessionConflict = errors.New("router-core: another administrative session is active")
	// ErrUnexpectedResponse: the response did not match any known shape.
	ErrUnexpectedResponse = errors.New("router-core: unexpected response shape")
	// ErrUnreachable: transport-level failure (timeout, refused, DNS).
	ErrUnreachable = errors.New("router-core: device unreachable")
	// ErrWriteForbidden: something attempted a mutating request.
	ErrWriteForbidden = errors.New("router-core: write operations are forbidden")
	// ErrCaptureMissing: the operation depends on captured traffic that this
	// repository does not contain, so it is not implemented rather than
	// guessed. See BLOCKED_CAPTURE.md.
	ErrCaptureMissing = errors.New("router-core: required capture is missing")
	// ErrUnverifiedEndpoint: the endpoint recipe has not been confirmed
	// against real captured traffic and unverified live access is not enabled.
	ErrUnverifiedEndpoint = errors.New("router-core: endpoint is unverified against captured traffic")
)
