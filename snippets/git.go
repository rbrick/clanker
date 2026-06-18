package snippets

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

	commands := [][]string{
		{"git", "init"},
		{"git", "config", "user.name", "Clanker"},
		{"git", "config", "user.email", "clanker@localhost"},
		{"git", "add", "."},
		{"git", "commit", "-m", "Initial snippet"},
		{"git", "clone", "--bare", workDir, bareRepo},
		{"git", "--git-dir", bareRepo, "update-server-info"},
	}
	for _, args := range commands {
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Dir = workDir
		if out, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("%s failed: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
		}
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

func cleanSnippetPath(name string) string {
	name = filepath.Clean(strings.TrimPrefix(name, "/"))
	if name == "." || strings.HasPrefix(name, "..") || filepath.IsAbs(name) || strings.HasPrefix(name, ".git") || strings.Contains(name, string(filepath.Separator)+".git"+string(filepath.Separator)) {
		return "snippet"
	}
	return name
}
