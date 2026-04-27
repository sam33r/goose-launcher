// Package daemon defines the IPC protocol between the goose-launcher CLI
// client and the resident goose-launcherd daemon process.
//
// Wire format
//
// Each message is a single framed envelope:
//
//	[1-byte tag][4-byte big-endian length N][N payload bytes]
//
// Payload is JSON-encoded for tagged message bodies (Hello, StdinChunk,
// Response). MsgStdinEOF carries no payload (length=0).
//
// Conversation shape (one connection per client invocation):
//
//	client -> daemon : MsgHello
//	client -> daemon : MsgStdinChunk*  (zero or more, batched)
//	client -> daemon : MsgStdinEOF
//	daemon -> client : MsgResponse
//	connection closed
//
// The daemon shows the window as soon as Hello arrives — items stream in
// over the chunk frames while the user types. Selection / cancel completes
// the request even before EOF; the daemon then closes the socket, which
// signals the client's stdin-forwarder goroutine (and the upstream producer
// via SIGPIPE) to terminate.
package daemon

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ProtocolVersion is bumped on every wire-format change. Mismatched versions
// are a hard error — the daemon does not attempt backward compatibility.
const ProtocolVersion = 2

// MaxFrameSize caps a single frame at 256 MiB to prevent a malicious or
// buggy peer from forcing the other side into an OOM. The launcher's actual
// usage is many orders of magnitude below this.
const MaxFrameSize = 256 * 1024 * 1024

// Message tag values. Stable across protocol versions within v2.
const (
	MsgTagHello      uint8 = 1
	MsgTagStdinChunk uint8 = 2
	MsgTagStdinEOF   uint8 = 3
	MsgTagResponse   uint8 = 4
)

// Hello is the client's first message. Carries argv (everything after argv[0])
// and the protocol version it speaks. The daemon parses Args the same way the
// standalone binary used to.
type Hello struct {
	Version int      `json:"version"`
	Args    []string `json:"args"`
}

// StdinChunk carries a batch of stdin lines. The client batches lines (by
// count, byte size, or short idle interval) before sending; the daemon
// appends them to the live launcher items as each chunk arrives.
type StdinChunk struct {
	Lines []string `json:"lines"`
}

// Response carries the user's selection and the exit code the client should
// propagate. Error is set when the daemon couldn't process the request at
// all (parse error, internal panic, etc.); the client prints it to stderr.
type Response struct {
	Selection string `json:"selection"`
	ExitCode  int    `json:"exit_code"`
	Error     string `json:"error,omitempty"`
}

// WriteMsg writes a tagged, length-prefixed frame.
func WriteMsg(w io.Writer, tag uint8, payload []byte) error {
	if len(payload) > MaxFrameSize {
		return fmt.Errorf("daemon: frame size %d exceeds max %d", len(payload), MaxFrameSize)
	}
	var hdr [5]byte
	hdr[0] = tag
	binary.BigEndian.PutUint32(hdr[1:], uint32(len(payload)))
	if _, err := w.Write(hdr[:]); err != nil {
		return fmt.Errorf("daemon: write frame header: %w", err)
	}
	if len(payload) > 0 {
		if _, err := w.Write(payload); err != nil {
			return fmt.Errorf("daemon: write frame body: %w", err)
		}
	}
	return nil
}

// ReadMsg reads one tagged frame. Returns the tag byte and the payload bytes.
// The payload may be empty (e.g. MsgStdinEOF).
func ReadMsg(r io.Reader) (uint8, []byte, error) {
	var hdr [5]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return 0, nil, fmt.Errorf("daemon: read frame header: %w", err)
	}
	tag := hdr[0]
	n := binary.BigEndian.Uint32(hdr[1:])
	if n > MaxFrameSize {
		return 0, nil, fmt.Errorf("daemon: frame size %d exceeds max %d", n, MaxFrameSize)
	}
	if n == 0 {
		return tag, nil, nil
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return 0, nil, fmt.Errorf("daemon: read frame body: %w", err)
	}
	return tag, buf, nil
}

// WriteHello sends a MsgHello frame.
func WriteHello(w io.Writer, h *Hello) error {
	b, err := json.Marshal(h)
	if err != nil {
		return fmt.Errorf("daemon: marshal hello: %w", err)
	}
	return WriteMsg(w, MsgTagHello, b)
}

// DecodeHello decodes the payload of a MsgHello frame.
func DecodeHello(payload []byte) (*Hello, error) {
	var h Hello
	if err := json.Unmarshal(payload, &h); err != nil {
		return nil, fmt.Errorf("daemon: unmarshal hello: %w", err)
	}
	return &h, nil
}

// WriteChunk sends a MsgStdinChunk frame.
func WriteChunk(w io.Writer, c *StdinChunk) error {
	b, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("daemon: marshal chunk: %w", err)
	}
	return WriteMsg(w, MsgTagStdinChunk, b)
}

// DecodeChunk decodes the payload of a MsgStdinChunk frame.
func DecodeChunk(payload []byte) (*StdinChunk, error) {
	var c StdinChunk
	if err := json.Unmarshal(payload, &c); err != nil {
		return nil, fmt.Errorf("daemon: unmarshal chunk: %w", err)
	}
	return &c, nil
}

// WriteEOF sends a MsgStdinEOF frame (no payload).
func WriteEOF(w io.Writer) error {
	return WriteMsg(w, MsgTagStdinEOF, nil)
}

// WriteResponse sends a MsgResponse frame.
func WriteResponse(w io.Writer, r *Response) error {
	b, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("daemon: marshal response: %w", err)
	}
	return WriteMsg(w, MsgTagResponse, b)
}

// DecodeResponse decodes the payload of a MsgResponse frame.
func DecodeResponse(payload []byte) (*Response, error) {
	var r Response
	if err := json.Unmarshal(payload, &r); err != nil {
		return nil, fmt.Errorf("daemon: unmarshal response: %w", err)
	}
	return &r, nil
}

// ReadResponse is a convenience for the client: reads one frame and
// requires it to be MsgResponse.
func ReadResponse(r io.Reader) (*Response, error) {
	tag, payload, err := ReadMsg(r)
	if err != nil {
		return nil, err
	}
	if tag != MsgTagResponse {
		return nil, fmt.Errorf("daemon: expected response frame (tag %d), got tag %d", MsgTagResponse, tag)
	}
	return DecodeResponse(payload)
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
