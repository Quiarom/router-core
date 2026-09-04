// Command gavetero is the user-facing orchestrator CLI for the
// Gavetero project. It hides the implementation detail of the
// three transitional binaries (router-core, router-core-agent,
// router-core-learn) and provides a small set of high-level
// commands that the end user actually runs.
//
// Subcommands:
//
//	setup     store the GMI Cloud API key once, securely
//	ask       investigate a network question with MiniMax M3
//	inspect   show the current router observations
//	doctor    run 6 checks; report which surface is ready
//	version   print the version
//
// This is a thin orchestrator. It does not reimplement the
// runtime, the agent, or the lab. It calls those binaries with
// the right flags, captures their output, and presents it.
//
// The transitional binaries still own their implementation;
// gavetero is the entry point.
package main

import (
	"github.com/Quiarom/router-core/cmd/gavetero/cmd"
)

func main() {
	cmd.Execute()
}
