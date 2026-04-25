package models

import "time"

type Snippet struct {
	ID        int       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	Content   string    `json:"content" gorm:"column:content;type:text"`
	Language  string    `json:"language" gorm:"column:language;type:text;"` // the programming language of the snippet, e.g. "python", "javascript", etc.
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
}

func (Snippet) TableName() string {
	return "snippets"
}
