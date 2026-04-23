// Package daemon defines the IPC protocol between the goose-launcher CLI
// client and the resident goose-launcherd daemon process.
//
// Wire format (one request per connection):
//
//	client → daemon: WriteFrame(Request)
//	daemon → client: WriteFrame(Response)
//	connection closed
//
// Each frame is a 4-byte big-endian length followed by JSON-encoded payload.
// Stdin is carried verbatim inside Request.Stdin — JSON handles arbitrary
// strings (including embedded newlines) via escapes; the cost is ~1.3x size
// inflation in pathological cases. The launcher's typical stdin is small
// enough (a few hundred KB) that this is a non-issue. If we ever need to
// stream gigabytes, switch to a length-prefixed binary frame for stdin.
package daemon

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// MaxFrameSize caps a single frame at 256 MiB to prevent a malicious or
// buggy peer from forcing the other side into an OOM. The launcher's actual
// usage is many orders of magnitude below this.
const MaxFrameSize = 256 * 1024 * 1024

// Request is what the client sends. Args are CLI flags as the user typed
// them (everything after argv[0]); Stdin is the raw input the user piped in.
// The daemon parses both the same way the standalone binary used to —
// keeps the client minimal and ensures behavior parity.
type Request struct {
	Args  []string `json:"args"`
	Stdin string   `json:"stdin"`
}

// Response carries the user's selection and the exit code the client should
// propagate. Error is set when the daemon couldn't process the request at
// all (parse error, internal panic, etc.); the client prints it to stderr.
type Response struct {
	Selection string `json:"selection"`
	ExitCode  int    `json:"exit_code"`
	Error     string `json:"error,omitempty"`
}

// WriteFrame writes payload as a length-prefixed frame.
func WriteFrame(w io.Writer, payload []byte) error {
	if len(payload) > MaxFrameSize {
		return fmt.Errorf("daemon: frame size %d exceeds max %d", len(payload), MaxFrameSize)
	}
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(payload)))
	if _, err := w.Write(hdr[:]); err != nil {
		return fmt.Errorf("daemon: write frame header: %w", err)
	}
	if _, err := w.Write(payload); err != nil {
		return fmt.Errorf("daemon: write frame body: %w", err)
	}
	return nil
}

// ReadFrame reads one length-prefixed frame.
func ReadFrame(r io.Reader) ([]byte, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, fmt.Errorf("daemon: read frame header: %w", err)
	}
	n := binary.BigEndian.Uint32(hdr[:])
	if n > MaxFrameSize {
		return nil, fmt.Errorf("daemon: frame size %d exceeds max %d", n, MaxFrameSize)
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, fmt.Errorf("daemon: read frame body: %w", err)
	}
	return buf, nil
}

// WriteRequest is a typed convenience wrapper.
func WriteRequest(w io.Writer, req *Request) error {
	b, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("daemon: marshal request: %w", err)
	}
	return WriteFrame(w, b)
}

// ReadRequest reads and decodes a Request.
func ReadRequest(r io.Reader) (*Request, error) {
	b, err := ReadFrame(r)
	if err != nil {
		return nil, err
	}
	var req Request
	if err := json.Unmarshal(b, &req); err != nil {
		return nil, fmt.Errorf("daemon: unmarshal request: %w", err)
	}
	return &req, nil
}

// WriteResponse is a typed convenience wrapper.
func WriteResponse(w io.Writer, resp *Response) error {
	b, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("daemon: marshal response: %w", err)
	}
	return WriteFrame(w, b)
}

// ReadResponse reads and decodes a Response.
func ReadResponse(r io.Reader) (*Response, error) {
	b, err := ReadFrame(r)
	if err != nil {
		return nil, err
	}
	var resp Response
	if err := json.Unmarshal(b, &resp); err != nil {
		return nil, fmt.Errorf("daemon: unmarshal response: %w", err)
	}
	return &resp, nil
}

// DefaultSocketPath is the canonical Unix-socket path for the user's daemon.
// Lives under ~/Library/Caches/ on macOS so it's cleaned out on disk pressure.
// Path length must stay under 104 bytes (sun_path limit on macOS) — the
// default is well within that.
func DefaultSocketPath() string {
	if v := os.Getenv("GOOSE_LAUNCHER_SOCKET"); v != "" {
		return v
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		// Fallback: tmp. Not ideal (multi-user collision risk) but better
		// than failing to start.
		return filepath.Join(os.TempDir(), "goose-launcher.sock")
	}
	return filepath.Join(cache, "goose-launcher.sock")
}
