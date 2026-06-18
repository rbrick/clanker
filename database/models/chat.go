package models

type ChatMessage struct {
	ID             int    `json:"id" gorm:"primaryKey;autoIncrement;column:id"`
	Platform       string `json:"platform" gorm:"index;column:platform"`
	ChatID         int    `json:"chat_id" gorm:"index;column:chat_id"`
	ReplyToID      *int   `json:"reply_to_id" gorm:"column:reply_to_id;default:null"`
	SenderID       int    `json:"sender_id" gorm:"column:sender_id"`
	SenderUsername string `json:"sender_username" gorm:"column:sender_username;index"`
	SenderName     string `json:"sender_name" gorm:"column:sender_name;index"`
	Message        string `json:"message" gorm:"column:message"`
	Timestamp      int64  `json:"timestamp" gorm:"column:timestamp"`
}

func (ChatMessage) TableName() string {
	return "chat_messages"
}
