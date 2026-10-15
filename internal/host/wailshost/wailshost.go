// Package wailshost implements host.Host with Wails v3 windows. On Windows it
// adds the native Win32 layer for what Wails lacks (window ownership) and for
// ground truth (what the OS says the rectangles really are); elsewhere it uses
// Wails' own screen and bounds APIs.
package wailshost

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/evobytes/lumen/internal/host"
	"github.com/evobytes/lumen/internal/win32"
)

const (
	floatW, floatH = 480, 270 // logical (DIP) size of the floating output window
	settleTries    = 4
	settleWait     = 120 * time.Millisecond
)

// Host controls the control and output windows.
type Host struct {
	app  *application.App
	ctrl *application.WebviewWindow
	out  *application.WebviewWindow

	mu      sync.Mutex
	last    placement // what the output was last asked to do, re-applied after page load
	toolkit any       // what the toolkit reported for the last toolkit-mode placement
}

type placement struct {
	set      bool
	floating bool
	screen   string
	how      host.Placement
}

var _ host.Host = (*Host)(nil)

// New wraps the application and its two windows.
func New(app *application.App, control, output *application.WebviewWindow) *Host {
	return &Host{app: app, ctrl: control, out: output}
}

func hwnd(w *application.WebviewWindow) uintptr { return uintptr(w.NativeWindow()) }

// Capabilities implements host.Host. Only Windows is verified: Wayland forbids positioning
// windows, and X11 and macOS placement have not been run.
func (h *Host) Capabilities() host.Capabilities {
	win := runtime.GOOS == "windows"
	return host.Capabilities{ExactPlacement: win, Ownership: win, Topmost: win}
}

// Ready reports whether both native windows exist.
func (h *Host) Ready() bool { return hwnd(h.ctrl) != 0 && hwnd(h.out) != 0 }

// Attach makes the output window an owned child of the control window (Windows
// only: elsewhere the toolkit has no equivalent and it is a no-op).
func (h *Host) Attach() error {
	if runtime.GOOS != "windows" {
		return nil
	}
	if !h.Ready() {
		return errors.New("windows not created yet")
	}
	return win32.SetOwner(hwnd(h.out), hwnd(h.ctrl))
}

// ShowOutput shows the output window.
func (h *Host) ShowOutput() { application.InvokeSync(func() { h.out.Show() }) }

// Reapply repeats the last output placement. It is called when the output page
// has finished loading: the toolkit's own first-show positioning can otherwise
// land after the first placement (seen once as a 24 px offset in the spike).
func (h *Host) Reapply(ctx context.Context) error {
	h.mu.Lock()
	p := h.last
	h.mu.Unlock()
	switch {
	case !p.set:
		return nil
	case p.floating:
		return h.FloatOutput(ctx)
	default:
		return h.PlaceOutput(ctx, p.screen, p.how)
	}
}

// Screens implements host.Host.
func (h *Host) Screens(context.Context) ([]host.Screen, error) {
	if runtime.GOOS == "windows" {
		return win32.Monitors()
	}
	return h.toolkitScreens(), nil
}

func (h *Host) toolkitScreens() []host.Screen {
	var out []host.Screen
	for _, s := range h.app.Screen.GetAll() {
		// Names are not unique (WSLg reports "rdp" for every monitor); the toolkit's ID is.
		id := s.Name + "#" + s.ID
		scale := float64(s.ScaleFactor)
		if scale <= 0 {
			scale = 1
		}
		out = append(out, host.Screen{
			ID:      id,
			Rect:    host.Rect{X: s.PhysicalBounds.X, Y: s.PhysicalBounds.Y, W: s.PhysicalBounds.Width, H: s.PhysicalBounds.Height},
			DPI:     int(96 * scale),
			Scale:   scale,
			Primary: s.IsPrimary,
		})
	}
	return out
}

func (h *Host) find(ctx context.Context, id string) (host.Screen, error) {
	screens, err := h.Screens(ctx)
	if err != nil {
		return host.Screen{}, err
	}
	for _, s := range screens {
		if s.ID == id {
			return s, nil
		}
	}
	return host.Screen{}, fmt.Errorf("unknown screen %q", id)
}

// PlaceOutput implements host.Host.
func (h *Host) PlaceOutput(ctx context.Context, screenID string, how host.Placement) error {
	scr, err := h.find(ctx, screenID)
	if err != nil {
		return err
	}
	h.mu.Lock()
	h.last = placement{set: true, screen: screenID, how: how}
	h.mu.Unlock()
	switch how {
	case host.PlaceWin32:
		if runtime.GOOS != "windows" {
			return win32.ErrUnsupported
		}
		return h.settle(ctx, scr.Rect)
	case host.PlaceToolkit:
		return h.placeViaToolkit(scr)
	default:
		return fmt.Errorf("unknown placement %q", how)
	}
}

