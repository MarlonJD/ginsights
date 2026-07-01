package report

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/multica-ai/ginsights/internal/analyze"
)

var updateGolden = flag.Bool("update", false, "update golden test fixtures")

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

func TestHTMLRendersDenseCommitHeatmap(t *testing.T) {
	snap := analyze.Snapshot{
		RepoName:    "demo",
		RepoPath:    "/tmp/demo",
		GeneratedAt: time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
		Daily: []analyze.DayStat{
			{Date: time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC), Commits: 1},
			{Date: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), Commits: 4},
		},
	}
	html, err := HTML(snap)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`class="heatmap-shell"`,
		`aria-label="commit activity heatmap, 7 days"`,
		`title="2026-06-30: 0 commits"`,
		`class="heat l0"`,
		`class="heat l4"`,
		`class="heatmap-legend"`,
		`2 active days · 5 commits`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("HTML missing dense heatmap marker %q", want)
		}
	}
}

func TestHTMLRendersDenseCodeFrequency(t *testing.T) {
	snap := analyze.Snapshot{
		RepoName:    "demo",
		RepoPath:    "/tmp/demo",
		GeneratedAt: time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
		Weekly: []analyze.WeekStat{
			{WeekStart: time.Date(2026, 6, 22, 0, 0, 0, 0, time.UTC), Commits: 1, Additions: 50, Deletions: 5},
			{WeekStart: time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC), Commits: 3, Additions: 180, Deletions: 42},
		},
	}
	html, err := HTML(snap)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`class="code-frequency-table"`,
		`<th>Net</th>`,
		`class="frequency-bars" title="2026-06-29: &#43;180 additions, -42 deletions, &#43;138 net across 3 commits"`,
		`class="frequency-bar additions"`,
		`class="frequency-bar deletions"`,
		`&#43;138`,
		`2 weeks · &#43;230/-47 · net &#43;183`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("HTML missing dense code-frequency marker %q", want)
		}
	}
}

func TestHTMLRendersGitHubAPISectionWithProvenance(t *testing.T) {
	snap := analyze.Snapshot{
		RepoName:    "demo",
		RepoPath:    "/tmp/demo",
		GeneratedAt: time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC),
		GitHub: &analyze.GitHubMetrics{
			Repository: "acme/widgets",
			Stars:      42,
			Forks:      7,
			OpenIssues: 3,
			Views:      &analyze.GitHubTrafficMetric{Count: 120, Uniques: 45},
			Clones:     &analyze.GitHubTrafficMetric{Count: 12, Uniques: 8},
		},
	}
	html, err := HTML(snap)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`GitHub API`,
		`github_api`,
		`acme/widgets`,
		`42`,
		`120 views`,
		`12 clones`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("HTML missing GitHub API marker %q", want)
		}
	}
}

func TestHTMLMatchesGoldenFixture(t *testing.T) {
	html, err := HTML(goldenSnapshot())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("testdata", "non_empty_report.golden.html")
	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(html), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden fixture %s: %v; run go test ./internal/report -run TestHTMLMatchesGoldenFixture -update", path, err)
	}
	if html != string(want) {
		t.Fatalf("HTML output does not match golden fixture %s; first diff at byte %d", path, firstDiff(html, string(want)))
	}
}

func goldenSnapshot() analyze.Snapshot {
	baseDate := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	return analyze.Snapshot{
		RepoName:    "demo",
		RepoPath:    "/tmp/demo",
		GeneratedAt: baseDate,
		Totals: analyze.Totals{
			Commits:      3,
			Authors:      2,
			FilesChanged: 3,
			Additions:    180,
			Deletions:    42,
			NetLines:     138,
		},
		Authors: []analyze.AuthorStat{
			{
				Name:         "Ada Lovelace",
				Email:        "ada@example.com",
				Commits:      2,
				Additions:    150,
				Deletions:    25,
				FilesTouched: 2,
				FirstCommit:  time.Date(2026, 6, 29, 9, 0, 0, 0, time.UTC),
				LastCommit:   time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC),
			},
			{
				Name:         "Grace Hopper",
				Email:        "grace@example.com",
				Commits:      1,
				Additions:    30,
				Deletions:    17,
				FilesTouched: 1,
				FirstCommit:  time.Date(2026, 6, 30, 15, 0, 0, 0, time.UTC),
				LastCommit:   time.Date(2026, 6, 30, 15, 0, 0, 0, time.UTC),
			},
		},
		Weekly: []analyze.WeekStat{
			{WeekStart: time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC), Commits: 3, Additions: 180, Deletions: 42},
		},
		Daily: []analyze.DayStat{
			{Date: time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC), Commits: 1},
			{Date: time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC), Commits: 2},
			{Date: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), Commits: 3},
		},
		HotFiles: []analyze.FileStat{
			{Path: "internal/report/report.go", Commits: 2, Additions: 120, Deletions: 30, Churn: 150},
			{Path: "README.md", Commits: 1, Additions: 60, Deletions: 12, Churn: 72},
		},
		Languages: []analyze.LanguageStat{
			{Name: "Go", Bytes: 12000, Percent: 75},
			{Name: "Markdown", Bytes: 4000, Percent: 25},
		},
		Recent: []analyze.RecentCommit{
			{
				Hash:         "abcdef1234567890",
				ShortHash:    "abcdef1",
				AuthorName:   "Ada Lovelace",
				Date:         time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC),
				Subject:      "Improve dashboard renderer",
				FilesChanged: 2,
			},
			{
				Hash:         "1234567890abcdef",
				ShortHash:    "1234567",
				AuthorName:   "Grace Hopper",
				Date:         time.Date(2026, 6, 30, 15, 0, 0, 0, time.UTC),
				Subject:      "Document local insights scope",
				FilesChanged: 1,
			},
		},
		Health: []analyze.HealthSignal{
			{Name: "README", Present: true, Detail: "README.md"},
			{Name: "CI workflow", Present: false, Detail: "missing"},
			{Name: "Tests", Present: true, Detail: "test files detected"},
		},
		Provenance: []analyze.ProvenanceRow{
			{Metric: "commits/authors/code frequency/file churn", Source: "local_git"},
			{Metric: "languages/repo health", Source: "working_tree"},
		},
	}
}

func firstDiff(a, b string) int {
	limit := len(a)
	if len(b) < limit {
		limit = len(b)
	}
	for i := 0; i < limit; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return limit
}
