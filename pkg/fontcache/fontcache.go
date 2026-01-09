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
	once    sync.Once
	err     error
}

// Global font cache instance
var globalCache = &Cache{}

// ParseFonts parses the font bytes and caches the results
// Subsequent calls return the cached faces
func (c *Cache) ParseFonts(regularBytes, boldBytes []byte) (regular, bold font.Face, err error) {
	c.once.Do(func() {
		// Parse regular font
		c.regular, c.err = opentype.Parse(regularBytes)
		if c.err != nil {
			c.err = fmt.Errorf("failed to parse regular font: %w", c.err)
			return
		}

		// Parse bold font
		c.bold, c.err = opentype.Parse(boldBytes)
		if c.err != nil {
			c.err = fmt.Errorf("failed to parse bold font: %w", c.err)
			return
		}
	})

	if c.err != nil {
		return nil, nil, c.err
	}

	return c.regular, c.bold, nil
}

// GetFonts returns cached fonts or parses them if not cached
// This is a convenience wrapper around the global cache
func GetFonts(regularBytes, boldBytes []byte) (regular, bold font.Face, err error) {
	return globalCache.ParseFonts(regularBytes, boldBytes)
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
