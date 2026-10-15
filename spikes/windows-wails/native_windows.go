//go:build windows

package main

import (
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
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
	gwlpHwndParent = ^uintptr(7)  // GWLP_HWNDPARENT (-8): for a top-level window this is its *owner*
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

type monitorInfoEx struct {
	Size    uint32
	Monitor winRect
	Work    winRect
	Flags   uint32
	Device  [32]uint16
}

func (r winRect) rect() Rect {
	return Rect{X: int(r.Left), Y: int(r.Top), W: int(r.Right - r.Left), H: int(r.Bottom - r.Top)}
}

var (
	enumMu  sync.Mutex
	enumOut []ScreenInfo
	// windows.NewCallback allocations are never freed, so create the callback exactly once.
	enumCallback = sync.OnceValue(func() uintptr {
		return windows.NewCallback(func(hMonitor, _hdc, _rect, _lparam uintptr) uintptr {
			mi := monitorInfoEx{Size: uint32(unsafe.Sizeof(monitorInfoEx{}))}
			if r, _, _ := procGetMonitorInfoW.Call(hMonitor, uintptr(unsafe.Pointer(&mi))); r == 0 {
				return 1 // skip this one, keep enumerating
			}
			var dpiX, dpiY uint32
			dpi := 96
			if hr, _, _ := procGetDpiForMonitor.Call(hMonitor, 0 /* MDT_EFFECTIVE_DPI */, uintptr(unsafe.Pointer(&dpiX)), uintptr(unsafe.Pointer(&dpiY))); hr == 0 && dpiX > 0 {
				dpi = int(dpiX)
			}
			enumOut = append(enumOut, ScreenInfo{
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

func nativeMonitors() ([]ScreenInfo, error) {
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

func nativeWindowRect(hwnd uintptr) (Rect, error) {
	var r winRect
	if ok, _, err := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r))); ok == 0 {
		return Rect{}, fmt.Errorf("GetWindowRect: %w", err)
	}
	return r.rect(), nil
}

func nativeExStyle(hwnd uintptr) uintptr {
	v, _, _ := procGetWindowLongPtrW.Call(hwnd, gwlExStyle)
	return v
}

func nativeIsTopmost(hwnd uintptr) bool    { return nativeExStyle(hwnd)&wsExTopmost != 0 }
func nativeIsToolWindow(hwnd uintptr) bool { return nativeExStyle(hwnd)&wsExToolWindow != 0 }

func nativeIsVisible(hwnd uintptr) bool {
	v, _, _ := procIsWindowVisible.Call(hwnd)
	return v != 0
}

func nativeDPI(hwnd uintptr) int {
	v, _, _ := procGetDpiForWindow.Call(hwnd)
	return int(v)
}

// nativeSetOwner makes owner the Win32 owner of child: the child then always stays above its
// owner, is minimised/restored/destroyed with it, and gets no taskbar button.
func nativeSetOwner(child, owner uintptr) error {
	if child == 0 || owner == 0 {
		return errors.New("nil window handle")
	}
	procSetWindowLongPtrW.Call(child, gwlpHwndParent, owner)
	if got := nativeOwner(child); got != owner {
		return fmt.Errorf("owner not applied (got %#x, want %#x)", got, owner)
	}
	return nil
}

func nativeOwner(hwnd uintptr) uintptr {
	v, _, _ := procGetWindow.Call(hwnd, gwOwner)
	return v
}

// nativeSetBounds positions in physical pixels, without stealing focus.
func nativeSetBounds(hwnd uintptr, r Rect, topmost bool) error {
	after := hwndNoTopmost
	if topmost {
		after = hwndTopmost
	}
	ok, _, err := procSetWindowPos.Call(hwnd, after, uintptr(int32(r.X)), uintptr(int32(r.Y)),
		uintptr(int32(r.W)), uintptr(int32(r.H)), swpNoActivate|swpShowWindow)
	if ok == 0 {
		return fmt.Errorf("SetWindowPos: %w", err)
	}
	return nil
}