// settle sets the native rectangle and verifies it stuck, retrying briefly.
func (h *Host) settle(ctx context.Context, want host.Rect) error {
	var got host.Rect
	for range settleTries {
		var serr error
		application.InvokeSync(func() { serr = win32.SetBounds(hwnd(h.out), want, true) })
		if serr != nil {
			return serr
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("placing output: %w", ctx.Err())
		case <-time.After(settleWait):
		}
		var err error
		if got, err = win32.WindowRect(hwnd(h.out)); err != nil {
			return err
		}
		if got == want {
			return nil
		}
	}
	return fmt.Errorf("output did not settle at %+v (is at %+v)", want, got)
}

func (h *Host) placeViaToolkit(scr host.Screen) error {
	var match *application.Screen
	for _, s := range h.app.Screen.GetAll() {
		pb := s.PhysicalBounds
		if pb.X == scr.Rect.X && pb.Y == scr.Rect.Y && pb.Width == scr.Rect.W && pb.Height == scr.Rect.H {
			match = s
			break
		}
	}
	if match == nil {
		return errors.New("no toolkit screen matches that monitor")
	}
	h.mu.Lock()
	h.toolkit = map[string]any{"matchedScreen": match, "setBounds": match.Bounds}
	h.mu.Unlock()
	application.InvokeSync(func() {
		h.out.SetAlwaysOnTop(true)
		h.out.SetBounds(match.Bounds) // logical (DIP) bounds
		h.out.Show()
	})
	return nil
}

// FloatOutput implements host.Host.
func (h *Host) FloatOutput(context.Context) error {
	h.mu.Lock()
	h.last = placement{set: true, floating: true}
	h.mu.Unlock()
	if runtime.GOOS != "windows" {
		var cb application.Rect
		application.InvokeSync(func() {
			cb = h.ctrl.Bounds()
			h.out.SetBounds(application.Rect{X: cb.X + cb.Width - floatW - 24, Y: cb.Y + 80, Width: floatW, Height: floatH})
			h.out.SetAlwaysOnTop(true)
			h.out.Show()
		})
		return nil
	}
	cr, err := win32.WindowRect(hwnd(h.ctrl))
	if err != nil {
		return err
	}
	scale := 1.0
	if dpi := win32.DPI(hwnd(h.ctrl)); dpi > 0 {
		scale = float64(dpi) / 96
	}
	w, hh := int(floatW*scale), int(floatH*scale)
	r := host.Rect{X: cr.X + cr.W - w - int(24*scale), Y: cr.Y + int(80*scale), W: w, H: hh}
	var serr error
	application.InvokeSync(func() { serr = win32.SetBounds(hwnd(h.out), r, true) })
	return serr
}

func (h *Host) window(w *application.WebviewWindow, screens []host.Screen) host.Window {
	if runtime.GOOS != "windows" {
		b := w.Bounds()
		return host.Window{Rect: host.Rect{X: b.X, Y: b.Y, W: b.Width, H: b.Height}, Visible: w.IsVisible()}
	}
	id := hwnd(w)
	r, _ := win32.WindowRect(id)
	win := host.Window{
		Rect:       r,
		Visible:    win32.IsVisible(id),
		Topmost:    win32.IsTopmost(id),
		ToolWindow: win32.IsToolWindow(id),
		DPI:        win32.DPI(id),
	}
	cx, cy := r.Center()
	for _, s := range screens {
		if s.Rect.Contains(cx, cy) {
			win.Monitor = s.ID
		}
	}
	return win
}

// Report implements host.Host.
func (h *Host) Report(ctx context.Context) (host.Report, error) {
	if !h.Ready() {
		return host.Report{}, errors.New("windows not created yet")
	}
	screens, err := h.Screens(ctx)
	if err != nil {
		return host.Report{}, err
	}
	rep := host.Report{Control: h.window(h.ctrl, screens), Output: h.window(h.out, screens), OutputOwnedBy: "none"}
	if runtime.GOOS == "windows" {
		switch owner := win32.Owner(hwnd(h.out)); owner {
		case 0:
		case hwnd(h.ctrl):
			rep.OutputOwnedBy = "control"
		default:
			rep.OutputOwnedBy = "other"
		}
	}
	h.mu.Lock()
	rep.Toolkit = map[string]any{"screens": h.app.Screen.GetAll(), "lastPlacement": h.toolkit}
	h.mu.Unlock()
	return rep, nil
}

// Quit implements host.Host.
func (h *Host) Quit() { h.app.Quit() }
