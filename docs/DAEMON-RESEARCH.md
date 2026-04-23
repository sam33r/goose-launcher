# Daemon-mode research

Tracked: 2026-04-22. Goal: eliminate the ~200 ms cold-start by keeping a
launcher process resident and showing/hiding the window on demand. The
existing fzf-shaped contract (stdin items, stdout selection, exit 1 on cancel)
must be preserved exactly so Goose's `LAUNCHER_CMD` integration is unchanged.

## TL;DR

**There are three viable paths**, ranked by complexity:

| Path | Approach | Est. saving | Cost |
|---|---|---|---|
| **A. Destroy + recreate** | Single `app.Main()`; new `app.Window` per summon, `ActionClose` to dismiss. Public Gio APIs only. | ~70 ms (saves dyld + Go runtime + font load) | Low. Maybe ~130 ms remains because we still pay Gio per-window init each show. |
| **B. cgo hide/show shim** | One persistent `app.Window`; we add a tiny cgo shim that calls `[NSWindow orderOut:]` and `[NSWindow makeKeyAndOrderFront:]` directly (Gio doesn't expose these). | ~170 ms (saves nearly everything per show) | Moderate. ~30–50 lines of cgo, fragile across Gio versions, and we're touching a window Gio thinks it owns. |
| **C. Fork Gio** | Add a public `Hidden` window mode and `Accessory` activation policy upstream-of-our-fork. | ~170 ms, plus clean UX. | High. Maintain a fork or land a PR; long timeline. |

**Showstopper for all three:** Gio v0.9.0 hardcodes
`[NSApp setActivationPolicy:NSApplicationActivationPolicyRegular]` at
`os_macos.m:421`. Without intervention every summon flashes a Dock icon and
steals menubar focus. Mitigations exist (LSUIElement plist in an `.app`
bundle, post-init `setActivationPolicy:Accessory` via cgo) but none is
fully clean without modifying Gio.

**Recommended next step:** prototype path A first. It's the cheapest way to
verify (1) Gio actually allows sequential windows in a single `app.Main()`
without hangs, (2) the activation-policy/dock-icon UX is acceptable, (3) the
IPC + client architecture is sound. If A's measured saving (likely 70–100 ms)
isn't enough to justify the complexity, revisit B once the architecture is
proven. **Don't start with B**: combining "first time anyone has done a Gio
daemon on macOS" with "cgo around Gio's window" is two unknowns at once.

## Why the previous attempt likely hung

The repo's CLAUDE.md mentions a `TEST-DAEMON.md` postmortem (the file is no
longer in tree). Given Gio's actual architecture, the most plausible failures:

1. **Calling `app.Main()` more than once.** `gio_main` (`os_macos.m:426-452`)
   does `[NSApp run]` which never returns; `[NSApp stop]` is not called
   anywhere in Gio. Calling it twice would re-`close()` an already-closed
   `launched` channel (`os_macos.go:379,997`) and panic.
2. **Running `window.Run()` on the main goroutine.** `osMain`
   (`os_macos.go:1065-1067`) panics if not on main; `window.Run` on main
   would deadlock the cgo bridge before the panic fires.
3. **Reusing a destroyed `app.Window` value.** `DestroyEvent`
   (`window.go:638-651`) sets `w.driver = nil`. Subsequent `Event()` calls on
   that struct may behave unexpectedly. The safe pattern is to throw the
   struct away and instantiate a fresh `app.Window`.
