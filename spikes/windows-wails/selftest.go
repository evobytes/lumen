package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

type placement struct {
	Screen  string    `json:"screen"`
	Mode    PlaceMode `json:"mode"`
	Want    Rect      `json:"want"`
	Got     Rect      `json:"got"`
	Exact   bool      `json:"exact"`
	DeltaX  int       `json:"deltaX"`
	DeltaY  int       `json:"deltaY"`
	DeltaW  int       `json:"deltaW"`
	DeltaH  int       `json:"deltaH"`
	Monitor string    `json:"monitorHoldingWindow"`
	Topmost bool      `json:"topmost"`
	Owned   string    `json:"ownedBy"`
	OutDPI  int       `json:"outputDPI"`
	Error   string    `json:"error,omitempty"`
	Samples []Rect    `json:"samplesAt100ms_700ms_2000ms"`
	Run     int       `json:"run"`
}

type selfTestReport struct {
	Started    string          `json:"started"`
	Screens    []ScreenInfo    `json:"screens"`
	Initial    WindowReport    `json:"initialWindows"`
	Placements []placement     `json:"placements"`
	Floating   WindowReport    `json:"floating"`
	Media      json.RawMessage `json:"media"`
	MediaError string          `json:"mediaError,omitempty"`
	Summary    map[string]any  `json:"summary"`
}

func runSelfTest(h *wailsHost, reports *Reports, path string) {
	rep := selfTestReport{Started: time.Now().Format(time.RFC3339)}
	defer func() {
		b, _ := json.MarshalIndent(rep, "", "  ")
		if err := os.WriteFile(path, b, 0o644); err != nil {
			log.Printf("write report: %v", err)
		}
		log.Printf("self-test report written to %s", path)
		h.Quit()
	}()

	var err error
	if rep.Screens, err = h.Screens(); err != nil {
		log.Printf("screens: %v", err)
		return
	}
	rep.Initial, _ = h.Report()

	for _, s := range rep.Screens {
		for _, mode := range []PlaceMode{PlaceWin32, PlaceWails} {
			for run := 1; run <= 3; run++ {
				p := placement{Screen: s.ID, Mode: mode, Want: s.Rect, Run: run}
				// Start each run from the floating state so every run is a real move.
				_ = h.FloatOutput()
				time.Sleep(300 * time.Millisecond)
				if err := h.PlaceOutput(s.ID, mode); err != nil {
					p.Error = err.Error()
				}
				last := time.Now()
				for _, d := range []time.Duration{100, 700, 2000} {
					time.Sleep(time.Until(last.Add(0)) + d*time.Millisecond - time.Since(last))
					w, _ := h.Report()
					p.Samples = append(p.Samples, w.Output.Rect)
				}
				w, _ := h.Report()
				p.Got = w.Output.Rect
				p.Exact = p.Got == p.Want
				p.DeltaX, p.DeltaY = p.Got.X-p.Want.X, p.Got.Y-p.Want.Y
				p.DeltaW, p.DeltaH = p.Got.W-p.Want.W, p.Got.H-p.Want.H
				p.Monitor, p.Topmost, p.Owned, p.OutDPI = w.Output.Monitor, w.Output.Topmost, w.OutputOwnedBy, w.Output.DPI
				rep.Placements = append(rep.Placements, p)
			}
		}
	}
	if err := h.FloatOutput(); err != nil {
		log.Printf("float: %v", err)
	}
	time.Sleep(500 * time.Millisecond)
	rep.Floating, _ = h.Report()

	// The output page runs its media probe on load and POSTs the result.
	deadline := time.Now().Add(40 * time.Second)
	for time.Now().Before(deadline) {
		if m, ok := reports.Get("media"); ok {
			rep.Media = m
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	if rep.Media == nil {
		rep.MediaError = "no media report received within 40s"
	}

	exact, total := 0, 0
	for _, p := range rep.Placements {
		total++
		if p.Exact && p.Error == "" {
			exact++
		}
	}
	rep.Summary = map[string]any{
		"monitors":                len(rep.Screens),
		"placementsExact":         fmt.Sprintf("%d/%d", exact, total),
		"outputOwnedByControl":    rep.Initial.OutputOwnedBy == "control",
		"outputTopmost":           rep.Initial.Output.Topmost,
		"outputHiddenFromTaskbar": rep.Initial.Output.ToolWindow,
		"mediaReceived":           rep.Media != nil,
	}
}

// runDemo parks the output on a screen so a screenshot can be taken from outside.
func runDemo(h *wailsHost, want string, mode PlaceMode, hold time.Duration) {
	screens, _ := h.Screens()
	id := want
	if want == "auto" {
		for _, s := range screens {
			if !s.Primary {
				id = s.ID
			}
		}
	}
	if err := h.PlaceOutput(id, mode); err != nil {
		log.Printf("demo place %s (%s): %v", id, mode, err)
	}
	log.Printf("demo: output on %s via %s", id, mode)
	if hold > 0 {
		time.Sleep(hold)
		h.Quit()
	}
}
