package app

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/multica-ai/ginsights/internal/analyze"
	historycache "github.com/multica-ai/ginsights/internal/cache"
	"github.com/multica-ai/ginsights/internal/doclint"
	"github.com/multica-ai/ginsights/internal/githubapi"
	"github.com/multica-ai/ginsights/internal/gitlog"
	"github.com/multica-ai/ginsights/internal/report"
	"github.com/multica-ai/ginsights/internal/repository"
	"github.com/multica-ai/ginsights/internal/server"
)

const defaultPort = 43117

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stdout)
		return 0
	}

	ctx := context.Background()
	switch args[0] {
	case "help", "-h", "--help":
		printUsage(stdout)
		return 0
	case "serve":
		return runServe(ctx, args[1:], stdout, stderr)
	case "build":
		return runBuild(ctx, args[1:], stdout, stderr)
	case "json":
		return runJSON(ctx, args[1:], stdout, stderr)
	case "cache-clear":
		return runCacheClear(args[1:], stdout, stderr)
	case "doctor":
		return runDoctor(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runServe(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(stderr)
	port := fs.Int("port", defaultPort, "port to listen on; use 0 for a random free port")
	noCache := fs.Bool("no-cache", false, "disable the disposable local analysis cache")
	sinceValue := fs.String("since", "", "only include commits on or after YYYY-MM-DD")
	githubRepo := fs.String("github-api", "", "opt-in GitHub API repository as owner/name")
	workspaceMode := fs.Bool("workspace", false, "discover and aggregate nested Git repositories")
	if err := fs.Parse(normalizeFlagArgs(args, map[string]bool{"port": true, "since": true, "github-api": true})); err != nil {
		return 2
	}
	opts, err := snapshotOptionsFromFlags(*sinceValue, *noCache, *githubRepo, *workspaceMode)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	repo := firstArg(fs.Args(), ".")

	snap, err := snapshot(ctx, repo, opts)
	if err != nil {
		fmt.Fprintf(stderr, "analyze %s: %v\n", repo, err)
		return 1
	}
	if err := server.Serve(ctx, *port, snap, stdout); err != nil {
		fmt.Fprintf(stderr, "serve: %v\n", err)
		return 1
	}
	return 0
}

func runBuild(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("out", "report", "output directory")
	noCache := fs.Bool("no-cache", false, "disable the disposable local analysis cache")
	sinceValue := fs.String("since", "", "only include commits on or after YYYY-MM-DD")
	githubRepo := fs.String("github-api", "", "opt-in GitHub API repository as owner/name")
	workspaceMode := fs.Bool("workspace", false, "discover and aggregate nested Git repositories")
	if err := fs.Parse(normalizeFlagArgs(args, map[string]bool{"out": true, "since": true, "github-api": true})); err != nil {
		return 2
	}
	opts, err := snapshotOptionsFromFlags(*sinceValue, *noCache, *githubRepo, *workspaceMode)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	repo := firstArg(fs.Args(), ".")

	snap, err := snapshot(ctx, repo, opts)
	if err != nil {
		fmt.Fprintf(stderr, "analyze %s: %v\n", repo, err)
		return 1
	}
	if err := report.Write(*out, snap); err != nil {
		fmt.Fprintf(stderr, "write report: %v\n", err)
		return 1
	}
	abs, _ := filepath.Abs(*out)
	fmt.Fprintf(stdout, "Report written: %s\n", abs)
	return 0
}

func runJSON(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("json", flag.ContinueOnError)
	fs.SetOutput(stderr)
	noCache := fs.Bool("no-cache", false, "disable the disposable local analysis cache")
	sinceValue := fs.String("since", "", "only include commits on or after YYYY-MM-DD")
	githubRepo := fs.String("github-api", "", "opt-in GitHub API repository as owner/name")
	workspaceMode := fs.Bool("workspace", false, "discover and aggregate nested Git repositories")
	if err := fs.Parse(normalizeFlagArgs(args, map[string]bool{"since": true, "github-api": true})); err != nil {
		return 2
	}
	opts, err := snapshotOptionsFromFlags(*sinceValue, *noCache, *githubRepo, *workspaceMode)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	repo := firstArg(fs.Args(), ".")

	snap, err := snapshot(ctx, repo, opts)
	if err != nil {
		fmt.Fprintf(stderr, "analyze %s: %v\n", repo, err)
		return 1
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(snap); err != nil {
		fmt.Fprintf(stderr, "encode json: %v\n", err)
		return 1
	}
	return 0
}

func runCacheClear(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("cache-clear", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	repo := firstArg(fs.Args(), ".")
	if err := historycache.Clear(repo); err != nil {
		fmt.Fprintf(stderr, "cache-clear %s: %v\n", repo, err)
		return 1
	}
	abs, _ := filepath.Abs(historycache.DefaultDir(repo))
	fmt.Fprintf(stdout, "Cache cleared: %s\n", abs)
	return 0
}

func runDoctor(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return 2
	}
	root := firstArg(fs.Args(), ".")

	result := doclint.Check(root)
	for _, warning := range result.Warnings {
		fmt.Fprintf(stdout, "warning: %s\n", warning)
	}
	if len(result.Errors) > 0 {
		for _, issue := range result.Errors {
			fmt.Fprintf(stderr, "error: %s\n", issue)
		}
		return 1
	}
	fmt.Fprintln(stdout, "doctor ok")
	return 0
}

type snapshotOptions struct {
	Since      time.Time
	NoCache    bool
	GitHubRepo string
	Workspace  bool
}

func snapshotOptionsFromFlags(sinceValue string, noCache bool, githubRepo string, workspaceMode bool) (snapshotOptions, error) {
	opts := snapshotOptions{NoCache: noCache, GitHubRepo: strings.TrimSpace(githubRepo), Workspace: workspaceMode}
	if opts.Workspace && opts.GitHubRepo != "" {
		return snapshotOptions{}, fmt.Errorf("--workspace cannot be combined with --github-api; analyze that repository separately")
	}
	if sinceValue == "" {
		return opts, nil
	}
	since, err := parseSince(sinceValue)
	if err != nil {
		return snapshotOptions{}, err
	}
	opts.Since = since
	return opts, nil
}

func parseSince(value string) (time.Time, error) {
	since, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid --since %q: use YYYY-MM-DD, for example --since 2026-07-01", value)
	}
	return since, nil
}

func snapshot(ctx context.Context, repo string, opts snapshotOptions) (analyze.Snapshot, error) {
	if opts.Workspace {
		return workspaceSnapshot(ctx, repo, opts)
	}
	return repositorySnapshot(ctx, repo, opts, time.Now())
}

func repositorySnapshot(ctx context.Context, repo string, opts snapshotOptions, generatedAt time.Time) (analyze.Snapshot, error) {
	collector := gitlog.NewCollector(repo)
	var history []gitlog.Commit
	var err error
	if opts.NoCache {
		history, err = collector.Collect(ctx)
	} else {
		history, err = historycache.Collect(ctx, collector, historycache.Store{Dir: historycache.DefaultDir(repo)})
	}
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return analyze.Snapshot{}, fmt.Errorf("git repository not found: %w", err)
		}
		return analyze.Snapshot{}, err
	}
	if !opts.Since.IsZero() {
		history = filterCommitsSince(history, opts.Since)
	}
	snap := analyze.BuildSnapshot(repo, history, generatedAt)
	if opts.GitHubRepo != "" {
		mergeGitHubMetrics(ctx, &snap, opts.GitHubRepo)
	}
	return snap, nil
}

