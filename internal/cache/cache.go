package cache

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/multica-ai/ginsights/internal/gitlog"
)

const (
	dirName   = ".ginsights-cache"
	fileName  = "commits-v1.json"
	version   = 1
	tempPerm  = 0o644
	cachePerm = 0o755
)

type Source interface {
	ListHashes(context.Context) ([]string, error)
	CollectHashes(context.Context, []string) ([]gitlog.Commit, error)
}

type Store struct {
	Dir string
}

type cacheFile struct {
	Version   int             `json:"version"`
	UpdatedAt time.Time       `json:"updated_at"`
	Commits   []gitlog.Commit `json:"commits"`
}

func DefaultDir(repo string) string {
	return filepath.Join(repo, dirName)
}

func Clear(repo string) error {
	return os.RemoveAll(DefaultDir(repo))
}

func Collect(ctx context.Context, source Source, store Store) ([]gitlog.Commit, error) {
	hashes, err := source.ListHashes(ctx)
	if err != nil {
		return nil, err
	}
	cached, err := store.Load()
	if err != nil {
		return nil, err
	}

	var missing []string
	for _, hash := range hashes {
		if _, ok := cached[hash]; !ok {
			missing = append(missing, hash)
		}
	}
	if len(missing) > 0 {
		fetched, err := source.CollectHashes(ctx, missing)
		if err != nil {
			return nil, err
		}
		for _, commit := range fetched {
			cached[commit.Hash] = commit
		}
	}

	ordered := make([]gitlog.Commit, 0, len(hashes))
	for _, hash := range hashes {
		commit, ok := cached[hash]
		if !ok {
			continue
		}
		ordered = append(ordered, commit)
	}
	if err := store.Save(ordered); err != nil {
		return nil, err
	}
	return ordered, nil
}

func (s Store) Load() (map[string]gitlog.Commit, error) {
	commits := map[string]gitlog.Commit{}
	data, err := os.ReadFile(filepath.Join(s.Dir, fileName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return commits, nil
		}
		return nil, err
	}
	var file cacheFile
	if err := json.Unmarshal(data, &file); err != nil {
		return commits, nil
	}
	if file.Version != version {
		return commits, nil
	}
	for _, commit := range file.Commits {
		if commit.Hash != "" {
			commits[commit.Hash] = commit
		}
	}
	return commits, nil
}

func (s Store) Save(commits []gitlog.Commit) error {
	if err := os.MkdirAll(s.Dir, cachePerm); err != nil {
		return err
	}
	file := cacheFile{Version: version, UpdatedAt: time.Now().UTC(), Commits: commits}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(s.Dir, fileName)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, tempPerm); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
