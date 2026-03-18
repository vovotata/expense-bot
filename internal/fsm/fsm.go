package fsm

import (
	"context"
	"sync"
	"time"
)

// StateStore manages wizard states for users.
type StateStore interface {
	Get(ctx context.Context, userID int64) (*WizardState, error)
	Set(ctx context.Context, state *WizardState) error
	Delete(ctx context.Context, userID int64) error
}

// MemoryStore is an in-memory StateStore backed by sync.Map with TTL cleanup.
type MemoryStore struct {
	data  sync.Map
	locks sync.Map // per-user mutexes to prevent race conditions
	ttl   time.Duration
	done  chan struct{}
}

// NewMemoryStore creates a new in-memory store with background TTL cleanup.
func NewMemoryStore(ttl time.Duration) *MemoryStore {
	ms := &MemoryStore{
		ttl:  ttl,
		done: make(chan struct{}),
	}
	go ms.cleanupLoop()
	return ms
}

func (ms *MemoryStore) getUserLock(userID int64) *sync.Mutex {
	val, _ := ms.locks.LoadOrStore(userID, &sync.Mutex{})
	return val.(*sync.Mutex)
}

// Lock acquires a per-user lock. Must be paired with Unlock.
func (ms *MemoryStore) Lock(userID int64) {
	ms.getUserLock(userID).Lock()
}

// Unlock releases a per-user lock.
func (ms *MemoryStore) Unlock(userID int64) {
	ms.getUserLock(userID).Unlock()
}

func (ms *MemoryStore) Get(_ context.Context, userID int64) (*WizardState, error) {
	val, ok := ms.data.Load(userID)
	if !ok {
		return nil, nil
	}
	state := val.(*WizardState)
	if time.Since(state.LastActiveAt) > ms.ttl {
		ms.data.Delete(userID)
		return nil, nil
	}
	// Return a copy to prevent concurrent mutation
	cp := *state
	return &cp, nil
}

func (ms *MemoryStore) Set(_ context.Context, state *WizardState) error {
	state.LastActiveAt = time.Now()
	// Store a copy
	cp := *state
	ms.data.Store(state.UserID, &cp)
	return nil
}

func (ms *MemoryStore) Delete(_ context.Context, userID int64) error {
	ms.data.Delete(userID)
	return nil
}

// Stop terminates the background cleanup goroutine.
func (ms *MemoryStore) Stop() {
	close(ms.done)
}

func (ms *MemoryStore) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ms.data.Range(func(key, value any) bool {
				state := value.(*WizardState)
				if time.Since(state.LastActiveAt) > ms.ttl {
					ms.data.Delete(key)
				}
				return true
			})
		case <-ms.done:
			return
		}
	}
}
