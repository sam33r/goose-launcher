// macwin: AppKit shim that lets us drive a Gio-managed NSWindow's
// visibility from outside Gio. See cmd/spike/ for the throwaway code that
// validated this approach (phase 0 spike, 2026-04-22) and
// docs/DAEMON-RESEARCH.md for the architecture rationale.
//
// All AppKit calls are dispatched onto the main thread because NSWindow
// methods aren't thread-safe. Gio's app.Main() owns the main thread, so
// dispatch_sync from a Go goroutine works.

#import <Cocoa/Cocoa.h>

// macwin_findFirstWindow returns a retained pointer to the first non-nil
// NSWindow in [NSApp windows]. Caller must macwin_releaseWindow when done.
// Returns NULL if no window exists yet (Gio hasn't created one).
//
// Why "first": the launcher daemon owns exactly one Gio window. NSApp may
// have additional internal windows (notably the menu); we filter by taking
// the first NSWindow whose title matches the one Gio set.
void* macwin_findWindowByTitle(const char *titleC) {
    __block void *result = NULL;
    NSString *want = [NSString stringWithUTF8String:titleC];
    dispatch_sync(dispatch_get_main_queue(), ^{
        for (NSWindow *w in [NSApp windows]) {
            if (w == nil) continue;
            if ([[w title] isEqualToString:want]) {
                result = (void *)CFBridgingRetain(w);
                break;
            }
        }
    });
    return result;
}

void macwin_orderOut(void *win) {
    if (win == NULL) return;
    NSWindow *w = (__bridge NSWindow *)win;
    dispatch_sync(dispatch_get_main_queue(), ^{
        [w orderOut:nil];
    });
}

void macwin_makeKeyAndOrderFront(void *win) {
    if (win == NULL) return;
    NSWindow *w = (__bridge NSWindow *)win;
    dispatch_sync(dispatch_get_main_queue(), ^{
        // activateIgnoringOtherApps wakes us if another app currently has
        // the menubar; without this the new window may appear behind the
        // active app's windows.
        [NSApp activateIgnoringOtherApps:YES];
        [w makeKeyAndOrderFront:nil];
    });
}

// macwin_setAccessoryPolicy switches the running app to
// NSApplicationActivationPolicyAccessory — no Dock icon, no menu bar by
// default, but can show NSWindows and take focus. Override Gio's hardcoded
// Regular policy.
//
// Apple docs note that transitions toward more-restrictive policies aren't
// guaranteed once the dock icon has appeared. In practice on modern macOS
// it works for menu-bar utilities like ours; the spike (phase 0) confirmed
// no dock icon appears in steady state.
void macwin_setAccessoryPolicy(void) {
    dispatch_sync(dispatch_get_main_queue(), ^{
        [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
    });
}

void macwin_releaseWindow(void *win) {
    if (win != NULL) {
        CFRelease(win);
    }
}
