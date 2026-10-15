// Package app wires Lumen together: the web API server, the control and output
// windows (Wails), and the optional self-test and demo modes.
package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"github.com/evobytes/lumen/internal/api"
	"github.com/evobytes/lumen/internal/config"
	"github.com/evobytes/lumen/internal/host"
	"github.com/evobytes/lumen/internal/host/wailshost"
	"github.com/evobytes/lumen/internal/selftest"
	"github.com/evobytes/lumen/internal/state"
)

// ErrSelfTestFailed is returned by Run when the self-test verdict is a failure.
var ErrSelfTestFailed = errors.New("self-test failed")

// SelfTest asks Run to run the self-test, write its JSON report and exit.
type SelfTest struct {
	ReportPath string
	// Modes are the placement modes to exercise; default is every mode the platform supports.
	Modes []host.Placement
}

// Demo asks Run to park the output on a screen. Hold, if positive, quits afterwards.
type Demo struct {
	Screen    string
	Placement host.Placement
	Hold      time.Duration
}

// Options configures Run.
type Options struct {
	Config   config.Config
	Logger   *slog.Logger
	SelfTest *SelfTest
	Demo     *Demo
}

// Run starts the API server and the windows and blocks until the application quits.
func Run(ctx context.Context, o Options) error {
	cfg, log := o.Config, o.Logger
	if o.SelfTest != nil {
		cfg.Server.Listen = "127.0.0.1:0" // never collide with a real instance
	}

	var media fs.FS
	if o.SelfTest != nil {
		media = selftest.Media()
	}
	srv := api.New(api.Options{
		Logger:    log,
		State:     state.New(state.Sample()),
		Token:     cfg.Auth.Token,
		Placement: host.Placement(cfg.Output.Placement),
		Media:     media,
	})

	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", cfg.Server.Listen)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", cfg.Server.Listen, err)
	}
	base := "http://" + ln.Addr().String()
	// Long-lived event streams hang off the request contexts, which derive from BaseContext:
	// cancelling it ends them, so a graceful shutdown does not wait on open browsers.
	streams, endStreams := context.WithCancel(context.Background())
	httpSrv := &http.Server{
		Handler:           srv.Handler(),
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout.Std(),
		BaseContext:       func(net.Listener) context.Context { return streams },
	}
	go func() {
		if err := httpSrv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("web server stopped", "err", err)
		}
	}()
	log.Info("web API and pages", "url", base)
	defer func() {
		endStreams()
		sctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout.Std())
		defer cancel()
		if err := httpSrv.Shutdown(sctx); err != nil {
			log.Warn("web server shutdown", "err", err)
		}
	}()

	wapp := application.New(application.Options{
		Name:        "Lumen",
		Description: "Presentation projector",
		Assets:      application.AssetOptions{Handler: http.NotFoundHandler()}, // pages come from our own server
		Mac:         application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
	})

	outQuery := url.Values{}
	if o.SelfTest != nil {
		outQuery.Set("probe", "1")
	}
	if o.Demo != nil {
		outQuery.Set("debug", "1")
	}
	ctrl := wapp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:   "control",
		Title:  "Lumen",
		Width:  1000,
		Height: 700,
		URL:    pageURL(base, "/control", cfg.Auth.Token, nil),
	})
	out := wapp.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "output",
		Title:            "Lumen output",
		URL:              pageURL(base, "/output", cfg.Auth.Token, outQuery),
		Width:            480,
		Height:           270,
		Frameless:        true,
		AlwaysOnTop:      true,
		Hidden:           true,
		BackgroundColour: application.NewRGB(0, 0, 0),
		Windows: application.WindowsWindow{
			DisableFramelessWindowDecorations: true, // no shadow or rounded corners on the output
			HiddenOnTaskbar:                   true,
		},
	})
	h := wailshost.New(wapp, ctrl, out)

	// Re-assert the output placement once its page has loaded: the toolkit's own
	// first-show positioning can otherwise land after the first placement.
	out.OnWindowEvent(loadFinishedEvent(), func(*application.WindowEvent) {
		go func() {
			if err := h.Reapply(ctx); err != nil {
				log.Warn("re-apply output placement", "err", err)
			}
		}()
	})

	var verdict error
	done := make(chan struct{})
	wapp.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		go func() {
			defer close(done)
			startWindows(ctx, h, srv, cfg, log)
			switch {
			case o.SelfTest != nil:
				verdict = runSelfTest(ctx, h, srv, o.SelfTest, log)
				h.Quit()
			case o.Demo != nil:
				if err := selftest.Demo(ctx, h, o.Demo.Screen, o.Demo.Placement, o.Demo.Hold, log); err != nil {
					log.Error("demo", "err", err)
				}
			}
		}()
	})
	go func() { // a signal (Ctrl-C, service stop) ends the application
		<-ctx.Done()
		wapp.Quit()
	}()

	if err := wapp.Run(); err != nil {
		return fmt.Errorf("run windows: %w", err)
	}
	<-done
	return verdict
}

