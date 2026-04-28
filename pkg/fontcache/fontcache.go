package fontcache

import (
	"fmt"
	"sync"

	"gioui.org/font"
	"gioui.org/font/opentype"
)

// Cache holds parsed font faces
type Cache struct {
	regular font.Face
	bold    font.Face
	italic  font.Face
	emoji   font.Face
	once    sync.Once
	err     error
}

// Global font cache instance
var globalCache = &Cache{}

// ParseFonts parses the font bytes and caches the results. Subsequent calls
// return the cached faces. italicBytes and emojiBytes may be nil; the
// corresponding faces will also be nil and callers must skip registering them.
func (c *Cache) ParseFonts(regularBytes, boldBytes, italicBytes, emojiBytes []byte) (regular, bold, italic, emoji font.Face, err error) {
	c.once.Do(func() {
		c.regular, c.err = opentype.Parse(regularBytes)
		if c.err != nil {
			c.err = fmt.Errorf("failed to parse regular font: %w", c.err)
			return
		}
		c.bold, c.err = opentype.Parse(boldBytes)
		if c.err != nil {
			c.err = fmt.Errorf("failed to parse bold font: %w", c.err)
			return
		}
		if italicBytes != nil {
			c.italic, c.err = opentype.Parse(italicBytes)
			if c.err != nil {
				c.err = fmt.Errorf("failed to parse italic font: %w", c.err)
				return
			}
		}
		if emojiBytes != nil {
			c.emoji, c.err = opentype.Parse(emojiBytes)
			if c.err != nil {
				c.err = fmt.Errorf("failed to parse emoji font: %w", c.err)
				return
			}
		}
	})

	if c.err != nil {
		return nil, nil, nil, nil, c.err
	}

	return c.regular, c.bold, c.italic, c.emoji, nil
}

// GetFonts returns cached fonts or parses them if not cached.
// italicBytes and emojiBytes may be nil; the corresponding returned faces
// will then also be nil.
func GetFonts(regularBytes, boldBytes, italicBytes, emojiBytes []byte) (regular, bold, italic, emoji font.Face, err error) {
	return globalCache.ParseFonts(regularBytes, boldBytes, italicBytes, emojiBytes)
}

// Reset clears the cache (useful for testing)
func Reset() {
	globalCache = &Cache{}
}

// IsCached returns true if fonts are already cached
func IsCached() bool {
	// If once.Do has been called, fonts are cached
	// We can't directly check sync.Once state, so we check if faces are set
	return globalCache.regular != nil
}
