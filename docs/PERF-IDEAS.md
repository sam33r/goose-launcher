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

### 1. Lazy / parallel font loading
`NewWindow` calls `fontcache.GetFonts(...)` synchronously and registers Regular,
Bold, Italic before the first frame. Two angles:
- **Defer Bold + Italic.** Frame 0 only needs Regular (the prompt and item
  text). Bold/Italic only matter once a Pango-styled item is rendered. Register
  Regular eagerly; load the others on a goroutine and swap them in via
  `text.NewShaper` reload. Estimated saving: 30–40 ms of the 56 ms creation
  cost.
- **Parallel parse.** Three faces in parallel. Halves the parse cost on cold
  cache; near-free on warm. Cheap to write.

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

### 6. Skip `app.Maximized.Option()` if not needed
A maximized window may force the OS into more compositor work. If the launcher
should be modal/centered (more fzf-like), removing maximize might cut layout
cost.

### 7. Replace `material.Body1` with hand-written text widget
Material widgets do per-call theming work. The single visible frame's first
layout may benefit from a hand-written widget that bypasses Material's machinery.
Profiling required to confirm.

### 8. Subset JetBrains Mono glyphs
TTFs are ~250 KB each. A subset containing printable ASCII + box-drawing chars
is <30 KB. Faster parse, faster cache load. Edge case is non-ASCII items, which
fall back to system font (Gio handles).

## Recommended next steps (in order)

1. **#1 — Lazy/parallel font loading.** ~1 hour, expected 30+ ms win, no UX change.
2. **Profile the layout stage** (`go tool pprof`, Gio CPU profiler). Decides
   whether the 110 ms of layout work is text shaping, OS window creation, or
   widget tree. Without that data, #2/#5 are guesses.
3. **#4 — Daemon mode**, after research in `DAEMON-RESEARCH.md` lands.
   Largest payoff, highest implementation cost.
