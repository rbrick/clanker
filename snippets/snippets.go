package snippets

import (
	"context"
	"log"
	"strings"

	"github.com/google/uuid"
	"github.com/rbrick/clanker/database"
	"github.com/rbrick/clanker/database/models"
	"github.com/rbrick/clanker/objectstore"
)

type File struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Language string `json:"language"`
}

type Snippets struct {
	snippetRepo database.Repository[models.Snippet]
	fileRepo    database.Repository[models.SnippetFile]
	gitFileRepo database.Repository[models.SnippetGitFile]
	objectStore objectstore.Store
	gitStore    *GitStore
	publicURL   string
}

type Option func(*Snippets)

func WithGitStore(store *GitStore, gitFileRepo database.Repository[models.SnippetGitFile], publicURL string) Option {
	return func(s *Snippets) {
		s.gitStore = store
		s.gitFileRepo = gitFileRepo
		s.publicURL = strings.TrimRight(publicURL, "/")
	}
}

func WithGitObjectStore(store *GitStore, objectStore objectstore.Store, publicURL string) Option {
	return func(s *Snippets) {
		s.gitStore = store
		s.objectStore = objectStore
		s.publicURL = strings.TrimRight(publicURL, "/")
	}
}

func NewSnippets(snippetRepo database.Repository[models.Snippet], fileRepo database.Repository[models.SnippetFile], opts ...Option) *Snippets {
	s := &Snippets{snippetRepo: snippetRepo, fileRepo: fileRepo}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Snippets) CreateSnippet(content, language string) (*models.Snippet, error) {
	return s.CreateSnippetWithFiles([]File{{Path: "snippet", Content: content, Language: language}})
}

func (s *Snippets) CreateSnippetWithFiles(files []File) (*models.Snippet, error) {
	if len(files) == 0 {
		files = []File{{Path: "snippet"}}
	}

	snippet := &models.Snippet{}
	if err := s.snippetRepo.Create(snippet); err != nil {
		return nil, err
	}

	normalizedFiles := make([]File, 0, len(files))
	for _, file := range files {
		path := file.Path
		if path == "" {
			path = "snippet"
		}
		file.Path = path
		normalizedFiles = append(normalizedFiles, file)

		if err := s.fileRepo.Create(&models.SnippetFile{
			SnippetID: snippet.ID,
			Path:      path,
			Content:   file.Content,
			Language:  file.Language,
		}); err != nil {
			return nil, err
		}
	}

	if err := s.createGitRepo(context.Background(), snippet.ID, normalizedFiles); err != nil {
		// A git clone URL is a convenience feature for snippets, not the source of
		// truth. Keep the web snippet usable even if git repo materialization or
		// object storage has a transient/configuration failure.
		log.Printf("failed to create git repo for snippet %s; continuing with web snippet: %v", snippet.ID, err)
	}

	return s.GetSnippetByID(snippet.ID)
}

func (s *Snippets) GetSnippetByID(id uuid.UUID) (*models.Snippet, error) {
	snippets, err := s.snippetRepo.Where("id = ?", id)
	if err != nil {
		return nil, err
	}
	if len(snippets) == 0 {
		return nil, nil
	}

	files, err := s.fileRepo.Where("snippet_id = ?", id)
	if err != nil {
		return nil, err
	}
	snippets[0].Files = files
	snippets[0].GitURL = s.GitURL(snippets[0].ID)
	return &snippets[0], nil
}

func (s *Snippets) GitURL(id uuid.UUID) string {
	if s.publicURL == "" {
		return ""
	}
	return s.publicURL + "/git/" + id.String() + ".git"
}

func (s *Snippets) GetGitFile(ctx context.Context, id uuid.UUID, path string) ([]byte, string, error) {
	if s.objectStore != nil {
		return s.objectStore.Get(ctx, gitObjectKey(id, path))
	}
	if s.gitFileRepo == nil {
		return nil, "", nil
	}
	files, err := s.gitFileRepo.Where("snippet_id = ? AND path = ?", id, path)
	if err != nil || len(files) == 0 {
		return nil, "", err
	}
	return files[0].Content, "application/octet-stream", nil
}

func (s *Snippets) createGitRepo(ctx context.Context, id uuid.UUID, files []File) error {
	if s.gitStore == nil || (s.gitFileRepo == nil && s.objectStore == nil) {
		return nil
	}
	repoFiles, err := s.gitStore.BuildRepo(ctx, id, files)
	if err != nil {
		return err
	}
	for path, content := range repoFiles {
		if s.objectStore != nil {
			if err := s.objectStore.Put(ctx, gitObjectKey(id, path), content, "application/octet-stream"); err != nil {
				return err
			}
			continue
		}
		if err := s.gitFileRepo.Create(&models.SnippetGitFile{SnippetID: id, Path: path, Content: content}); err != nil {
			return err
		}
	}
	return nil
}

func gitObjectKey(id uuid.UUID, path string) string {
	return "snippets/" + id.String() + ".git/" + strings.TrimPrefix(path, "/")
}
