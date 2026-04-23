package daemon

import (
	"bytes"
	"strings"
	"testing"
)

func TestRoundTripRequest(t *testing.T) {
	in := &Request{
		Args:  []string{"--rank", "--bind", "tab:replace-query"},
		Stdin: "line one\nline two\nline three with \"quotes\" and \x00 nulls\n",
	}
	var buf bytes.Buffer
	if err := WriteRequest(&buf, in); err != nil {
		t.Fatalf("WriteRequest: %v", err)
	}
	out, err := ReadRequest(&buf)
	if err != nil {
		t.Fatalf("ReadRequest: %v", err)
	}
	if strings.Join(in.Args, "|") != strings.Join(out.Args, "|") {
		t.Errorf("args mismatch: %v vs %v", in.Args, out.Args)
	}
	if in.Stdin != out.Stdin {
		t.Errorf("stdin mismatch:\n in: %q\nout: %q", in.Stdin, out.Stdin)
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

func TestFrameSizeLimit(t *testing.T) {
	var buf bytes.Buffer
	huge := make([]byte, MaxFrameSize+1)
	if err := WriteFrame(&buf, huge); err == nil {
		t.Error("expected error for oversized frame, got nil")
	}
}
