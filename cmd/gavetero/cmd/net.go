//go:build !windows

package cmd

import "net"

// netListenLoopback wraps net.Listen so it can be replaced in
// tests. Production code calls net.Listen("tcp", addr).
func init() {
	netListenLoopback = func(addr string) (net.Listener, error) {
		return net.Listen("tcp", addr)
	}
}
