package analyze

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/multica-ai/ginsights/internal/gitlog"
)

type Snapshot struct {
	RepoPath    string          `json:"repo_path"`
	RepoName    string          `json:"repo_name"`
	GeneratedAt time.Time       `json:"generated_at"`
	Totals      Totals          `json:"totals"`
	Authors     []AuthorStat    `json:"authors"`
	Weekly      []WeekStat      `json:"weekly"`
	Daily       []DayStat       `json:"daily"`
	HotFiles    []FileStat      `json:"hot_files"`
	Languages   []LanguageStat  `json:"languages"`
	Recent      []RecentCommit  `json:"recent"`
	Health      []HealthSignal  `json:"health"`
	Provenance  []ProvenanceRow `json:"provenance"`
}

type Totals struct {
	Commits      int `json:"commits"`
	Authors      int `json:"authors"`
	FilesChanged int `json:"files_changed"`
	Additions    int `json:"additions"`
	Deletions    int `json:"deletions"`
	NetLines     int `json:"net_lines"`
}

type AuthorStat struct {
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	Commits      int       `json:"commits"`
	Additions    int       `json:"additions"`
	Deletions    int       `json:"deletions"`
	FilesTouched int       `json:"files_touched"`
	FirstCommit  time.Time `json:"first_commit"`
	LastCommit   time.Time `json:"last_commit"`
}

type WeekStat struct {
	WeekStart time.Time `json:"week_start"`
	Commits   int       `json:"commits"`
	Additions int       `json:"additions"`
	Deletions int       `json:"deletions"`
}

type DayStat struct {
	Date    time.Time `json:"date"`
	Commits int       `json:"commits"`
}

