package server

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/multica-ai/ginsights/internal/analyze"
	"github.com/multica-ai/ginsights/internal/report"
)

func Serve(ctx context.Context, port int, interval time.Duration, load func(context.Context) (analyze.Snapshot, error), stdout io.Writer) error {
	dashboard, err := newDashboard(ctx, interval, load)
	if err != nil {
		return err
	}

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return err
	}
	srv := &http.Server{
		Handler:           dashboard,
		ReadHeaderTimeout: 5 * time.Second,
	}

	stopped := make(chan struct{})
	defer close(stopped)
	go func() {
		select {
		case <-ctx.Done():
		case <-stopped:
			return
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	fmt.Fprintf(stdout, "Local dashboard: http://%s\n", ln.Addr().String())
	err = srv.Serve(ln)
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

type dashboard struct {
	mu          sync.Mutex
	interval    time.Duration
	load        func(context.Context) (analyze.Snapshot, error)
	lastChecked time.Time
	html        string
	data        []byte
	etag        string
	refreshErr  error
}

func newDashboard(ctx context.Context, interval time.Duration, load func(context.Context) (analyze.Snapshot, error)) (*dashboard, error) {
	if interval < time.Millisecond {
		return nil, fmt.Errorf("refresh interval must be at least 1ms")
	}
	d := &dashboard{interval: interval, load: load}
	if err := d.refresh(ctx); err != nil {
		return nil, err
	}
	d.lastChecked = time.Now()
	return d, nil
}

// refresh publishes HTML and JSON together only after analysis and rendering succeed.
func (d *dashboard) refresh(ctx context.Context) error {
	snap, err := d.load(ctx)
	if err != nil {
		return err
	}
	html, err := report.HTML(snap)
	if err != nil {
		return err
	}
	data, err := report.JSON(snap)
	if err != nil {
		return err
	}
	etag, err := snapshotETag(snap)
	if err != nil {
		return err
	}
	d.html = liveHTML(html, etag, d.interval)
	d.data = data
	d.etag = etag
	return nil
}

func (d *dashboard) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/data.json" && r.URL.Path != "/healthz" && r.URL.Path != "/favicon.ico" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "use GET or HEAD", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if r.URL.Path == "/favicon.ico" {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.URL.Path == "/healthz" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if r.Method == http.MethodGet {
			_, _ = io.WriteString(w, "ok\n")
		}
		return
	}

	d.mu.Lock()
	if time.Since(d.lastChecked) >= d.interval {
		d.refreshErr = d.refresh(r.Context())
		d.lastChecked = time.Now()
	}
	html, data, etag, refreshErr := d.html, d.data, d.etag, d.refreshErr
	d.mu.Unlock()

	if r.URL.Path == "/data.json" {
		if refreshErr != nil {
			http.Error(w, "refresh report: "+refreshErr.Error(), http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("ETag", etag)
		for _, candidate := range strings.Split(r.Header.Get("If-None-Match"), ",") {
			if candidate = strings.TrimSpace(candidate); candidate == etag || candidate == "*" {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}
		if r.Method == http.MethodGet {
			_, _ = w.Write(data)
		}
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if refreshErr != nil {
		html = strings.Replace(html, liveStatus, liveErrorStatus, 1)
	}
	if r.Method == http.MethodGet {
		_, _ = io.WriteString(w, html)
	}
}

func snapshotETag(snap analyze.Snapshot) (string, error) {
	data, err := report.JSON(withoutGeneratedTimes(snap))
	if err != nil {
		return "", err
	}
	// Ignore volatile timestamps, but advance the activity calendar on a new day.
	data = append([]byte(snap.GeneratedAt.Format("2006-01-02")), data...)
	return fmt.Sprintf("W/\"%x\"", sha256.Sum256(data)), nil
}

func withoutGeneratedTimes(snap analyze.Snapshot) analyze.Snapshot {
	snap.GeneratedAt = time.Time{}
	if snap.Workspace != nil {
		workspace := *snap.Workspace
		workspace.Repositories = append([]analyze.WorkspaceRepository(nil), workspace.Repositories...)
		for i := range workspace.Repositories {
			workspace.Repositories[i].Snapshot = withoutGeneratedTimes(workspace.Repositories[i].Snapshot)
		}
		snap.Workspace = &workspace
	}
	return snap
}
