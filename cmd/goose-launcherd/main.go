// goose-launcherd is the resident daemon process. The CLI client
// (cmd/goose-launcher) dials its Unix socket and forwards each user
// invocation; the daemon runs the launcher window and returns the selection.
//
// Phase 1 architecture: one app.Window per request, destroyed at the end.
// This still amortizes dyld + Go runtime + font init across invocations
// (~70 ms saved each call) but pays Gio's per-window setup every time.
// Phase 2 will reuse a single hidden window via the AppKit cgo shim
// validated in cmd/spike.
package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gioui.org/app"

	"github.com/sam33r/goose-launcher/pkg/config"
	"github.com/sam33r/goose-launcher/pkg/daemon"
	"github.com/sam33r/goose-launcher/pkg/input"
	"github.com/sam33r/goose-launcher/pkg/ui"
)

func main() {
	socketPath := daemon.DefaultSocketPath()
	if v := os.Getenv("GOOSE_LAUNCHER_SOCKET"); v != "" {
		socketPath = v
	}

	listener, err := listen(socketPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "goose-launcherd: %v\n", err)
		os.Exit(1)
	}
	log.Printf("listening on %s", socketPath)

	// Accept loop runs on a goroutine; app.Main owns the main thread.
	// The work mutex serializes window creation — Gio supports sequential
	// app.Window instances under one app.Main, but only one at a time.
	go acceptLoop(listener)

	app.Main()
}

// listen unlinks any stale socket file (left over from a SIGKILL'd daemon)
// then binds. Mode 0600: only the user can talk to their daemon.
func listen(path string) (net.Listener, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("mkdir socket dir: %w", err)
	}
	// Best-effort cleanup of stale socket. If a live daemon is running,
	// the bind below will fail — that's how we detect it (single-instance
	// enforcement comes properly in phase 4 via flock).
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

// workMu serializes window creation. macOS + Gio require app.Window to be
// created/destroyed sequentially under one app.Main; concurrent clients
// queue here.
var workMu sync.Mutex

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

	req, err := daemon.ReadRequest(conn)
	if err != nil {
		// EOF on read just means the client disconnected; log others.
		if !errors.Is(err, io.EOF) {
			log.Printf("read request: %v", err)
		}
		return
	}

	resp := serveRequest(req)
	if err := daemon.WriteResponse(conn, resp); err != nil {
		log.Printf("write response: %v", err)
	}
}

// serveRequest runs the launcher for a single client request. Mirrors what
// the standalone cmd/goose-launcher main used to do, minus the OS-process
// boundary.
func serveRequest(req *daemon.Request) *daemon.Response {
	cfg, err := config.ParseFlags(req.Args)
	if err != nil {
		return &daemon.Response{ExitCode: 2, Error: fmt.Sprintf("parse flags: %v", err)}
	}

	reader := input.NewReader(strings.NewReader(req.Stdin), cfg.Markup)
	items, err := reader.ReadAll()
	if err != nil {
		return &daemon.Response{ExitCode: 2, Error: fmt.Sprintf("read items: %v", err)}
	}

	// Serialize window creation/run. Phase 2 keeps a single window alive
	// across requests; for now every request is a fresh app.Window.
	workMu.Lock()
	defer workMu.Unlock()

	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in window: %v", r)
		}
	}()

	log.Printf("serving request: %d items, args=%v", len(items), req.Args)
	window := ui.NewWindow(items, cfg.HighlightMatches, cfg.ExactMode, cfg.Rank)
	selected, err := window.Run()
	if err != nil {
		log.Printf("window error: %v", err)
		return &daemon.Response{ExitCode: 1, Error: fmt.Sprintf("window: %v", err)}
	}
	log.Printf("request done: selection=%q", selected)

	// Match the standalone binary's contract: exit 0 always; selection on
	// stdout (here: in the response). Cancel = empty selection, exit 0.
	return &daemon.Response{Selection: selected, ExitCode: 0}
}
