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
	"github.com/multica-ai/ginsights/internal/doclint"
	"github.com/multica-ai/ginsights/internal/gitlog"
	"github.com/multica-ai/ginsights/internal/report"
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
	sinceValue := fs.String("since", "", "only include commits on or after YYYY-MM-DD")
	if err := fs.Parse(normalizeFlagArgs(args, map[string]bool{"port": true, "since": true})); err != nil {
		return 2
	}
	opts, err := snapshotOptionsFromFlags(*sinceValue)
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
	sinceValue := fs.String("since", "", "only include commits on or after YYYY-MM-DD")
	if err := fs.Parse(normalizeFlagArgs(args, map[string]bool{"out": true, "since": true})); err != nil {
		return 2
	}
	opts, err := snapshotOptionsFromFlags(*sinceValue)
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
	sinceValue := fs.String("since", "", "only include commits on or after YYYY-MM-DD")
	if err := fs.Parse(normalizeFlagArgs(args, map[string]bool{"since": true})); err != nil {
		return 2
	}
	opts, err := snapshotOptionsFromFlags(*sinceValue)
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
	Since time.Time
}

func snapshotOptionsFromFlags(sinceValue string) (snapshotOptions, error) {
	if sinceValue == "" {
		return snapshotOptions{}, nil
	}
	since, err := parseSince(sinceValue)
	if err != nil {
		return snapshotOptions{}, err
	}
	return snapshotOptions{Since: since}, nil
}

func parseSince(value string) (time.Time, error) {
	since, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid --since %q: use YYYY-MM-DD, for example --since 2026-07-01", value)
	}
	return since, nil
}

func snapshot(ctx context.Context, repo string, opts snapshotOptions) (analyze.Snapshot, error) {
	collector := gitlog.NewCollector(repo)
	history, err := collector.Collect(ctx)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return analyze.Snapshot{}, fmt.Errorf("git repository not found: %w", err)
		}
		return analyze.Snapshot{}, err
	}
	if !opts.Since.IsZero() {
		history = filterCommitsSince(history, opts.Since)
	}
	return analyze.BuildSnapshot(repo, history, time.Now()), nil
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
  ginsights serve [repo] [--port 43117] [--since YYYY-MM-DD]
  ginsights build [repo] --out report [--since YYYY-MM-DD]
  ginsights json [repo] [--since YYYY-MM-DD]
  ginsights doctor [repo]

Examples:
  ginsights serve .
  ginsights build ~/src/project --out report --since 2026-07-01
  ginsights json . --since 2026-07-01 > insights.json`)
}
