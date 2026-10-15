package api_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/evobytes/lumen/internal/api"
	"github.com/evobytes/lumen/internal/host"
	"github.com/evobytes/lumen/internal/host/hosttest"
	"github.com/evobytes/lumen/internal/state"
)

var twoScreens = []host.Screen{
	{ID: `\\.\DISPLAY1`, Rect: host.Rect{X: 0, Y: 0, W: 2256, H: 1504}, DPI: 96, Scale: 1, Primary: true},
	{ID: `\\.\DISPLAY2`, Rect: host.Rect{X: -2560, Y: -90, W: 2560, H: 1600}, DPI: 96, Scale: 1},
}

func newServer(t *testing.T, token string) (*api.Server, *hosttest.Fake, http.Handler) {
	t.Helper()
	st := state.New([]state.Slide{{Text: "one"}, {Title: "Two", Text: "two"}, {Text: "three"}})
	srv := api.New(api.Options{
		State: st, Token: token, Placement: host.PlaceWin32,
		Media: fstest.MapFS{"clip.webm": {Data: []byte("x")}},
	})
	fake := hosttest.New(twoScreens...)
	srv.SetHost(fake)
	return srv, fake, srv.Handler()
}

func do(t *testing.T, h http.Handler, method, path, body string, hdr ...string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for i := 0; i+1 < len(hdr); i += 2 {
		req.Header.Set(hdr[i], hdr[i+1])
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// statusOf performs a real HTTP request against a test server and returns the status, closing the body.
func statusOf(t *testing.T, method, url string) int {
	t.Helper()
	res := fetch(t, method, url) //nolint:bodyclose // fetch closes the body when the test ends
	_ = res.Body.Close()
	return res.StatusCode
}

// fetch performs a real HTTP request against a test server with a context; the body is closed on
// cleanup, which lets streaming responses (server-sent events) be read while the test runs.
func fetch(t *testing.T, method, url string) *http.Response {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = res.Body.Close() })
	return res
}

func snap(t *testing.T, rec *httptest.ResponseRecorder) state.Snapshot {
	t.Helper()
	var s state.Snapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &s); err != nil {
		t.Fatalf("decode snapshot: %v\n%s", err, rec.Body.String())
	}
	return s
}

func TestStatusAndNavigation(t *testing.T) {
	t.Parallel()
	_, _, h := newServer(t, "")
	s := snap(t, do(t, h, "GET", "/api/v1/status", ""))
	if s.Index != 0 || s.Count != 3 || s.Slide == nil || s.Slide.Text != "one" || s.Slides[1].Label != "Two" {
		t.Fatalf("status = %+v", s)
	}
	for _, step := range []struct {
		path string
		want int
	}{{"/api/v1/next", 1}, {"/api/v1/next", 2}, {"/api/v1/next", 2}, {"/api/v1/prev", 1}} {
		if got := snap(t, do(t, h, "POST", step.path, "")).Index; got != step.want {
			t.Fatalf("after POST %s: index %d, want %d", step.path, got, step.want)
		}
	}
}

func TestShow(t *testing.T) {
	t.Parallel()
	_, _, h := newServer(t, "")
	if got := snap(t, do(t, h, "POST", "/api/v1/slides/2/show", "")).Index; got != 2 {
		t.Fatalf("show 2: index %d", got)
	}
	tests := []struct {
		path string
		want int
	}{{"/api/v1/slides/9/show", 404}, {"/api/v1/slides/-1/show", 404}, {"/api/v1/slides/x/show", 400}}
	for _, tt := range tests {
		rec := do(t, h, "POST", tt.path, "")
		if rec.Code != tt.want {
			t.Errorf("POST %s = %d, want %d", tt.path, rec.Code, tt.want)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
			t.Errorf("POST %s content type %q", tt.path, ct)
		}
	}
}

func TestBlank(t *testing.T) {
	t.Parallel()
	_, _, h := newServer(t, "")
	tests := []struct {
		body string
		want bool
	}{{"", true}, {"", false}, {`{"blank":true}`, true}, {`{"blank":true}`, true}, {`{"blank":false}`, false}, {`{}`, true}}
	for _, tt := range tests {
		if got := snap(t, do(t, h, "POST", "/api/v1/blank", tt.body)).Blanked; got != tt.want {
			t.Fatalf("blank body %q: blanked=%v, want %v", tt.body, got, tt.want)
		}
	}
	if rec := do(t, h, "POST", "/api/v1/blank", `{"blank":"yes"}`); rec.Code != 400 {
		t.Errorf("bad type = %d, want 400", rec.Code)
	}
	if rec := do(t, h, "POST", "/api/v1/blank", `{"nope":1}`); rec.Code != 400 {
		t.Errorf("unknown field = %d, want 400", rec.Code)
	}
}

