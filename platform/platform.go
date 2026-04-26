package platform

import (
	"context"

	"github.com/rbrick/clanker/text"
)

type PlatformMessageAdapter func(ctx context.Context, msg any) (*text.Message, error)

type PlatformConfig struct {
	//MessageAdapter Transforms the message into an internal platform agnostic format before being passed to the chat pipeline
	MessageAdapter PlatformMessageAdapter
	//Instructions define platform specific instructions for the agent to follow when responding to messages from this platform
	Instructions string
}

type Platform interface {
	Init() error
	Start(ctx context.Context) error
	Config() *PlatformConfig
}
