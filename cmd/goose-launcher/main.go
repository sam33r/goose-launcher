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
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/sam33r/goose-launcher/pkg/daemon"
)

const (
	dialTimeout    = 250 * time.Millisecond
	startupTimeout = 5 * time.Second
)

func main() {
	stdin, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "goose-launcher: read stdin: %v\n", err)
		os.Exit(2)
	}

	socket := daemon.DefaultSocketPath()
	conn, err := dialWithAutostart(socket)
	if err != nil {
		fmt.Fprintf(os.Stderr, "goose-launcher: %v\n", err)
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

// dialWithAutostart connects to the daemon, launching it on demand if the
// socket is missing or unreachable. Returns an error only if both the
// initial dial and the post-spawn retries fail.
func dialWithAutostart(socket string) (net.Conn, error) {
	// Fast path: daemon already running.
	if conn, err := net.DialTimeout("unix", socket, dialTimeout); err == nil {
		return conn, nil
	}

	// Spawn the daemon. We resolve goose-launcherd via PATH, falling back
	// to the same directory as our own binary (for development workflows
	// where both are at repo root).
	exePath, err := findDaemonBinary()
	if err != nil {
		return nil, fmt.Errorf("autostart: %w", err)
	}

	cmd := exec.Command(exePath)
	// Detach: own session, drop std fds. The daemon writes its own log via
	// log.Printf to stderr — discard for the autostart path so the client
	// terminal isn't polluted.
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("autostart: spawn %s: %w", exePath, err)
	}
	// Don't wait — the daemon runs forever. Releasing avoids a zombie.
	go func() { _ = cmd.Wait() }()

	// Poll the socket until ready or deadline. Bootstrap (window create +
	// font load + first frame + orderOut) takes ~250 ms cold; budget 5 s
	// to be safe on slow systems.
	deadline := time.Now().Add(startupTimeout)
	var lastErr error
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("unix", socket, dialTimeout)
		if err == nil {
			return conn, nil
		}
		lastErr = err
		time.Sleep(50 * time.Millisecond)
	}
	return nil, fmt.Errorf("autostart: daemon never came online within %v: %w", startupTimeout, lastErr)
}

// findDaemonBinary locates goose-launcherd. PATH first; if not found, look
// next to our own binary (development workflow).
func findDaemonBinary() (string, error) {
	if p, err := exec.LookPath("goose-launcherd"); err == nil {
		return p, nil
	}
	self, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locate self: %w", err)
	}
	candidate := filepath.Join(filepath.Dir(self), "goose-launcherd")
	if _, err := os.Stat(candidate); err == nil {
		return candidate, nil
	}
	return "", errors.New("goose-launcherd not found in PATH or alongside goose-launcher")
}
