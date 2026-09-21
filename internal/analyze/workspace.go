package analyze

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// BuildWorkspaceSnapshot aggregates independently analyzed repositories while
// retaining every repository snapshot for inspection and JSON consumers.
func BuildWorkspaceSnapshot(root string, repositories []WorkspaceRepository, failures []WorkspaceError, now time.Time) Snapshot {
	abs, err := filepath.Abs(root)
	if err == nil {
		root = abs
	}

	authors := map[string]*AuthorStat{}
	weeks := map[time.Time]*WeekStat{}
	days := map[time.Time]*DayStat{}
	languages := map[string]int64{}
	var hotFiles []FileStat
	var recent []RecentCommit
	var health []HealthSignal
	totals := Totals{}

	for _, repository := range repositories {
		snap := repository.Snapshot
		label := workspaceRepositoryLabel(repository)
		totals.Commits += snap.Totals.Commits
		totals.FilesChanged += snap.Totals.FilesChanged
		totals.Additions += snap.Totals.Additions
		totals.Deletions += snap.Totals.Deletions

		for _, author := range snap.Authors {
			key := strings.ToLower(author.Email)
			if key == "" {
				key = strings.ToLower(author.Name)
			}
			aggregate := authors[key]
			if aggregate == nil {
				copy := author
				authors[key] = &copy
				continue
			}
			aggregate.Commits += author.Commits
			aggregate.Additions += author.Additions
			aggregate.Deletions += author.Deletions
			aggregate.FilesTouched += author.FilesTouched
			if author.FirstCommit.Before(aggregate.FirstCommit) {
				aggregate.FirstCommit = author.FirstCommit
			}
			if author.LastCommit.After(aggregate.LastCommit) {
				aggregate.LastCommit = author.LastCommit
			}
		}

		for _, week := range snap.Weekly {
			aggregate := weeks[week.WeekStart]
			if aggregate == nil {
				copy := week
				weeks[week.WeekStart] = &copy
				continue
			}
			aggregate.Commits += week.Commits
			aggregate.Additions += week.Additions
			aggregate.Deletions += week.Deletions
		}
		for _, day := range snap.Daily {
			aggregate := days[day.Date]
			if aggregate == nil {
				copy := day
				days[day.Date] = &copy
				continue
			}
			aggregate.Commits += day.Commits
		}
		for _, file := range snap.HotFiles {
			file.Repository = label
			hotFiles = append(hotFiles, file)
		}
		for _, commit := range snap.Recent {
			commit.Repository = label
			recent = append(recent, commit)
		}
		for _, language := range snap.Languages {
			languages[language.Name] += language.Bytes
		}

		present := 0
		for _, signal := range snap.Health {
			if signal.Present {
				present++
			}
		}
		health = append(health, HealthSignal{
			Name:    label,
			Present: len(snap.Health) > 0 && present == len(snap.Health),
			Detail:  workspaceHealthDetail(present, len(snap.Health)),
		})
	}
	for _, failure := range failures {
		health = append(health, HealthSignal{Name: failure.RelativePath, Present: false, Detail: failure.Error})
	}

	totals.Authors = len(authors)
	totals.NetLines = totals.Additions - totals.Deletions

	return Snapshot{
		RepoPath:    root,
		RepoName:    repoName(root),
		GeneratedAt: now,
		Totals:      totals,
		Authors:     sortedAuthors(authors),
		Weekly:      sortedWeeks(weeks),
		Daily:       sortedDays(days),
		HotFiles:    sortedWorkspaceFiles(hotFiles, 25),
		Languages:   workspaceLanguages(languages),
		Recent:      sortedWorkspaceRecent(recent, 12),
		Health:      health,
		Provenance: []ProvenanceRow{
			{Metric: "workspace repository discovery", Source: "local_filesystem"},
			{Metric: "commits/authors/code frequency/file churn", Source: "local_git_per_repository"},
			{Metric: "languages/repository health", Source: "working_tree_per_repository"},
		},
		Workspace: &Workspace{
			RepositoryCount: len(repositories) + len(failures),
			Repositories:    repositories,
			Errors:          failures,
		},
	}
}

func workspaceRepositoryLabel(repository WorkspaceRepository) string {
	if repository.RelativePath == "." || repository.RelativePath == "" {
		return repository.Snapshot.RepoName
	}
	return filepath.ToSlash(repository.RelativePath)
}

func workspaceHealthDetail(present, total int) string {
	if total == 0 {
		return "no repository health signals"
	}
	return strconv.Itoa(present) + "/" + strconv.Itoa(total) + " repository health signals present"
}

func sortedWorkspaceFiles(files []FileStat, limit int) []FileStat {
	sort.Slice(files, func(i, j int) bool {
		if files[i].Churn == files[j].Churn {
			if files[i].Repository == files[j].Repository {
				return files[i].Path < files[j].Path
			}
			return files[i].Repository < files[j].Repository
		}
		return files[i].Churn > files[j].Churn
	})
	if limit > 0 && len(files) > limit {
		return files[:limit]
	}
	return files
}

func sortedWorkspaceRecent(commits []RecentCommit, limit int) []RecentCommit {
	sort.Slice(commits, func(i, j int) bool {
		if commits[i].Date.Equal(commits[j].Date) {
			if commits[i].Repository == commits[j].Repository {
				return commits[i].Hash < commits[j].Hash
			}
			return commits[i].Repository < commits[j].Repository
		}
		return commits[i].Date.After(commits[j].Date)
	})
	if limit > 0 && len(commits) > limit {
		return commits[:limit]
	}
	return commits
}

func workspaceLanguages(input map[string]int64) []LanguageStat {
	var total int64
	for _, bytes := range input {
		total += bytes
	}
	out := make([]LanguageStat, 0, len(input))
	for name, bytes := range input {
		percent := 0.0
		if total > 0 {
			percent = float64(bytes) / float64(total) * 100
		}
		out = append(out, LanguageStat{Name: name, Bytes: bytes, Percent: percent})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Bytes == out[j].Bytes {
			return out[i].Name < out[j].Name
		}
		return out[i].Bytes > out[j].Bytes
	})
	return out
}
