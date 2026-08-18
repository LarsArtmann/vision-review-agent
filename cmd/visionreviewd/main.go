// visionreviewd is the event-sourced UI review daemon CLI.
//
// Usage:
//
//	visionreviewd run       [-config PATH]                 # daemon loop
//	visionreviewd once      [-config PATH]                 # single pass
//	visionreviewd discover  ROOT                            # suggest config JSON
//	visionreviewd compare   [-config PATH] -project P A B   # manual A/B compare
//	visionreviewd events    [-config PATH]                  # print event journal
//	visionreviewd replay    [-config PATH]                  # rebuild reviews dir
//	visionreviewd doctor    [-config PATH]                  # health checks
//	visionreviewd version                                   # print version
package main

import (
	"fmt"
	"io"
	"os"
)

// version is the daemon version string. It is a var (not a const) so release
// tooling can override it at build time via -ldflags "-X main.version=...".
var version = "0.7.0-dev"

// Exit codes: usage errors are distinguishable from runtime failures so
// scripts can treat 2 as "operator fixable" and 1 as "something broke".
const (
	exitOK     = 0
	exitFailed = 1
	exitUsage  = 2
)

// Subcommand names, shared by dispatch and tests.
const (
	cmdRun      = "run"
	cmdOnce     = "once"
	cmdDiscover = "discover"
	cmdCompare  = "compare"
	cmdEvents   = "events"
	cmdReplay   = "replay"
	cmdDoctor   = "doctor"
	cmdVersion  = "version"
	cmdHelp     = "help"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run dispatches a subcommand and returns the process exit code. Parsing and
// dispatch never call os.Exit so tests can drive every path.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)

		return exitUsage
	}

	command, rest := args[0], args[1:]

	switch command {
	case cmdRun:
		return runDaemonCommand(rest, stdout, stderr)
	case cmdOnce:
		return runOnceCommand(rest, stdout, stderr)
	case cmdDiscover:
		return runDiscoverCommand(rest, stdout, stderr)
	case cmdCompare:
		return runCompareCommand(rest, stdout, stderr)
	case cmdEvents:
		return runEventsCommand(rest, stdout, stderr)
	case cmdReplay:
		return runReplayCommand(rest, stdout, stderr)
	case cmdDoctor:
		return runDoctorCommand(rest, stdout, stderr)
	case cmdVersion, "--version", "-v":
		fmt.Fprintln(stdout, "visionreviewd", version)

		return exitOK
	case cmdHelp, "--help", "-h":
		printUsage(stdout)

		return exitOK
	default:
		fmt.Fprintf(stderr, "visionreviewd: unknown command %q\n\n", command)
		printUsage(stderr)

		return exitUsage
	}
}

// printUsage writes the subcommand overview.
func printUsage(w io.Writer) {
	fmt.Fprint(w, `Usage: visionreviewd COMMAND [flags]

Commands:
  run       Run the review daemon: scan every interval, review changes
  once      Run a single review pass now
  discover  Walk a directory tree and print a suggested config JSON
  compare   Manually compare a BEFORE and AFTER screenshot of one view
  events    Print the recorded event journal
  replay    Rebuild the reviews directory from the event journal
  doctor    Check config, directories, globs, and the model endpoint
  version   Print the version

Most commands accept -config PATH (default ~/.config/visionreviewd/config.json).
`)
}
