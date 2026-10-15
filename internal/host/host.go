// Package host defines the window-management seam. The rest of the application
// (API, self-test, state) talks only to Host, so the windowing toolkit (Wails
// today) can be replaced without touching them. Rectangles are in physical
// pixels in virtual-desktop coordinates; the origin may be negative.
package host

import "context"

// Rect is a rectangle in physical pixels.
type Rect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

// Center returns the middle point of r.
func (r Rect) Center() (x, y int) { return r.X + r.W/2, r.Y + r.H/2 }

// Contains reports whether the point is inside r.
func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

// Screen is the operating system's own view of a monitor.
type Screen struct {
	ID      string  `json:"id"` // Windows device name, e.g. \\.\DISPLAY2
	Rect    Rect    `json:"rect"`
	DPI     int     `json:"dpi"` // effective DPI; 96 is 100%
	Scale   float64 `json:"scale"`
	Primary bool    `json:"primary"`
}

// Placement selects how the output window is moved.
type Placement string

// Placement modes. PlaceWin32 sets the native window rectangle directly and is
// the deterministic default on Windows; PlaceToolkit uses the windowing toolkit's own API.
const (
	PlaceWin32   Placement = "win32"
	PlaceToolkit Placement = "toolkit"
)

// Window describes one native window as the OS reports it.
type Window struct {
	Rect       Rect   `json:"rect"`
	Visible    bool   `json:"visible"`
	Topmost    bool   `json:"topmost"`
	ToolWindow bool   `json:"toolWindow"`
	DPI        int    `json:"dpi"`
	Monitor    string `json:"monitor"` // the screen containing the window's centre
}

// Report is the ground truth about both windows.
type Report struct {
	Control       Window `json:"control"`
	Output        Window `json:"output"`
	OutputOwnedBy string `json:"outputOwnedBy"` // "control", "none" or "other"
	Toolkit       any    `json:"toolkit,omitempty"`
}

// Capabilities says what the platform lets the application do and verify. The self-test only
// gates on what a platform can do: Wayland, for one, forbids applications from positioning windows.
type Capabilities struct {
	// ExactPlacement is true when the output window can be placed on a monitor and the placement
	// read back from the OS (verified on Windows only).
	ExactPlacement bool `json:"exactPlacement"`
	// Ownership is true when the output window can be a native owned child of the control window.
	Ownership bool `json:"ownership"`
	// Topmost is true when the always-on-top state can be read back from the OS.
	Topmost bool `json:"topmost"`
}

// Host controls the control and output windows.
type Host interface {
	// Capabilities reports what this platform supports; it never changes during a run.
	Capabilities() Capabilities
	// Screens lists the connected monitors as the OS sees them.
	Screens(ctx context.Context) ([]Screen, error)
	// PlaceOutput makes the output window cover the screen with the given ID.
	PlaceOutput(ctx context.Context, screenID string, how Placement) error
	// FloatOutput returns the output window to a small floating window over the control window.
	FloatOutput(ctx context.Context) error
	// Report describes both windows as the OS sees them.
	Report(ctx context.Context) (Report, error)
	// Quit ends the application.
	Quit()
}
