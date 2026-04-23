//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa

void* spike_findFirstWindow(void);
long  spike_windowCount(void);
void  spike_orderOut(void *win);
void  spike_makeKeyAndOrderFront(void *win);
void  spike_setAccessoryPolicy(void);
void  spike_releaseWindow(void *win);
*/
import "C"

import "unsafe"

func findFirstWindow() unsafe.Pointer { return C.spike_findFirstWindow() }
func windowCount() int                { return int(C.spike_windowCount()) }
func orderOut(p unsafe.Pointer)       { C.spike_orderOut(p) }
func makeKeyAndOrderFront(p unsafe.Pointer) {
	C.spike_makeKeyAndOrderFront(p)
}
func setAccessoryPolicy()         { C.spike_setAccessoryPolicy() }
func releaseWindow(p unsafe.Pointer) { C.spike_releaseWindow(p) }
