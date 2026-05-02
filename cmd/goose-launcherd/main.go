// goose-launcherd is the resident daemon process. The CLI client
// (cmd/goose-launcher) dials its Unix socket and forwards each user
// invocation; the daemon runs the launcher window and returns the selection.
//
// Phase 2 architecture: ONE app.Window kept alive for the daemon's lifetime.
// Per-request show/hide goes through pkg/macwin's AppKit cgo shim
// ([NSWindow orderOut:] / [NSWindow makeKeyAndOrderFront:]). Eliminates
// per-summon Cocoa first-frame cost (~170 ms) — see DAEMON-RESEARCH.md.
//
// Daemon startup briefly flashes a window: Gio creates the NSWindow visible
// by default; we orderOut as fast as possible after the first FrameEvent so
// we have a valid NSWindow* to work with. Acceptable since the daemon
// starts once per login (later: via launchd).
package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gioui.org/app"

	"github.com/sam33r/goose-launcher/pkg/config"
	"github.com/sam33r/goose-launcher/pkg/daemon"
	"github.com/sam33r/goose-launcher/pkg/input"
	"github.com/sam33r/goose-launcher/pkg/macwin"
	"github.com/sam33r/goose-launcher/pkg/ui"
)

// daemon-wide state. Initialized in main; protected by stateMu where
// concurrent access is possible.
var (
	stateMu sync.Mutex // serializes per-request access to window + handle
	window  *ui.Window
	handle  *macwin.Handle

	// Closed when bootstrap is complete (handle set, accessory policy
	// applied, window hidden). Requests block on this before serving.
	bootstrapDone = make(chan struct{})

	// Single-flight: only one window onscreen at a time. Multiple clients
	// queue here.
	workMu sync.Mutex
)

func main() {
	// Single-instance enforcement. flock auto-releases on process exit, so
	// no stale-PID problems even after SIGKILL. Take this BEFORE binding the
	// socket so a second daemon doesn't unlink the first daemon's socket.
	lock, err := daemon.AcquireLock(daemon.DefaultLockPath())
	if err != nil {
		if errors.Is(err, daemon.ErrAlreadyRunning) {
			fmt.Fprintf(os.Stderr, "goose-launcherd: another daemon is already running\n")
			os.Exit(0) // Not an error condition for autostart use.
		}
		fmt.Fprintf(os.Stderr, "goose-launcherd: %v\n", err)
		os.Exit(1)
	}
	defer lock.Release()

	socketPath := daemon.DefaultSocketPath()
	listener, err := listen(socketPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "goose-launcherd: %v\n", err)
		os.Exit(1)
	}
	log.Printf("listening on %s", socketPath)

	// Create the persistent window. Gio shows it visible by default — we
	// hide it as soon as the first frame fires (so we have a valid
	// NSWindow* and the layout cost is paid up-front, not on first user
	// summon).
	stateMu.Lock()
	window = ui.NewWindowEmpty()
	stateMu.Unlock()

	// Event loop on its own goroutine.
	go func() {
		if err := window.RunForever(); err != nil {
			log.Printf("RunForever returned: %v", err)
		}
		os.Exit(0)
	}()

	// Bootstrap: wait for first frame so NSWindow exists, then locate it,
	// switch to Accessory policy, hide.
	go bootstrapWindow()

	// Listener loop on its own goroutine.
	go acceptLoop(listener)

	// app.Main owns the main thread (macOS requirement).
	app.Main()
}

// bootstrapWindow runs once at daemon startup. Blocks until Gio has made
// the NSWindow real, then captures the pointer + hides the window.
func bootstrapWindow() {
	window.WaitForFirstFrame()
	log.Printf("first frame done; locating NSWindow")

	h, err := macwin.FindWindowByTitle("Goose Launcher", 2*time.Second)
	if err != nil {
		log.Fatalf("bootstrap: %v", err)
	}
	stateMu.Lock()
	handle = h
	stateMu.Unlock()

	macwin.SetAccessoryPolicy()
	// Make the launcher follow the user's current Space when summoned.
	// Without this, macOS switches back to the Space the window was first
	// shown in, which is jarring when summoning from another desktop.
	h.SetLauncherCollectionBehavior()
	h.OrderOut()
	// Dismiss when the user clicks another window/app — same convention as
	// Spotlight/Alfred/Raycast. The Obj-C side ignores resign-key events
	// while the window is hidden, so this won't fire on our own OrderOut.
	h.OnResignKey(func() { window.Cancel() })
	close(bootstrapDone)
	log.Printf("daemon ready; window hidden, accessory policy set")
}

func listen(path string) (net.Listener, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("mkdir socket dir: %w", err)
	}
	// Best-effort cleanup of stale socket. Phase 4 adds proper flock-based
	// single-instance enforcement.
	_ = os.Remove(path)
	l, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("listen %s: %w", path, err)
	}
	if err := os.Chmod(path, 0600); err != nil {
		l.Close()
		return nil, fmt.Errorf("chmod socket: %w", err)
	}
	return l, nil
}