4. **`runOnMain` deadlock.** `newWindow` (`os_macos.go:1003,1031`) marshals
   onto the main thread and waits. If the main goroutine is blocked
   somewhere (e.g. on a channel from the worker that's also blocked), the
   whole process hangs.

Mitigation pattern for path A: keep one main goroutine running `app.Main()`,
spawn one worker goroutine that loops `accept(socket) → newWindow → run →
close`, never touch the main goroutine from worker logic.

## 1. IPC mechanism

| Option | Latency | Streaming | Complexity | macOS gotchas |
|---|---|---|---|---|
| **Unix domain socket** | ~10–50 µs | full duplex | low — `net.Listen("unix", path)` | `SUN_PATH_LEN` = 104 bytes. Stale socket files survive crashes — unlink on startup. fs perms enforce single-user (`mode 0600`). |
| Named pipe (FIFO) | similar | half duplex per FIFO | moderate (need pair) | One direction per FIFO; need two. No SO_PEERCRED on macOS. |
| macOS XPC | ~µs | message-based, bidirectional | high — needs Mach service registration, codesigning recommended | Idiomatic only via cgo wrapper around libxpc. Worth it only if we want sandboxing. |
| Shared memory + sem | <1 µs | manual framing | very high | macOS POSIX semaphores fragile across crashes. Overkill. |

**Pick UDS.** It's what `tmux`, `gpg-agent`, `fish` universal-vars, and most
macOS userspace daemons use. Wire format: `uvarint(length) + json`. One
in-flight request (the launcher window is modal anyway). Frame messages:

```
client → daemon:  RequestEnvelope { flags, items[] }
daemon → client:  ResponseEnvelope { selection, exit_code }
```

Goose pipes up to ~10 MB of items in extreme cases. UDS handles that fine;
the daemon must read into memory before showing the window — same as today.

## 2. macOS application activation

Three relevant `NSApplicationActivationPolicy` values:

| Policy | Dock icon | Menu bar | Can show NSWindow | Can take key focus |
|---|---|---|---|---|
| **Regular** (Gio's default) | yes | yes | yes | yes |
| **Accessory** | no | only when active | yes | yes |
| **Prohibited** | no | no | no | no |

What we need: **Accessory**. Alfred, Raycast, Spotlight, Bartender, and
1Password's quick-access window all run as Accessory.

Two paths to Accessory:

1. **`Info.plist` `LSUIElement = 1`** in an `.app` bundle. Set before Cocoa
   starts. Standard for menu-bar apps. Requires distributing as
   `Goose Launcher.app/Contents/MacOS/goose-launcher` rather than a bare
   binary. Ref: [Apple LSUIElement][lsui].
2. **Runtime call to `setActivationPolicy:Accessory`** *before* a window is
   shown. Apple's [`NSApplication.activationPolicy(_:)`][nsapp-policy] docs
   note transitions toward more-restrictive (Regular → Accessory) are not
   guaranteed once the dock icon has appeared. Spotlight-class apps are
   bundled with `LSUIElement` for this reason.

**Gio v0.9.0 issue:** `os_macos.m:421` calls
`setActivationPolicy:NSApplicationActivationPolicyRegular` *unconditionally*
inside `applicationDidFinishLaunching` — which fires after `[NSApp run]`
starts, i.e. inside `app.Main()`. Even if our process is bundled as
LSUIElement, Gio overrides it the moment `app.Main()` is called. Options:

- **Maintain a one-line fork**: skip the line, or gate it on an env var.
- **cgo shim that calls `setActivationPolicy:Accessory` *after* `app.Main()`
  starts**: race-prone but probably works because the policy change takes
  effect synchronously on the main thread.
- **Upstream a `WithActivationPolicy(...)` option** to Gio. Right thing to
  do but adds delay.

A bare CLI binary in `/usr/local/bin` *can* show a focused NSWindow with no
bundle (Gio already proves this). Bundling is only required for the policy
plist; the focus-grab works either way (`raiseWindow` at `os_macos.go:283`
does `activateIgnoringOtherApps:YES` + `makeKeyAndOrderFront:`).

[lsui]: https://developer.apple.com/documentation/bundleresources/information_property_list/lsuielement
[nsapp-policy]: https://developer.apple.com/documentation/appkit/nsapplication/activationpolicy

## 3. Hide / show: what Gio actually exposes

This is the part that determines whether path A or path B is feasible.

**Show**: `w.Perform(system.ActionRaise)` — calls `raiseWindow`
(`os_macos.go:279-288`) → `activateWithOptions:` + `makeKeyAndOrderFront:`.
Public API, works.

**Hide**: there is no clean API. Available approaches:

- `w.Option(app.Minimized.Option())` — calls `hideWindow`
  (`os_macos.go:216-221`) → `[window miniaturize:]`. **Animates to the Dock**;
  visible to the user. Wrong UX.
- `w.Perform(system.ActionClose)` — calls `closeWindow`
  (`os_macos.go:170-175`) → `performClose:` → `windowWillClose` →
  `gio_onDestroy` → `DestroyEvent` → `w.driver = nil`. **Destroys** the
  window; cannot be reopened on the same `Window` value.

`[NSWindow orderOut:]` (the right primitive — instant, invisible, no Dock
animation, window not destroyed) is **not exposed by Gio v0.9.0** anywhere.
Verified by grep across `app/os_macos.go` and `app/os_macos.m`.

**Implication for path A:** "hide" = `ActionClose` = destroy. Each summon
constructs a brand-new `app.Window`, runs to first frame (~110 ms of Gio's
160 ms first-layout cost — not amortized across summons because the GPU
context is per-window), then destroys. Saves only dyld + Go runtime + font
load.

**Implication for path B:** add a cgo file that calls `orderOut` /
`makeKeyAndOrderFront` directly on the `NSWindow*`. We need to get the
window pointer out of Gio — `windowForView` (`os_macos.go`) returns the
`NSWindow*` from a `gioView`, but that's package-internal. We'd duplicate
that helper in our cgo file using the view pointer Gio gives us via
`app.Window.Driver()` (if exposed) or by intercepting at the FrameEvent.
**Not trivial** — requires reverse-engineering Gio's internals to get the
right window handle.

## 4. Lifecycle / autostart

Survey of similar tools on macOS:

- **skhd** ([install logic][skhd]) ships `bin/skhd --install-service` which
  writes `~/Library/LaunchAgents/com.koekeishiya.skhd.plist` with
  `RunAtLoad=true`, `KeepAlive=true`. Headless — no windows.
- **yabai** — same pattern, also headless.
- **karabiner-elements** — system-wide LaunchDaemon + user LaunchAgent for
  the menu-bar UI (`LSUIElement` `.app`).
- **Alfred / Raycast** — bundled `LSUIElement` `.app`s started via Login
  Items API; main process stays resident.

For us: **LaunchAgent with socket activation**. The kernel-level handoff
means the daemon doesn't even start until the first time the user presses
the keybinding. Plist sketch:

```xml
<key>Label</key><string>dev.goose.launcher</string>
<key>ProgramArguments</key>
  <array><string>/usr/local/bin/goose-launcher-daemon</string></array>
<key>RunAtLoad</key><false/>
<key>KeepAlive</key><dict><key>SuccessfulExit</key><false/></dict>
<key>Sockets</key>
  <dict><key>Listener</key>
    <dict><key>SockPathName</key>
      <string>/Users/USER/Library/Caches/goose-launcher.sock</string>
      <key>SockPathMode</key><integer>384</integer></dict></dict>
```

Install with:
```sh
launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/dev.goose.launcher.plist
```

The daemon retrieves the listening fd via `launch_activate_socket("Listener", ...)`
([Apple docs][launchd-socket]). Refs: [`launchd.plist(5)`][launchd-plist] and
the [Daemons and Services Programming Guide][daemons-guide].

[skhd]: https://github.com/koekeishiya/skhd/blob/master/src/manifest.c
[launchd-socket]: https://developer.apple.com/documentation/xpc/1505523-launch_activate_socket
[launchd-plist]: https://www.manpagez.com/man/5/launchd.plist/
[daemons-guide]: https://developer.apple.com/library/archive/documentation/MacOSX/Conceptual/BPSystemStartup/

## 5. Single-instance enforcement

- **launchd handles it natively** if we use the LaunchAgent path — refuses
  to start a second instance with the same `Label`. **Recommended.**
- Otherwise: `flock(LOCK_EX | LOCK_NB)` on a pidfile in
  `~/Library/Caches/goose-launcher/pid`. POSIX advisory lock auto-releases
  on process exit (kernel-managed) — no stale-PID problem. Wraps via
  `golang.org/x/sys/unix.Flock`.
- Bind-and-fail on the UDS path also works but leaves stale sockets after
  SIGKILL. Mitigation: `if dial(path) fails: unlink + listen`.

Avoid pure PID-file checks (`kill -0`) — race-prone.

## 6. Existing prior art

- **fzf `--listen`** ([server.go][fzf-server]): fzf already has a long-running
  mode that exposes an HTTP control API on a TCP port. Items are still piped
  via stdin at startup; subsequent `change-query`, `reload`, etc. are HTTP
  POSTs. Different shape (no per-request items list) but proves the
  resident-process pattern.
- **choose / choose-gui** (Rust): no daemon mode; small + skia-less so
  cold-start is acceptable.
- **rofi**: Linux X11; per-invocation, not daemonized. Plugins wrap.
- **Alfred / Raycast**: closed-source; Activity Monitor confirms a single
  resident process per user, `LSUIElement=1`, no Dock icon, started by
  Login Item.
- **Go-based GUI launcher with daemon mode**: none mature. Would be novel
  territory in pure-Go land.

[fzf-server]: https://github.com/junegunn/fzf/blob/master/src/server.go

## Recommended path A architecture (concrete)

```
~/Library/LaunchAgents/dev.goose.launcher.plist  (Sockets entry, no RunAtLoad)
       │
       ▼ launchd hands fd on first connect
Goose Launcher.app/Contents/MacOS/goose-launcher-daemon  (LSUIElement=1)
       │  socket loop on worker goroutine:
       │    accept connection
       │    read RequestEnvelope (flags + items)
       │    create app.Window, configure, run
       │    on selection or cancel: write ResponseEnvelope, ActionClose
       │  app.Main() runs forever on main goroutine
       ▼
/usr/local/bin/goose-launcher  (CLI client, ~2 MB, no Gio)
       │  dial UDS, send request, block on reply
       └─► stdout = selected line, exit code 0/1
```

Compatibility: Goose's `LAUNCHER_CMD` continues to point at
`/usr/local/bin/goose-launcher`. The user-facing contract is unchanged.

## Validation experiments to run before committing

The whole architecture rests on assumptions that Gio v0.9.0's source supports
but no one has demonstrated end-to-end on macOS. Run these in order:

1. **Sequential `app.Window` instances under one `app.Main()`.** Build a
   smallest-possible test that creates a window, processes one frame, calls
   `ActionClose`, waits for `DestroyEvent`, then creates a new `app.Window`
   and runs another frame. Verify (a) it doesn't hang, (b) the second window
   actually appears, (c) keyboard focus works, (d) measure the per-window
   setup cost vs the cold-start baseline.
2. **Activation policy override**: try setting `LSUIElement=1` in an `.app`
   bundle and check whether Gio's `setActivationPolicy:Regular` overrides
   it. If yes, write the cgo shim to call `setActivationPolicy:Accessory`
   after `app.Main()` starts and confirm the dock icon disappears.
3. **Socket activation handshake**: prototype `launch_activate_socket` from
   Go (cgo to `<launch.h>`). Several existing Go libraries do this — verify
   one works on current macOS.
4. **End-to-end timing**: with all three above working, time a full
   summon-to-first-frame cycle and compare against the current ~200 ms
   cold-start. If the saving is <70 ms, path A isn't worth the complexity.

## Risks / open questions

- **Gio not designed for window reuse.** `app.Window.Invalidate()` and the
  event loop assume one logical window lifetime. Validation experiment #1
  exists specifically to catch this.
- **Activation-policy race.** Even with the cgo shim, if Gio's
  `setActivationPolicy:Regular` runs first, the user may see a Dock icon
  flash. Acceptable? Probably — it's transient.
- **Daemon crash mid-request.** Client hangs on `Read`. Mitigation:
  client-side timeout (30 s), daemon sends keepalive frames during long
  sessions.
- **Big stdin payloads.** Goose can pipe 100k+ items (~10 MB). UDS handles
  it but daemon must drain into memory before showing the window — same as
  current single-shot binary. No streaming-render benefit.
- **Updating the daemon binary.** Once resident, a new release won't take
  effect until the user kills the daemon. Add `goose-launcher --restart`
  command and document.

## References

- Apple, [`NSApplication.activationPolicy`][nsapp-policy]
- Apple, [LSUIElement][lsui]
- Apple, [`launch_activate_socket`][launchd-socket]
- Apple, [Daemons and Services Programming Guide][daemons-guide]
- [`launchd.plist(5)`][launchd-plist]
- [fzf `--listen` server][fzf-server]
- [skhd install logic][skhd]
- Gio v0.9.0 source (under `/Users/samahuja/go/pkg/mod/gioui.org@v0.9.0/`):
  - `app/app.go:135` — `app.Main`
  - `app/os_macos.m:421` — hardcoded Regular activation policy
  - `app/os_macos.m:426-452` — `gio_main` / `[NSApp run]`
  - `app/os_macos.go:170-228` — `closeWindow` / `hideWindow` (miniaturize)
  - `app/os_macos.go:279-288` — `raiseWindow`
  - `app/os_macos.go:516-534` — `Perform` (only Center/Raise/Close on macOS)
  - `app/os_macos.go:1000-1032` — `newWindow`
  - `app/window.go:638-651` — DestroyEvent handling
