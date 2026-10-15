// Package api serves the web API (/api/v1), the control and output pages and
// the server-sent event stream that keeps every renderer in step with the state.
//
// The route table is data (see Routes) so documentation is generated from the
// same list the router is built from.
package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/evobytes/lumen/internal/host"
	"github.com/evobytes/lumen/internal/state"
	"github.com/evobytes/lumen/internal/ui"
	"github.com/evobytes/lumen/internal/version"
)

const (
	maxBodyBytes  = 1 << 20
	maxSlides     = 10_000
	maxSlideBytes = 64 << 10
	maxReportName = 64
)

// Options configures a Server.
type Options struct {
	Logger *slog.Logger // defaults to a discarding logger
	State  *state.State
	// Token is the shared secret for /api/v1; empty disables the check.
	Token string
	// Placement is used when PUT /api/v1/output does not name one.
	Placement host.Placement
	// Media, when set, is served read-only at /media/ (the self-test clips).
	Media fs.FS
}

// Server is the HTTP handler set. Create it with New.
type Server struct {
	o       Options
	log     *slog.Logger
	reports *Reports

	mu   sync.RWMutex
	host host.Host
}

// New returns a Server. The window host is attached later with SetHost, because
// the windows load pages from this very server and so are created after it starts.
func New(o Options) *Server {
	log := o.Logger
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if o.Placement == "" {
		o.Placement = host.PlaceToolkit
	}
	return &Server{o: o, log: log, reports: &Reports{}}
}

// SetHost attaches the window host. Until it is set, host routes answer 503.
func (s *Server) SetHost(h host.Host) {
	s.mu.Lock()
	s.host = h
	s.mu.Unlock()
}

// Reports returns the diagnostics store written by POST /api/v1/report/{name}.
func (s *Server) Reports() *Reports { return s.reports }

func (s *Server) hostOrNil() host.Host {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.host
}

type route struct {
	method, path, summary string
	api                   bool // under /api/v1 and so subject to the token
	handler               http.HandlerFunc
}

// RouteDoc describes one route for generated documentation.
type RouteDoc struct {
	Method, Path, Summary string
	Auth                  bool
}

// Routes lists every route with its summary, in registration order.
func Routes() []RouteDoc {
	rs := New(Options{}).routes()
	docs := make([]RouteDoc, len(rs))
	for i, r := range rs {
		docs[i] = RouteDoc{Method: r.method, Path: r.path, Summary: r.summary, Auth: r.api}
	}
	return docs
}

func (s *Server) routes() []route {
	return []route{
		{"GET", "/healthz", "liveness probe; always 200 ok", false, s.healthz},
		{"GET", "/{$}", "redirect to the control page", false, s.index},
		{"GET", "/control", "control page (HTML)", false, s.page("control.html")},
		{"GET", "/output", "output page (HTML); ?debug adds an overlay, ?probe runs the media probe", false, s.page("output.html")},
		{"GET", "/assets/{file...}", "static CSS and JavaScript for the pages", false, s.assets()},
		{"GET", "/media/{file...}", "self-test media clips (only when the self-test is running)", false, s.media()},
		{"GET", "/api/v1/version", "build version, commit and build stamp", true, s.version},
		{"GET", "/api/v1/status", "position, blank flag, current slide and all slides", true, s.status},
		{"POST", "/api/v1/next", "advance one slide (stops at the last)", true, s.next},
		{"POST", "/api/v1/prev", "go back one slide (stops at the first)", true, s.prev},
		{"POST", "/api/v1/slides/{index}/show", "show slide {index}; 404 if it does not exist", true, s.show},
		{"PUT", "/api/v1/slides", `replace all slides: {"slides":[{"title":"","text":"..."}]}`, true, s.putSlides},
		{"POST", "/api/v1/blank", `blank {"blank":true}, unblank {"blank":false}, or toggle with an empty body`, true, s.blank},
		{"GET", "/api/v1/screens", "connected monitors as the OS reports them", true, s.screens},
		{"PUT", "/api/v1/output", `move the output: {"screen":"<id>","placement":"win32|toolkit"} or {"screen":null} for the floating window`, true, s.putOutput},
		{"GET", "/api/v1/windows", "both windows as the OS reports them (rectangles, topmost, owner, DPI, monitor)", true, s.windows},
		{"GET", "/api/v1/events", "server-sent events: an event named state after every change (token may be ?token=)", true, s.events},
		{"POST", "/api/v1/report/{name}", "store a diagnostics JSON document (used by the self-test)", true, s.postReport},
		{"GET", "/api/v1/report", "read all stored diagnostics documents", true, s.getReport},
	}
}

