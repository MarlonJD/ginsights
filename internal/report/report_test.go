package report

import (
	"strings"
	"testing"
	"time"

	"github.com/multica-ai/ginsights/internal/analyze"
)

func TestHTMLRendersCoreSections(t *testing.T) {
	snap := analyze.Snapshot{
		RepoName:    "demo",
		RepoPath:    "/tmp/demo",
		GeneratedAt: time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
		Totals:      analyze.Totals{Commits: 1, Authors: 1, FilesChanged: 1, Additions: 10, Deletions: 2},
		Authors:     []analyze.AuthorStat{{Name: "Ada", Email: "ada@example.com", Commits: 1}},
		Weekly:      []analyze.WeekStat{{WeekStart: time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC), Commits: 1, Additions: 10, Deletions: 2}},
		HotFiles:    []analyze.FileStat{{Path: "main.go", Commits: 1, Churn: 12, Additions: 10, Deletions: 2}},
	}
	html, err := HTML(snap)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Pulse", "Contributors", "Code frequency", "Hot files", "Metric provenance"} {
		if !strings.Contains(html, want) {
			t.Fatalf("HTML missing %q", want)
		}
	}
}
