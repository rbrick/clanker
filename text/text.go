package text

type Chatter struct {
	ID       string `json:"id"`
	Username string `json:"username"`
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

type Message struct {
	// Message ID from the given platform, used for replying to the message
	ID        string   `json:"id"`
	Platform  string   `json:"platform"`
	RepliedTo *Chatter `json:"replied_to,omitempty"` // The message this message is replying to, if any
	Sender    *Chatter `json:"sender,omitempty"`
	Content   *Content `json:"content,omitempty"`
	Chat      *Chat    `json:"chat,omitempty"`
}
