package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/multica-ai/ginsights/internal/analyze"
	"github.com/multica-ai/ginsights/internal/report"
)

func TestDashboardRefreshesHTMLAndJSONTogether(t *testing.T) {
	snap := analyze.Snapshot{RepoName: "before", GeneratedAt: time.Now(), Totals: analyze.Totals{Commits: 1}}
	loads := 0
	d, err := newDashboard(context.Background(), time.Hour, func(context.Context) (analyze.Snapshot, error) {
		loads++
		return snap, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	before := requestDashboard(d, http.MethodGet, "/data.json", "")
	snap.RepoName = "after"
	snap.Totals.Commits = 2
	if got := requestDashboard(d, http.MethodGet, "/data.json", ""); got.Body.String() != before.Body.String() || loads != 1 {
		t.Fatalf("request within interval reloaded data: loads = %d, body = %s", loads, got.Body.String())
	}
	expireDashboard(d)
	after := requestDashboard(d, http.MethodGet, "/data.json", "")
	var got analyze.Snapshot
	if err := json.Unmarshal(after.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Totals.Commits != 2 || loads != 2 || after.Header().Get("ETag") == before.Header().Get("ETag") {
		t.Fatalf("refreshed snapshot = %+v, loads = %d, headers = %v", got, loads, after.Header())
	}
	html := requestDashboard(d, http.MethodGet, "/", "")
	if !strings.Contains(html.Body.String(), "after") || !strings.Contains(html.Body.String(), `id="live-status"`) {
		t.Fatalf("HTML did not reflect the refreshed snapshot and live status")
	}
	if html.Header().Get("Cache-Control") != "no-store" || after.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("live responses must not be cached")
	}
}

func TestDashboardIgnoresVolatileWorkspaceTimestamps(t *testing.T) {
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	snap := analyze.Snapshot{RepoName: "workspace", GeneratedAt: now, Workspace: &analyze.Workspace{
		RepositoryCount: 1,
		Repositories: []analyze.WorkspaceRepository{{RelativePath: "child", Snapshot: analyze.Snapshot{
			GeneratedAt: now, Totals: analyze.Totals{Commits: 1},
		}}},
	}}
	d, err := newDashboard(context.Background(), time.Hour, func(context.Context) (analyze.Snapshot, error) {
		return snap, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !snap.Workspace.Repositories[0].Snapshot.GeneratedAt.Equal(now) {
		t.Fatal("computing the ETag mutated the input snapshot")
	}
	etag := requestDashboard(d, http.MethodGet, "/data.json", "").Header().Get("ETag")
	snap.GeneratedAt = now.Add(time.Minute)
	snap.Workspace.Repositories[0].Snapshot.GeneratedAt = now.Add(time.Minute)
	expireDashboard(d)
	unchanged := requestDashboard(d, http.MethodHead, "/data.json", etag)
	if unchanged.Code != http.StatusNotModified || unchanged.Body.Len() != 0 {
		t.Fatalf("timestamp-only change = %d, body = %q", unchanged.Code, unchanged.Body.String())
	}
	snap.Workspace.Repositories[0].Snapshot.Totals.Commits++
	expireDashboard(d)
	changed := requestDashboard(d, http.MethodHead, "/data.json", etag)
	if changed.Code != http.StatusOK || changed.Header().Get("ETag") == etag || changed.Body.Len() != 0 {
		t.Fatalf("nested content change = %d, headers = %v, body = %q", changed.Code, changed.Header(), changed.Body.String())
	}
	newETag := changed.Header().Get("ETag")
	snap.GeneratedAt = now.AddDate(0, 0, 1)
	expireDashboard(d)
	if nextDay := requestDashboard(d, http.MethodHead, "/data.json", newETag); nextDay.Code != http.StatusOK {
		t.Fatalf("activity calendar did not advance on a new day: %d", nextDay.Code)
	}
}

func TestDashboardRetainsReportAndRecoversAfterRefreshFailure(t *testing.T) {
	failure := error(nil)
	snap := analyze.Snapshot{RepoName: "last-good", GeneratedAt: time.Now()}
	d, err := newDashboard(context.Background(), time.Hour, func(context.Context) (analyze.Snapshot, error) {
		return snap, failure
	})
	if err != nil {
		t.Fatal(err)
	}
	failure = errors.New("git temporarily unavailable")
	expireDashboard(d)
	data := requestDashboard(d, http.MethodHead, "/data.json", "")
	if data.Code != http.StatusServiceUnavailable {
		t.Fatalf("failed refresh status = %d, want 503", data.Code)
	}
	html := requestDashboard(d, http.MethodGet, "/", "")
	if html.Code != http.StatusOK || !strings.Contains(html.Body.String(), "last-good") || !strings.Contains(html.Body.String(), liveErrorStatus) {
		t.Fatal("failed refresh did not retain the last report with a visible error")
	}
	failure = nil
	snap.RepoName = "recovered"
	expireDashboard(d)
	recovered := requestDashboard(d, http.MethodGet, "/", "")
	if !strings.Contains(recovered.Body.String(), "recovered") || d.refreshErr != nil {
		t.Fatal("refresh did not recover on the next interval")
	}
}

func TestConcurrentDashboardRequestsShareOneRefresh(t *testing.T) {
	loads := 0
	d, err := newDashboard(context.Background(), time.Hour, func(context.Context) (analyze.Snapshot, error) {
		loads++
		time.Sleep(5 * time.Millisecond)
		return analyze.Snapshot{GeneratedAt: time.Now()}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	expireDashboard(d)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if got := requestDashboard(d, http.MethodGet, "/data.json", ""); got.Code != http.StatusOK {
				t.Errorf("concurrent request status = %d", got.Code)
			}
		}()
	}
	wg.Wait()
	if loads != 2 {
		t.Fatalf("loads = %d, want startup plus one refresh", loads)
	}
}

func TestDashboardStartupAndRouting(t *testing.T) {
	load := func(context.Context) (analyze.Snapshot, error) { return analyze.Snapshot{}, nil }
	if _, err := newDashboard(context.Background(), 0, load); err == nil {
		t.Fatal("zero refresh interval was accepted")
	}
	if _, err := newDashboard(context.Background(), time.Second, func(context.Context) (analyze.Snapshot, error) {
		return analyze.Snapshot{}, errors.New("initial analysis failed")
	}); err == nil {
		t.Fatal("startup analysis failure was ignored")
	}
	d, err := newDashboard(context.Background(), time.Second, load)
	if err != nil {
		t.Fatal(err)
	}
	if got := requestDashboard(d, http.MethodGet, "/healthz", ""); got.Code != http.StatusOK || got.Body.String() != "ok\n" {
		t.Fatalf("health response = %d %q", got.Code, got.Body.String())
	}
	if got := requestDashboard(d, http.MethodPost, "/data.json", ""); got.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status = %d, want 405", got.Code)
	}
	if got := requestDashboard(d, http.MethodGet, "/missing", ""); got.Code != http.StatusNotFound {
		t.Fatalf("unknown route status = %d, want 404", got.Code)
	}
}

func TestStaticReportDoesNotPoll(t *testing.T) {
	html, err := report.HTML(analyze.Snapshot{GeneratedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html, "live-status") || strings.Contains(html, "fetch(") {
		t.Fatal("static HTML contains live refresh behavior")
	}
}

func requestDashboard(d *dashboard, method, path, etag string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, nil)
	if etag != "" {
		r.Header.Set("If-None-Match", etag)
	}
	w := httptest.NewRecorder()
	d.ServeHTTP(w, r)
	return w
}

func expireDashboard(d *dashboard) {
	d.mu.Lock()
	d.lastChecked = time.Time{}
	d.mu.Unlock()
}
