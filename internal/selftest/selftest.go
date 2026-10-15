// Package selftest drives the real, running application through its own host
// seam and web page and writes a machine-readable report. It answers, on the
// machine it runs on: can the output window be placed exactly on every
// monitor, is it owned/topmost/hidden from the taskbar, and what media can the
// webview play? `lumen -selftest` runs it; CI uses it to verify packaged builds.
package selftest

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"time"

	"github.com/evobytes/lumen/internal/api"
	"github.com/evobytes/lumen/internal/host"
)

//go:embed media
var mediaFS embed.FS

// Media returns the small test clips the output page's ?probe plays (H.264,
// HEVC, AV1, VP9 and VP9 with alpha), rooted so /media/<name> resolves.
func Media() fs.FS {
	sub, err := fs.Sub(mediaFS, "media")
	if err != nil {
		panic(err) // embedded at build time
	}
	return sub
}

// Options tunes a run. The zero value uses sensible defaults.
type Options struct {
	// Modes are the placement modes to exercise on every screen.
	Modes []host.Placement
	// Repeats is how many times each screen/mode pair is placed (default 3).
	Repeats int
	// Samples are the delays after a placement at which the window rectangle
	// is read, to catch asynchronous drift (default 100ms, 700ms, 2s).
	Samples []time.Duration
	// MediaWait bounds the wait for the page's media report (default 40s).
	MediaWait time.Duration
	Logger    *slog.Logger
}

func (o *Options) defaults() {
	if len(o.Modes) == 0 {
		o.Modes = []host.Placement{host.PlaceToolkit}
	}
	if o.Repeats <= 0 {
		o.Repeats = 3
	}
	if len(o.Samples) == 0 {
		o.Samples = []time.Duration{100 * time.Millisecond, 700 * time.Millisecond, 2 * time.Second}
	}
	if o.MediaWait <= 0 {
		o.MediaWait = 40 * time.Second
	}
	if o.Logger == nil {
		o.Logger = slog.New(slog.DiscardHandler)
	}
}

// Placed is the outcome of one placement of the output window.
type Placed struct {
	Screen  string         `json:"screen"`
	Mode    host.Placement `json:"mode"`
	Run     int            `json:"run"`
	Want    host.Rect      `json:"want"`
	Got     host.Rect      `json:"got"`
	Exact   bool           `json:"exact"`
	Samples []host.Rect    `json:"samples"`
	Monitor string         `json:"monitorHoldingWindow"`
	Topmost bool           `json:"topmost"`
	OwnedBy string         `json:"ownedBy"`
	Error   string         `json:"error,omitempty"`
}

// Report is the full self-test result.
type Report struct {
	Started      string            `json:"started"`
	Capabilities host.Capabilities `json:"capabilities"`
	Screens      []host.Screen     `json:"screens"`
	Initial      host.Report       `json:"initialWindows"`
	Placements   []Placed          `json:"placements"`
	Floating     host.Report       `json:"floating"`
	Media        json.RawMessage   `json:"media,omitempty"`
	MediaError   string            `json:"mediaError,omitempty"`
	Summary      Summary           `json:"summary"`
}

// Summary is the at-a-glance verdict.
type Summary struct {
	Monitors        int    `json:"monitors"`
	PlacementsExact string `json:"placementsExact"` // "exact/total"
	OutputTopmost   bool   `json:"outputTopmost"`
	MediaReceived   bool   `json:"mediaReceived"`
	// PlacementGated is false when the platform cannot place windows exactly (host.Capabilities):
	// placements are then recorded but do not decide Pass.
	PlacementGated bool `json:"placementGated"`
	// Pass requires that the output page loaded and reported (MediaReceived: the app, its server and
	// the window really ran) and, where the platform can do it (PlacementGated), that every
	// placement was exact and the output is always-on-top. Media codec support is reported but never
	// gates, because it varies by machine.
	Pass bool `json:"pass"`
}

