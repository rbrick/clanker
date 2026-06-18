package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SnippetGitFile struct {
	ID        uuid.UUID `json:"id" gorm:"column:id;type:uuid;primaryKey"`
	SnippetID uuid.UUID `json:"snippet_id" gorm:"column:snippet_id;type:uuid;index:idx_snippet_git_file,unique"`
	Path      string    `json:"path" gorm:"column:path;type:text;index:idx_snippet_git_file,unique"`
	Content   []byte    `json:"-" gorm:"column:content"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

func (f *SnippetGitFile) BeforeCreate(tx *gorm.DB) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	return nil
}

func (SnippetGitFile) TableName() string {
	return "snippet_git_files"
}
