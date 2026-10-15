package config_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/evobytes/lumen/internal/config"
)

func noEnv(string) string { return "" }

func env(kv map[string]string) func(string) string { return func(k string) string { return kv[k] } }

func TestDefaultsAreValid(t *testing.T) {
	t.Parallel()
	if err := config.Default().Validate(); err != nil {
		t.Fatalf("default config invalid: %v", err)
	}
}

func TestDefaultForPlacement(t *testing.T) {
	t.Parallel()
	tests := map[string]string{"windows": "win32", "linux": "toolkit", "darwin": "toolkit"}
	for goos, want := range tests {
		if got := config.DefaultFor(goos).Output.Placement; got != want {
			t.Errorf("DefaultFor(%q).Output.Placement = %q, want %q", goos, got, want)
		}
	}
}

func TestLoadPrecedence(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	file := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(file, []byte("[server]\nlisten = \"127.0.0.1:9000\"\n[log]\nlevel = \"debug\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(file, true, env(map[string]string{"LUMEN_LOG_LEVEL": "warn"}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Listen != "127.0.0.1:9000" {
		t.Errorf("file should beat default: listen = %q", cfg.Server.Listen)
	}
	if cfg.Log.Level != "warn" {
		t.Errorf("env should beat file: level = %q", cfg.Log.Level)
	}
	if cfg.Log.Format != "text" {
		t.Errorf("untouched key keeps its default: format = %q", cfg.Log.Format)
	}
}

func TestLoadMissingFile(t *testing.T) {
	t.Parallel()
	missing := filepath.Join(t.TempDir(), "nope.toml")
	if _, err := config.Load(missing, false, noEnv); err != nil {
		t.Errorf("implicit missing file should fall back to defaults: %v", err)
	}
	if _, err := config.Load(missing, true, noEnv); err == nil {
		t.Error("explicit missing file should be an error")
	}
}

func TestLoadRejectsUnknownKeys(t *testing.T) {
	t.Parallel()
	file := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(file, []byte("[server]\nlisen = \"x\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := config.Load(file, true, noEnv)
	if err == nil || !strings.Contains(err.Error(), "unknown key") {
		t.Fatalf("err = %v, want unknown key error", err)
	}
}

func TestLoadBadEnvDuration(t *testing.T) {
	t.Parallel()
	_, err := config.Load("", false, env(map[string]string{"LUMEN_SERVER_SHUTDOWN_TIMEOUT": "soon"}))
	if err == nil || !strings.Contains(err.Error(), "LUMEN_SERVER_SHUTDOWN_TIMEOUT") {
		t.Fatalf("err = %v, want the variable named", err)
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		mutate  func(*config.Config)
		wantErr string
	}{
		{"ok loopback no token", func(*config.Config) {}, ""},
		{"ok port zero", func(c *config.Config) { c.Server.Listen = "127.0.0.1:0" }, ""},
		{"ok ipv6 loopback", func(c *config.Config) { c.Server.Listen = "[::1]:8765" }, ""},
		{"non-loopback needs token", func(c *config.Config) { c.Server.Listen = "0.0.0.0:8765" }, "auth.token"},
		{"non-loopback with token", func(c *config.Config) { c.Server.Listen = "0.0.0.0:8765"; c.Auth.Token = "s3cret" }, ""},
		{"bad listen", func(c *config.Config) { c.Server.Listen = "nonsense" }, "server.listen"},
		{"bad port", func(c *config.Config) { c.Server.Listen = "127.0.0.1:70000" }, "port"},
		{"bad level", func(c *config.Config) { c.Log.Level = "loud" }, "log.level"},
		{"bad format", func(c *config.Config) { c.Log.Format = "xml" }, "log.format"},
		{"bad placement", func(c *config.Config) { c.Output.Placement = "magic" }, "output.placement"},
		{"empty screen", func(c *config.Config) { c.Output.Screen = "" }, "output.screen"},
		{"zero timeout", func(c *config.Config) { c.Server.ShutdownTimeout = 0 }, "shutdown_timeout"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := config.Default()
			tt.mutate(&cfg)
			err := cfg.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want it to mention %q", err, tt.wantErr)
			}
		})
	}
}

func TestValidateWin32OnlyOnWindows(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	cfg.Output.Placement = "win32"
	err := cfg.Validate()
	if runtime.GOOS == "windows" && err != nil {
		t.Fatalf("win32 placement must be valid on windows: %v", err)
	}
	if runtime.GOOS != "windows" && (err == nil || !strings.Contains(err.Error(), "Windows")) {
		t.Fatalf("win32 placement must be rejected off windows, got %v", err)
	}
}

func TestRedactedHidesToken(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	cfg.Auth.Token = "super-secret"
	out, err := cfg.Redacted()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "super-secret") || !strings.Contains(out, "<redacted>") {
		t.Fatalf("token not redacted:\n%s", out)
	}
}