func runtimeIsWindows() bool { return runtime.GOOS == "windows" }

func pageURL(base, path, token string, extra url.Values) string {
	q := url.Values{}
	for k, v := range extra {
		q[k] = v
	}
	if token != "" {
		q.Set("token", token)
	}
	if len(q) == 0 {
		return base + path
	}
	return base + path + "?" + q.Encode()
}

// startWindows waits for the native windows, attaches the output to the control
// window, applies the configured initial placement, shows the output and opens
// the host routes of the API.
func startWindows(ctx context.Context, h *wailshost.Host, srv *api.Server, cfg config.Config, log *slog.Logger) {
	deadline := time.Now().Add(10 * time.Second)
	for !h.Ready() && time.Now().Before(deadline) {
		time.Sleep(50 * time.Millisecond)
	}
	if err := h.Attach(); err != nil {
		log.Warn("attach output to control window", "err", err)
	}
	initialOutput(ctx, h, cfg, log)
	h.ShowOutput()
	srv.SetHost(h)
}

// initialOutput applies output.screen: floating, auto (first non-primary), or a monitor id.
func initialOutput(ctx context.Context, h host.Host, cfg config.Config, log *slog.Logger) {
	how := host.Placement(cfg.Output.Placement)
	target := cfg.Output.Screen
	if target == "auto" {
		target = ""
		if screens, err := h.Screens(ctx); err == nil {
			for _, s := range screens {
				if !s.Primary {
					target = s.ID
					break
				}
			}
		}
	}
	if target != "" && target != "floating" {
		err := h.PlaceOutput(ctx, target, how)
		if err == nil {
			return
		}
		log.Warn("place output; falling back to the floating window", "screen", target, "err", err)
	}
	if err := h.FloatOutput(ctx); err != nil {
		log.Warn("float output", "err", err)
	}
}

func runSelfTest(ctx context.Context, h host.Host, srv *api.Server, st *SelfTest, log *slog.Logger) error {
	modes := st.Modes
	if len(modes) == 0 {
		modes = []host.Placement{host.PlaceToolkit}
		if runtimeIsWindows() {
			modes = []host.Placement{host.PlaceWin32, host.PlaceToolkit}
		}
	}
	rep, err := selftest.Run(ctx, h, srv.Reports(), selftest.Options{Modes: modes, Logger: log})
	if err != nil {
		return fmt.Errorf("self-test: %w", err)
	}
	b, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return fmt.Errorf("encode self-test report: %w", err)
	}
	if err := os.WriteFile(st.ReportPath, b, 0o600); err != nil {
		return fmt.Errorf("write self-test report: %w", err)
	}
	log.Info("self-test finished", "report", st.ReportPath, "pass", rep.Summary.Pass, "placements", rep.Summary.PlacementsExact, "media", rep.Summary.MediaReceived)
	if !rep.Summary.Pass {
		return ErrSelfTestFailed
	}
	return nil
}
