package main

// Host is the window-management seam. The rest of the app (API, state, self-test) only
// talks to this interface, so Wails could be replaced without touching them.

// Rect is in physical pixels, in virtual-desktop coordinates (origin may be negative).
type Rect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

func (r Rect) Center() (int, int) { return r.X + r.W/2, r.Y + r.H/2 }
func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

// ScreenInfo is the OS's own view of a monitor (Win32 EnumDisplayMonitors + GetDpiForMonitor).
type ScreenInfo struct {
	ID      string  `json:"id"` // Windows device name, e.g. \\.\DISPLAY2
	Rect    Rect    `json:"rect"`
	DPI     int     `json:"dpi"` // effective DPI; 96 = 100%
	Scale   float64 `json:"scale"`
	Primary bool    `json:"primary"`
}

// PlaceMode selects how the output window is moved: through the Wails API or straight Win32.
type PlaceMode string

const (
	PlaceWails PlaceMode = "wails"
	PlaceWin32 PlaceMode = "win32"
)

type WinInfo struct {
	Rect       Rect   `json:"rect"`
	Visible    bool   `json:"visible"`
	Topmost    bool   `json:"topmost"`
	ToolWindow bool   `json:"toolWindow"`
	DPI        int    `json:"dpi"`
	Monitor    string `json:"monitor"` // which monitor contains the window centre
}

type WindowReport struct {
	Control        WinInfo `json:"control"`
	Output         WinInfo `json:"output"`
	OutputOwnedBy  string  `json:"outputOwnedBy"` // "control" | "none" | "other"
	WailsOutBounds any     `json:"wailsOutputBounds,omitempty"`
	WailsScreens   any     `json:"wailsScreens,omitempty"`
}

type Host interface {
	Screens() ([]ScreenInfo, error)
	// PlaceOutput makes the output window cover the given screen.
	PlaceOutput(screenID string, mode PlaceMode) error
	// FloatOutput returns the output window to a small floating window over the control window.
	FloatOutput() error
	Report() (WindowReport, error)
	Quit()
}
