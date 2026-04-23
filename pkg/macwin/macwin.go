// Package macwin is a thin AppKit shim that drives a Gio-managed NSWindow's
// visibility and activation policy from outside Gio. Used by the launcher
// daemon to keep one window alive across many user invocations and toggle
// its visibility instantly.
//
// Why this exists: Gio v0.9.0 doesn't expose NSWindow show/hide. The only
// public dismissal API destroys the window. By dispatching `[NSWindow
// orderOut:]` and `[NSWindow makeKeyAndOrderFront:]` directly via cgo we
// keep the window object alive while toggling its on-screen state. See
// docs/DAEMON-RESEARCH.md and cmd/spike/ for the rationale + validation.
//
// macOS-only by design.
//
//go:build darwin

package macwin

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>

void* macwin_findWindowByTitle(const char *titleC);
void  macwin_orderOut(void *win);
void  macwin_makeKeyAndOrderFront(void *win);
void  macwin_setAccessoryPolicy(void);
void  macwin_releaseWindow(void *win);
*/
import "C"

import (
	"errors"
	"time"
	"unsafe"
)

// Handle is an opaque retained reference to an NSWindow. Release with Free
// when no longer needed (typically: never, since the daemon holds one for
// its lifetime).
type Handle struct {
	ptr unsafe.Pointer
}

// FindWindowByTitle locates the NSWindow with the given title in [NSApp
// windows]. Polls briefly because the window may not yet be created when
// the caller tries — Gio creates the NSWindow lazily inside its event
// loop after the first FrameEvent.
//
// Returns an error if no matching window appears within timeout.
func FindWindowByTitle(title string, timeout time.Duration) (*Handle, error) {
	cTitle := C.CString(title)
	defer C.free(unsafe.Pointer(cTitle))

	deadline := time.Now().Add(timeout)
	for {
		ptr := C.macwin_findWindowByTitle(cTitle)
		if ptr != nil {
			return &Handle{ptr: ptr}, nil
		}
		if time.Now().After(deadline) {
			return nil, errors.New("macwin: no window with matching title within timeout")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// OrderOut instantly removes the window from the screen without destroying
// it. Cheap (~1 ms). Safe to call multiple times.
func (h *Handle) OrderOut() {
	if h == nil {
		return
	}
	C.macwin_orderOut(h.ptr)
}

// MakeKeyAndOrderFront shows the window and gives it keyboard focus.
//
// IMPORTANT: callers must follow this with `gioWin.Invalidate()` to wake
// Gio's event loop — otherwise macOS won't schedule a paint and Gio sits
// parked in Window.Event(). This was the key learning from the phase 0
// spike (see docs/DAEMON-RESEARCH.md).
func (h *Handle) MakeKeyAndOrderFront() {
	if h == nil {
		return
	}
	C.macwin_makeKeyAndOrderFront(h.ptr)
}

// Free releases the retained NSWindow reference. Idempotent.
func (h *Handle) Free() {
	if h == nil || h.ptr == nil {
		return
	}
	C.macwin_releaseWindow(h.ptr)
	h.ptr = nil
}

// SetAccessoryPolicy switches the process to NSApplicationActivationPolicyAccessory:
// no Dock icon, no menu bar, but can show NSWindows and take focus.
// Overrides Gio's hardcoded Regular policy.
func SetAccessoryPolicy() {
	C.macwin_setAccessoryPolicy()
}
