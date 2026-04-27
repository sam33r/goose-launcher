# Goose Launcher Usage Guide

## Installation

### From Source

```bash
git clone https://github.com/sam33r/goose-launcher
cd goose-launcher
./install.sh
```

### Manual Build

```bash
make build-macos
sudo cp goose-launcher /usr/local/bin/
```

## Integration with Goose

Add to `~/.config/goose`:

```bash
# Use native launcher instead of fzf
LAUNCHER_CMD="goose-launcher --no-sort --height=100 --layout=reverse"
```

> **Migration note:** the `--bind KEY:ACTION` flag was removed (the bindings
> were silently ignored anyway). If your `LAUNCHER_CMD` still passes `--bind`,
> the launcher will exit with `flag provided but not defined: -bind`. The
> behaviors that those binds described — `tab:replace-query`,
> `ctrl-u:page-up`, `ctrl-d:page-down` — are now hardcoded as default
> bindings, so dropping the `--bind` flags retains the behavior people
> actually got.

## Command-Line Options

```
-e, --exact           Exact match mode (default: true)
--fuzzy               Fuzzy match mode (overrides --exact)
--rank                Rank results by match quality (default: false)
--no-sort             Filter only; preserve input order (default; kept for compatibility)
--markup=FORMAT       Parse stdin markup; currently only 'pango' is supported
--height=N            Window height percentage (default: 100)
--layout=STYLE        Layout style: default|reverse
```

The launcher streams stdin: the window appears as soon as you invoke the
binary (no waiting for the producer to close stdin), and items flow in as
they arrive. Selecting or pressing ESC closes the connection — the upstream
producer (e.g. `find /`) gets SIGPIPE on its next write and terminates.

## Key Bindings

All bindings are hardcoded; the launcher does not currently support
user-configurable key bindings.

- `↑` / `↓` — Navigate up / down
- `Ctrl+K` / `Ctrl+J` — Navigate up / down (vim-style)
- `Ctrl+U` / `Ctrl+D` — Page up / page down (jumps by visible-row count)
- `Enter` — Select highlighted item; if no matches, output the typed query
- `Shift+Enter` — Output the typed query (regardless of selection)
- `Tab` — Replace search input with the selected item's raw text
- `ESC` — Cancel
- `Cmd+Q` — Quit
- Mouse wheel — Scrolls the list AND moves the highlighted row by the same
  amount, so the cursor's position within the visible window stays stable.
- Mouse click on an item — Select it.

## Examples

### Basic Usage

```bash
echo -e "Item 1\nItem 2\nItem 3" | goose-launcher
```

### With Fuzzy Search

```bash
find . -type f | goose-launcher
# Type "mk" to match "Makefile"
```

### Exact Mode

```bash
ls | goose-launcher -e
# Only matches exact substrings
```

## Markup

With `--markup=pango`, each input line may contain a small subset of Pango markup:

- `<b>…</b>` — bold (rendered)
- `<i>…</i>` — italic (rendered)
- `<span foreground="#RRGGBB">…</span>` — foreground color (rendered). `fg` is an alias for `foreground`. Named colors (`red`, `green`, `blue`, `yellow`, `cyan`, `magenta`, `white`, `black`, plus `light*`/`dark*` variants) are accepted.
- `<u>…</u>` — parsed but not yet rendered
- `<span background="…">…</span>` — parsed but not yet rendered

Matching and selection use the plain (markup-stripped) text, so markup never leaks to stdout. Malformed markup falls back to literal text for that line.

```bash
printf '<b>ERROR</b>    . connection refused\n<span foreground="#4ec9b0">OK</span>       . ready\n' \
  | goose-launcher --markup=pango
```

## Troubleshooting

**Window doesn't appear:**
- Check that you're running from a graphical environment
- Try: `echo "test" | goose-launcher`

**No items shown:**
- Verify stdin is providing data: `echo "test" | goose-launcher`
- Check for errors: `goose-launcher 2>&1`

**Performance issues:**
- For >100k items, consider filtering before piping to launcher
- Disable animations (future feature)

## Development

Run tests:
```bash
make test
```

Build for development:
```bash
make build
./goose-launcher
```