// Handler returns the full handler: routing, auth, panic recovery and logging.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	for _, r := range s.routes() {
		h := http.Handler(r.handler)
		if r.api {
			h = s.requireToken(h)
		}
		mux.Handle(r.method+" "+r.path, securityHeaders(h))
	}
	return s.recoverer(s.logRequests(mux))
}

// ---- middleware -------------------------------------------------------------

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		// No inline script or style anywhere: the pages are static files.
		h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; media-src 'self' blob:; object-src 'none'; base-uri 'none'; frame-ancestors 'none'")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.o.Token == "" {
			next.ServeHTTP(w, r)
			return
		}
		got := r.Header.Get("X-API-Token")
		if got == "" {
			if v, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
				got = v
			}
		}
		// EventSource cannot set headers, so the event stream alone accepts ?token=.
		if got == "" && r.URL.Path == "/api/v1/events" {
			got = r.URL.Query().Get("token")
		}
		if subtle.ConstantTimeCompare([]byte(got), []byte(s.o.Token)) != 1 {
			problem(w, http.StatusUnauthorized, "unauthorized", "missing or invalid API token")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				s.log.Error("panic in handler", "method", r.Method, "path", r.URL.Path, "panic", fmt.Sprint(v))
				problem(w, http.StatusInternalServerError, "internal error", "the request failed")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	code int
}

func (w *statusWriter) WriteHeader(code int) { w.code = code; w.ResponseWriter.WriteHeader(code) }

// Flush keeps server-sent events working through the wrapper.
func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &statusWriter{ResponseWriter: w, code: http.StatusOK}
		start := time.Now()
		next.ServeHTTP(sw, r)
		// Never log the query string: it can carry the event-stream token.
		s.log.Debug("request", "method", r.Method, "path", r.URL.Path, "status", sw.code, "took", time.Since(start).Round(time.Microsecond))
	})
}

// ---- helpers ----------------------------------------------------------------

// problem writes an RFC 9457 problem+json error.
func problem(w http.ResponseWriter, status int, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "about:blank", "title": title, "status": status, "detail": detail})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// decode reads a bounded JSON body into v, rejecting unknown fields. An empty
// body is not an error and leaves v untouched; it reports whether a body was present.
func decode(w http.ResponseWriter, r *http.Request, v any) (present bool, err error) {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		if errors.Is(err, io.EOF) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *Server) needHost(w http.ResponseWriter) (host.Host, bool) {
	h := s.hostOrNil()
	if h == nil {
		problem(w, http.StatusServiceUnavailable, "windows not ready", "the control and output windows are still starting")
		return nil, false
	}
	return h, true
}

// ---- handlers ---------------------------------------------------------------

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = io.WriteString(w, "ok\n")
}

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	target := "/control"
	if t := r.URL.Query().Get("token"); t != "" {
		target += "?token=" + url.QueryEscape(t)
	}
	http.Redirect(w, r, target, http.StatusFound)
}

func (s *Server) page(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := fs.ReadFile(ui.Assets(), name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(b)
	}
}

func (s *Server) assets() http.HandlerFunc {
	files := http.StripPrefix("/assets/", http.FileServerFS(ui.Assets()))
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		files.ServeHTTP(w, r)
	}
}

func (s *Server) media() http.HandlerFunc {
	if s.o.Media == nil {
		return http.NotFound
	}
	return http.StripPrefix("/media/", http.FileServerFS(s.o.Media)).ServeHTTP
}

func (s *Server) version(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, version.Get())
}

func (s *Server) status(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.o.State.Snapshot())
}

func (s *Server) next(w http.ResponseWriter, _ *http.Request) {
	s.o.State.Next()
	writeJSON(w, http.StatusOK, s.o.State.Snapshot())
}

func (s *Server) prev(w http.ResponseWriter, _ *http.Request) {
	s.o.State.Prev()
	writeJSON(w, http.StatusOK, s.o.State.Snapshot())
}

func (s *Server) show(w http.ResponseWriter, r *http.Request) {
	i, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		problem(w, http.StatusBadRequest, "bad index", "the slide index must be an integer")
		return
	}
	if err := s.o.State.Goto(i); err != nil {
		problem(w, http.StatusNotFound, "no such slide", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, s.o.State.Snapshot())
}

type slidesBody struct {
	Slides []state.Slide `json:"slides"`
}

func (s *Server) putSlides(w http.ResponseWriter, r *http.Request) {
	var body slidesBody
	if _, err := decode(w, r, &body); err != nil {
		problem(w, http.StatusBadRequest, "bad request body", err.Error())
		return
	}
	if len(body.Slides) > maxSlides {
		problem(w, http.StatusRequestEntityTooLarge, "too many slides", fmt.Sprintf("at most %d slides", maxSlides))
		return
	}
	for i, sl := range body.Slides {
		if len(sl.Text) > maxSlideBytes || len(sl.Title) > maxSlideBytes {
			problem(w, http.StatusRequestEntityTooLarge, "slide too large", fmt.Sprintf("slide %d exceeds %d bytes", i, maxSlideBytes))
			return
		}
	}
	s.o.State.SetSlides(body.Slides)
	writeJSON(w, http.StatusOK, s.o.State.Snapshot())
}

