package platform

import (
	"context"
	"errors"
	"log"
	"sync"
)

type Manager struct {
	platforms []Platform
}

func NewManager(platforms ...Platform) *Manager { return &Manager{platforms: platforms} }

func (m *Manager) Start(ctx context.Context) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(m.platforms))

	for _, p := range m.platforms {
		if p == nil {
			continue
		}
		if err := p.Init(); err != nil {
			return err
		}
		wg.Add(1)
		go func(p Platform) {
			defer wg.Done()
			log.Printf("starting platform %T", p)
			if err := p.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
				errCh <- err
			}
		}(p)
	}

	go func() { wg.Wait(); close(errCh) }()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err, ok := <-errCh:
		if !ok {
			return nil
		}
		return err
	}
}
