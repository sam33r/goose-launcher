// +build darwin
//
// Minimal cgo shim for the daemon spike. We deliberately keep this tiny —
// the goal is to learn whether driving an arbitrary NSWindow's visibility
// from outside Gio works. NOT meant for production.

#import <Cocoa/Cocoa.h>

// Returns the first non-nil NSWindow in [NSApp windows], retained.
// Caller must CFRelease when done. Returns NULL if no window exists yet.
void* spike_findFirstWindow(void) {
    __block void *result = NULL;
    dispatch_sync(dispatch_get_main_queue(), ^{
        NSArray<NSWindow*> *wins = [NSApp windows];
        for (NSWindow *w in wins) {
            if (w != nil) {
                result = (void *)CFBridgingRetain(w);
                break;
            }
        }
    });
    return result;
}

// Returns the count of windows currently owned by NSApp. Useful sanity check.
long spike_windowCount(void) {
    __block long n = 0;
    dispatch_sync(dispatch_get_main_queue(), ^{
        n = (long)[[NSApp windows] count];
    });
    return n;
}

void spike_orderOut(void *win) {
    NSWindow *w = (__bridge NSWindow *)win;
    dispatch_sync(dispatch_get_main_queue(), ^{
        [w orderOut:nil];
    });
}

void spike_makeKeyAndOrderFront(void *win) {
    NSWindow *w = (__bridge NSWindow *)win;
    dispatch_sync(dispatch_get_main_queue(), ^{
        [NSApp activateIgnoringOtherApps:YES];
        [w makeKeyAndOrderFront:nil];
    });
}

// Try to switch activation policy to Accessory (no Dock icon). May or may
// not "stick" after Gio set it to Regular — this is what we want to learn.
void spike_setAccessoryPolicy(void) {
    dispatch_sync(dispatch_get_main_queue(), ^{
        [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
    });
}

void spike_releaseWindow(void *win) {
    if (win != NULL) {
        CFRelease(win);
    }
}