func (s *Server) blank(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Blank *bool `json:"blank"`
	}
	if _, err := decode(w, r, &body); err != nil {
		problem(w, http.StatusBadRequest, "bad request body", err.Error())
		return
	}
	s.o.State.SetBlank(body.Blank) // nil toggles
	writeJSON(w, http.StatusOK, s.o.State.Snapshot())
}

func (s *Server) screens(w http.ResponseWriter, r *http.Request) {
	h, ok := s.needHost(w)
	if !ok {
		return
	}
	screens, err := h.Screens(r.Context())
	if err != nil {
		problem(w, http.StatusInternalServerError, "cannot list screens", err.Error())
		return
	}
	if screens == nil {
		screens = []host.Screen{}
	}
	writeJSON(w, http.StatusOK, screens)
}

func (s *Server) putOutput(w http.ResponseWriter, r *http.Request) {
	h, ok := s.needHost(w)
	if !ok {
		return
	}
	var body struct {
		Screen    *string        `json:"screen"` // null selects the floating window
		Placement host.Placement `json:"placement"`
	}
	if _, err := decode(w, r, &body); err != nil {
		problem(w, http.StatusBadRequest, "bad request body", err.Error())
		return
	}
	how := body.Placement
	if how == "" {
		how = s.o.Placement
	}
	if how != host.PlaceWin32 && how != host.PlaceToolkit {
		problem(w, http.StatusBadRequest, "bad placement", `placement must be "win32" or "toolkit"`)
		return
	}
	var err error
	if body.Screen == nil {
		err = h.FloatOutput(r.Context())
	} else {
		err = h.PlaceOutput(r.Context(), *body.Screen, how)
	}
	if err != nil {
		problem(w, http.StatusUnprocessableEntity, "cannot place the output", err.Error())
		return
	}
	s.windows(w, r)
}

func (s *Server) windows(w http.ResponseWriter, r *http.Request) {
	h, ok := s.needHost(w)
	if !ok {
		return
	}
	rep, err := h.Report(r.Context())
	if err != nil {
		problem(w, http.StatusInternalServerError, "cannot read windows", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		problem(w, http.StatusInternalServerError, "streaming unsupported", "the connection cannot stream")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	ch, cancel := s.o.State.Subscribe()
	defer cancel()
	_, _ = io.WriteString(w, "event: state\ndata: {}\n\n")
	fl.Flush()
	keepAlive := time.NewTicker(15 * time.Second)
	defer keepAlive.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ch:
			_, _ = io.WriteString(w, "event: state\ndata: {}\n\n")
		case <-keepAlive.C:
			_, _ = io.WriteString(w, ": keep-alive\n\n")
		}
		fl.Flush()
	}
}

func (s *Server) postReport(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" || len(name) > maxReportName || strings.ContainsAny(name, "/\\ ") {
		problem(w, http.StatusBadRequest, "bad report name", "use a short name without slashes or spaces")
		return
	}
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	if err != nil || !json.Valid(b) {
		problem(w, http.StatusBadRequest, "bad request body", "the body must be a JSON document")
		return
	}
	s.reports.Set(name, b)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getReport(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.reports.All())
}

// Reports is a small in-memory store of named JSON documents that pages post
// for diagnostics. It is safe for concurrent use.
type Reports struct {
	mu sync.Mutex
	m  map[string]json.RawMessage
}

// Set stores a document under name.
func (r *Reports) Set(name string, v json.RawMessage) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.m == nil {
		r.m = map[string]json.RawMessage{}
	}
	r.m[name] = append(json.RawMessage(nil), v...)
}

// Get returns the document stored under name.
func (r *Reports) Get(name string) (json.RawMessage, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.m[name]
	return v, ok
}

// All returns a copy of every stored document.
func (r *Reports) All() map[string]json.RawMessage {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[string]json.RawMessage, len(r.m))
	for k, v := range r.m {
		out[k] = v
	}
	return out
}

// Wait polls for a document until it exists or ctx ends.
func (r *Reports) Wait(ctx context.Context, name string) (json.RawMessage, error) {
	t := time.NewTicker(100 * time.Millisecond)
	defer t.Stop()
	for {
		if v, ok := r.Get(name); ok {
			return v, nil
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("waiting for report %q: %w", name, ctx.Err())
		case <-t.C:
		}
	}
}