type FileStat struct {
	Path      string `json:"path"`
	Commits   int    `json:"commits"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Churn     int    `json:"churn"`
}

type LanguageStat struct {
	Name    string  `json:"name"`
	Bytes   int64   `json:"bytes"`
	Percent float64 `json:"percent"`
}

type RecentCommit struct {
	Hash         string    `json:"hash"`
	ShortHash    string    `json:"short_hash"`
	AuthorName   string    `json:"author_name"`
	Date         time.Time `json:"date"`
	Subject      string    `json:"subject"`
	FilesChanged int       `json:"files_changed"`
}

type HealthSignal struct {
	Name    string `json:"name"`
	Present bool   `json:"present"`
	Detail  string `json:"detail"`
}

type ProvenanceRow struct {
	Metric string `json:"metric"`
	Source string `json:"source"`
}

func BuildSnapshot(repo string, commits []gitlog.Commit, now time.Time) Snapshot {
	abs, err := filepath.Abs(repo)
	if err == nil {
		repo = abs
	}
	authors := map[string]*AuthorStat{}
	authorFiles := map[string]map[string]struct{}{}
	files := map[string]*FileStat{}
	weeks := map[time.Time]*WeekStat{}
	days := map[time.Time]*DayStat{}
	uniqueFiles := map[string]struct{}{}
	totals := Totals{Commits: len(commits)}

	for _, commit := range commits {
		key := strings.ToLower(commit.AuthorEmail)
		if key == "" {
			key = strings.ToLower(commit.AuthorName)
		}
		author := authors[key]
		if author == nil {
			author = &AuthorStat{Name: commit.AuthorName, Email: commit.AuthorEmail, FirstCommit: commit.Date, LastCommit: commit.Date}
			authors[key] = author
			authorFiles[key] = map[string]struct{}{}
		}
		author.Commits++
		if commit.Date.Before(author.FirstCommit) {
			author.FirstCommit = commit.Date
		}
		if commit.Date.After(author.LastCommit) {
			author.LastCommit = commit.Date
		}

		weekStart := startOfWeek(commit.Date)
		week := weeks[weekStart]
		if week == nil {
			week = &WeekStat{WeekStart: weekStart}
			weeks[weekStart] = week
		}
		week.Commits++

		dayStart := startOfDay(commit.Date)
		day := days[dayStart]
		if day == nil {
			day = &DayStat{Date: dayStart}
			days[dayStart] = day
		}
		day.Commits++

		for _, change := range commit.Files {
			totals.Additions += change.Additions
			totals.Deletions += change.Deletions
			author.Additions += change.Additions
			author.Deletions += change.Deletions
			week.Additions += change.Additions
			week.Deletions += change.Deletions
			uniqueFiles[change.Path] = struct{}{}
			authorFiles[key][change.Path] = struct{}{}

			file := files[change.Path]
			if file == nil {
				file = &FileStat{Path: change.Path}
				files[change.Path] = file
			}
			file.Commits++
			file.Additions += change.Additions
			file.Deletions += change.Deletions
			file.Churn += change.Additions + change.Deletions
		}
	}

	for key, touched := range authorFiles {
		authors[key].FilesTouched = len(touched)
	}
	totals.Authors = len(authors)
	totals.FilesChanged = len(uniqueFiles)
	totals.NetLines = totals.Additions - totals.Deletions

	return Snapshot{
		RepoPath:    repo,
		RepoName:    repoName(repo),
		GeneratedAt: now,
		Totals:      totals,
		Authors:     sortedAuthors(authors),
		Weekly:      sortedWeeks(weeks),
		Daily:       sortedDays(days),
		HotFiles:    sortedFiles(files, 25),
		Languages:   DetectLanguages(repo),
		Recent:      recentCommits(commits, 12),
		Health:      DetectHealth(repo),
		Provenance: []ProvenanceRow{
			{Metric: "commits/authors/code frequency/file churn", Source: "local_git"},
			{Metric: "languages/repo health", Source: "working_tree"},
		},
	}
}

func startOfWeek(t time.Time) time.Time {
	y, m, d := t.Date()
	day := time.Date(y, m, d, 0, 0, 0, 0, t.Location())
	offset := (int(day.Weekday()) + 6) % 7
	return day.AddDate(0, 0, -offset)
}

func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func repoName(path string) string {
	clean := filepath.Clean(path)
	base := filepath.Base(clean)
	if base == "." || base == string(filepath.Separator) {
		return "repository"
	}
	return base
}

func sortedAuthors(input map[string]*AuthorStat) []AuthorStat {
	out := make([]AuthorStat, 0, len(input))
	for _, stat := range input {
		out = append(out, *stat)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Commits == out[j].Commits {
			return out[i].Name < out[j].Name
		}
		return out[i].Commits > out[j].Commits
	})
	return out
}

func sortedWeeks(input map[time.Time]*WeekStat) []WeekStat {
	out := make([]WeekStat, 0, len(input))
	for _, stat := range input {
		out = append(out, *stat)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].WeekStart.Before(out[j].WeekStart) })
	return out
}

func sortedDays(input map[time.Time]*DayStat) []DayStat {
	out := make([]DayStat, 0, len(input))
	for _, stat := range input {
		out = append(out, *stat)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date.Before(out[j].Date) })
	return out
}

func sortedFiles(input map[string]*FileStat, limit int) []FileStat {
	out := make([]FileStat, 0, len(input))
	for _, stat := range input {
		out = append(out, *stat)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Churn == out[j].Churn {
			return out[i].Path < out[j].Path
		}
		return out[i].Churn > out[j].Churn
	})
	if limit > 0 && len(out) > limit {
		return out[:limit]
	}
	return out
}

func recentCommits(commits []gitlog.Commit, limit int) []RecentCommit {
	out := make([]RecentCommit, 0, len(commits))
	for _, commit := range commits {
		short := commit.Hash
		if len(short) > 7 {
			short = short[:7]
		}
		out = append(out, RecentCommit{
			Hash:         commit.Hash,
			ShortHash:    short,
			AuthorName:   commit.AuthorName,
			Date:         commit.Date,
			Subject:      commit.Subject,
			FilesChanged: len(commit.Files),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date.After(out[j].Date) })
	if limit > 0 && len(out) > limit {
		return out[:limit]
	}
	return out
}

func DetectHealth(repo string) []HealthSignal {
	checks := []struct {
		name  string
		paths []string
	}{
		{"README", []string{"README.md", "README"}},
		{"LICENSE", []string{"LICENSE", "LICENSE.md"}},
		{"CONTRIBUTING", []string{"CONTRIBUTING.md", ".github/CONTRIBUTING.md"}},
		{"CODE_OF_CONDUCT", []string{"CODE_OF_CONDUCT.md", ".github/CODE_OF_CONDUCT.md"}},
		{"SECURITY", []string{"SECURITY.md", "docs/SECURITY.md", ".github/SECURITY.md"}},
		{"CI workflow", []string{".github/workflows/ci.yml", ".github/workflows/ci.yaml"}},
		{"Agent map", []string{"AGENTS.md"}},
		{"Architecture doc", []string{"ARCHITECTURE.md"}},
	}
	out := make([]HealthSignal, 0, len(checks)+1)
	for _, check := range checks {
		present, detail := anyExists(repo, check.paths)
		out = append(out, HealthSignal{Name: check.name, Present: present, Detail: detail})
	}
	testsPresent := hasTests(repo)
	detail := "no tests detected"
	if testsPresent {
		detail = "test files detected"
	}
	out = append(out, HealthSignal{Name: "Tests", Present: testsPresent, Detail: detail})
	return out
}

func anyExists(repo string, paths []string) (bool, string) {
	for _, rel := range paths {
		if _, err := os.Stat(filepath.Join(repo, rel)); err == nil {
			return true, rel
		}
	}
	return false, "missing"
}

func hasTests(repo string) bool {
	found := false
	_ = filepath.WalkDir(repo, func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "vendor" || name == "node_modules" || name == "report" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), "_test.go") || strings.HasSuffix(d.Name(), ".test.ts") || strings.HasSuffix(d.Name(), ".spec.ts") {
			found = true
		}
		return nil
	})
	return found
}