// Run executes the self-test. It never returns an error for a failed check;
// failures are recorded in the report. An error means the run itself could not proceed.
func Run(ctx context.Context, h host.Host, reports *api.Reports, o Options) (Report, error) {
	o.defaults()
	rep := Report{Started: time.Now().UTC().Format(time.RFC3339), Capabilities: h.Capabilities()}

	screens, err := h.Screens(ctx)
	if err != nil {
		return rep, fmt.Errorf("list screens: %w", err)
	}
	rep.Screens = screens
	rep.Initial, _ = h.Report(ctx)

	for _, s := range screens {
		for _, mode := range o.Modes {
			for run := 1; run <= o.Repeats; run++ {
				if err := ctx.Err(); err != nil {
					return rep, err
				}
				rep.Placements = append(rep.Placements, placeOnce(ctx, h, s, mode, run, o))
			}
		}
	}

	if err := h.FloatOutput(ctx); err != nil {
		o.Logger.Warn("float output", "err", err)
	}
	time.Sleep(settle(o))
	rep.Floating, _ = h.Report(ctx)

	mctx, cancel := context.WithTimeout(ctx, o.MediaWait)
	defer cancel()
	if m, err := reports.Wait(mctx, "media"); err != nil {
		rep.MediaError = err.Error()
	} else {
		rep.Media = m
	}

	rep.Summary = summarise(rep)
	return rep, nil
}

func settle(o Options) time.Duration { return o.Samples[0] }

func placeOnce(ctx context.Context, h host.Host, s host.Screen, mode host.Placement, run int, o Options) Placed {
	p := Placed{Screen: s.ID, Mode: mode, Run: run, Want: s.Rect}
	_ = h.FloatOutput(ctx) // start every run from the floating state so each is a real move
	time.Sleep(settle(o))
	if err := h.PlaceOutput(ctx, s.ID, mode); err != nil {
		p.Error = err.Error()
	}
	var elapsed time.Duration
	for _, at := range o.Samples {
		time.Sleep(at - elapsed)
		elapsed = at
		if w, err := h.Report(ctx); err == nil {
			p.Samples = append(p.Samples, w.Output.Rect)
		}
	}
	w, err := h.Report(ctx)
	if err != nil {
		p.Error = joinErr(p.Error, err.Error())
		return p
	}
	p.Got, p.Monitor, p.Topmost, p.OwnedBy = w.Output.Rect, w.Output.Monitor, w.Output.Topmost, w.OutputOwnedBy
	p.Exact = p.Error == "" && p.Got == p.Want
	for _, r := range p.Samples { // the window must also have stayed put
		if r != p.Want {
			p.Exact = false
		}
	}
	return p
}

func joinErr(a, b string) string {
	if a == "" {
		return b
	}
	return a + "; " + b
}

func summarise(rep Report) Summary {
	exact := 0
	for _, p := range rep.Placements {
		if p.Exact {
			exact++
		}
	}
	gated := rep.Capabilities.ExactPlacement
	sum := Summary{
		Monitors:        len(rep.Screens),
		PlacementsExact: fmt.Sprintf("%d/%d", exact, len(rep.Placements)),
		OutputTopmost:   rep.Initial.Output.Topmost,
		MediaReceived:   rep.Media != nil,
		PlacementGated:  gated,
	}
	sum.Pass = sum.MediaReceived
	if gated {
		sum.Pass = sum.Pass && len(rep.Placements) > 0 && exact == len(rep.Placements) && rep.Initial.Output.Topmost
	}
	return sum
}

// Demo parks the output on a screen so a person (or a screenshot) can look at
// it. screen "auto" picks the first non-primary monitor. A positive hold quits
// afterwards; zero leaves the application running.
func Demo(ctx context.Context, h host.Host, screen string, how host.Placement, hold time.Duration, log *slog.Logger) error {
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	screens, err := h.Screens(ctx)
	if err != nil {
		return err
	}
	id := screen
	if screen == "auto" {
		id = ""
		for _, s := range screens {
			if !s.Primary {
				id = s.ID
			}
		}
		if id == "" {
			return fmt.Errorf("demo: no non-primary screen among %d", len(screens))
		}
	}
	if err := h.PlaceOutput(ctx, id, how); err != nil {
		return fmt.Errorf("demo: place on %s: %w", id, err)
	}
	log.Info("demo: output placed", "screen", id, "placement", string(how))
	if hold > 0 {
		select {
		case <-time.After(hold):
		case <-ctx.Done():
		}
		h.Quit()
	}
	return nil
}
