// Package config loads and validates Lumen configuration.
//
// Precedence (low to high): defaults, config file, LUMEN_* environment, flags.
// Flags are applied by the caller after Load, then Validate is called.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// EnvPrefix starts the name of every environment variable that sets a config key.
const EnvPrefix = "LUMEN_"

// Duration is a time.Duration that reads and writes as text ("5s").
type Duration time.Duration

// Std returns the value as a time.Duration.
func (d Duration) Std() time.Duration { return time.Duration(d) }

// UnmarshalText implements encoding.TextUnmarshaler.
func (d *Duration) UnmarshalText(b []byte) error {
	v, err := time.ParseDuration(strings.TrimSpace(string(b)))
	if err != nil {
		return fmt.Errorf("duration %q: use a number with a unit, for example \"5s\" or \"1m30s\"", string(b))
	}
	*d = Duration(v)
	return nil
}

// MarshalText implements encoding.TextMarshaler.
func (d Duration) MarshalText() ([]byte, error) { return []byte(time.Duration(d).String()), nil }

// Config is the full application configuration.
type Config struct {
	Server Server `toml:"server"`
	Auth   Auth   `toml:"auth"`
	Output Output `toml:"output"`
	Log    Log    `toml:"log"`
}

// Server configures the embedded web API and pages.
//
// Field tags: toml is the key; doc is the description rendered into
// docs/configuration.md and the shipped config file (scripts/docgen).
type Server struct {
	Listen            string   `toml:"listen" doc:"Listen address as host:port for the web API, the control page and the output page. Keep it on loopback unless auth.token is set; use port 0 for any free port."`
	ReadHeaderTimeout Duration `toml:"read_header_timeout" doc:"Maximum time to read the request headers."`
	ShutdownTimeout   Duration `toml:"shutdown_timeout" doc:"Grace period to drain in-flight requests when quitting."`
}

// Auth configures API access.
type Auth struct {
	Token string `toml:"token" doc:"Shared secret required on /api/v1 as the X-API-Token header or an Authorization: Bearer header. Empty disables the check, which is allowed only while server.listen is on loopback. Prefer the LUMEN_AUTH_TOKEN environment variable to writing it in this file."`
}

// Output configures the audience-facing output window.
type Output struct {
	Screen    string `toml:"screen" doc:"Where the output window starts: auto (the first non-primary monitor, else a floating window), floating, or a monitor id such as \\\\.\\DISPLAY2 as GET /api/v1/screens shows it."`
	Placement string `toml:"placement" doc:"How the output window is moved onto a monitor: win32 (set the native window rectangle directly; Windows only; the default there) or toolkit (the windowing toolkit's own API; the default elsewhere)."`
}

// Log configures slog.
type Log struct {
	Level  string `toml:"level" doc:"Minimum level: debug, info, warn or error."`
	Format string `toml:"format" doc:"Output format on stderr: json or text."`
}

const redacted = "<redacted>"

// DefaultPath is the platform's standard per-user config file location.
func DefaultPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return filepath.Join("dev", "config.toml")
	}
	return filepath.Join(dir, "Lumen", "config.toml")
}

// Default returns the compiled-in, credential-free defaults for this platform.
func Default() Config { return DefaultFor(runtime.GOOS) }

// DefaultFor returns the defaults for goos. Docs generation uses it to render
// the shipped config file, so it must not read the environment.
func DefaultFor(goos string) Config {
	placement := string(placementToolkit)
	if goos == "windows" {
		placement = string(placementWin32)
	}
	return Config{
		Server: Server{
			Listen:            "127.0.0.1:8765",
			ReadHeaderTimeout: Duration(5 * time.Second),
			ShutdownTimeout:   Duration(5 * time.Second),
		},
		Output: Output{Screen: "auto", Placement: placement},
		Log:    Log{Level: "info", Format: "text"},
	}
}

type placementMode string

const (
	placementWin32   placementMode = "win32"
	placementToolkit placementMode = "toolkit"
)

