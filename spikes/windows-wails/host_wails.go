package main

import (
	"errors"
	"fmt"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const floatW, floatH = 480, 270 // logical (DIP) size of the floating output window

// wailsHost implements Host with Wails windows plus a thin native layer for what Wails lacks
// (window ownership) and for ground truth (what Windows says the rectangles really are).
type wailsHost struct {
	app  *application.App
	ctrl *application.WebviewWindow
	out  *application.WebviewWindow

	mu          sync.Mutex
	lastWailsOp any // what Wails reported for the last Wails-mode placement
}

func (h *wailsHost) hwnd(w *application.WebviewWindow) uintptr { return uintptr(w.NativeWindow()) }

func (h *wailsHost) ready() bool { return h.hwnd(h.ctrl) != 0 && h.hwnd(h.out) != 0 }

// Attach makes the output window an owned child of the control window.
func (h *wailsHost) Attach() error {
	if !h.ready() {
		return errors.New("windows not created yet")
	}
	return nativeSetOwner(h.hwnd(h.out), h.hwnd(h.ctrl))
}

func (h *wailsHost) Screens() ([]ScreenInfo, error) { return nativeMonitors() }

func (h *wailsHost) find(id string) (ScreenInfo, error) {
	screens, err := nativeMonitors()
	if err != nil {
		return ScreenInfo{}, err
	}
	for _, s := range screens {
		if s.ID == id {
			return s, nil
		}
	}
	return ScreenInfo{}, fmt.Errorf("unknown screen %q", id)
}

func (h *wailsHost) PlaceOutput(screenID string, mode PlaceMode) error {
	scr, err := h.find(screenID)
	if err != nil {
		return err
	}
	var perr error
	application.InvokeSync(func() {
		switch mode {
		case PlaceWin32:
			perr = nativeSetBounds(h.hwnd(h.out), scr.Rect, true)
		case PlaceWails:
			perr = h.placeViaWails(scr)
		default:
			perr = fmt.Errorf("unknown mode %q", mode)
		}
	})
	return perr
}

// placeViaWails finds the Wails screen that matches the OS monitor and uses Wails' own API.
func (h *wailsHost) placeViaWails(scr ScreenInfo) error {
	var match *application.Screen
	for _, s := range h.app.Screen.GetAll() {
		pb := s.PhysicalBounds
		if pb.X == scr.Rect.X && pb.Y == scr.Rect.Y && pb.Width == scr.Rect.W && pb.Height == scr.Rect.H {
			match = s
			break
		}
	}
	if match == nil {
		return errors.New("no Wails screen matches that monitor's physical bounds")
	}
	h.mu.Lock()
	h.lastWailsOp = map[string]any{"matchedScreen": match, "setBounds": match.Bounds}
	h.mu.Unlock()
	h.out.SetAlwaysOnTop(true)
	h.out.SetBounds(match.Bounds) // logical (DIP) bounds
	h.out.Show()
	return nil
}

func (h *wailsHost) FloatOutput() error {
	cr, err := nativeWindowRect(h.hwnd(h.ctrl))
	if err != nil {
		return err
	}
	scale := 1.0
	if dpi := nativeDPI(h.hwnd(h.ctrl)); dpi > 0 {
		scale = float64(dpi) / 96
	}
	w, hh := int(floatW*scale), int(floatH*scale)
	r := Rect{X: cr.X + cr.W - w - int(24*scale), Y: cr.Y + int(80*scale), W: w, H: hh}
	var perr error
	application.InvokeSync(func() { perr = nativeSetBounds(h.hwnd(h.out), r, true) })
	return perr
}

func (h *wailsHost) info(hwnd uintptr, screens []ScreenInfo) WinInfo {
	r, _ := nativeWindowRect(hwnd)
	wi := WinInfo{
		Rect:       r,
		Visible:    nativeIsVisible(hwnd),
		Topmost:    nativeIsTopmost(hwnd),
		ToolWindow: nativeIsToolWindow(hwnd),
		DPI:        nativeDPI(hwnd),
	}
	cx, cy := r.Center()
	for _, s := range screens {
		if s.Rect.Contains(cx, cy) {
			wi.Monitor = s.ID
		}
	}
	return wi
}

func (h *wailsHost) Report() (WindowReport, error) {
	if !h.ready() {
		return WindowReport{}, errors.New("windows not created yet")
	}
	screens, err := nativeMonitors()
	if err != nil {
		return WindowReport{}, err
	}
	rep := WindowReport{
		Control:      h.info(h.hwnd(h.ctrl), screens),
		Output:       h.info(h.hwnd(h.out), screens),
		WailsScreens: h.app.Screen.GetAll(),
	}
	switch owner := nativeOwner(h.hwnd(h.out)); owner {
	case 0:
		rep.OutputOwnedBy = "none"
	case h.hwnd(h.ctrl):
		rep.OutputOwnedBy = "control"
	default:
		rep.OutputOwnedBy = "other"
	}
	h.mu.Lock()
	rep.WailsOutBounds = h.lastWailsOp
	h.mu.Unlock()
	return rep, nil
}

func (h *wailsHost) Quit() { h.app.Quit() }