func TestPutSlides(t *testing.T) {
	t.Parallel()
	_, _, h := newServer(t, "")
	rec := do(t, h, "PUT", "/api/v1/slides", `{"slides":[{"title":"T","text":"x"}]}`)
	s := snap(t, rec)
	if rec.Code != 200 || s.Count != 1 || s.Index != 0 || s.Slides[0].Label != "T" {
		t.Fatalf("PUT slides = %d %+v", rec.Code, s)
	}
	big := `{"slides":[{"text":"` + strings.Repeat("x", 70<<10) + `"}]}`
	if rec := do(t, h, "PUT", "/api/v1/slides", big); rec.Code != 413 {
		t.Errorf("oversized slide = %d, want 413", rec.Code)
	}
	if rec := do(t, h, "PUT", "/api/v1/slides", `not json`); rec.Code != 400 {
		t.Errorf("garbage = %d, want 400", rec.Code)
	}
}

func TestScreensAndOutput(t *testing.T) {
	t.Parallel()
	_, fake, h := newServer(t, "")
	var screens []host.Screen
	rec := do(t, h, "GET", "/api/v1/screens", "")
	if err := json.Unmarshal(rec.Body.Bytes(), &screens); err != nil || len(screens) != 2 {
		t.Fatalf("screens = %d %v %s", rec.Code, err, rec.Body.String())
	}
	rec = do(t, h, "PUT", "/api/v1/output", `{"screen":"\\\\.\\DISPLAY2"}`)
	var rep host.Report
	if err := json.Unmarshal(rec.Body.Bytes(), &rep); err != nil || rec.Code != 200 {
		t.Fatalf("PUT output = %d %v %s", rec.Code, err, rec.Body.String())
	}
	if rep.Output.Rect != twoScreens[1].Rect || rep.Output.Monitor != `\\.\DISPLAY2` {
		t.Fatalf("output not on DISPLAY2: %+v", rep.Output)
	}
	if rec := do(t, h, "PUT", "/api/v1/output", `{"screen":null}`); rec.Code != 200 {
		t.Fatalf("float = %d", rec.Code)
	}
	if rec := do(t, h, "PUT", "/api/v1/output", `{"screen":"\\\\.\\DISPLAY9"}`); rec.Code != 422 {
		t.Errorf("unknown screen = %d, want 422", rec.Code)
	}
	if rec := do(t, h, "PUT", "/api/v1/output", `{"screen":"x","placement":"magic"}`); rec.Code != 400 {
		t.Errorf("bad placement = %d, want 400", rec.Code)
	}
	if got := strings.Join(fake.Calls, ","); !strings.Contains(got, "PlaceOutput:") || !strings.Contains(got, "FloatOutput") {
		t.Errorf("calls = %s", got)
	}
}

func TestHostNotReady(t *testing.T) {
	t.Parallel()
	srv := api.New(api.Options{State: state.New(nil)})
	h := srv.Handler()
	for _, p := range []string{"/api/v1/screens", "/api/v1/windows"} {
		if rec := do(t, h, "GET", p, ""); rec.Code != 503 {
			t.Errorf("GET %s = %d, want 503", p, rec.Code)
		}
	}
	if rec := do(t, h, "GET", "/api/v1/status", ""); rec.Code != 200 {
		t.Errorf("status must work before windows exist, got %d", rec.Code)
	}
}

func TestAuth(t *testing.T) {
	t.Parallel()
	_, _, h := newServer(t, "s3cret")
	tests := []struct {
		name string
		path string
		hdr  []string
		want int
	}{
		{"no token", "/api/v1/status", nil, 401},
		{"wrong token", "/api/v1/status", []string{"X-API-Token", "nope"}, 401},
		{"header token", "/api/v1/status", []string{"X-API-Token", "s3cret"}, 200},
		{"bearer token", "/api/v1/status", []string{"Authorization", "Bearer s3cret"}, 200},
		{"bearer wrong", "/api/v1/status", []string{"Authorization", "Bearer nope"}, 401},
		{"query token rejected on normal routes", "/api/v1/status?token=s3cret", nil, 401},
		{"healthz is open", "/healthz", nil, 200},
		{"pages are open", "/control", nil, 200},
		{"assets are open", "/assets/common.js", nil, 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := do(t, h, "GET", tt.path, "", tt.hdr...).Code; got != tt.want {
				t.Fatalf("GET %s = %d, want %d", tt.path, got, tt.want)
			}
		})
	}
}

