//go:build windows

package cmd

import "net"

func init() {
	netListenLoopback = func(addr string) (net.Listener, error) {
		return net.Listen("tcp", addr)
	}
}
