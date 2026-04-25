package models

type ChatMessage struct {
	ID             int    `gorm:"primaryKey;autoIncrement;column:id"`
	ChatID         int    `gorm:"index:index:idx_chat_message;unique;column:chat_id"`
	ReplyToID      *int   `gorm:"index:index:idx_chat_message;unique;column:reply_to_id;default:null"`
	SenderID       int    `gorm:"index:index:idx_chat_message;unique;column:sender_id"`
	SenderUsername string `gorm:"index:index:idx_chat_message;unique;column:sender_username"`
	Message        string `gorm:"column:message"`
	Timestamp      int64  `gorm:"column:timestamp"`
}

func (ChatMessage) TableName() string {
	return "chat_messages"
}
