//go:build windows

package app

import "github.com/wailsapp/wails/v3/pkg/events"

// loadFinishedEvent is the toolkit's "page finished loading" window event; it is
// platform-specific in Wails v3.
func loadFinishedEvent() events.WindowEventType { return events.Windows.WebViewNavigationCompleted }
