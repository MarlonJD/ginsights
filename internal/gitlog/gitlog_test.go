package gitlog

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestParseLog(t *testing.T) {
	input := []byte("\x1eabc\x1fAda\x1fada@example.com\x1f2026-07-01T12:00:00+03:00\x1fInitial commit\n" +
		"10\t2\tmain.go\n" +
		"-\t-\timage.png\n" +
		"\x1edef\x1fLinus\x1flinus@example.com\x1f2026-07-02T12:00:00+03:00\x1fSecond\n" +
		"3\t4\tREADME.md\n")

	commits, err := parseLog(input)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(commits), 2; got != want {
		t.Fatalf("len(commits) = %d, want %d", got, want)
	}
	if got, want := commits[0].Files[0].Path, "main.go"; got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
	if commits[0].Files[1].Additions != 0 || commits[0].Files[1].Deletions != 0 {
		t.Fatalf("binary file stats should become zero: %+v", commits[0].Files[1])
	}
}

func TestCollectorListsHashesAndCollectsSelectedHashes(t *testing.T) {
	repo := testGitRepo(t)
	commitFile(t, repo, "old.go", "package demo\n", "2026-06-30T12:00:00+00:00", "old commit")
	commitFile(t, repo, "new.go", "package demo\nfunc New() {}\n", "2026-07-01T12:00:00+00:00", "new commit")
	collector := NewCollector(repo)

	hashes, err := collector.ListHashes(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(hashes) != 2 {
		t.Fatalf("hashes = %#v, want two commits", hashes)
	}

	commits, err := collector.CollectHashes(context.Background(), []string{hashes[0]})
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 1 {
		t.Fatalf("commits = %+v, want one selected commit", commits)
	}
	if commits[0].Hash != hashes[0] || commits[0].Subject != "new commit" {
		t.Fatalf("commit = %+v, want selected newest commit", commits[0])
	}
	if len(commits[0].Files) != 1 || commits[0].Files[0].Path != "new.go" {
		t.Fatalf("files = %+v, want new.go numstat", commits[0].Files)
	}
}

func BenchmarkParseLog(b *testing.B) {
	input := benchmarkLogFixture(500)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		commits, err := parseLog(input)
		if err != nil {
			b.Fatal(err)
		}
		if len(commits) != 500 {
			b.Fatalf("len(commits) = %d, want 500", len(commits))
		}
	}
}

func testGitRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.name", "Ada")
	runGit(t, repo, "config", "user.email", "ada@example.com")
	return repo
}

func commitFile(t *testing.T, repo, name, body, date, message string) {
	t.Helper()
	path := filepath.Join(repo, name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	runGit(t, repo, "add", name)
	cmd := exec.Command("git", "-C", repo, "commit", "-m", message)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Ada",
		"GIT_AUTHOR_EMAIL=ada@example.com",
		"GIT_COMMITTER_NAME=Ada",
		"GIT_COMMITTER_EMAIL=ada@example.com",
		"GIT_AUTHOR_DATE="+date,
		"GIT_COMMITTER_DATE="+date,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git commit %q failed: %v\n%s", message, err, string(out))
	}
}

func runGit(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmdArgs := append([]string{"-C", repo}, args...)
	out, err := exec.Command("git", cmdArgs...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(out))
	}
}

func benchmarkLogFixture(commits int) []byte {
	var b strings.Builder
	for i := 0; i < commits; i++ {
		hash := fmt.Sprintf("%040d", i)
		day := i%28 + 1
		b.WriteString(recordSep)
		b.WriteString(hash)
		b.WriteString(unitSep)
		b.WriteString("Ada")
		b.WriteString(unitSep)
		b.WriteString("ada@example.com")
		b.WriteString(unitSep)
		b.WriteString(fmt.Sprintf("2026-07-%02dT12:00:00+00:00", day))
		b.WriteString(unitSep)
		b.WriteString("Commit ")
		b.WriteString(strconv.Itoa(i))
		b.WriteByte('\n')
		for file := 0; file < 3; file++ {
			b.WriteString("10\t2\tinternal/file")
			b.WriteString(strconv.Itoa(file))
			b.WriteString(".go\n")
		}
	}
	return []byte(b.String())
}
