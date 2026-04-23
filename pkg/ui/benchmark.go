package ui

import (
	"time"
)

// StartupMetrics tracks window startup performance.
//
// Timeline (each later field happens-after each earlier field):
//
//	LaunchStart          parent harness called exec() (set via LAUNCH_START_NS env)
//	ProcessStart         first instant inside our binary (after Go runtime init)
//	StdinReadStart/End   bracket os.Stdin -> []input.Item
//	WindowCreationStart  NewWindow() entry (theme + font setup)
//	WindowCreationEnd    NewWindow() return
//	FirstLayoutTime      first layout() call
//	FirstFrameTime       first frame submitted to the OS compositor
//
// LaunchStart and ProcessStart are zero when the parent harness didn't set
// LAUNCH_START_NS — fall back gracefully and skip the corresponding deltas.
type StartupMetrics struct {
	LaunchStart         time.Time
	ProcessStart        time.Time
	StdinReadStart      time.Time
	StdinReadEnd        time.Time
	WindowCreationStart time.Time
	WindowCreationEnd   time.Time
	FirstLayoutTime     time.Time
	FirstFrameTime      time.Time
}

// BenchmarkMode enables startup timing collection
var BenchmarkMode = false

// GetStartupDuration returns the time from window creation to first frame.
// This is the legacy "startup" number reported by the BENCHMARK line — kept
// for continuity with older recorded baselines.
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

// GetPrelaunchDuration returns the time spent in dyld + Go runtime init
// (LaunchStart -> ProcessStart). Zero when the parent didn't set
// LAUNCH_START_NS.
func (m *StartupMetrics) GetPrelaunchDuration() time.Duration {
	if m.LaunchStart.IsZero() || m.ProcessStart.IsZero() {
		return 0
	}
	return m.ProcessStart.Sub(m.LaunchStart)
}

// GetStdinReadDuration returns the time spent draining stdin into items.
func (m *StartupMetrics) GetStdinReadDuration() time.Duration {
	if m.StdinReadStart.IsZero() || m.StdinReadEnd.IsZero() {
		return 0
	}
	return m.StdinReadEnd.Sub(m.StdinReadStart)
}

// GetTotalDuration returns user-perceived launch latency: from the parent's
// pre-exec timestamp to first frame. Falls back to ProcessStart -> first frame
// when LaunchStart isn't set, so this number is always meaningful.
func (m *StartupMetrics) GetTotalDuration() time.Duration {
	if m.FirstFrameTime.IsZero() {
		return 0
	}
	start := m.LaunchStart
	if start.IsZero() {
		start = m.ProcessStart
	}
	if start.IsZero() {
		return 0
	}
	return m.FirstFrameTime.Sub(start)
}
