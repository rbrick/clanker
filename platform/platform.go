package platform

import (
	"context"

	"github.com/rbrick/clanker/text"
)

type Platform interface {
	HandleMessage(ctx context.Context, msg *text.Message) error
	Init() error
	Start(ctx context.Context) error
}
