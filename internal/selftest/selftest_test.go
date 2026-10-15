package selftest_test

import (
	"context"
	"io/fs"
	"testing"
	"time"

	"github.com/evobytes/lumen/internal/api"
	"github.com/evobytes/lumen/internal/host"
	"github.com/evobytes/lumen/internal/host/hosttest"
	"github.com/evobytes/lumen/internal/selftest"
)

var screens = []host.Screen{
	{ID: "A", Rect: host.Rect{X: 0, Y: 0, W: 1920, H: 1080}, Primary: true},
	{ID: "B", Rect: host.Rect{X: -2560, Y: -90, W: 2560, H: 1600}},
}

func fast() selftest.Options {
	return selftest.Options{
		Modes:     []host.Placement{host.PlaceWin32, host.PlaceToolkit},
		Repeats:   2,
		Samples:   []time.Duration{time.Millisecond, 2 * time.Millisecond},
		MediaWait: time.Second,
	}
}

func TestRunPasses(t *testing.T) {
	t.Parallel()
	reports := &api.Reports{}
	reports.Set("media", []byte(`{"ok":true}`))
	rep, err := selftest.Run(context.Background(), hosttest.New(screens...), reports, fast())
	if err != nil {
		t.Fatal(err)
	}
	if got := len(rep.Placements); got != 2*2*2 {
		t.Fatalf("placements = %d, want 8", got)
	}
	if !rep.Summary.Pass || rep.Summary.PlacementsExact != "8/8" || !rep.Summary.MediaReceived || rep.Summary.Monitors != 2 {
		t.Fatalf("summary = %+v", rep.Summary)
	}
	for _, p := range rep.Placements {
		if !p.Exact || p.Monitor != p.Screen || len(p.Samples) != 2 {
			t.Errorf("placement %+v", p)
		}
	}
}

func TestRunRecordsMissingMedia(t *testing.T) {
	t.Parallel()
	o := fast()
	o.MediaWait = 20 * time.Millisecond
	rep, err := selftest.Run(context.Background(), hosttest.New(screens...), &api.Reports{}, o)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Summary.MediaReceived || rep.MediaError == "" {
		t.Fatalf("expected a media error, got %+v / %q", rep.Summary, rep.MediaError)
	}
	if rep.Summary.Pass {
		t.Error("no media report means the page never loaded: the verdict must fail")
	}
}

// misplacing is a host whose output never lands on the screen it was told to use.
type misplacing struct{ *hosttest.Fake }

func (m misplacing) PlaceOutput(ctx context.Context, id string, how host.Placement) error {
	_ = m.Fake.PlaceOutput(ctx, id, how)
	return m.FloatOutput(ctx) // moves it back: the placement did not stick
}

func TestRunDetectsMisplacement(t *testing.T) {
	t.Parallel()
	o := fast()
	reports := &api.Reports{}
	reports.Set("media", []byte(`{}`))
	rep, err := selftest.Run(context.Background(), misplacing{hosttest.New(screens...)}, reports, o)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Summary.Pass || rep.Summary.PlacementsExact != "0/8" {
		t.Fatalf("a misplaced output must fail: %+v", rep.Summary)
	}
}

func TestRunStopsOnCancel(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := selftest.Run(ctx, hosttest.New(screens...), &api.Reports{}, fast()); err == nil {
		t.Fatal("expected a context error")
	}
}

func TestDemo(t *testing.T) {
	t.Parallel()
	fake := hosttest.New(screens...)
	if err := selftest.Demo(context.Background(), fake, "auto", host.PlaceWin32, time.Millisecond, nil); err != nil {
		t.Fatal(err)
	}
	if !fake.Quitted() {
		t.Error("demo with a hold should quit")
	}
	if err := selftest.Demo(context.Background(), hosttest.New(screens[0]), "auto", host.PlaceWin32, 0, nil); err == nil {
		t.Error("auto with only a primary screen should fail")
	}
}

func TestMediaClipsEmbedded(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"h264_aac.mp4", "vp9_opus.webm", "hevc.mp4", "av1.mp4", "vp9_alpha.webm"} {
		if _, err := fs.Stat(selftest.Media(), name); err != nil {
			t.Errorf("missing clip %s: %v", name, err)
		}
	}
}

// limited is a host on a platform that cannot place windows exactly (for example Wayland).
type limited struct{ *hosttest.Fake }

func (limited) Capabilities() host.Capabilities { return host.Capabilities{} }

func (l limited) PlaceOutput(ctx context.Context, id string, how host.Placement) error {
	_ = l.Fake.PlaceOutput(ctx, id, how)
	return l.FloatOutput(ctx) // the platform ignores the request
}

func TestPlacementDoesNotGateWhereThePlatformCannotPlace(t *testing.T) {
	t.Parallel()
	reports := &api.Reports{}
	reports.Set("media", []byte(`{}`))
	rep, err := selftest.Run(context.Background(), limited{hosttest.New(screens...)}, reports, fast())
	if err != nil {
		t.Fatal(err)
	}
	if rep.Summary.PlacementGated || !rep.Summary.Pass || rep.Summary.PlacementsExact != "0/8" {
		t.Fatalf("placements must be recorded but not gate: %+v", rep.Summary)
	}
	// ...but a page that never loaded still fails, even where nothing is gated.
	rep, err = selftest.Run(context.Background(), limited{hosttest.New(screens...)}, &api.Reports{}, func() selftest.Options {
		o := fast()
		o.MediaWait = 10 * time.Millisecond
		return o
	}())
	if err != nil {
		t.Fatal(err)
	}
	if rep.Summary.Pass {
		t.Fatal("no page, no pass")
	}
}
