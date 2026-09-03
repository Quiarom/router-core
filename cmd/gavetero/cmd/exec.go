package cmd

import (
	"os/exec"
)

// execCmdResult is a thin wrapper around os/exec.Cmd that
// delays the actual os/exec import to this file. This lets
// the doctor test mock the subprocess layer without touching
// the platform-specific os/exec call sites.
type execCmdResult struct {
	cmd *exec.Cmd
}

func (r *execCmdResult) Run() error { return r.cmd.Run() }

func newExecCmd(name string, args ...string) *execCmdResult {
	return &execCmdResult{cmd: exec.Command(name, args...)}
}
