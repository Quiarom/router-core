// Package cmd: tiny indirection over the keyring package so
// the agent process can be spawned without importing the
// keyring at the top of the file (it is only used for the
// keyring get at startup; failures fall back to env var).
package cmd

import "github.com/zalando/go-keyring"

func keyringGet(service, account string) (string, error) {
	return keyring.Get(service, account)
}
