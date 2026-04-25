package snippets

import (
	"github.com/rbrick/clanker/database"
	"github.com/rbrick/clanker/database/models"
)

type Snippets struct {
	repo database.Repository[models.Snippet]
}

func NewSnippets(repo database.Repository[models.Snippet]) *Snippets {
	return &Snippets{repo: repo}
}

func (s *Snippets) CreateSnippet(content, language string) (*models.Snippet, error) {
	snippet := &models.Snippet{
		Content:  content,
		Language: language,
	}

	if err := s.repo.Create(snippet); err != nil {
		return nil, err
	}

	return snippet, nil
}

func (s *Snippets) GetAllSnippets() ([]models.Snippet, error) {
	return s.repo.FindAll()
}

func (s *Snippets) GetSnippetsByLanguage(language string) ([]models.Snippet, error) {
	return s.repo.Where("language = ?", language)
}

func (s *Snippets) GetSnippetByID(id int) (*models.Snippet, error) {
	snippets, err := s.repo.Where("id = ?", id)
	if err != nil {
		return nil, err
	}
	if len(snippets) == 0 {
		return nil, nil
	}
	return &snippets[0], nil
}
