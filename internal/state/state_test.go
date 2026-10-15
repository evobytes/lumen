package state_test

import (
	"errors"
	"testing"
	"time"

	"github.com/evobytes/lumen/internal/state"
)

func three() *state.State {
	return state.New([]state.Slide{{Text: "a"}, {Text: "b"}, {Title: "T", Text: "c"}})
}

func TestNavigationStopsAtEnds(t *testing.T) {
	t.Parallel()
	s := three()
	s.Prev()
	if got := s.Snapshot().Index; got != 0 {
		t.Fatalf("prev at start: index %d", got)
	}
	s.Next()
	s.Next()
	s.Next()
	if got := s.Snapshot().Index; got != 2 {
		t.Fatalf("next past end: index %d", got)
	}
}

func TestGoto(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		index   int
		wantErr bool
	}{
		{"first", 0, false},
		{"last", 2, false},
		{"too high", 3, true},
		{"negative", -1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := three()
			err := s.Goto(tt.index)
			if tt.wantErr {
				if !errors.Is(err, state.ErrOutOfRange) {
					t.Fatalf("err = %v, want ErrOutOfRange", err)
				}
				return
			}
			if err != nil || s.Snapshot().Index != tt.index {
				t.Fatalf("err = %v, index = %d", err, s.Snapshot().Index)
			}
		})
	}
}

func TestBlank(t *testing.T) {
	t.Parallel()
	s := three()
	yes, no := true, false
	s.SetBlank(nil)
	if !s.Snapshot().Blanked {
		t.Fatal("toggle should blank")
	}
	s.SetBlank(nil)
	if s.Snapshot().Blanked {
		t.Fatal("second toggle should unblank")
	}
	s.SetBlank(&yes)
	s.SetBlank(&yes)
	if !s.Snapshot().Blanked {
		t.Fatal("set true twice stays blank")
	}
	s.SetBlank(&no)
	if s.Snapshot().Blanked {
		t.Fatal("set false unblanks")
	}
}

func TestSetSlidesResetsPositionAndCopies(t *testing.T) {
	t.Parallel()
	s := three()
	_ = s.Goto(2)
	in := []state.Slide{{Text: "x"}}
	s.SetSlides(in)
	in[0].Text = "mutated"
	snap := s.Snapshot()
	if snap.Index != 0 || snap.Count != 1 || snap.Slide.Text != "x" {
		t.Fatalf("snapshot = %+v", snap)
	}
}

func TestEmpty(t *testing.T) {
	t.Parallel()
	snap := state.New(nil).Snapshot()
	if snap.Slide != nil || snap.Count != 0 {
		t.Fatalf("empty snapshot = %+v", snap)
	}
}

func TestLabel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		slide state.Slide
		want  string
	}{
		{state.Slide{Text: "hello\nworld"}, "hello"},
		{state.Slide{Title: "T", Text: "hello"}, "T"},
		{state.Slide{Text: "  \n  "}, "(empty)"},
		{state.Slide{Text: "\n\nlate"}, "late"},
	}
	for _, tt := range tests {
		if got := tt.slide.Label(); got != tt.want {
			t.Errorf("Label(%+v) = %q, want %q", tt.slide, got, tt.want)
		}
	}
}

func TestSubscribeCoalescesAndCancels(t *testing.T) {
	t.Parallel()
	s := three()
	ch, cancel := s.Subscribe()
	s.Next()
	s.Next() // coalesces with the first signal
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatal("no notification")
	}
	select {
	case <-ch:
		t.Fatal("expected a single coalesced signal")
	default:
	}
	cancel()
	s.Next() // must not block or panic with the subscriber gone
}
