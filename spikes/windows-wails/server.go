package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

//go:embed web
var webFS embed.FS

// Reports collects JSON posted by the pages (e.g. the media probe) for the self-test.
type Reports struct {
	mu sync.Mutex
	m  map[string]json.RawMessage
}

func (r *Reports) Set(name string, v json.RawMessage) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.m == nil {
		r.m = map[string]json.RawMessage{}
	}
	r.m[name] = v
}

func (r *Reports) Get(name string) (json.RawMessage, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, ok := r.m[name]
	return v, ok
}

func (r *Reports) All() map[string]json.RawMessage {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[string]json.RawMessage, len(r.m))
	for k, v := range r.m {
		out[k] = v
	}
	return out
}

// hostRef lets the server start (and the windows load its URL) before the Host exists.
type hostRef struct {
	mu sync.RWMutex
	h  Host
}

func (r *hostRef) Set(h Host) { r.mu.Lock(); r.h = h; r.mu.Unlock() }
func (r *hostRef) Get() Host  { r.mu.RLock(); defer r.mu.RUnlock(); return r.h }

func newHandler(st *State, hr *hostRef, reports *Reports) http.Handler {
	mux := http.NewServeMux()
	writeJSON := func(w http.ResponseWriter, code int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		_ = json.NewEncoder(w).Encode(v)
	}
	needHost := func(w http.ResponseWriter) Host {
		h := hr.Get()
		if h == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "windows not ready"})
		}
		return h
	}

	mux.HandleFunc("GET /api/v1/status", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, st.Status())
	})
	mux.HandleFunc("POST /api/v1/next", func(w http.ResponseWriter, r *http.Request) {
		st.Move(+1)
		writeJSON(w, 200, st.Status())
	})
	mux.HandleFunc("POST /api/v1/prev", func(w http.ResponseWriter, r *http.Request) {
		st.Move(-1)
		writeJSON(w, 200, st.Status())
	})
	mux.HandleFunc("POST /api/v1/blank", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Blank *bool `json:"blank"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body) // empty body toggles
		st.SetBlank(body.Blank)
		writeJSON(w, 200, st.Status())
	})

	// Server-sent events: tell renderers to re-read state.
	mux.HandleFunc("GET /api/v1/events", func(w http.ResponseWriter, r *http.Request) {
		fl, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", 500)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		ch, cancel := st.Subscribe()
		defer cancel()
		fmt.Fprint(w, "event: state\ndata: {}\n\n")
		fl.Flush()
		keepAlive := time.NewTicker(15 * time.Second)
		defer keepAlive.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case <-ch:
				fmt.Fprint(w, "event: state\ndata: {}\n\n")
			case <-keepAlive.C:
				fmt.Fprint(w, ": keep-alive\n\n")
			}
			fl.Flush()
		}
	})

	mux.HandleFunc("GET /api/v1/screens", func(w http.ResponseWriter, r *http.Request) {
		h := needHost(w)
		if h == nil {
			return
		}
		s, err := h.Screens()
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, s)
	})
	mux.HandleFunc("PUT /api/v1/output", func(w http.ResponseWriter, r *http.Request) {
		h := needHost(w)
		if h == nil {
			return
		}
		var body struct {
			Screen *string   `json:"screen"` // null = floating
			Mode   PlaceMode `json:"mode"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		var err error
		if body.Screen == nil {
			err = h.FloatOutput()
		} else {
			if body.Mode == "" {
				body.Mode = PlaceWin32
			}
			err = h.PlaceOutput(*body.Screen, body.Mode)
		}
		if err != nil {
			writeJSON(w, 422, map[string]string{"error": err.Error()})
			return
		}
		rep, _ := h.Report()
		writeJSON(w, 200, rep)
	})
	mux.HandleFunc("GET /api/v1/windows", func(w http.ResponseWriter, r *http.Request) {
		h := needHost(w)
		if h == nil {
			return
		}
		rep, err := h.Report()
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, rep)
	})
	mux.HandleFunc("POST /api/v1/report/{name}", func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil || !json.Valid(b) {
			writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
			return
		}
		reports.Set(r.PathValue("name"), b)
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /api/v1/report", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, reports.All())
	})

	sub, _ := fs.Sub(webFS, "web")
	files := http.FileServer(http.FS(sub))
	mux.HandleFunc("GET /output", func(w http.ResponseWriter, r *http.Request) { serveFile(w, r, sub, "output.html") })
	mux.HandleFunc("GET /control", func(w http.ResponseWriter, r *http.Request) { serveFile(w, r, sub, "control.html") })
	mux.Handle("GET /media/", files)
	return logRequests(mux)
}

// logRequests logs page loads (not the chatty API/SSE traffic) so window loading can be traced.
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/output" || r.URL.Path == "/control" || len(r.URL.Path) > 11 && r.URL.Path[:11] == "/api/v1/rep" {
			log.Printf("%s %s?%s ua=%.60s", r.Method, r.URL.Path, r.URL.RawQuery, r.UserAgent())
		}
		next.ServeHTTP(w, r)
	})
}

func serveFile(w http.ResponseWriter, r *http.Request, fsys fs.FS, name string) {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(b)
}

// listen binds 127.0.0.1:port (0 = any free port) and returns the listener and base URL.
func listen(port int) (net.Listener, string, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, "", err
	}
	return ln, "http://" + ln.Addr().String(), nil
}
