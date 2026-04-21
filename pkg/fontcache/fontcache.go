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
	once    sync.Once
	err     error
}

// Global font cache instance
var globalCache = &Cache{}

// ParseFonts parses the font bytes and caches the results.
// Subsequent calls return the cached faces. italicBytes may be nil, in which
// case the italic face is also nil and callers must skip registering it.
func (c *Cache) ParseFonts(regularBytes, boldBytes, italicBytes []byte) (regular, bold, italic font.Face, err error) {
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
	})

	if c.err != nil {
		return nil, nil, nil, c.err
	}

	return c.regular, c.bold, c.italic, nil
}

// GetFonts returns cached fonts or parses them if not cached.
// italicBytes may be nil; the returned italic face will then also be nil.
func GetFonts(regularBytes, boldBytes, italicBytes []byte) (regular, bold, italic font.Face, err error) {
	return globalCache.ParseFonts(regularBytes, boldBytes, italicBytes)
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
