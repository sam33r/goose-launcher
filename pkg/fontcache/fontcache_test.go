package fontcache

import (
	"testing"
)

func TestFontCacheParsing(t *testing.T) {
	// Reset cache for clean test
	Reset()

	// Mock font bytes (minimal valid TTF headers)
	mockRegular := make([]byte, 100)
	mockBold := make([]byte, 100)

	// First call should parse
	if IsCached() {
		t.Error("cache should be empty initially")
	}

	_, _, err := GetFonts(mockRegular, mockBold)
	// We expect an error because mock bytes aren't valid fonts
	if err == nil {
		t.Error("expected error with invalid font bytes")
	}
}

func TestFontCacheReuse(t *testing.T) {
	Reset()

	mockBytes := make([]byte, 100)

	// Parse once (will fail, but that's OK for this test)
	GetFonts(mockBytes, mockBytes)

	// Second call should return cached result (same error)
	_, _, err1 := GetFonts(mockBytes, mockBytes)
	_, _, err2 := GetFonts(mockBytes, mockBytes)

	if err1 == nil || err2 == nil {
		t.Error("expected errors with invalid fonts")
	}

	if err1.Error() != err2.Error() {
		t.Error("expected same error from cache")
	}
}

func TestReset(t *testing.T) {
	Reset()

	mockBytes := make([]byte, 100)

	// Parse once
	GetFonts(mockBytes, mockBytes)

	if !IsCached() {
		t.Error("cache should be populated after first parse")
	}

	// Reset
	Reset()

	if IsCached() {
		t.Error("cache should be empty after reset")
	}
}
