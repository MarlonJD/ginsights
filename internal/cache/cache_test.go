package cache

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/multica-ai/ginsights/internal/gitlog"
)

func TestCollectPopulatesCacheAndReusesUnchangedCommits(t *testing.T) {
	ctx := context.Background()
	store := Store{Dir: t.TempDir()}
	first := &fakeSource{
		hashes: []string{"new", "old"},
		commits: map[string]gitlog.Commit{
			"new": {Hash: "new", Date: time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC), Subject: "new"},
			"old": {Hash: "old", Date: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), Subject: "old"},
		},
	}

	got, err := Collect(ctx, first, store)
	if err != nil {
		t.Fatal(err)
	}
	if hashesOf(got) != "new,old" {
		t.Fatalf("first collect hashes = %s, want new,old", hashesOf(got))
	}
	if !reflect.DeepEqual(first.collectHashCalls, [][]string{{"new", "old"}}) {
		t.Fatalf("first collect calls = %#v, want all missing hashes", first.collectHashCalls)
	}

	second := &fakeSource{hashes: []string{"new", "old"}}
	got, err = Collect(ctx, second, store)
	if err != nil {
		t.Fatal(err)
	}
	if hashesOf(got) != "new,old" {
		t.Fatalf("second collect hashes = %s, want new,old", hashesOf(got))
	}
	if len(second.collectHashCalls) != 0 {
		t.Fatalf("second collect calls = %#v, want cache hit without reparsing", second.collectHashCalls)
	}
}

func TestCollectFetchesOnlyMissingCommitsAndPrunesStaleCache(t *testing.T) {
	ctx := context.Background()
	store := Store{Dir: t.TempDir()}
	if err := store.Save([]gitlog.Commit{
		{Hash: "cached", Subject: "cached"},
		{Hash: "stale", Subject: "stale"},
	}); err != nil {
		t.Fatal(err)
	}
	source := &fakeSource{
		hashes: []string{"missing", "cached"},
		commits: map[string]gitlog.Commit{
			"missing": {Hash: "missing", Subject: "missing"},
		},
	}

	got, err := Collect(ctx, source, store)
	if err != nil {
		t.Fatal(err)
	}
	if hashesOf(got) != "missing,cached" {
		t.Fatalf("collect hashes = %s, want missing,cached", hashesOf(got))
	}
	if !reflect.DeepEqual(source.collectHashCalls, [][]string{{"missing"}}) {
		t.Fatalf("collect calls = %#v, want only missing hash", source.collectHashCalls)
	}

	cached, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cached["stale"]; ok {
		t.Fatalf("stale commit remained in cache: %#v", cached)
	}
}

func TestClearRemovesDisposableCache(t *testing.T) {
	repo := t.TempDir()
	cacheDir := DefaultDir(repo)
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "commits-v1.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Clear(repo); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Fatalf("cache dir still exists or stat failed with unexpected error: %v", err)
	}
}

type fakeSource struct {
	hashes           []string
	commits          map[string]gitlog.Commit
	collectHashCalls [][]string
}

func (s *fakeSource) ListHashes(context.Context) ([]string, error) {
	return append([]string(nil), s.hashes...), nil
}

func (s *fakeSource) CollectHashes(_ context.Context, hashes []string) ([]gitlog.Commit, error) {
	s.collectHashCalls = append(s.collectHashCalls, append([]string(nil), hashes...))
	commits := make([]gitlog.Commit, 0, len(hashes))
	for _, hash := range hashes {
		commits = append(commits, s.commits[hash])
	}
	return commits, nil
}

func hashesOf(commits []gitlog.Commit) string {
	hashes := make([]byte, 0)
	for i, commit := range commits {
		if i > 0 {
			hashes = append(hashes, ',')
		}
		hashes = append(hashes, commit.Hash...)
	}
	return string(hashes)
}
