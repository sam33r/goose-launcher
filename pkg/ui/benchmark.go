package ui

import (
	"time"
)

// StartupMetrics tracks window startup performance
type StartupMetrics struct {
	WindowCreationStart time.Time
	WindowCreationEnd   time.Time
	FirstFrameTime      time.Time
	FirstLayoutTime     time.Time
}

// BenchmarkMode enables startup timing collection
var BenchmarkMode = false

// GetStartupDuration returns the time from window creation to first frame
func (m *StartupMetrics) GetStartupDuration() time.Duration {
	if m.FirstFrameTime.IsZero() {
		return 0
	}
	return m.FirstFrameTime.Sub(m.WindowCreationStart)
}

// GetCreationDuration returns window object creation time
func (m *StartupMetrics) GetCreationDuration() time.Duration {
	if m.WindowCreationEnd.IsZero() {
		return 0
	}
	return m.WindowCreationEnd.Sub(m.WindowCreationStart)
}

// GetTimeToFirstLayout returns time to first layout operation
func (m *StartupMetrics) GetTimeToFirstLayout() time.Duration {
	if m.FirstLayoutTime.IsZero() {
		return 0
	}
	return m.FirstLayoutTime.Sub(m.WindowCreationStart)
}
