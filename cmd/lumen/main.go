// Command lumen is the Lumen presentation projector: a control window, a
// floating output window and an embedded web API.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/evobytes/lumen/internal/app"
	"github.com/evobytes/lumen/internal/config"
	"github.com/evobytes/lumen/internal/host"
	"github.com/evobytes/lumen/internal/logging"
	"github.com/evobytes/lumen/internal/version"
)

const appName = "lumen"

func main() {
	os.Exit(run(os.Args[1:], os.Getenv, os.Stdout, os.Stderr, app.Run))
}

// runner starts the application; it is injected so run can be tested without windows.
type runner func(context.Context, app.Options) error

// run is main with injectable I/O so it can be tested.
func run(args []string, getenv func(string) string, stdout, stderr io.Writer, start runner) int {
	fs := flag.NewFlagSet(appName, flag.ContinueOnError)
	fs.SetOutput(stderr)
	var (
		cfgPath   = fs.String("config", config.DefaultPath(), "config file path")
		listen    = fs.String("listen", "", "web API listen address (overrides config)")
		logLevel  = fs.String("log-level", "", "log level (overrides config)")
		showVer   = fs.Bool("version", false, "print version and exit")
		checkOnly = fs.Bool("check-config", false, "validate config, print it (secrets redacted) and exit")

		// Developer and CI modes.
		selftest  = fs.Bool("selftest", false, "run the automated window/monitor/media checks, write a report and exit")
		report    = fs.String("report", "selftest-report.json", "self-test report path")
		demo      = fs.String("demo", "", "park the output on this monitor id (or 'auto') and stay up")
		placement = fs.String("placement", "", "placement mode for -demo: win32 or toolkit (default from config)")
		hold      = fs.Duration("hold", 0, "with -demo: quit after this long (0 runs until closed)")
	)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if *showVer {
		fmt.Fprintln(stdout, version.String())
		return 0
	}

	explicit := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "config" {
			explicit = true
		}
	})
	cfg, err := config.Load(*cfgPath, explicit, getenv)
	if err != nil {
		fmt.Fprintln(stderr, "config:", err)
		return 1
	}
	if *listen != "" {
		cfg.Server.Listen = *listen
	}
	if *logLevel != "" {
		cfg.Log.Level = *logLevel
	}
	if err := cfg.Validate(); err != nil {
		fmt.Fprintln(stderr, "config invalid:\n"+err.Error())
		return 1
	}
	if *checkOnly {
		out, err := cfg.Redacted()
		if err != nil {
			fmt.Fprintln(stderr, "config:", err)
			return 1
		}
		fmt.Fprint(stdout, out)
		return 0
	}
	if *selftest && *demo != "" {
		fmt.Fprintln(stderr, "-selftest and -demo are mutually exclusive")
		return 2
	}

	log, err := logging.New(stderr, cfg.Log.Level, cfg.Log.Format)
	if err != nil {
		fmt.Fprintln(stderr, "logging:", err)
		return 1
	}
	opts := app.Options{Config: cfg, Logger: log}
	if *selftest {
		opts.SelfTest = &app.SelfTest{ReportPath: *report}
	}
	if *demo != "" {
		how := host.Placement(cfg.Output.Placement)
		if *placement != "" {
			how = host.Placement(*placement)
		}
		opts.Demo = &app.Demo{Screen: *demo, Placement: how, Hold: *hold}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	log.Info("starting", "version", version.String(), "config", *cfgPath)
	if err := start(ctx, opts); err != nil {
		log.Error("lumen stopped", "err", err)
		return 1
	}
	return 0
}
