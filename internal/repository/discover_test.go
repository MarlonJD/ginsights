package repository

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverFindsRootNestedAndGitFileRepositories(t *testing.T) {
	workspace := t.TempDir()
	mustMkdir(t, filepath.Join(workspace, ".git"))
	mustMkdir(t, filepath.Join(workspace, "apps", "surveil", ".git"))
	worktree := filepath.Join(workspace, "shared", "auth")
	mustMkdir(t, worktree)
	if err := os.WriteFile(filepath.Join(worktree, ".git"), []byte("gitdir: /tmp/example\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustMkdir(t, filepath.Join(workspace, "node_modules", "ignored", ".git"))

	roots, err := Discover(workspace)
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) != 3 {
		t.Fatalf("roots = %+v, want 3 repositories", roots)
	}
	want := []string{".", filepath.Join("apps", "surveil"), filepath.Join("shared", "auth")}
	for i, root := range roots {
		if root.RelativePath != want[i] {
			t.Fatalf("roots[%d].RelativePath = %q, want %q", i, root.RelativePath, want[i])
		}
	}
}

func TestDiscoverRejectsDirectoryWithoutRepositories(t *testing.T) {
	_, err := Discover(t.TempDir())
	if err == nil {
		t.Fatal("Discover succeeded, want no repositories error")
	}
}

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}
