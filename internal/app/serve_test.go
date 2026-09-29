package app

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/multica-ai/ginsights/internal/analyze"
)

func TestRunServeRefreshesWorkspaceDiscoveryAndCommitHistory(t *testing.T) {
	workspace := testGitRepo(t)
	commitFile(t, workspace, "README.md", "# Workspace\n", "2026-09-20T12:00:00+00:00", "excluded root commit")
	child := filepath.Join(workspace, "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	initGitRepo(t, child)
	commitFile(t, child, "main.go", "package demo\n", "2026-09-21T12:00:00+00:00", "initial child commit")

	url := startServe(t, workspace, "--workspace", "--since", "2026-09-21")
	initial := readServedSnapshot(t, url)
	if initial.Workspace.RepositoryCount != 2 || initial.Totals.Commits != 1 {
		t.Fatalf("initial workspace = %+v, totals = %+v", initial.Workspace, initial.Totals)
	}
	commitFile(t, child, "main.go", "package demo\nfunc Added() {}\n", "2026-09-22T12:00:00+00:00", "live child commit")
	updated := readServedSnapshot(t, url)
	if updated.Totals.Commits != 2 || updated.Recent[0].Subject != "live child commit" {
		t.Fatalf("nested commit did not refresh: totals = %+v, recent = %+v", updated.Totals, updated.Recent)
	}
	response, err := http.Get(url + "/")
	if err != nil {
		t.Fatal(err)
	}
	html, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || !strings.Contains(string(html), "live child commit") {
		t.Fatalf("served HTML did not refresh: %v", err)
	}

	added := filepath.Join(workspace, "added")
	if err := os.MkdirAll(added, 0o755); err != nil {
		t.Fatal(err)
	}
	initGitRepo(t, added)
	commitFile(t, added, "new.py", "print('new repository')\n", "2026-09-23T12:00:00+00:00", "new repository commit")
	discovered := readServedSnapshot(t, url)
	if discovered.Workspace.RepositoryCount != 3 || discovered.Totals.Commits != 3 {
		t.Fatalf("added repository did not refresh: workspace = %+v, totals = %+v", discovered.Workspace, discovered.Totals)
	}
	if err := os.RemoveAll(added); err != nil {
		t.Fatal(err)
	}
	removed := readServedSnapshot(t, url)
	if removed.Workspace.RepositoryCount != 2 || removed.Totals.Commits != 2 {
		t.Fatalf("removed repository remained in report: workspace = %+v, totals = %+v", removed.Workspace, removed.Totals)
	}
}

func TestRunServeRefreshesSingleRepositoryWithoutCache(t *testing.T) {
	repo := testGitRepo(t)
	commitFile(t, repo, "main.go", "package demo\n", "2026-09-21T12:00:00+00:00", "initial")
	url := startServe(t, repo, "--no-cache")
	if initial := readServedSnapshot(t, url); initial.Totals.Commits != 1 {
		t.Fatalf("initial totals = %+v", initial.Totals)
	}
	commitFile(t, repo, "main.go", "package demo\nfunc Live() {}\n", "2026-09-22T12:00:00+00:00", "live update")
	if updated := readServedSnapshot(t, url); updated.Totals.Commits != 2 {
		t.Fatalf("single-repository totals did not refresh: %+v", updated.Totals)
	}
	if _, err := os.Stat(filepath.Join(repo, ".ginsights-cache")); !os.IsNotExist(err) {
		t.Fatalf("--no-cache created a cache or stat failed: %v", err)
	}
}

func TestRunServeRejectsInvalidRefresh(t *testing.T) {
	for _, value := range []string{"0", "-1s", "500us", "invalid"} {
		t.Run(value, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if code := Run([]string{"serve", ".", "--refresh", value}, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "refresh") {
				t.Fatalf("exit = %d, stderr = %q; want invalid refresh guidance", code, stderr.String())
			}
		})
	}
}

func startServe(t *testing.T, repo string, options ...string) string {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	addresses := make(serveAddress, 1)
	done := make(chan struct{})
	code := 0
	var stderr bytes.Buffer
	args := append([]string{repo, "--port", "0", "--refresh", "1ms"}, options...)
	go func() {
		code = runServe(ctx, args, addresses, &stderr)
		close(done)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
			if code != 0 {
				t.Errorf("serve exit = %d, stderr = %s", code, stderr.String())
			}
		case <-time.After(5 * time.Second):
			t.Error("serve did not stop after context cancellation")
		}
	})
	select {
	case url := <-addresses:
		return url
	case <-done:
		t.Fatalf("serve exited before startup: exit = %d, stderr = %s", code, stderr.String())
		return ""
	case <-time.After(10 * time.Second):
		t.Fatal("serve did not publish its local address")
		return ""
	}
}

type serveAddress chan string

func (a serveAddress) Write(p []byte) (int, error) {
	a <- strings.TrimSpace(strings.TrimPrefix(string(p), "Local dashboard: "))
	return len(p), nil
}

func readServedSnapshot(t *testing.T, url string) analyze.Snapshot {
	t.Helper()
	// Allow the minimum refresh interval to elapse after filesystem-only changes.
	time.Sleep(2 * time.Millisecond)
	response, err := http.Get(url + "/data.json")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("data.json status = %d, body = %s", response.StatusCode, body)
	}
	var snap analyze.Snapshot
	if err := json.NewDecoder(response.Body).Decode(&snap); err != nil {
		t.Fatal(err)
	}
	return snap
}
