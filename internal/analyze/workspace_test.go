package analyze

import (
	"testing"
	"time"
)

func TestBuildWorkspaceSnapshotAggregatesRepositoriesWithoutLosingIdentity(t *testing.T) {
	now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	week := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)
	repositories := []WorkspaceRepository{
		{
			RelativePath: ".",
			Snapshot: Snapshot{
				RepoName:  "workspace",
				Totals:    Totals{Commits: 2, Authors: 1, FilesChanged: 1, Additions: 10, Deletions: 2, NetLines: 8},
				Authors:   []AuthorStat{{Name: "Ada", Email: "ada@example.com", Commits: 2, Additions: 10, Deletions: 2, FilesTouched: 1, FirstCommit: now.Add(-2 * time.Hour), LastCommit: now.Add(-time.Hour)}},
				Weekly:    []WeekStat{{WeekStart: week, Commits: 2, Additions: 10, Deletions: 2}},
				Daily:     []DayStat{{Date: week, Commits: 2}},
				HotFiles:  []FileStat{{Path: "README.md", Commits: 2, Additions: 10, Deletions: 2, Churn: 12}},
				Recent:    []RecentCommit{{Hash: "root", ShortHash: "root", AuthorName: "Ada", Date: now.Add(-time.Hour), Subject: "root change"}},
				Languages: []LanguageStat{{Name: "Markdown", Bytes: 100, Percent: 100}},
				Health:    []HealthSignal{{Name: "README", Present: true}},
			},
		},
		{
			RelativePath: "apps/surveil",
			Snapshot: Snapshot{
				RepoName:  "surveil",
				Totals:    Totals{Commits: 3, Authors: 1, FilesChanged: 2, Additions: 30, Deletions: 5, NetLines: 25},
				Authors:   []AuthorStat{{Name: "Ada", Email: "ada@example.com", Commits: 3, Additions: 30, Deletions: 5, FilesTouched: 2, FirstCommit: now.Add(-3 * time.Hour), LastCommit: now}},
				Weekly:    []WeekStat{{WeekStart: week, Commits: 3, Additions: 30, Deletions: 5}},
				Daily:     []DayStat{{Date: week, Commits: 3}},
				HotFiles:  []FileStat{{Path: "main.go", Commits: 3, Additions: 30, Deletions: 5, Churn: 35}},
				Recent:    []RecentCommit{{Hash: "child", ShortHash: "child", AuthorName: "Ada", Date: now, Subject: "child change"}},
				Languages: []LanguageStat{{Name: "Go", Bytes: 300, Percent: 100}},
				Health:    []HealthSignal{{Name: "README", Present: false}},
			},
		},
	}

	snap := BuildWorkspaceSnapshot("/tmp/workspace", repositories, nil, now)
	if snap.Workspace == nil || snap.Workspace.RepositoryCount != 2 {
		t.Fatalf("workspace = %+v, want two repositories", snap.Workspace)
	}
	if snap.Totals.Commits != 5 || snap.Totals.Authors != 1 || snap.Totals.FilesChanged != 3 || snap.Totals.NetLines != 33 {
		t.Fatalf("totals = %+v, want aggregated totals", snap.Totals)
	}
	if len(snap.Authors) != 1 || snap.Authors[0].FilesTouched != 3 {
		t.Fatalf("authors = %+v, want merged author with namespaced files", snap.Authors)
	}
	if len(snap.Recent) != 2 || snap.Recent[0].Repository != "apps/surveil" {
		t.Fatalf("recent = %+v, want repository-labeled chronological commits", snap.Recent)
	}
	if len(snap.HotFiles) != 2 || snap.HotFiles[0].Repository != "apps/surveil" {
		t.Fatalf("hot files = %+v, want repository identity retained", snap.HotFiles)
	}
	if len(snap.Languages) != 2 || snap.Languages[0].Name != "Go" || snap.Languages[0].Percent != 75 {
		t.Fatalf("languages = %+v, want byte-weighted workspace languages", snap.Languages)
	}
}
