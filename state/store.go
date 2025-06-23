// state/store.go
// Clean generic store implementation - just storage, no complexity

package state

import (
	"sync"
	"sync/atomic"
	"time"
)

/*
	Store[T] - Simple Generic Storage

	Just a concurrent key-value store with:
	- O(1) access using sync.Map
	- Atomic counting
	- Generic type support
	- Clean API: Set, Get, Forget, Clear, Count, GetAll
*/

// Store provides generic thread-safe storage
type Store[T any] struct {
	data  sync.Map
	count int64 // atomic counter
}

// NewStore creates a new generic store
func NewStore[T any]() *Store[T] {
	return &Store[T]{}
}

// Set stores an item with the given key
func (s *Store[T]) Set(key string, value T) {
	// Check if key already exists
	_, exists := s.data.Load(key)

	// Store the value
	s.data.Store(key, value)

	// Update counter only if it's a new key
	if !exists {
		atomic.AddInt64(&s.count, 1)
	}
}

// Get retrieves an item by key
func (s *Store[T]) Get(key string) (T, bool) {
	if value, exists := s.data.Load(key); exists {
		return value.(T), true
	}

	var zero T
	return zero, false
}

// Forget removes an item by key
func (s *Store[T]) Forget(key string) bool {
	if _, exists := s.data.LoadAndDelete(key); exists {
		atomic.AddInt64(&s.count, -1)
		return true
	}
	return false
}

// Clear removes all items
func (s *Store[T]) Clear() {
	s.data.Range(func(key, value interface{}) bool {
		s.data.Delete(key)
		return true
	})
	atomic.StoreInt64(&s.count, 0)
}

// Count returns the number of items in the store
func (s *Store[T]) Count() int64 {
	return atomic.LoadInt64(&s.count)
}

// GetAll returns all items as a slice
func (s *Store[T]) GetAll() []T {
	var items []T

	s.data.Range(func(key, value interface{}) bool {
		items = append(items, value.(T))
		return true
	})

	return items
}

// GetKeys returns all keys as a slice
func (s *Store[T]) GetKeys() []string {
	var keys []string

	s.data.Range(func(key, value interface{}) bool {
		keys = append(keys, key.(string))
		return true
	})

	return keys
}

// Exists checks if a key exists
func (s *Store[T]) Exists(key string) bool {
	_, exists := s.data.Load(key)
	return exists
}

// ForEach iterates over all items
func (s *Store[T]) ForEach(fn func(key string, value T) bool) {
	s.data.Range(func(key, value interface{}) bool {
		return fn(key.(string), value.(T))
	})
}

// === CORE DATA TYPES ===

// Subscriber represents a function handler for an action
type Subscriber struct {
	ID        string      `json:"id"`
	ChannelID string      `json:"actionId"`
	Handler   HandlerFunc `json:"-"` // Function not serializable
	CreatedAt time.Time   `json:"createdAt"`
}

// Payload represents stored payload data
type Payload struct {
	ID      string      `json:"id"`
	Payload interface{} `json:"payload"`
}

// Timer represents a scheduled timer in the timeline
type Timer struct {
	ID            string                                     `json:"id"`
	ActionID      string                                     `json:"actionId"`
	Type          string                                     `json:"type"`
	Interval      time.Duration                              `json:"interval"`
	Delay         time.Duration                              `json:"delay"`
	Repeat        int                                        `json:"repeat"`
	Executed      int                                        `json:"executed"`
	Status        string                                     `json:"status"`
	NextExecution time.Time                                  `json:"nextExecution"`
	LastExecution time.Time                                  `json:"lastExecution"`
	CreatedAt     time.Time                                  `json:"createdAt"`
	ExecuteFunc   func(actionID string, payload interface{}) `json:"-"`
	Payload       interface{}                                `json:"payload"`
}

// BranchStore represents a workflow branch
type BranchStore struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Actions    []string               `json:"actions"`
	Conditions map[string]interface{} `json:"conditions"`
	Status     string                 `json:"status"`
	CreatedAt  time.Time              `json:"createdAt"`
	UpdatedAt  time.Time              `json:"updatedAt"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// HandlerFunc represents a function that handles action execution
type HandlerFunc func(payload interface{}) interface{}
