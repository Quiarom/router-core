package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestCLIGoldenOutput(t *testing.T) {
	fixtures := "../../fixtures/synthetic/tplink-wr841n-v8"
	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{
			"probe", []string{"probe", "--fixtures", fixtures},
			"TP-Link TL-WR841N\nHardware: WR841N v8 00000000\nFirmware: 3.13.33 Build 130429 Rel.55978n\nHost: fixture\nAuthentication: success\n",
		},
		{
			"inspect", []string{"inspect", "--fixtures", fixtures},
			"Reachable: true\nWAN: connected\nClients: 2\nWPS: true\nDMZ: false\nUPnP: true\nRemote management: false\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=TestCLIHelper")
			cmd.Env = append(os.Environ(), "ROUTER_CORE_HELPER=1", "ROUTER_CORE_ARGS="+strings.Join(test.args, "\x1f"))
			var output bytes.Buffer
			cmd.Stdout = &output
			cmd.Stderr = &output
			if err := cmd.Run(); err != nil {
				t.Fatalf("command error: %v\n%s", err, output.String())
			}
			if output.String() != test.want {
				t.Fatalf("output:\n%s\nwant:\n%s", output.String(), test.want)
			}
		})
	}
}

func TestCLIHelper(t *testing.T) {
	if os.Getenv("ROUTER_CORE_HELPER") != "1" {
		return
	}
	args := strings.Split(os.Getenv("ROUTER_CORE_ARGS"), "\x1f")
	if err := run(args); err != nil {
		t.Fatal(err)
	}
	os.Exit(0)
}
