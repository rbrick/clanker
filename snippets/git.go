package snippets

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/uuid"
)

type GitStore struct{}

func NewGitStore() *GitStore { return &GitStore{} }

func (g *GitStore) BuildRepo(ctx context.Context, id uuid.UUID, files []File) (map[string][]byte, error) {
	workDir, err := os.MkdirTemp("", "clanker-snippet-work-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(workDir)

	bareDir, err := os.MkdirTemp("", "clanker-snippet-bare-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(bareDir)
	bareRepo := filepath.Join(bareDir, id.String()+".git")

	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		name := cleanSnippetPath(file.Path)
		if name == "" {
			name = "snippet"
		}
		fullPath := filepath.Join(workDir, name)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(fullPath, []byte(file.Content), 0o644); err != nil {
			return nil, err
		}
	}

	repo, err := git.PlainInit(workDir, false)
	if err != nil {
		return nil, fmt.Errorf("git init failed: %w", err)
	}
	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("git worktree failed: %w", err)
	}
	if _, err := worktree.Add("."); err != nil {
		return nil, fmt.Errorf("git add failed: %w", err)
	}
	commitHash, err := worktree.Commit("Initial snippet", &git.CommitOptions{
		Author: &object.Signature{Name: "Clanker", Email: "clanker@localhost", When: time.Now()},
	})
	if err != nil {
		return nil, fmt.Errorf("git commit failed: %w", err)
	}

	if _, err := git.PlainCloneContext(ctx, bareRepo, true, &git.CloneOptions{URL: workDir}); err != nil {
		return nil, fmt.Errorf("git clone --bare failed: %w", err)
	}
	if err := writeDumbHTTPInfo(bareRepo, commitHash.String()); err != nil {
		return nil, fmt.Errorf("git update-server-info failed: %w", err)
	}

	repoFiles := map[string][]byte{}
	if err := filepath.WalkDir(bareRepo, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(bareRepo, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		repoFiles[filepath.ToSlash(rel)] = data
		return nil
	}); err != nil {
		return nil, err
	}

	return repoFiles, nil
}

func writeDumbHTTPInfo(bareRepo, headHash string) error {
	infoDir := filepath.Join(bareRepo, "info")
	objectsInfoDir := filepath.Join(bareRepo, "objects", "info")
	if err := os.MkdirAll(infoDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(objectsInfoDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(infoDir, "refs"), []byte(headHash+"\trefs/heads/master\n"), 0o644); err != nil {
		return err
	}

	packEntries, err := filepath.Glob(filepath.Join(bareRepo, "objects", "pack", "*.pack"))
	if err != nil {
		return err
	}
	var packs strings.Builder
	for _, pack := range packEntries {
		packs.WriteString("P ")
		packs.WriteString(filepath.Base(pack))
		packs.WriteByte('\n')
	}
	return os.WriteFile(filepath.Join(objectsInfoDir, "packs"), []byte(packs.String()), 0o644)
}

func cleanSnippetPath(name string) string {
	name = filepath.Clean(strings.TrimPrefix(name, "/"))
	if name == "." || strings.HasPrefix(name, "..") || filepath.IsAbs(name) || strings.HasPrefix(name, ".git") || strings.Contains(name, string(filepath.Separator)+".git"+string(filepath.Separator)) {
		return "snippet"
	}
	return name
}
