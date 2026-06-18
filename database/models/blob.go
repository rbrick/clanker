package models

import "time"

type Blob struct {
	ID        string    `json:"id" gorm:"primaryKey;column:id"`
	MediaType string    `json:"media_type" gorm:"column:media_type;index"`
	Data      string    `json:"-" gorm:"column:data;type:text"` // Base64-encoded blob data.
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`
}

func (Blob) TableName() string {
	return "blobs"
}
