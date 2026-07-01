package app

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/multica-ai/ginsights/internal/analyze"
	"github.com/multica-ai/ginsights/internal/cache"
	"github.com/multica-ai/ginsights/internal/gitlog"
)

func TestRunJSONFiltersCommitsSince(t *testing.T) {
	repo := testGitRepo(t)
	commitFile(t, repo, "old.go", "package demo\n", "2026-06-30T12:00:00+00:00", "old commit")
	commitFile(t, repo, "new.go", "package demo\nfunc New() {}\n", "2026-07-01T12:00:00+00:00", "new commit")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"json", repo, "--since", "2026-07-01"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run json exit = %d, stderr = %s", code, stderr.String())
	}

	var snap analyze.Snapshot
	if err := json.Unmarshal(stdout.Bytes(), &snap); err != nil {
		t.Fatalf("decode json output: %v\n%s", err, stdout.String())
	}
	if got, want := snap.Totals.Commits, 1; got != want {
		t.Fatalf("commits = %d, want %d; output:\n%s", got, want, stdout.String())
	}
	if len(snap.Recent) != 1 || snap.Recent[0].Subject != "new commit" {
		t.Fatalf("recent = %+v, want only new commit", snap.Recent)
	}
}

func TestRunBuildFiltersCommitsSince(t *testing.T) {
	repo := testGitRepo(t)
	commitFile(t, repo, "old.go", "package demo\n", "2026-06-30T12:00:00+00:00", "old commit")
	commitFile(t, repo, "new.go", "package demo\nfunc New() {}\n", "2026-07-01T12:00:00+00:00", "new commit")
	outDir := filepath.Join(t.TempDir(), "report")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"build", repo, "--out", outDir, "--since", "2026-07-01"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run build exit = %d, stderr = %s", code, stderr.String())
	}

	data, err := os.ReadFile(filepath.Join(outDir, "data.json"))
	if err != nil {
		t.Fatalf("read data.json: %v", err)
	}
	var snap analyze.Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		t.Fatalf("decode data.json: %v\n%s", err, string(data))
	}
	if got, want := snap.Totals.Commits, 1; got != want {
		t.Fatalf("commits = %d, want %d; data.json:\n%s", got, want, string(data))
	}
}

func TestRunJSONRejectsInvalidSince(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"json", "--since", "07/01/2026", "."}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("Run json exit = %d, want 2; stdout = %s stderr = %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), `invalid --since "07/01/2026": use YYYY-MM-DD`) {
		t.Fatalf("stderr = %q, want practical YYYY-MM-DD error", stderr.String())
	}
}

func TestRunJSONWithGitHubAPIMissingTokenDegradesGracefully(t *testing.T) {
	t.Setenv("GINSIGHTS_GITHUB_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	repo := testGitRepo(t)
	commitFile(t, repo, "main.go", "package demo\n", "2026-07-01T12:00:00+00:00", "initial")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"json", repo, "--github-api", "acme/widgets", "--no-cache"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run json exit = %d, stderr = %s", code, stderr.String())
	}
	var raw map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &raw); err != nil {
		t.Fatalf("decode json output: %v\n%s", err, stdout.String())
	}
	github, ok := raw["github"].(map[string]any)
	if !ok {
		t.Fatalf("github field missing from JSON output:\n%s", stdout.String())
	}
	if github["repository"] != "acme/widgets" {
		t.Fatalf("github.repository = %v, want acme/widgets", github["repository"])
	}
	if !strings.Contains(github["error"].(string), "GINSIGHTS_GITHUB_TOKEN") {
		t.Fatalf("github.error = %v, want missing token guidance", github["error"])
	}
	if !strings.Contains(stdout.String(), `"source": "github_api"`) {
		t.Fatalf("provenance missing github_api source:\n%s", stdout.String())
	}
}

func TestRunJSONPopulatesCacheByDefault(t *testing.T) {
	repo := testGitRepo(t)
	commitFile(t, repo, "main.go", "package demo\n", "2026-07-01T12:00:00+00:00", "initial")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"json", repo}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run json exit = %d, stderr = %s", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(cache.DefaultDir(repo), "commits-v1.json")); err != nil {
		t.Fatalf("cache file was not created: %v", err)
	}
}

func TestRunJSONNoCacheDoesNotCreateCache(t *testing.T) {
	repo := testGitRepo(t)
	commitFile(t, repo, "main.go", "package demo\n", "2026-07-01T12:00:00+00:00", "initial")

	var stdout, stderr bytes.Buffer
	code := Run([]string{"json", repo, "--no-cache"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run json exit = %d, stderr = %s", code, stderr.String())
	}
	if _, err := os.Stat(cache.DefaultDir(repo)); !os.IsNotExist(err) {
		t.Fatalf("cache dir exists or stat failed with unexpected error: %v", err)
	}
}

func TestRunCacheClearRemovesCache(t *testing.T) {
	repo := testGitRepo(t)
	cacheDir := cache.DefaultDir(repo)
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "commits-v1.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := Run([]string{"cache-clear", repo}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run cache-clear exit = %d, stderr = %s", code, stderr.String())
	}
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Fatalf("cache dir exists or stat failed with unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Cache cleared:") {
		t.Fatalf("stdout = %q, want cache-clear confirmation", stdout.String())
	}
}

func TestFilterCommitsSinceIncludesBoundary(t *testing.T) {
	since := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	commits := []gitlog.Commit{
		{Hash: "old", Date: since.Add(-time.Nanosecond)},
		{Hash: "boundary", Date: since},
		{Hash: "new", Date: since.Add(time.Hour)},
	}

	got := filterCommitsSince(commits, since)
	if len(got) != 2 || got[0].Hash != "boundary" || got[1].Hash != "new" {
		t.Fatalf("filtered commits = %+v, want boundary and new", got)
	}
}

func testGitRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.name", "Ada")
	runGit(t, repo, "config", "user.email", "ada@example.com")
	return repo
}

func commitFile(t *testing.T, repo, name, body, date, message string) {
	t.Helper()
	path := filepath.Join(repo, name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	runGit(t, repo, "add", name)
	cmd := exec.Command("git", "-C", repo, "commit", "-m", message)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Ada",
		"GIT_AUTHOR_EMAIL=ada@example.com",
		"GIT_COMMITTER_NAME=Ada",
		"GIT_COMMITTER_EMAIL=ada@example.com",
		"GIT_AUTHOR_DATE="+date,
		"GIT_COMMITTER_DATE="+date,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git commit %q failed: %v\n%s", message, err, string(out))
	}
}

func runGit(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmdArgs := append([]string{"-C", repo}, args...)
	out, err := exec.Command("git", cmdArgs...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(out))
	}
}
