// Package hosttest provides an in-memory host.Host for tests.
package hosttest

import (
	"context"
	"fmt"
	"sync"

	"github.com/evobytes/lumen/internal/host"
)

// Fake is a host.Host with scripted screens that records where the output went.
// "Placing" the output moves its rectangle exactly onto the screen.
type Fake struct {
	mu      sync.Mutex
	screens []host.Screen
	output  host.Rect
	quit    bool
	// Calls records method names in call order.
	Calls []string
}

// New returns a Fake with the given screens; the output starts floating.
func New(screens ...host.Screen) *Fake {
	return &Fake{screens: screens, output: host.Rect{X: 100, Y: 100, W: 480, H: 270}}
}

// Capabilities implements host.Host: the fake can do everything.
func (f *Fake) Capabilities() host.Capabilities {
	return host.Capabilities{ExactPlacement: true, Ownership: true, Topmost: true}
}

// Screens implements host.Host.
func (f *Fake) Screens(context.Context) ([]host.Screen, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = append(f.Calls, "Screens")
	return append([]host.Screen(nil), f.screens...), nil
}

// PlaceOutput implements host.Host.
func (f *Fake) PlaceOutput(_ context.Context, id string, _ host.Placement) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = append(f.Calls, "PlaceOutput:"+id)
	for _, s := range f.screens {
		if s.ID == id {
			f.output = s.Rect
			return nil
		}
	}
	return fmt.Errorf("unknown screen %q", id)
}

// FloatOutput implements host.Host.
func (f *Fake) FloatOutput(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = append(f.Calls, "FloatOutput")
	f.output = host.Rect{X: 100, Y: 100, W: 480, H: 270}
	return nil
}

// Report implements host.Host.
func (f *Fake) Report(context.Context) (host.Report, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	r := host.Report{OutputOwnedBy: "control"}
	r.Output = host.Window{Rect: f.output, Visible: true, Topmost: true, ToolWindow: true, DPI: 96}
	cx, cy := f.output.Center()
	for _, s := range f.screens {
		if s.Rect.Contains(cx, cy) {
			r.Output.Monitor = s.ID
		}
	}
	return r, nil
}

// Quit implements host.Host.
func (f *Fake) Quit() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.quit = true
}

// Quitted reports whether Quit was called.
func (f *Fake) Quitted() bool { f.mu.Lock(); defer f.mu.Unlock(); return f.quit }
