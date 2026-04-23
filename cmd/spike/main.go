// Phase 0 daemon spike. Throwaway code: validates whether we can drive a
// Gio-managed NSWindow's visibility from outside Gio.
//
// What this program does:
//   - Opens a single Gio window (default size, undecorated).
//   - Locates its NSWindow via [NSApp windows] (no Gio internals touched).
//   - Cycles 60 times: orderOut → wait → makeKeyAndOrderFront → measure
//     time until next FrameEvent arrives.
//   - Prints per-cycle latency + summary stats.
//   - Logs key events to verify keyboard input works after re-show.
//
// Go/no-go for the daemon plan:
//   - All cycles complete without hang/crash.
//   - Show-to-frame latency stays well under 50 ms median.
//   - Keys work after re-show (visually verify by typing).
//
// Run from repo root:
//   go run ./cmd/spike
//
// Type characters in the window after each show; the spike prints them.
// Press q to quit (or Cmd+Q).
package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"os"
	"sort"
	"time"
	"unsafe"

	"gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"
)

const (
	cycles      = 60
	hideHold    = 100 * time.Millisecond
	maxShowWait = 500 * time.Millisecond
)

func main() {
	go runWindow()
	app.Main()
}

type frameSignal struct{}
type readySignal struct{}

var (
	frameCh = make(chan frameSignal, 64)
	readyCh = make(chan readySignal, 1)
	quitCh  = make(chan struct{})

	// Set by runWindow once the *app.Window exists, read by runDriver to
	// kick the event loop after we makeKeyAndOrderFront (otherwise macOS
	// won't schedule a redraw on a previously hidden window and Gio's
	// event loop stays parked in Event()).
	gioWin *app.Window
)

func runWindow() {
	w := new(app.Window)
	gioWin = w
	w.Option(
		app.Title("Daemon Spike"),
		app.Decorated(false),
		app.Size(unit.Dp(600), unit.Dp(200)),
	)
	theme := material.NewTheme()
	theme.Shaper = text.NewShaper(text.NoSystemFonts())
	theme.Bg = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	theme.Fg = color.NRGBA{R: 220, G: 220, B: 220, A: 255}

	go runDriver()

	var ops op.Ops
	cycle := 0
	notifiedReady := false
	for {
		e := w.Event()
		switch e := e.(type) {
		case app.DestroyEvent:
			log.Printf("destroy: %v", e.Err)
			os.Exit(0)
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			// Register a key listener over the entire window area so we
			// see key events regardless of focus. (No editor in this spike.)
			area := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
			event.Op(gtx.Ops, w)
			area.Pop()
			for {
				ev, ok := gtx.Event(key.Filter{})
				if !ok {
					break
				}
				if ke, ok := ev.(key.Event); ok && ke.State == key.Press {
					log.Printf("key: %s (mods=%s)", ke.Name, ke.Modifiers)
					if ke.Name == "Q" {
						close(quitCh)
					}
				}
			}

			paint.Fill(gtx.Ops, theme.Bg)
			label := material.H6(theme, fmt.Sprintf("Spike cycle %d / %d", cycle, cycles))
			label.Color = theme.Fg
			layout.Center.Layout(gtx, label.Layout)
			e.Frame(&ops)

			cycle++

			if !notifiedReady {
				notifiedReady = true
				readyCh <- readySignal{}
			}

			// Non-blocking signal so the driver can wait for the next frame.
			select {
			case frameCh <- frameSignal{}:
			default:
			}
		}
	}
}

func runDriver() {
	// Wait for the first frame so we know the NSWindow exists.
	<-readyCh
	log.Printf("window ready; NSApp windows count = %d", windowCount())

	winPtr := findFirstWindow()
	if winPtr == nil {
		log.Fatal("FAIL: spike_findFirstWindow returned NULL")
	}
	defer releaseWindow(winPtr)
	log.Printf("got NSWindow ptr = %p", winPtr)

	// Try to switch to Accessory policy. Whether this hides the dock
	// icon (after Gio set Regular) is one of the things we want to learn.
	setAccessoryPolicy()
	log.Printf("set Accessory policy (dock icon should disappear if it works)")

	// Cycle hide/show.
	latencies := make([]time.Duration, 0, cycles)
	for i := 1; i <= cycles; i++ {
		select {
		case <-quitCh:
			log.Printf("quit signal received after %d cycles", i-1)
			summarize(latencies)
			os.Exit(0)
		default:
		}

		orderOut(winPtr)
		time.Sleep(hideHold)

		// Drain any spurious frame events that fired during hide.
		drainFrames()

		t0 := time.Now()
		makeKeyAndOrderFront(winPtr)
		// Kick Gio's event loop — orderOut leaves the loop parked in
		// Event(); makeKeyAndOrderFront alone doesn't queue a paint.
		gioWin.Invalidate()

		select {
		case <-frameCh:
			lat := time.Since(t0)
			latencies = append(latencies, lat)
			fmt.Printf("cycle %2d: show→frame %6.2f ms\n", i, ms(lat))
		case <-time.After(maxShowWait):
			fmt.Printf("cycle %2d: TIMEOUT (no frame in %v)\n", i, maxShowWait)
		}

		// Hold visible briefly so a human can verify focus/keys.
		time.Sleep(50 * time.Millisecond)
	}

	summarize(latencies)
	log.Printf("done.")
	os.Exit(0)
}

func drainFrames() {
	for {
		select {
		case <-frameCh:
		default:
			return
		}
	}
}

func summarize(d []time.Duration) {
	if len(d) == 0 {
		fmt.Println("no measurements")
		return
	}
	sort.Slice(d, func(i, j int) bool { return d[i] < d[j] })
	min := d[0]
	max := d[len(d)-1]
	median := d[len(d)/2]
	var sum float64
	for _, x := range d {
		sum += float64(x)
	}
	mean := time.Duration(sum / float64(len(d)))
	var variance float64
	for _, x := range d {
		variance += math.Pow(float64(x-mean), 2)
	}
	stddev := time.Duration(math.Sqrt(variance / float64(len(d))))

	fmt.Println("\n=== show→frame latency ===")
	fmt.Printf("samples: %d\n", len(d))
	fmt.Printf("min:     %.2f ms\n", ms(min))
	fmt.Printf("median:  %.2f ms\n", ms(median))
	fmt.Printf("mean:    %.2f ms\n", ms(mean))
	fmt.Printf("max:     %.2f ms\n", ms(max))
	fmt.Printf("stddev:  %.2f ms\n", ms(stddev))
}

func ms(d time.Duration) float64 { return d.Seconds() * 1000 }

// silence "unused" warnings for hypothetical future use
var _ = unsafe.Pointer(nil)
