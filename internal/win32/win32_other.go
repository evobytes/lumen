//go:build !windows

// Package win32 is the thin Windows edge: ground-truth monitor and window
// facts, plus the few Win32 operations the windowing toolkit lacks (window
// ownership, exact placement). It is CGO-free and has an inert twin elsewhere.
package win32

import "github.com/evobytes/lumen/internal/host"

// Monitors is not available off Windows.
func Monitors() ([]host.Screen, error) { return nil, ErrUnsupported }

// WindowRect is not available off Windows.
func WindowRect(uintptr) (host.Rect, error) { return host.Rect{}, ErrUnsupported }

// IsTopmost is not available off Windows.
func IsTopmost(uintptr) bool { return false }

// IsToolWindow is not available off Windows.
func IsToolWindow(uintptr) bool { return false }

// IsVisible is not available off Windows.
func IsVisible(uintptr) bool { return false }

// DPI is not available off Windows.
func DPI(uintptr) int { return 0 }

// SetOwner is not available off Windows.
func SetOwner(_, _ uintptr) error { return ErrUnsupported }

// Owner is not available off Windows.
func Owner(uintptr) uintptr { return 0 }

// SetBounds is not available off Windows.
func SetBounds(uintptr, host.Rect, bool) error { return ErrUnsupported }