func acceptLoop(l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			log.Printf("accept: %v", err)
			continue
		}
		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	defer conn.Close()

	// First frame must be Hello.
	tag, payload, err := daemon.ReadMsg(conn)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			log.Printf("read hello: %v", err)
		}
		return
	}
	if tag != daemon.MsgTagHello {
		writeResponseLogged(conn, &daemon.Response{
			ExitCode: 2,
			Error:    fmt.Sprintf("expected hello (tag %d), got tag %d", daemon.MsgTagHello, tag),
		})
		return
	}
	hello, err := daemon.DecodeHello(payload)
	if err != nil {
		writeResponseLogged(conn, &daemon.Response{ExitCode: 2, Error: err.Error()})
		return
	}
	if hello.Version != daemon.ProtocolVersion {
		writeResponseLogged(conn, &daemon.Response{
			ExitCode: 2,
			Error: fmt.Sprintf("unsupported protocol version %d (daemon speaks %d)",
				hello.Version, daemon.ProtocolVersion),
		})
		return
	}

	// serveRequest owns the response write and conn close so it can sequence
	// them correctly with the streaming chunk reader.
	serveRequest(conn, hello)
}

func writeResponseLogged(conn net.Conn, resp *daemon.Response) {
	if err := daemon.WriteResponse(conn, resp); err != nil {
		log.Printf("write response: %v", err)
	}
}

// serveRequest runs the launcher for a single client request. Configures
// the persistent window with no items, shows it, streams stdin into the
// live launcher via a dedicated reader goroutine while the user interacts,
// writes the response, then closes the connection (which terminates the
// chunk reader).
func serveRequest(conn net.Conn, hello *daemon.Hello) {
	cfg, err := config.ParseFlags(hello.Args)
	if err != nil {
		writeResponseLogged(conn, &daemon.Response{
			ExitCode: 2,
			Error:    fmt.Sprintf("parse flags: %v", err),
		})
		return
	}

	// Serialize. Concurrent clients queue here; the user only ever sees one
	// window at a time.
	workMu.Lock()
	defer workMu.Unlock()

	// Wait for bootstrap (window created + handle located + accessory
	// applied). On the autostart path the client connects before bootstrap
	// finishes; on subsequent requests this is already closed and returns
	// instantly.
	<-bootstrapDone
	stateMu.Lock()
	w := window
	h := handle
	stateMu.Unlock()
	if w == nil || h == nil {
		writeResponseLogged(conn, &daemon.Response{
			ExitCode: 1,
			Error:    "daemon: bootstrap incomplete",
		})
		return
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in window: %v", r)
		}
	}()

	w.ConfigureEmpty(cfg.HighlightMatches, cfg.ExactMode, cfg.Rank, cfg.Multi)
	log.Printf("serving streaming request")

	t0 := time.Now()
	h.MakeKeyAndOrderFront()
	w.GioWindow().Invalidate() // Wake event loop — see DAEMON-RESEARCH.md.

	// GOOSE_AUTOCANCEL_MS: if set, auto-cancel the window N ms after show.
	// For benchmarking show→done latency without human interaction. Off by
	// default; daemon never cancels the user's window in normal operation.
	if v := os.Getenv("GOOSE_AUTOCANCEL_MS"); v != "" {
		if delay, err := time.ParseDuration(v + "ms"); err == nil {
			go func() {
				time.Sleep(delay)
				w.Cancel()
			}()
		}
	}

	// Stream stdin into the window. The reader goroutine exits cleanly when
	// it sees MsgStdinEOF or a read error — the latter happens when we close
	// the socket below after writing the response.
	chunkReaderDone := make(chan int)
	go streamChunks(conn, w, cfg.Markup, chunkReaderDone)

	selected := w.WaitForSelection()

	// Write response BEFORE closing the conn — closing first would race the
	// reader goroutine and turn the response write into a "use of closed
	// connection" error.
	writeResponseLogged(conn, &daemon.Response{Selection: selected, ExitCode: 0})

	// Now signal the chunk reader to exit by closing the conn. The defer in
	// handleConn will Close again; net.Conn.Close is idempotent.
	_ = conn.Close()

	// Bound the wait so a misbehaving client can't block window hide.
	var streamedCount int
	select {
	case streamedCount = <-chunkReaderDone:
	case <-time.After(500 * time.Millisecond):
		log.Printf("warning: chunk reader did not exit within 500ms")
	}

	log.Printf("request done in %.1f ms: %d items streamed, selection=%q",
		time.Since(t0).Seconds()*1000, streamedCount, selected)

	h.OrderOut()
}

// streamChunks reads MsgStdinChunk frames off conn and appends the parsed
// items to w via AppendItems. Exits when:
//   - MsgStdinEOF arrives (clean termination by client),
//   - any read error occurs (connection closed by daemon after selection,
//     or client disconnected unexpectedly),
//   - a frame with an unexpected tag arrives.
//
// Reports total items streamed via doneC, then closes it.
func streamChunks(conn net.Conn, w *ui.Window, markupFormat string, doneC chan<- int) {
	defer close(doneC)
	index := 0
	for {
		tag, payload, err := daemon.ReadMsg(conn)
		if err != nil {
			doneC <- index
			return
		}
		switch tag {
		case daemon.MsgTagStdinChunk:
			chunk, err := daemon.DecodeChunk(payload)
			if err != nil {
				log.Printf("decode chunk: %v", err)
				doneC <- index
				return
			}
			batch := make([]input.Item, 0, len(chunk.Lines))
			for _, line := range chunk.Lines {
				batch = append(batch, input.ParseLine(line, index, markupFormat))
				index++
			}
			w.AppendItems(batch)
		case daemon.MsgTagStdinEOF:
			doneC <- index
			return
		default:
			log.Printf("unexpected frame tag %d during stream", tag)
			doneC <- index
			return
		}
	}
}
