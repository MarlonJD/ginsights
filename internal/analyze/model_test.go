package analyze

import (
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
