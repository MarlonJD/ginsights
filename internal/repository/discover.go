package repository

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// Root identifies one Git repository inside a workspace.
type Root struct {
	Path         string
	RelativePath string
}

// Discover returns every Git repository rooted at or below root. It looks for
// filesystem Git boundaries directly so repositories ignored by a parent Git
// repository are still included.
func Discover(root string) ([]Root, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve workspace root %s: %w", root, err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("inspect workspace root %s: %w", abs, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("workspace root is not a directory: %s", abs)
	}

	var roots []Root
	err = filepath.WalkDir(abs, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() {
			return nil
		}
		if path != abs && shouldPrune(entry.Name()) {
			return filepath.SkipDir
		}
		if !IsRoot(path) {
			return nil
		}
		rel, err := filepath.Rel(abs, path)
		if err != nil {
			return fmt.Errorf("resolve repository path %s: %w", path, err)
		}
		roots = append(roots, Root{Path: path, RelativePath: rel})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("discover repositories under %s: %w", abs, err)
	}
	if len(roots) == 0 {
		return nil, fmt.Errorf("no Git repositories found under %s", abs)
	}

	sort.Slice(roots, func(i, j int) bool {
		if roots[i].RelativePath == "." {
			return true
		}
		if roots[j].RelativePath == "." {
			return false
		}
		return roots[i].RelativePath < roots[j].RelativePath
	})
	return roots, nil
}

// IsRoot reports whether path contains a .git directory or file. A file marker
// covers linked worktrees and submodules as well as ordinary repositories.
func IsRoot(path string) bool {
	_, err := os.Lstat(filepath.Join(path, ".git"))
	return err == nil
}

func shouldPrune(name string) bool {
	switch name {
	case ".git", ".ginsights-cache", "node_modules", "vendor":
		return true
	default:
		return false
	}
}
