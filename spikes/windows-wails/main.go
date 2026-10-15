package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

func main() {
	port := flag.Int("port", 8765, "API/web port on 127.0.0.1 (0 = any free port)")
	selftest := flag.Bool("selftest", false, "run the automated window/monitor/media checks, write a report, exit")
	report := flag.String("report", "selftest-report.json", "self-test report path")
	demo := flag.String("demo", "", "place the output on this screen id (or 'auto' = first non-primary) and stay up")
	mode := flag.String("mode", "win32", "placement mode for -demo: win32 | wails")
	hold := flag.Duration("hold", 0, "with -demo: quit after this long (0 = run until closed)")
	flag.Parse()

	state := NewState()
	hr := &hostRef{}
	reports := &Reports{}

	ln, base, err := listen(*port)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	go func() {
		if err := http.Serve(ln, newHandler(state, hr, reports)); err != nil {
			log.Printf("http server stopped: %v", err)
		}
	}()
	log.Printf("API and pages on %s", base)

	app := application.New(application.Options{
		Name:        "lumen-spike",
		Description: "Lumen Windows-first spike",
		Assets:      application.AssetOptions{Handler: http.NotFoundHandler()},
		Mac:         application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
	})

	ctrl := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:   "control",
		Title:  "Lumen",
		Width:  1000,
		Height: 700,
		URL:    base + "/control",
	})

	outURL := base + "/output"
	switch {
	case *selftest:
		outURL += "?probe=1"
	case *demo != "":
		outURL += "?debug=1"
	}
	out := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name:             "output",
		Title:            "Lumen output",
		URL:              outURL,
		Width:            480,
		Height:           270,
		Frameless:        true,
		AlwaysOnTop:      true,
		Hidden:           true,
		BackgroundColour: application.NewRGB(0, 0, 0),
		Windows: application.WindowsWindow{
			DisableFramelessWindowDecorations: true, // no shadow / rounded corners on the output
			HiddenOnTaskbar:                   true,
		},
	})

	host := &wailsHost{app: app, ctrl: ctrl, out: out}

	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		go func() {
			// The native windows exist shortly after the app starts.
			deadline := time.Now().Add(10 * time.Second)
			for !host.ready() && time.Now().Before(deadline) {
				time.Sleep(50 * time.Millisecond)
			}
			if err := host.Attach(); err != nil {
				log.Printf("attach output to control: %v", err)
			}
			if err := host.FloatOutput(); err != nil {
				log.Printf("float output: %v", err)
			}
			application.InvokeSync(func() { out.Show() })
			hr.Set(host)

			switch {
			case *selftest:
				runSelfTest(host, reports, *report)
			case *demo != "":
				runDemo(host, *demo, PlaceMode(*mode), *hold)
			}
		}()
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
