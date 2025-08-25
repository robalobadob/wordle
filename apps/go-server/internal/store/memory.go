// apps/go-server/internal/store/memory.go
//
// In-memory implementation of the game.Store interface.
// This is a lightweight persistence layer used for ephemeral game sessions,
// primarily in development/testing, or when durability is not required.
//
// Characteristics:
//   - Stores *game.Game objects keyed by ID in a map.
//   - Concurrency-safe via RWMutex (concurrent reads allowed, writes exclusive).
//   - State is lost when the process restarts.
//   - Errors are returned for missing game IDs on Get().

package store

import (
	"context"
	"errors"
	"sync"

	"github.com/robalobadob/wordle/apps/go-server/internal/game"
)

// Store defines the persistence interface for game sessions.
// Implementations may be backed by memory (this package), Redis, SQL, etc.
type Store interface {
	// Save persists or updates a game state.
	Save(ctx context.Context, g *game.Game) error

	// Get retrieves a game by ID.
	// Returns an error if the game is not found.
	Get(ctx context.Context, id string) (*game.Game, error)
}

// memory is an in-memory map-based Store implementation.
type memory struct {
	mu    sync.RWMutex           // guards games map
	games map[string]*game.Game  // keyed by Game.ID
}

// NewMemoryStore constructs a new in-memory Store.
func NewMemoryStore() Store {
	return &memory{games: make(map[string]*game.Game)}
}

// Save adds or updates the game in the map.
func (m *memory) Save(ctx context.Context, g *game.Game) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.games[g.ID] = g
	return nil
}

// Get looks up a game by ID.
// Returns a pointer to the stored *game.Game or an error if missing.
func (m *memory) Get(ctx context.Context, id string) (*game.Game, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if g, ok := m.games[id]; ok {
		return g, nil
	}
	return nil, errors.New("not found")
}
