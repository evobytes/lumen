package main

import "sync"

type Slide struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

// State is the single source of truth. Renderers (the output page, the control page, OBS, ...)
// are pure functions of it and subscribe to changes.
type State struct {
	mu      sync.Mutex
	slides  []Slide
	index   int
	blanked bool
	subs    map[chan struct{}]struct{}
}

func NewState() *State {
	return &State{
		slides: []Slide{
			{"Welcome", "Lumen spike\nOutput window is driven by server state"},
			{"Second slide", "Placed on the chosen monitor"},
			{"Third slide", "Thank you"},
		},
		subs: map[chan struct{}]struct{}{},
	}
}

type Status struct {
	Index   int     `json:"index"`
	Count   int     `json:"count"`
	Blanked bool    `json:"blanked"`
	Slide   *Slide  `json:"slide"`
	Slides  []Slide `json:"slides"`
}

func (s *State) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := Status{Index: s.index, Count: len(s.slides), Blanked: s.blanked, Slides: s.slides}
	if len(s.slides) > 0 {
		sl := s.slides[s.index]
		st.Slide = &sl
	}
	return st
}

func (s *State) Move(delta int) {
	s.mu.Lock()
	if n := s.index + delta; n >= 0 && n < len(s.slides) {
		s.index = n
	}
	s.mu.Unlock()
	s.notify()
}

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

func (s *State) Subscribe() (chan struct{}, func()) {
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
		default: // a notification is already pending; the subscriber will re-read state
		}
	}
}
