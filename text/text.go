package text

type Chatter struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name,omitempty"`
}

type Content struct {
	Text     string `json:"text"`
	AudioURL string `json:"audio_url,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	VideoURL string `json:"video_url,omitempty"`
}

type Chat struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type ContextMessage struct {
	ID               string   `json:"id"`
	Timestamp        int64    `json:"timestamp,omitempty"`
	Sender           *Chatter `json:"sender,omitempty"`
	Content          *Content `json:"content,omitempty"`
	ReplyToMessageID string   `json:"reply_to_message_id,omitempty"`
}

type Message struct {
	// Message ID from the given platform, used for replying to the message
	ID               string           `json:"id"`
	Timestamp        int64            `json:"timestamp,omitempty"`
	Platform         string           `json:"platform"`
	ReplyToMessageID string           `json:"reply_to_message_id,omitempty"`
	RepliedTo        *Chatter         `json:"replied_to,omitempty"` // The sender of the message this message is replying to, if any
	Sender           *Chatter         `json:"sender,omitempty"`
	Content          *Content         `json:"content,omitempty"`
	Chat             *Chat            `json:"chat,omitempty"`
	Context          []ContextMessage `json:"context,omitempty"` // Recent messages in this chat, oldest first.
}
