package user

import (
	"context"
	"sync"
)

// Repository is defined here, by the consumer. PostgresRepository is the real
// implementation; MemoryRepository backs the unit tests.
type Repository interface {
	// Create stores the user with an already-hashed password.
	Create(ctx context.Context, u User, passwordHash string) error
	Get(ctx context.Context, id string) (User, error)
	// Credentials looks up login data by (normalized) email; ErrNotFound if none.
	Credentials(ctx context.Context, email string) (userID, passwordHash string, err error)
	Update(ctx context.Context, u User) error
	Delete(ctx context.Context, id string) error
}

type MemoryRepository struct {
	mu     sync.RWMutex
	users  map[string]User
	hashes map[string]string // user id -> password hash, kept off User so it can't leak into JSON
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{users: make(map[string]User), hashes: make(map[string]string)}
}

func (m *MemoryRepository) Create(_ context.Context, u User, passwordHash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.emailTaken(u.Email, u.ID) {
		return ErrEmailTaken
	}
	m.users[u.ID] = u
	m.hashes[u.ID] = passwordHash
	return nil
}

func (m *MemoryRepository) Credentials(_ context.Context, email string) (string, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for id, u := range m.users {
		if u.Email == email {
			return id, m.hashes[id], nil
		}
	}
	return "", "", ErrNotFound
}

func (m *MemoryRepository) Get(_ context.Context, id string) (User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.users[id]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

func (m *MemoryRepository) Update(_ context.Context, u User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, ok := m.users[u.ID]
	if !ok {
		return ErrNotFound
	}
	if m.emailTaken(u.Email, u.ID) {
		return ErrEmailTaken
	}
	u.CreatedAt = existing.CreatedAt // immutable, as in the Postgres UPDATE
	m.users[u.ID] = u
	return nil
}

func (m *MemoryRepository) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.users[id]; !ok {
		return ErrNotFound
	}
	delete(m.users, id)
	delete(m.hashes, id)
	return nil
}

// emailTaken mirrors the UNIQUE constraint. Caller must hold m.mu.
func (m *MemoryRepository) emailTaken(email, exceptID string) bool {
	for id, u := range m.users {
		if id != exceptID && u.Email == email {
			return true
		}
	}
	return false
}