func workspaceSnapshot(ctx context.Context, root string, opts snapshotOptions) (analyze.Snapshot, error) {
	roots, err := repository.Discover(root)
	if err != nil {
		return analyze.Snapshot{}, err
	}
	generatedAt := time.Now()
	repositories := make([]analyze.WorkspaceRepository, 0, len(roots))
	var failures []analyze.WorkspaceError
	for _, discovered := range roots {
		actualRoot, err := gitlog.NewCollector(discovered.Path).TopLevel(ctx)
		if err != nil {
			failures = append(failures, analyze.WorkspaceError{RelativePath: discovered.RelativePath, Error: err.Error()})
			continue
		}
		if !sameFilePath(actualRoot, discovered.Path) {
			failures = append(failures, analyze.WorkspaceError{
				RelativePath: discovered.RelativePath,
				Error:        fmt.Sprintf("Git marker did not resolve to this repository root; resolved to %s", actualRoot),
			})
			continue
		}
		snap, err := repositorySnapshot(ctx, discovered.Path, opts, generatedAt)
		if err != nil {
			failures = append(failures, analyze.WorkspaceError{RelativePath: discovered.RelativePath, Error: err.Error()})
			continue
		}
		repositories = append(repositories, analyze.WorkspaceRepository{
			RelativePath: discovered.RelativePath,
			Snapshot:     snap,
		})
	}
	if len(repositories) == 0 {
		return analyze.Snapshot{}, fmt.Errorf("analyze workspace %s: all %d repositories failed", root, len(roots))
	}
	return analyze.BuildWorkspaceSnapshot(root, repositories, failures, generatedAt), nil
}

