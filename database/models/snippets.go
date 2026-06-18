package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Snippet struct {
	ID        uuid.UUID     `json:"id" gorm:"column:id;type:uuid;primaryKey"`
	Files     []SnippetFile `json:"files" gorm:"foreignKey:SnippetID"`
	CreatedAt time.Time     `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time     `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

func (s *Snippet) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

func (Snippet) TableName() string {
	return "snippets"
}

type SnippetFile struct {
	ID        uuid.UUID `json:"id" gorm:"column:id;type:uuid;primaryKey"`
	SnippetID uuid.UUID `json:"snippet_id" gorm:"column:snippet_id;type:uuid;index"`
	Path      string    `json:"path" gorm:"column:path;type:text"`
	Content   string    `json:"content" gorm:"column:content;type:text"`
	Language  string    `json:"language" gorm:"column:language;type:text"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

func (f *SnippetFile) BeforeCreate(tx *gorm.DB) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	return nil
}

func (SnippetFile) TableName() string {
	return "snippet_files"
}
