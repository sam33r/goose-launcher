# Launch-latency ideas

Tracked: 2026-04-22. Numbers below are Apple M4 Pro, warm runs, 100 items,
30-iteration medians from `go run ./cmd/benchmark-startup`.

## Current breakdown

| Stage | Median | Notes |
|---|---|---|
| `prelaunch` (dyld + Go runtime init) | 15 ms | Cold dyld can spike to ~290 ms on first invocation; warm cache is consistent. |
| `stdin` (read + parse) | <1 ms | Already optimized — `Item.Init()` is the dominant cost and amortizes. |
| `creation` (`NewWindow`: theme + font) | 56 ms | Three JetBrains Mono faces parsed serially. |
| `layout` (NewWindow → first layout) | 168 ms | Gio cold-path setup; partly OS-side window creation. |
| `startup` (NewWindow → first frame) | 179 ms | Includes layout. |
| **total** wall-clock (hyperfine) | **~223 ms** | |

End-to-end scaling is gentle: 100 → 100k items adds only ~28 ms because the
matcher work was already cut in the [perf commit][perf] (1ed36bf).

[perf]: https://github.com/sam33r/goose-launcher/commit/1ed36bf

## Tier 1 — meaningful, modest risk

### 1. Lazy / parallel font loading — **TRIED 2026-04-22, no end-to-end win**
`NewWindow` calls `fontcache.GetFonts(...)` synchronously and registers Regular,
Bold, Italic before the first frame. Original hypothesis: defer Bold + Italic
to a goroutine, parallel-parse all three.

**What we measured:** `opentype.Parse` is only **~500 µs per face** (1.5 ms
serial total). The 56 ms "creation time" is almost entirely in
`text.NewShaper`'s call to `fontMap.UseSystemFonts(cacheDir)` — scanning
every font in `/System/Library/Fonts` and `~/Library/Fonts`.

Adding `text.NoSystemFonts()` collapses creation from ~65 ms to ~2 ms — but
the same ~63 ms reappears in **first-layout time** because Cocoa's CTFont
subsystem does the system-font work itself when the first frame is painted.
End-to-end startup is unchanged (256 ms median both ways). Parallel parse
alone saves <1 ms (parse isn't the bottleneck).

**Conclusion:** can't sidestep this from the Go side. The OS-managed font
subsystem owns the cost. Only daemon mode (#4) skips it on subsequent
invocations.

### 2. Pre-warm Gio's first-frame on a goroutine
Between `NewWindow` returning and the first `FrameEvent` arriving, the main
thread is idle waiting on the OS to deliver a Frame. Anything we can compute
speculatively (text shaping for `>` prompt, item-count label, the first ~30
visible rows) saves time on the critical path. Requires understanding Gio's
scheduler — moderate complexity.

### 3. Cache parsed `font.Face` on disk, not just raw TTF
`pkg/fontcache` today caches parsed bytes; if we serialized the post-parse
`opentype.Face`, we'd skip parsing entirely. Risky — the format isn't stable
across Gio versions, but a version-stamped cache file works. Could drop
`creation` to <10 ms.

## Tier 2 — bigger payoff, bigger investment

### 4. Persistent daemon mode
Keep one launcher process around; show/hide the window on demand. New
invocations send a request via Unix socket. Eliminates dyld + runtime + font
load on every invocation — drops total launch from ~200 ms to <20 ms.

A previous attempt (referenced in CLAUDE.md but not present in git history)
hung on macOS. Worth re-investigating because the payoff dwarfs everything else.
See [`DAEMON-RESEARCH.md`](DAEMON-RESEARCH.md) for the deep dive.

### 5. Pre-baked first frame
Goose users always see the same initial UI: dark bg, `>` prompt, item count,
first ~30 rows of stdin. Two variants:
- Show the window with a static pre-baked image while Gio's real first layout
  runs in background. Roughly what macOS Spotlight does.
- Hand-write a "fast-path layout" that bypasses Material widget overhead for
  frame 0 and substitutes raw text rendering.
Significant complexity; only justified if #1–#4 don't get us there.

## Tier 3 — small but clean

### 6. Skip `app.Maximized.Option()` if not needed — **TRIED 2026-04-22, no win**
Hypothesis: a maximized window forces the OS into a resize+composite cycle
that adds latency. **What we measured:** swapping `app.Maximized.Option()`
for `app.Size(900, 600)` produced no measurable startup change (255 vs 256 ms
median). Revisit only if profiling identifies window resize as a hot path.

### 7. Replace `material.Body1` with hand-written text widget
Material widgets do per-call theming work. The single visible frame's first
layout may benefit from a hand-written widget that bypasses Material's machinery.
Profiling required to confirm.

### 8. Subset JetBrains Mono glyphs
TTFs are ~250 KB each. A subset containing printable ASCII + box-drawing chars
is <30 KB. Faster parse, faster cache load. Edge case is non-ASCII items, which
fall back to system font (Gio handles).

## Recommended next steps (in order)

After trying #1 and #6 with no measurable wall-clock win, the picture is
clearer: **almost all of the launch cost lives outside Go-controllable
code.** The breakdown is approximately:

- ~30 ms: dyld + Go runtime init (fixed)
- ~63 ms: macOS font subsystem init (Cocoa-side; tries to defer with
  NoSystemFonts but reappears at first paint)
- ~170 ms: OS window creation + first-frame composition (Cocoa + GPU)
- ~few ms: our actual code (theme, widget tree, stdin parse)

This means **incremental tuning won't reach a 10× improvement.** The viable
options for serious launch-latency reduction are now:

1. **Profile the 230 ms layout stage** (Instruments "Time Profiler" with
   `--launch`, or Gio's CPU profiler). Confirms exactly where the OS-side
   cost lives. Without this data we're guessing.
2. **#4 — Daemon mode** ([`DAEMON-RESEARCH.md`](DAEMON-RESEARCH.md)). The
   only path that skips the OS-owned font/window costs entirely on every
   invocation after the first. Validation experiments listed in that doc.
3. **#5 — Pre-baked first frame.** If daemon mode proves infeasible, this
   is the fallback for hiding the latency from the user (show stale image
   while real frame composes).

Tier 2/3 micro-optimizations (#3, #7, #8) remain available but each is
likely worth single-digit milliseconds — not worth pursuing until profiling
shows a specific hot path they'd address.
