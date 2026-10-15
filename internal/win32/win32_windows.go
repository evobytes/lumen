//go:build windows

// Package win32 is the thin Windows edge: ground-truth monitor and window
// facts, plus the few Win32 operations the windowing toolkit lacks (window
// ownership, exact placement). It is CGO-free and has an inert twin elsewhere.
package win32

import (
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/evobytes/lumen/internal/host"
)

var (
	user32 = windows.NewLazySystemDLL("user32.dll")
	shcore = windows.NewLazySystemDLL("shcore.dll")

	procEnumDisplayMonitors = user32.NewProc("EnumDisplayMonitors")
	procGetMonitorInfoW     = user32.NewProc("GetMonitorInfoW")
	procGetWindowRect       = user32.NewProc("GetWindowRect")
	procSetWindowLongPtrW   = user32.NewProc("SetWindowLongPtrW")
	procGetWindowLongPtrW   = user32.NewProc("GetWindowLongPtrW")
	procGetWindow           = user32.NewProc("GetWindow")
	procIsWindowVisible     = user32.NewProc("IsWindowVisible")
	procSetWindowPos        = user32.NewProc("SetWindowPos")
	procGetDpiForWindow     = user32.NewProc("GetDpiForWindow")
	procGetDpiForMonitor    = shcore.NewProc("GetDpiForMonitor")
)

const (
	gwlpHwndParent = ^uintptr(7)  // GWLP_HWNDPARENT (-8): for a top-level window, its owner
	gwlExStyle     = ^uintptr(19) // GWL_EXSTYLE (-20)
	gwOwner        = 4

	wsExTopmost    = 0x00000008
	wsExToolWindow = 0x00000080

	swpNoActivate = 0x0010
	swpShowWindow = 0x0040

	monitorInfoPrimary = 1
)

var (
	hwndTopmost   = ^uintptr(0) // HWND_TOPMOST (-1)
	hwndNoTopmost = ^uintptr(1) // HWND_NOTOPMOST (-2)
)

type winRect struct{ Left, Top, Right, Bottom int32 }

func (r winRect) rect() host.Rect {
	return host.Rect{X: int(r.Left), Y: int(r.Top), W: int(r.Right - r.Left), H: int(r.Bottom - r.Top)}
}

type monitorInfoEx struct {
	Size    uint32
	Monitor winRect
	Work    winRect
	Flags   uint32
	Device  [32]uint16
}

var (
	enumMu  sync.Mutex
	enumOut []host.Screen
	// windows.NewCallback allocations are never freed, so create the callback exactly once.
	enumCallback = sync.OnceValue(func() uintptr {
		return windows.NewCallback(func(hMonitor, _, _, _ uintptr) uintptr {
			mi := monitorInfoEx{Size: uint32(unsafe.Sizeof(monitorInfoEx{}))}
			if r, _, _ := procGetMonitorInfoW.Call(hMonitor, uintptr(unsafe.Pointer(&mi))); r == 0 { //nolint:gosec // Win32 out-parameter
				return 1 // skip this monitor, keep enumerating
			}
			var dpiX, dpiY uint32
			dpi := 96
			if hr, _, _ := procGetDpiForMonitor.Call(hMonitor, 0 /* MDT_EFFECTIVE_DPI */, uintptr(unsafe.Pointer(&dpiX)), uintptr(unsafe.Pointer(&dpiY))); hr == 0 && dpiX > 0 { //nolint:gosec // Win32 out-parameters
				dpi = int(dpiX)
			}
			enumOut = append(enumOut, host.Screen{
				ID:      windows.UTF16ToString(mi.Device[:]),
				Rect:    mi.Monitor.rect(),
				DPI:     dpi,
				Scale:   float64(dpi) / 96,
				Primary: mi.Flags&monitorInfoPrimary != 0,
			})
			return 1
		})
	})
)

// Monitors lists the connected monitors with physical-pixel rectangles and effective DPI.
func Monitors() ([]host.Screen, error) {
	enumMu.Lock()
	defer enumMu.Unlock()
	enumOut = nil
	if r, _, err := procEnumDisplayMonitors.Call(0, 0, enumCallback(), 0); r == 0 {
		return nil, fmt.Errorf("EnumDisplayMonitors: %w", err)
	}
	out := enumOut
	enumOut = nil
	return out, nil
}

// WindowRect returns the window rectangle in physical pixels.
func WindowRect(hwnd uintptr) (host.Rect, error) {
	var r winRect
	if ok, _, err := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r))); ok == 0 { //nolint:gosec // Win32 out-parameter
		return host.Rect{}, fmt.Errorf("GetWindowRect: %w", err)
	}
	return r.rect(), nil
}

func exStyle(hwnd uintptr) uintptr {
	v, _, _ := procGetWindowLongPtrW.Call(hwnd, gwlExStyle)
	return v
}

// IsTopmost reports whether the window is always-on-top.
func IsTopmost(hwnd uintptr) bool { return exStyle(hwnd)&wsExTopmost != 0 }

// IsToolWindow reports whether the window is a tool window (no taskbar button).
func IsToolWindow(hwnd uintptr) bool { return exStyle(hwnd)&wsExToolWindow != 0 }

// IsVisible reports whether the window is visible.
func IsVisible(hwnd uintptr) bool {
	v, _, _ := procIsWindowVisible.Call(hwnd)
	return v != 0
}

// DPI returns the window's DPI (96 is 100%), or 0 if it cannot be read.
func DPI(hwnd uintptr) int {
	v, _, _ := procGetDpiForWindow.Call(hwnd)
	return int(v)
}

// SetOwner makes owner the Win32 owner of child: the child then always stays
// above its owner, is minimised, restored and destroyed with it, and gets no
// taskbar button.
func SetOwner(child, owner uintptr) error {
	if child == 0 || owner == 0 {
		return errors.New("nil window handle")
	}
	// The return value is the previous owner (0 is valid); success is verified through Owner below.
	_, _, _ = procSetWindowLongPtrW.Call(child, gwlpHwndParent, owner)
	if got := Owner(child); got != owner {
		return fmt.Errorf("owner not applied (got %#x, want %#x)", got, owner)
	}
	return nil
}

// Owner returns the window's owner, or 0.
func Owner(hwnd uintptr) uintptr {
	v, _, _ := procGetWindow.Call(hwnd, gwOwner)
	return v
}

// SetBounds positions the window in physical pixels without stealing focus.
func SetBounds(hwnd uintptr, r host.Rect, topmost bool) error {
	after := hwndNoTopmost
	if topmost {
		after = hwndTopmost
	}
	ok, _, err := procSetWindowPos.Call(hwnd, after, uintptr(int32(r.X)), uintptr(int32(r.Y)), //nolint:gosec // values come from monitor rectangles
		uintptr(int32(r.W)), uintptr(int32(r.H)), swpNoActivate|swpShowWindow) //nolint:gosec // as above
	if ok == 0 {
		return fmt.Errorf("SetWindowPos: %w", err)
	}
	return nil
}
