package daemon

import (
	"bytes"
	"strings"
	"testing"
)

func TestRoundTripHello(t *testing.T) {
	in := &Hello{
		Version: ProtocolVersion,
		Args:    []string{"--rank", "--bind", "tab:replace-query"},
	}
	var buf bytes.Buffer
	if err := WriteHello(&buf, in); err != nil {
		t.Fatalf("WriteHello: %v", err)
	}
	tag, payload, err := ReadMsg(&buf)
	if err != nil {
		t.Fatalf("ReadMsg: %v", err)
	}
	if tag != MsgTagHello {
		t.Fatalf("tag = %d, want %d", tag, MsgTagHello)
	}
	out, err := DecodeHello(payload)
	if err != nil {
		t.Fatalf("DecodeHello: %v", err)
	}
	if in.Version != out.Version {
		t.Errorf("version: %d vs %d", in.Version, out.Version)
	}
	if strings.Join(in.Args, "|") != strings.Join(out.Args, "|") {
		t.Errorf("args mismatch: %v vs %v", in.Args, out.Args)
	}
}

func TestRoundTripStdinChunk(t *testing.T) {
	in := &StdinChunk{
		Lines: []string{
			"line one",
			"line two",
			"line three with \"quotes\" and \x00 nulls",
			"",
		},
	}
	var buf bytes.Buffer
	if err := WriteChunk(&buf, in); err != nil {
		t.Fatalf("WriteChunk: %v", err)
	}
	tag, payload, err := ReadMsg(&buf)
	if err != nil {
		t.Fatalf("ReadMsg: %v", err)
	}
	if tag != MsgTagStdinChunk {
		t.Fatalf("tag = %d, want %d", tag, MsgTagStdinChunk)
	}
	out, err := DecodeChunk(payload)
	if err != nil {
		t.Fatalf("DecodeChunk: %v", err)
	}
	if len(in.Lines) != len(out.Lines) {
		t.Fatalf("line count: %d vs %d", len(in.Lines), len(out.Lines))
	}
	for i := range in.Lines {
		if in.Lines[i] != out.Lines[i] {
			t.Errorf("line %d: %q vs %q", i, in.Lines[i], out.Lines[i])
		}
	}
}

func TestRoundTripStdinEOF(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteEOF(&buf); err != nil {
		t.Fatalf("WriteEOF: %v", err)
	}
	tag, payload, err := ReadMsg(&buf)
	if err != nil {
		t.Fatalf("ReadMsg: %v", err)
	}
	if tag != MsgTagStdinEOF {
		t.Fatalf("tag = %d, want %d", tag, MsgTagStdinEOF)
	}
	if len(payload) != 0 {
		t.Errorf("EOF payload should be empty, got %d bytes", len(payload))
	}
}

func TestRoundTripResponse(t *testing.T) {
	in := &Response{Selection: "selected/path/file.go", ExitCode: 0}
	var buf bytes.Buffer
	if err := WriteResponse(&buf, in); err != nil {
		t.Fatalf("WriteResponse: %v", err)
	}
	out, err := ReadResponse(&buf)
	if err != nil {
		t.Fatalf("ReadResponse: %v", err)
	}
	if *in != *out {
		t.Errorf("response mismatch: %+v vs %+v", in, out)
	}
}

func TestReadResponseRejectsWrongTag(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteHello(&buf, &Hello{Version: ProtocolVersion}); err != nil {
		t.Fatalf("WriteHello: %v", err)
	}
	if _, err := ReadResponse(&buf); err == nil {
		t.Error("expected error reading Response when frame is Hello")
	}
}

func TestMixedFrameStream(t *testing.T) {
	// Simulate the daemon side: client sends Hello, two chunks, EOF.
	// Daemon reads frames in a loop and dispatches by tag.
	var buf bytes.Buffer
	if err := WriteHello(&buf, &Hello{Version: ProtocolVersion, Args: []string{"--exact"}}); err != nil {
		t.Fatalf("WriteHello: %v", err)
	}
	if err := WriteChunk(&buf, &StdinChunk{Lines: []string{"a", "b"}}); err != nil {
		t.Fatalf("WriteChunk 1: %v", err)
	}
	if err := WriteChunk(&buf, &StdinChunk{Lines: []string{"c"}}); err != nil {
		t.Fatalf("WriteChunk 2: %v", err)
	}
	if err := WriteEOF(&buf); err != nil {
		t.Fatalf("WriteEOF: %v", err)
	}

	gotTags := []uint8{}
	allLines := []string{}
	for {
		tag, payload, err := ReadMsg(&buf)
		if err != nil {
			break
		}
		gotTags = append(gotTags, tag)
		if tag == MsgTagStdinChunk {
			c, err := DecodeChunk(payload)
			if err != nil {
				t.Fatalf("DecodeChunk: %v", err)
			}
			allLines = append(allLines, c.Lines...)
		}
		if tag == MsgTagStdinEOF {
			break
		}
	}
	wantTags := []uint8{MsgTagHello, MsgTagStdinChunk, MsgTagStdinChunk, MsgTagStdinEOF}
	if len(gotTags) != len(wantTags) {
		t.Fatalf("tag sequence length: got %v, want %v", gotTags, wantTags)
	}
	for i := range gotTags {
		if gotTags[i] != wantTags[i] {
			t.Errorf("tag[%d] = %d, want %d", i, gotTags[i], wantTags[i])
		}
	}
	if strings.Join(allLines, "|") != "a|b|c" {
		t.Errorf("lines: got %v, want [a b c]", allLines)
	}
}

func TestWriteMsgRejectsOversizedFrame(t *testing.T) {
	var buf bytes.Buffer
	huge := make([]byte, MaxFrameSize+1)
	if err := WriteMsg(&buf, MsgTagStdinChunk, huge); err == nil {
		t.Error("expected error for oversized frame, got nil")
	}
}
