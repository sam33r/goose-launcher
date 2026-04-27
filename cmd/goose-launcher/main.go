// goose-launcher is the CLI client. It dials the resident daemon
// (goose-launcherd), sends a Hello frame immediately, then streams stdin
// line-by-line as MsgStdinChunk frames. The daemon shows the launcher
// window as soon as Hello arrives — items appear progressively as the user
// types. The daemon's MsgResponse signals completion; we print the
// selection and exit.
//
// Behavioral contract preserved from the previous standalone binary:
//   - Selected item printed on stdout (one line, no trailing whitespace
//     beyond the existing item content).
//   - Exit 0 on selection or cancel.
//   - Errors (daemon unreachable, IPC failure, etc.) printed to stderr,
//     exit 2.
//
// When the user picks/cancels before stdin EOF, the daemon closes the
// socket. Our stdin-forwarder goroutine then sees ErrClosed on its next
// write and exits silently. The upstream producer (e.g. find) gets SIGPIPE
// on the next write to our stdin pipe and terminates.
package main

import (
	"bufio"
	"errors"
	"fmt"
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

	// Chunk-size knobs. Pick small enough that the first batch lands within
	// one display frame (~16 ms), but large enough that high-throughput
	// producers don't wedge us on per-line frame overhead.
	chunkMaxLines = 64
	chunkMaxBytes = 16 * 1024
	chunkFlushIdle = 10 * time.Millisecond
)

func main() {
	socket := daemon.DefaultSocketPath()
	conn, err := dialWithAutostart(socket)
	if err != nil {
		fmt.Fprintf(os.Stderr, "goose-launcher: %v\n", err)
		os.Exit(2)
	}
	defer conn.Close()

	hello := &daemon.Hello{
		Version: daemon.ProtocolVersion,
		Args:    os.Args[1:],
	}
	if err := daemon.WriteHello(conn, hello); err != nil {
		fmt.Fprintf(os.Stderr, "goose-launcher: send hello: %v\n", err)
		os.Exit(2)
	}

	// Forward stdin in the background; main goroutine blocks on the daemon's
	// response. The forwarder exits cleanly on EOF (sends MsgStdinEOF) or on
	// any write error (daemon closed the socket — that's the cancel path).
	go forwardStdin(conn)

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

// forwardStdin reads os.Stdin line by line and ships batches over conn.
// Batches flush when they hit chunkMaxLines, chunkMaxBytes, or chunkFlushIdle
// has elapsed since the first line in the batch — the latter keeps slow
// producers (one item per second) interactive.
//
// On stdin EOF, sends a MsgStdinEOF frame. On any write error (most commonly
// the daemon closing the socket after the user picks an item), returns
// silently — losing the rest of stdin is the desired cancel behavior.
func forwardStdin(conn net.Conn) {
	scanner := bufio.NewScanner(os.Stdin)
	// Allow long lines (default 64 KB cap is too low for some launcher inputs).
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var batch []string
	var batchBytes int
	var firstAt time.Time

	flush := func() bool {
		if len(batch) == 0 {
			return true
		}
		if err := daemon.WriteChunk(conn, &daemon.StdinChunk{Lines: batch}); err != nil {
			return false
		}
		batch = batch[:0]
		batchBytes = 0
		return true
	}

	// Idle flusher: when a batch has been sitting for chunkFlushIdle, ship it.
	// We use a per-batch timer rather than a continuous ticker so an idle
	// stdin doesn't generate spurious wakeups.
	type lineMsg struct {
		line string
		ok   bool
	}
	lines := make(chan lineMsg)
	go func() {
		defer close(lines)
		for scanner.Scan() {
			lines <- lineMsg{line: scanner.Text(), ok: true}
		}
	}()

	for {
		var timeout <-chan time.Time
		if len(batch) > 0 {
			timeout = time.After(chunkFlushIdle - time.Since(firstAt))
		}
		select {
		case msg, ok := <-lines:
			if !ok {
				if !flush() {
					return
				}
				_ = daemon.WriteEOF(conn)
				return
			}
			if len(batch) == 0 {
				firstAt = time.Now()
			}
			batch = append(batch, msg.line)
			batchBytes += len(msg.line) + 1 // +1 for the newline accounting
			if len(batch) >= chunkMaxLines || batchBytes >= chunkMaxBytes {
				if !flush() {
					return
				}
			}
		case <-timeout:
			if !flush() {
				return
			}
		}
	}
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
