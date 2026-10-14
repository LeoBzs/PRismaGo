package lift

import (
	"context"
	"slices"
	"sync"
)

// Repository is defined here, by the consumer, not by the storage layer.
// Every method is scoped to one user: a lift owned by someone else behaves
// exactly like a lift that doesn't exist.
type Repository interface {
	Create(ctx context.Context, l Lift) error
	Get(ctx context.Context, userID, id string) (Lift, error)
	List(ctx context.Context, f ListFilter) ([]Lift, error)
	Delete(ctx context.Context, userID, id string) error
}

type ListFilter struct {
	UserID   string // required
	Exercise string // empty = all exercises
}

// MemoryRepository backs the unit tests. Unlike Postgres it has no users
// table, so Create and List never return ErrUserNotFound.
type MemoryRepository struct {
	mu    sync.RWMutex
	lifts map[string]Lift
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{lifts: make(map[string]Lift)}
}

func (m *MemoryRepository) Create(_ context.Context, l Lift) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lifts[l.ID] = l
	return nil
}

func (m *MemoryRepository) Get(_ context.Context, userID, id string) (Lift, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	l, ok := m.lifts[id]
	if !ok || l.UserID != userID {
		return Lift{}, ErrNotFound
	}
	return l, nil
}

func (m *MemoryRepository) List(_ context.Context, f ListFilter) ([]Lift, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Lift, 0, len(m.lifts))
	for _, l := range m.lifts {
		if l.UserID != f.UserID || (f.Exercise != "" && l.Exercise != f.Exercise) {
			continue
		}
		out = append(out, l)
	}
	slices.SortFunc(out, func(a, b Lift) int { return b.PerformedAt.Compare(a.PerformedAt) })
	return out, nil
}

func (m *MemoryRepository) Delete(_ context.Context, userID, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if l, ok := m.lifts[id]; !ok || l.UserID != userID {
		return ErrNotFound
	}
	delete(m.lifts, id)
	return nil
}