func TestPagesAndHeaders(t *testing.T) {
	t.Parallel()
	_, _, h := newServer(t, "")
	for _, p := range []string{"/control", "/output", "/assets/output.js", "/assets/control.js", "/assets/app.css", "/media/clip.webm"} {
		rec := do(t, h, "GET", p, "")
		if rec.Code != 200 {
			t.Errorf("GET %s = %d", p, rec.Code)
		}
		if csp := rec.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "default-src 'self'") || strings.Contains(csp, "unsafe-inline") {
			t.Errorf("GET %s CSP = %q", p, csp)
		}
	}
	rec := do(t, h, "GET", "/", "")
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/control" {
		t.Errorf("GET / = %d -> %q", rec.Code, rec.Header().Get("Location"))
	}
	rec = do(t, h, "GET", "/?token=a%26b", "")
	if loc := rec.Header().Get("Location"); loc != "/control?token=a%26b" {
		t.Errorf("token must be escaped in the redirect, got %q", loc)
	}
	if rec := do(t, h, "GET", "/nope", ""); rec.Code != 404 {
		t.Errorf("unknown path = %d, want 404", rec.Code)
	}
	body := do(t, h, "GET", "/control", "").Body.String()
	if strings.Contains(body, "<script>") || strings.Contains(body, "onclick") || strings.Contains(body, "style=") {
		t.Error("control page must not contain inline script or style")
	}
}

func TestVersion(t *testing.T) {
	t.Parallel()
	_, _, h := newServer(t, "")
	var v map[string]string
	if err := json.Unmarshal(do(t, h, "GET", "/api/v1/version", "").Body.Bytes(), &v); err != nil || v["version"] == "" {
		t.Fatalf("version = %v %v", v, err)
	}
}

func TestReports(t *testing.T) {
	t.Parallel()
	srv, _, h := newServer(t, "")
	if rec := do(t, h, "POST", "/api/v1/report/media", `{"ok":true}`); rec.Code != 204 {
		t.Fatalf("POST report = %d", rec.Code)
	}
	if v, ok := srv.Reports().Get("media"); !ok || string(v) != `{"ok":true}` {
		t.Fatalf("stored = %q %v", v, ok)
	}
	if rec := do(t, h, "POST", "/api/v1/report/media", `not json`); rec.Code != 400 {
		t.Errorf("bad json = %d", rec.Code)
	}
	if rec := do(t, h, "POST", "/api/v1/report/a%20b", `{}`); rec.Code != 400 {
		t.Errorf("bad name = %d", rec.Code)
	}
	var all map[string]json.RawMessage
	if err := json.Unmarshal(do(t, h, "GET", "/api/v1/report", "").Body.Bytes(), &all); err != nil || len(all) != 1 {
		t.Fatalf("all = %v %v", all, err)
	}
}

func TestEventsStream(t *testing.T) {
	t.Parallel()
	srv, _, _ := newServer(t, "")
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)                                      // cleanups run last-in first-out: the stream's body closes before the server
	res := fetch(t, http.MethodGet, ts.URL+"/api/v1/events") //nolint:bodyclose // closed by the cleanup in fetch, after the stream is read
	if ct := res.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content type %q", ct)
	}
	events := make(chan string, 4)
	go func() {
		sc := bufio.NewScanner(res.Body)
		for sc.Scan() {
			if strings.HasPrefix(sc.Text(), "event:") {
				events <- sc.Text()
			}
		}
	}()
	wait := func() {
		t.Helper()
		select {
		case <-events:
		case <-time.After(2 * time.Second):
			t.Fatal("no event")
		}
	}
	wait() // initial
	statusOf(t, http.MethodPost, ts.URL+"/api/v1/next")
	wait() // after the change
}

func TestEventsTokenQueryOnlyOnEvents(t *testing.T) {
	t.Parallel()
	srv, _, _ := newServer(t, "s3cret")
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close) // cleanups run last-in first-out: the stream's body closes before the server
	// The first request opens a stream; read only its status, then drop it with the test.
	if got := statusOf(t, http.MethodGet, ts.URL+"/api/v1/events?token=s3cret"); got != http.StatusOK {
		t.Fatalf("events with query token = %d", got)
	}
	if got := statusOf(t, http.MethodGet, ts.URL+"/api/v1/events"); got != http.StatusUnauthorized {
		t.Fatalf("events without token = %d, want 401", got)
	}
}

func TestRoutesTable(t *testing.T) {
	t.Parallel()
	routes := api.Routes()
	if len(routes) < 10 {
		t.Fatalf("only %d routes", len(routes))
	}
	seen := map[string]bool{}
	for _, r := range routes {
		key := r.Method + " " + r.Path
		if seen[key] {
			t.Errorf("duplicate route %s", key)
		}
		seen[key] = true
		if r.Summary == "" {
			t.Errorf("route %s has no summary", key)
		}
		if strings.HasPrefix(r.Path, "/api/v1/") != r.Auth {
			t.Errorf("route %s: Auth=%v does not match its path", key, r.Auth)
		}
	}
}
