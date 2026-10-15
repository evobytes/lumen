//go:build !windows

package main

import "errors"

var errUnsupported = errors.New("native window access is only implemented on Windows")

func nativeMonitors() ([]ScreenInfo, error)     { return nil, errUnsupported }
func nativeWindowRect(uintptr) (Rect, error)    { return Rect{}, errUnsupported }
func nativeIsTopmost(uintptr) bool              { return false }
func nativeIsToolWindow(uintptr) bool           { return false }
func nativeIsVisible(uintptr) bool              { return false }
func nativeDPI(uintptr) int                     { return 0 }
func nativeSetOwner(child, owner uintptr) error { return errUnsupported }
func nativeOwner(uintptr) uintptr               { return 0 }
func nativeSetBounds(uintptr, Rect, bool) error { return errUnsupported }
