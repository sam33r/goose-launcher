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
LAUNCHER_CMD="goose-launcher --bind alt-enter:print-query --bind tab:replace-query --bind=enter:replace-query+print-query --bind=ctrl-u:page-up --bind=ctrl-d:page-down --bind=ctrl-alt-u:pos(1) --bind=ctrl-alt-d:pos(-1) --no-sort --height=100 --layout=reverse"
```

## Command-Line Options

```
-e, --exact           Exact match mode (default: true)
--fuzzy               Fuzzy match mode (overrides --exact)
--no-sort             Preserve input order (default: true)
--height=N            Window height percentage (default: 100)
--layout=STYLE        Layout style: default|reverse
--bind=KEY:ACTION     Custom key binding (can be specified multiple times)
--interactive         Interactive mode (continuous stdin)
```

## Key Bindings

**Default bindings:**

- `↑` / `↓` - Navigate up/down
- `Ctrl+J` / `Ctrl+K` - Navigate down/up
- `Enter` - Select item
- `Shift+Enter` - Output current search query and exit
- `ESC` - Cancel (exit code 1)
- `Cmd+Q` - Quit application

**Custom bindings** (via --bind flag):

- `alt-enter:print-query` - Output typed query instead of selection
- `tab:replace-query` - Replace search with selected item
- `ctrl-u:page-up` - Scroll up one page
- `ctrl-d:page-down` - Scroll down one page

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