func sameFilePath(left, right string) bool {
	leftAbs, leftErr := filepath.Abs(left)
	rightAbs, rightErr := filepath.Abs(right)
	if leftErr != nil || rightErr != nil {
		return filepath.Clean(left) == filepath.Clean(right)
	}
	leftResolved, leftErr := filepath.EvalSymlinks(leftAbs)
	rightResolved, rightErr := filepath.EvalSymlinks(rightAbs)
	if leftErr == nil && rightErr == nil {
		return leftResolved == rightResolved
	}
	return filepath.Clean(leftAbs) == filepath.Clean(rightAbs)
}

func mergeGitHubMetrics(ctx context.Context, snap *analyze.Snapshot, repo string) {
	snap.Provenance = append(snap.Provenance, analyze.ProvenanceRow{Metric: "github repository metadata/traffic", Source: "github_api"})
	snap.GitHub = &analyze.GitHubMetrics{Repository: repo}
	token, _ := githubapi.EnvToken()
	if token == "" {
		snap.GitHub.Error = "github api token not configured: set GINSIGHTS_GITHUB_TOKEN or GITHUB_TOKEN"
		return
	}
	metrics, err := githubapi.DefaultClient(token).Fetch(ctx, repo)
	if err != nil {
		snap.GitHub.Error = err.Error()
		return
	}
	snap.GitHub.Repository = metrics.Repository
	snap.GitHub.Stars = metrics.Stars
	snap.GitHub.Forks = metrics.Forks
	snap.GitHub.OpenIssues = metrics.OpenIssues
	snap.GitHub.Warnings = metrics.Warnings
	if metrics.Views != nil {
		snap.GitHub.Views = &analyze.GitHubTrafficMetric{Count: metrics.Views.Count, Uniques: metrics.Views.Uniques}
	}
	if metrics.Clones != nil {
		snap.GitHub.Clones = &analyze.GitHubTrafficMetric{Count: metrics.Clones.Count, Uniques: metrics.Clones.Uniques}
	}
}

func filterCommitsSince(commits []gitlog.Commit, since time.Time) []gitlog.Commit {
	filtered := make([]gitlog.Commit, 0, len(commits))
	for _, commit := range commits {
		if !commit.Date.Before(since) {
			filtered = append(filtered, commit)
		}
	}
	return filtered
}

func normalizeFlagArgs(args []string, flagsWithValues map[string]bool) []string {
	if len(args) == 0 {
		return args
	}
	var flags []string
	var positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positionals = append(positionals, args[i+1:]...)
			break
		}
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)
			name := strings.TrimLeft(arg, "-")
			if before, _, ok := strings.Cut(name, "="); ok {
				name = before
			}
			if flagsWithValues[name] && !strings.Contains(arg, "=") && i+1 < len(args) {
				flags = append(flags, args[i+1])
				i++
			}
			continue
		}
		positionals = append(positionals, arg)
	}
	return append(flags, positionals...)
}

func firstArg(args []string, fallback string) string {
	if len(args) == 0 || args[0] == "" {
		return fallback
	}
	return args[0]
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, `ginsights - GitHub-style local repository insights

Usage:
  ginsights serve [repo] [--port 43117] [--since YYYY-MM-DD] [--no-cache] [--workspace] [--github-api owner/name]
  ginsights build [repo] --out report [--since YYYY-MM-DD] [--no-cache] [--workspace] [--github-api owner/name]
  ginsights json [repo] [--since YYYY-MM-DD] [--no-cache] [--workspace] [--github-api owner/name]
  ginsights cache-clear [repo]
  ginsights doctor [repo]

Examples:
  ginsights serve .
  ginsights serve . --workspace
  ginsights build ~/src/project --out report --since 2026-07-01
  ginsights json . --since 2026-07-01 > insights.json`)
}
