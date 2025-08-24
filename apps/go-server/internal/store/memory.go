package store

import (
	"context"
	"errors"
	"sync"

	"github.com/robalobadob/wordle/apps/go-server/internal/game"
)

type Store interface {
	Save(ctx context.Context, g *game.Game) error
	Get(ctx context.Context, id string) (*game.Game, error)
}

type memory struct {
	mu    sync.RWMutex
	games map[string]*game.Game
}

func NewMemoryStore() Store {
	return &memory{games: make(map[string]*game.Game)}
}

func (m *memory) Save(ctx context.Context, g *game.Game) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.games[g.ID] = g
	return nil
}

func (m *memory) Get(ctx context.Context, id string) (*game.Game, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if g, ok := m.games[id]; ok {
		return g, nil
	}
	return nil, errors.New("not found")
}
