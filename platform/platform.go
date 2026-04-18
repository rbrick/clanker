package platform

type Sender struct {
	Platform string
	ID       string
	Username string
}

type Content struct {
	Text     string
	AudioURL string
	ImageURL string
}

type Message struct {
	// Message ID from the given platform, used for replying to the message
	ID      string
	Sender  *Sender
	Content *Content
}

type Platform interface {
	HandleMessage(*Message)
	Reply(*Message, string)

	Init() error
}
