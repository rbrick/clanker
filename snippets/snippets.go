package snippets

import (
	"github.com/google/uuid"
	"github.com/rbrick/clanker/database"
	"github.com/rbrick/clanker/database/models"
)

type File struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Language string `json:"language"`
}

type Snippets struct {
	snippetRepo database.Repository[models.Snippet]
	fileRepo    database.Repository[models.SnippetFile]
}

func NewSnippets(snippetRepo database.Repository[models.Snippet], fileRepo database.Repository[models.SnippetFile]) *Snippets {
	return &Snippets{snippetRepo: snippetRepo, fileRepo: fileRepo}
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

	for _, file := range files {
		path := file.Path
		if path == "" {
			path = "snippet"
		}
		if err := s.fileRepo.Create(&models.SnippetFile{
			SnippetID: snippet.ID,
			Path:      path,
			Content:   file.Content,
			Language:  file.Language,
		}); err != nil {
			return nil, err
		}
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
	return &snippets[0], nil
}
