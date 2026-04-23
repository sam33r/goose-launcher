// goose-launcher is the CLI client. It dials the resident daemon
// (goose-launcherd), forwards the user's argv + stdin, blocks for the
// daemon to render the window and return a selection, then prints the
// selection to stdout.
//
// Behavioral contract preserved from the previous standalone binary:
//   - Selected item printed on stdout (one line, no trailing whitespace
//     beyond the existing item content).
//   - Exit 0 on selection or cancel.
//   - Errors (daemon unreachable, IPC failure, etc.) printed to stderr,
//     exit 2.
package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/sam33r/goose-launcher/pkg/daemon"
)

const dialTimeout = 2 * time.Second

func main() {
	stdin, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "goose-launcher: read stdin: %v\n", err)
		os.Exit(2)
	}

	socket := daemon.DefaultSocketPath()
	conn, err := net.DialTimeout("unix", socket, dialTimeout)
	if err != nil {
		// Phase 1: the user starts the daemon manually. Phase 4 will
		// fork-exec it on demand and retry the dial.
		fmt.Fprintf(os.Stderr,
			"goose-launcher: cannot reach daemon at %s: %v\n"+
				"Start it first:  goose-launcherd &\n",
			socket, err)
		os.Exit(2)
	}
	defer conn.Close()

	req := &daemon.Request{
		Args:  os.Args[1:],
		Stdin: string(stdin),
	}
	if err := daemon.WriteRequest(conn, req); err != nil {
		fmt.Fprintf(os.Stderr, "goose-launcher: send request: %v\n", err)
		os.Exit(2)
	}

	resp, err := daemon.ReadResponse(conn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "goose-launcher: read response: %v\n", err)
		os.Exit(2)
	}

	if resp.Error != "" {
		fmt.Fprintf(os.Stderr, "goose-launcher: %s\n", resp.Error)
	}
	if resp.Selection != "" {
		fmt.Println(resp.Selection)
	}
	os.Exit(resp.ExitCode)
}
