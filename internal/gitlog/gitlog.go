package gitlog

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	recordSep = "\x1e"
	unitSep   = "\x1f"
	logFormat = "--format=" + recordSep + "%H" + unitSep + "%an" + unitSep + "%ae" + unitSep + "%ad" + unitSep + "%s"
)

type Collector struct {
	repo string
}

type Commit struct {
	Hash        string       `json:"hash"`
	AuthorName  string       `json:"author_name"`
	AuthorEmail string       `json:"author_email"`
	Date        time.Time    `json:"date"`
	Subject     string       `json:"subject"`
	Files       []FileChange `json:"files"`
}

type FileChange struct {
	Path      string `json:"path"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

func NewCollector(repo string) Collector {
	return Collector{repo: repo}
}

func (c Collector) Collect(ctx context.Context) ([]Commit, error) {
	if err := c.ensureRepo(ctx); err != nil {
		return nil, err
	}
	args := []string{
		"-C", c.repo,
		"log",
		"--all",
		"--date=iso-strict",
		"--numstat",
		logFormat,
	}
	out, err := c.runGit(ctx, args)
	if err != nil {
		return nil, err
	}
	commits, err := parseLog(out)
	if err != nil {
		return nil, err
	}
	return commits, nil
}

func (c Collector) ListHashes(ctx context.Context) ([]string, error) {
	if err := c.ensureRepo(ctx); err != nil {
		return nil, err
	}
	out, err := c.runGit(ctx, []string{"-C", c.repo, "log", "--all", "--format=%H"})
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(bytes.NewReader(out))
	var hashes []string
	for scanner.Scan() {
		hash := strings.TrimSpace(scanner.Text())
		if hash != "" {
			hashes = append(hashes, hash)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan git hashes: %w", err)
	}
	return hashes, nil
}

func (c Collector) CollectHashes(ctx context.Context, hashes []string) ([]Commit, error) {
	if len(hashes) == 0 {
		return nil, nil
	}
	if err := c.ensureRepo(ctx); err != nil {
		return nil, err
	}
	args := []string{
		"-C", c.repo,
		"log",
		"--no-walk",
		"--date=iso-strict",
		"--numstat",
		logFormat,
	}
	args = append(args, hashes...)
	out, err := c.runGit(ctx, args)
	if err != nil {
		return nil, err
	}
	commits, err := parseLog(out)
	if err != nil {
		return nil, err
	}
	return commits, nil
}

func (c Collector) runGit(ctx context.Context, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

func (c Collector) ensureRepo(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "git", "-C", c.repo, "rev-parse", "--show-toplevel")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("not a git repository or git unavailable: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func parseLog(data []byte) ([]Commit, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	var commits []Commit
	var current *Commit
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, recordSep) {
			if current != nil {
				commits = append(commits, *current)
			}
			commit, err := parseHeader(strings.TrimPrefix(line, recordSep))
			if err != nil {
				return nil, err
			}
			current = &commit
			continue
		}
		if current == nil || strings.TrimSpace(line) == "" {
			continue
		}
		fc, ok := parseNumstat(line)
		if ok {
			current.Files = append(current.Files, fc)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan git log: %w", err)
	}
	if current != nil {
		commits = append(commits, *current)
	}
	return commits, nil
}

func parseHeader(header string) (Commit, error) {
	parts := strings.SplitN(header, unitSep, 5)
	if len(parts) != 5 {
		return Commit{}, fmt.Errorf("malformed git log header: %q", header)
	}
	dt, err := time.Parse(time.RFC3339, parts[3])
	if err != nil {
		return Commit{}, fmt.Errorf("parse commit date %q: %w", parts[3], err)
	}
	return Commit{
		Hash:        parts[0],
		AuthorName:  parts[1],
		AuthorEmail: parts[2],
		Date:        dt,
		Subject:     parts[4],
	}, nil
}

func parseNumstat(line string) (FileChange, bool) {
	parts := strings.SplitN(line, "\t", 3)
	if len(parts) != 3 {
		return FileChange{}, false
	}
	additions := parseStat(parts[0])
	deletions := parseStat(parts[1])
	path := strings.TrimSpace(parts[2])
	if path == "" {
		return FileChange{}, false
	}
	return FileChange{Path: path, Additions: additions, Deletions: deletions}, true
}

func parseStat(v string) int {
	if v == "-" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return n
}