// Load builds a Config from defaults, the file at path, and the environment.
// An empty path skips the file. A missing file is an error only if explicit.
func Load(path string, explicit bool, getenv func(string) string) (Config, error) {
	cfg := Default()
	if path != "" {
		b, err := os.ReadFile(path) //nolint:gosec // the config path is operator-supplied by design
		switch {
		case errors.Is(err, os.ErrNotExist) && !explicit:
			// fall through with defaults
		case err != nil:
			return Config{}, fmt.Errorf("read config: %w", err)
		default:
			dec := toml.NewDecoder(bytes.NewReader(b)).DisallowUnknownFields()
			if err := dec.Decode(&cfg); err != nil {
				var strict *toml.StrictMissingError
				if errors.As(err, &strict) {
					return Config{}, fmt.Errorf("parse %s: unknown key(s):\n%s", path, strict.String())
				}
				return Config{}, fmt.Errorf("parse %s: %w", path, err)
			}
		}
	}
	if err := applyEnv(&cfg, getenv); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

type envSetter func(*Config, string) error

func str(f func(*Config) *string) envSetter {
	return func(c *Config, v string) error { *f(c) = v; return nil }
}

func dur(f func(*Config) *Duration) envSetter {
	return func(c *Config, v string) error { return f(c).UnmarshalText([]byte(v)) }
}

var envKeys = map[string]envSetter{
	EnvPrefix + "SERVER_LISTEN":              str(func(c *Config) *string { return &c.Server.Listen }),
	EnvPrefix + "SERVER_READ_HEADER_TIMEOUT": dur(func(c *Config) *Duration { return &c.Server.ReadHeaderTimeout }),
	EnvPrefix + "SERVER_SHUTDOWN_TIMEOUT":    dur(func(c *Config) *Duration { return &c.Server.ShutdownTimeout }),
	EnvPrefix + "AUTH_TOKEN":                 str(func(c *Config) *string { return &c.Auth.Token }),
	EnvPrefix + "OUTPUT_SCREEN":              str(func(c *Config) *string { return &c.Output.Screen }),
	EnvPrefix + "OUTPUT_PLACEMENT":           str(func(c *Config) *string { return &c.Output.Placement }),
	EnvPrefix + "LOG_LEVEL":                  str(func(c *Config) *string { return &c.Log.Level }),
	EnvPrefix + "LOG_FORMAT":                 str(func(c *Config) *string { return &c.Log.Format }),
}

func applyEnv(c *Config, getenv func(string) string) error {
	var errs []error
	for name, set := range envKeys {
		if v := getenv(name); v != "" {
			if err := set(c, v); err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", name, err))
			}
		}
	}
	return errors.Join(errs...)
}

// Validate reports every problem with the configuration at once.
func (c Config) Validate() error {
	var errs []error
	host, port, err := net.SplitHostPort(c.Server.Listen)
	switch {
	case err != nil:
		errs = append(errs, fmt.Errorf("server.listen %q: %w", c.Server.Listen, err))
	default:
		if p, perr := strconv.Atoi(port); perr != nil || p < 0 || p > 65535 {
			errs = append(errs, fmt.Errorf("server.listen port %q: want 0-65535", port))
		}
		if c.Auth.Token == "" && !isLoopback(host) {
			errs = append(errs, fmt.Errorf("server.listen %q is not loopback: set auth.token (or LUMEN_AUTH_TOKEN)", c.Server.Listen))
		}
	}
	if c.Server.ReadHeaderTimeout <= 0 {
		errs = append(errs, errors.New("server.read_header_timeout must be positive"))
	}
	if c.Server.ShutdownTimeout <= 0 {
		errs = append(errs, errors.New("server.shutdown_timeout must be positive"))
	}
	switch c.Output.Placement {
	case string(placementWin32), string(placementToolkit):
	default:
		errs = append(errs, fmt.Errorf("output.placement %q: want win32 or toolkit", c.Output.Placement))
	}
	if c.Output.Placement == string(placementWin32) && runtime.GOOS != "windows" {
		errs = append(errs, errors.New("output.placement win32 is only available on Windows"))
	}
	if c.Output.Screen == "" {
		errs = append(errs, errors.New("output.screen must not be empty (auto, floating or a monitor id)"))
	}
	switch c.Log.Level {
	case "debug", "info", "warn", "error":
	default:
		errs = append(errs, fmt.Errorf("log.level %q: want debug, info, warn or error", c.Log.Level))
	}
	if c.Log.Format != "json" && c.Log.Format != "text" {
		errs = append(errs, fmt.Errorf("log.format %q: want json or text", c.Log.Format))
	}
	return errors.Join(errs...)
}

func isLoopback(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// Redacted returns the configuration as TOML with secrets masked, for -check-config.
func (c Config) Redacted() (string, error) {
	if c.Auth.Token != "" {
		c.Auth.Token = redacted
	}
	b, err := toml.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("render config: %w", err)
	}
	return string(b), nil
}
