package analyze

import (
	"fmt"
	"testing"
	"time"

	"github.com/multica-ai/ginsights/internal/gitlog"
)

func TestBuildSnapshotAggregates(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	commits := []gitlog.Commit{
		{
			Hash:        "abcdef123",
			AuthorName:  "Ada",
			AuthorEmail: "ada@example.com",
			Date:        now,
			Subject:     "Initial",
			Files: []gitlog.FileChange{
				{Path: "main.go", Additions: 10, Deletions: 2},
				{Path: "README.md", Additions: 3, Deletions: 0},
			},
		},
		{
			Hash:        "123456789",
			AuthorName:  "Ada",
			AuthorEmail: "ada@example.com",
			Date:        now.Add(24 * time.Hour),
			Subject:     "Update",
			Files: []gitlog.FileChange{
				{Path: "main.go", Additions: 4, Deletions: 1},
			},
		},
	}

	snap := BuildSnapshot(".", commits, now)
	if snap.Totals.Commits != 2 {
		t.Fatalf("commits = %d, want 2", snap.Totals.Commits)
	}
	if snap.Totals.Additions != 17 || snap.Totals.Deletions != 3 {
		t.Fatalf("line totals = +%d -%d, want +17 -3", snap.Totals.Additions, snap.Totals.Deletions)
	}
	if len(snap.Authors) != 1 || snap.Authors[0].FilesTouched != 2 {
		t.Fatalf("authors = %+v, want one author touching two files", snap.Authors)
	}
	if len(snap.HotFiles) == 0 || snap.HotFiles[0].Path != "main.go" {
		t.Fatalf("hot files = %+v, want main.go first", snap.HotFiles)
	}
}

func BenchmarkBuildSnapshot(b *testing.B) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	commits := benchmarkCommits(500)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		snap := BuildSnapshot(".", commits, now)
		if snap.Totals.Commits != len(commits) {
			b.Fatalf("commits = %d, want %d", snap.Totals.Commits, len(commits))
		}
	}
}

func benchmarkCommits(count int) []gitlog.Commit {
	base := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	commits := make([]gitlog.Commit, 0, count)
	for i := 0; i < count; i++ {
		commits = append(commits, gitlog.Commit{
			Hash:        fmt.Sprintf("%040d", i),
			AuthorName:  fmt.Sprintf("Author %d", i%8),
			AuthorEmail: fmt.Sprintf("author%d@example.com", i%8),
			Date:        base.Add(time.Duration(i) * time.Hour),
			Subject:     fmt.Sprintf("Commit %d", i),
			Files: []gitlog.FileChange{
				{Path: fmt.Sprintf("internal/file%d.go", i%25), Additions: 10 + i%7, Deletions: i % 5},
				{Path: "README.md", Additions: 1, Deletions: 0},
			},
		})
	}
	return commits
}
