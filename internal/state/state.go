// Package state is the single source of truth for what the audience sees:
// the slides, the current position and the blank flag. Everything that renders
// (the output window, the control window, OBS, a phone) is a pure function of
// it and subscribes to changes.
package state

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// ErrOutOfRange is returned when a slide index does not exist.
var ErrOutOfRange = errors.New("slide index out of range")

// Slide is one screenful of text.
type Slide struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

// Label is the short name used in lists: the title, else the first line of the text.
func (s Slide) Label() string {
	if s.Title != "" {
		return s.Title
	}
	for line := range strings.SplitSeq(strings.TrimSpace(s.Text), "\n") {
		if line != "" {
			return line
		}
	}
	return "(empty)"
}

// SlideView is a Slide as served by the API, with its derived label.
type SlideView struct {
	Slide
	Label string `json:"label"`
}

// Snapshot is a consistent copy of the state.
type Snapshot struct {
	Index   int         `json:"index"`
	Count   int         `json:"count"`
	Blanked bool        `json:"blanked"`
	Slide   *SlideView  `json:"slide"`
	Slides  []SlideView `json:"slides"`
}

// State holds the presentation. The zero value is not usable; call New.
type State struct {
	mu      sync.Mutex
	slides  []Slide
	index   int
	blanked bool
	subs    map[chan struct{}]struct{}
}

// New returns a State holding slides.
func New(slides []Slide) *State {
	return &State{slides: append([]Slide(nil), slides...), subs: map[chan struct{}]struct{}{}}
}

// Sample returns the slides a fresh install starts with.
func Sample() []Slide {
	return []Slide{
		{Title: "Welcome", Text: "Lumen\nThe output window is driven by server state"},
		{Title: "Second slide", Text: "Placed on the chosen monitor"},
		{Title: "Third slide", Text: "Thank you"},
	}
}

// Snapshot returns the current state.
func (s *State) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap := Snapshot{Index: s.index, Count: len(s.slides), Blanked: s.blanked, Slides: make([]SlideView, len(s.slides))}
	for i, sl := range s.slides {
		snap.Slides[i] = SlideView{Slide: sl, Label: sl.Label()}
	}
	if len(s.slides) > 0 {
		cur := snap.Slides[s.index]
		snap.Slide = &cur
	}
	return snap
}

// Next advances one slide; on the last slide it stays put.
func (s *State) Next() { s.move(+1) }

// Prev goes back one slide; on the first slide it stays put.
func (s *State) Prev() { s.move(-1) }

func (s *State) move(delta int) {
	s.mu.Lock()
	if n := s.index + delta; n >= 0 && n < len(s.slides) {
		s.index = n
	}
	s.mu.Unlock()
	s.notify()
}

// Goto shows the slide at index.
func (s *State) Goto(index int) error {
	s.mu.Lock()
	if index < 0 || index >= len(s.slides) {
		n := len(s.slides)
		s.mu.Unlock()
		return fmt.Errorf("%w: %d (have %d slides)", ErrOutOfRange, index, n)
	}
	s.index = index
	s.mu.Unlock()
	s.notify()
	return nil
}

// SetBlank blanks (true), unblanks (false) or, when v is nil, toggles.
func (s *State) SetBlank(v *bool) {
	s.mu.Lock()
	if v == nil {
		s.blanked = !s.blanked
	} else {
		s.blanked = *v
	}
	s.mu.Unlock()
	s.notify()
}

// SetSlides replaces the slides and returns to the first one.
func (s *State) SetSlides(slides []Slide) {
	s.mu.Lock()
	s.slides = append([]Slide(nil), slides...)
	s.index = 0
	s.mu.Unlock()
	s.notify()
}

// Subscribe returns a channel that receives a value after every change (changes
// coalesce: a slow reader sees at most one pending signal and must re-read the
// state), and a function that cancels the subscription.
func (s *State) Subscribe() (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	s.mu.Lock()
	s.subs[ch] = struct{}{}
	s.mu.Unlock()
	return ch, func() {
		s.mu.Lock()
		delete(s.subs, ch)
		s.mu.Unlock()
	}
}

func (s *State) notify() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.subs {
		select {
		case ch <- struct{}{}:
		default: // a signal is already pending
		}
	}
}
