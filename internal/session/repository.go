package session

import (
	"context"
	"sync"
	"time"
)

type Repository interface {
	Create(ctx context.Context, tokenHash []byte, userID string, expiresAt time.Time) error
	// UserID returns ErrUnauthenticated if the session is missing or expired at now.
	UserID(ctx context.Context, tokenHash []byte, now time.Time) (string, error)
	// Delete is idempotent: deleting a missing session is not an error.
	Delete(ctx context.Context, tokenHash []byte) error
}

type entry struct {
	userID    string
	expiresAt time.Time
}

// MemoryRepository backs the unit tests.
type MemoryRepository struct {
	mu       sync.RWMutex
	sessions map[string]entry // key: string(tokenHash)
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{sessions: make(map[string]entry)}
}

func (m *MemoryRepository) Create(_ context.Context, tokenHash []byte, userID string, expiresAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[string(tokenHash)] = entry{userID: userID, expiresAt: expiresAt}
	return nil
}

func (m *MemoryRepository) UserID(_ context.Context, tokenHash []byte, now time.Time) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	e, ok := m.sessions[string(tokenHash)]
	if !ok || !e.expiresAt.After(now) {
		return "", ErrUnauthenticated
	}
	return e.userID, nil
}

func (m *MemoryRepository) Delete(_ context.Context, tokenHash []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, string(tokenHash))
	return nil
}
