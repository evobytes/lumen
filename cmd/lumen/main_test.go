package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/evobytes/lumen/internal/app"
)

func noEnv(string) string { return "" }

// emptyConfig writes an empty config file: tests must not depend on (or read) the
// developer's real per-user config, and an explicit missing path is an error.
func emptyConfig(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(p, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

type result struct {
	code           int
	stdout, stderr string
	started        *app.Options
}

func runWith(t *testing.T, args []string, getenv func(string) string, startErr error) result {
	t.Helper()
	var out, errb bytes.Buffer
	var got *app.Options
	start := func(_ context.Context, o app.Options) error { got = &o; return startErr }
	code := run(args, getenv, &out, &errb, start)
	return result{code, out.String(), errb.String(), got}
}

func TestVersion(t *testing.T) {
	t.Parallel()
	r := runWith(t, []string{"-version"}, noEnv, nil)
	if r.code != 0 || !strings.HasPrefix(r.stdout, "lumen ") || r.started != nil {
		t.Fatalf("%+v", r)
	}
}

func TestBadFlag(t *testing.T) {
	t.Parallel()
	if r := runWith(t, []string{"-nope"}, noEnv, nil); r.code != 2 {
		t.Fatalf("code = %d, want 2", r.code)
	}
}

func TestCheckConfigRedactsToken(t *testing.T) {
	t.Parallel()
	env := func(k string) string {
		if k == "LUMEN_AUTH_TOKEN" {
			return "hunter2"
		}
		return ""
	}
	r := runWith(t, []string{"-check-config", "-config", emptyConfig(t)}, env, nil)
	if r.code != 0 || strings.Contains(r.stdout, "hunter2") || !strings.Contains(r.stdout, "<redacted>") || r.started != nil {
		t.Fatalf("%+v", r)
	}
}

func TestInvalidConfigIsRejected(t *testing.T) {
	t.Parallel()
	r := runWith(t, []string{"-listen", "0.0.0.0:8765", "-config", emptyConfig(t)}, noEnv, nil)
	if r.code != 1 || !strings.Contains(r.stderr, "auth.token") || r.started != nil {
		t.Fatalf("non-loopback without a token must be rejected: %+v", r)
	}
}

func TestExplicitMissingConfigFails(t *testing.T) {
	t.Parallel()
	r := runWith(t, []string{"-config", filepath.Join(t.TempDir(), "absent.toml")}, noEnv, nil)
	if r.code != 1 || !strings.Contains(r.stderr, "config:") {
		t.Fatalf("%+v", r)
	}
}

func TestFlagsOverrideFile(t *testing.T) {
	t.Parallel()
	file := filepath.Join(t.TempDir(), "c.toml")
	if err := os.WriteFile(file, []byte("[server]\nlisten = \"127.0.0.1:9000\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r := runWith(t, []string{"-config", file, "-listen", "127.0.0.1:9001", "-log-level", "warn"}, noEnv, nil)
	if r.code != 0 || r.started == nil {
		t.Fatalf("%+v", r)
	}
	if got := r.started.Config.Server.Listen; got != "127.0.0.1:9001" {
		t.Errorf("listen = %q, flag should win", got)
	}
	if got := r.started.Config.Log.Level; got != "warn" {
		t.Errorf("level = %q", got)
	}
}

func TestSelfTestAndDemoModes(t *testing.T) {
	t.Parallel()
	absent := emptyConfig(t)
	r := runWith(t, []string{"-selftest", "-report", "r.json", "-config", absent}, noEnv, nil)
	if r.started == nil || r.started.SelfTest == nil || r.started.SelfTest.ReportPath != "r.json" {
		t.Fatalf("%+v", r)
	}
	r = runWith(t, []string{"-demo", "auto", "-hold", "5s", "-config", absent}, noEnv, nil)
	if r.started == nil || r.started.Demo == nil || r.started.Demo.Screen != "auto" || r.started.Demo.Hold.Seconds() != 5 {
		t.Fatalf("%+v", r)
	}
	if r := runWith(t, []string{"-selftest", "-demo", "auto", "-config", absent}, noEnv, nil); r.code != 2 {
		t.Errorf("-selftest with -demo = %d, want 2", r.code)
	}
}

func TestRunnerErrorBecomesExitOne(t *testing.T) {
	t.Parallel()
	r := runWith(t, []string{"-config", emptyConfig(t)}, noEnv, errors.New("boom"))
	if r.code != 1 || !strings.Contains(r.stderr, "boom") {
		t.Fatalf("%+v", r)
	}
}
