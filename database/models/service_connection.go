package models

import "time"

// ServiceConnection stores OAuth/service credentials scoped to one chat.
type ServiceConnection struct {
	ID           int       `json:"id" gorm:"primaryKey;autoIncrement;column:id"`
	Platform     string    `json:"platform" gorm:"index:idx_service_chat,unique;column:platform"`
	ChatID       int       `json:"chat_id" gorm:"index:idx_service_chat,unique;column:chat_id"`
	Service      string    `json:"service" gorm:"index:idx_service_chat,unique;column:service"`
	AccessToken  string    `json:"-" gorm:"column:access_token"`
	RefreshToken string    `json:"-" gorm:"column:refresh_token"`
	ExpiresAt    time.Time `json:"expires_at" gorm:"column:expires_at"`
	Metadata     string    `json:"metadata" gorm:"column:metadata"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (ServiceConnection) TableName() string { return "service_connections" }

// ServiceOAuthState tracks pending chat-scoped OAuth flows.
type ServiceOAuthState struct {
	ID        int       `json:"id" gorm:"primaryKey;autoIncrement;column:id"`
	State     string    `json:"state" gorm:"uniqueIndex;column:state"`
	Platform  string    `json:"platform" gorm:"column:platform"`
	ChatID    int       `json:"chat_id" gorm:"column:chat_id"`
	Service   string    `json:"service" gorm:"column:service"`
	CreatedAt time.Time `json:"created_at"`
}

func (ServiceOAuthState) TableName() string { return "service_oauth_states" }
