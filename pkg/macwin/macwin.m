// macwin: AppKit shim that lets us drive a Gio-managed NSWindow's
// visibility from outside Gio. See cmd/spike/ for the throwaway code that
// validated this approach (phase 0 spike, 2026-04-22) and
// docs/DAEMON-RESEARCH.md for the architecture rationale.
//
// All AppKit calls are dispatched onto the main thread because NSWindow
// methods aren't thread-safe. Gio's app.Main() owns the main thread, so
// dispatch_sync from a Go goroutine works.

#import <Cocoa/Cocoa.h>

// Forward declaration of the Go-exported callback. cgo emits the prototype
// in _cgo_export.h; that header is generated for the Go file in this package
// and is implicitly available to .m files compiled in the same package.
extern void macwinDidResignKey(void *win);

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

// macwin_setLauncherCollectionBehavior sets NSWindow.collectionBehavior so
// the window:
//   - Follows the current Space when summoned (MoveToActiveSpace) — fixes
//     the bug where macOS switches back to the original Space if the user
//     summons the launcher from a different one.
//   - Doesn't show in Cmd+Tab / window cycling and doesn't restore on
//     relaunch (Transient) — appropriate for an ephemeral launcher.
//   - Can appear over fullscreen apps (FullScreenAuxiliary) — otherwise
//     summoning from a fullscreen window does nothing visible.
//
// Same combination Spotlight/Alfred/Raycast use.
void macwin_setLauncherCollectionBehavior(void *win) {
    if (win == NULL) return;
    NSWindow *w = (__bridge NSWindow *)win;
    dispatch_sync(dispatch_get_main_queue(), ^{
        [w setCollectionBehavior:
            NSWindowCollectionBehaviorMoveToActiveSpace |
            NSWindowCollectionBehaviorTransient |
            NSWindowCollectionBehaviorFullScreenAuxiliary];
    });
}

// macwin_observeResignKey registers an NSNotificationCenter observer that
// fires when the given NSWindow loses key status. Used to dismiss the
// launcher when the user clicks another window/app — same effect as ESC.
//
// The block runs on the main thread (notifications post to the same thread
// the event arrived on, which for window key changes is always main). It
// suppresses the notification that fires when the daemon itself hides the
// window via [orderOut:] after a selection: at that point the window is
// no longer visible, so isVisible returns NO.
//
// Observer lifetime: tied to the NSWindow's. We never call removeObserver
// because the daemon holds the window for its entire lifetime.
void macwin_observeResignKey(void *win) {
    if (win == NULL) return;
    NSWindow *w = (__bridge NSWindow *)win;
    void *winKey = win; // captured by value for the Go callback lookup
    dispatch_sync(dispatch_get_main_queue(), ^{
        [[NSNotificationCenter defaultCenter]
            addObserverForName:NSWindowDidResignKeyNotification
                        object:w
                         queue:[NSOperationQueue mainQueue]
                    usingBlock:^(NSNotification *note) {
                        NSWindow *src = (NSWindow *)note.object;
                        if (![src isVisible]) return;
                        macwinDidResignKey(winKey);
                    }];
    });
}

void macwin_releaseWindow(void *win) {
    if (win != NULL) {
        CFRelease(win);
    }
}
