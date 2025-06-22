// context/store.go
// Generic store implementation for Cyre Go
// Provides O(1) concurrent access with proper cleanup

package context

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"
)

/*
	Generic Store Implementation

	Provides thread-safe storage with:
	- O(1) concurrent access using sync.Map
	- Automatic cleanup and memory management
	- Generic type support for type safety
	- Count tracking with atomic operations
	- Proper cleanup and iteration
*/

// === GENERIC STORE ===

// Store provides generic thread-safe storage with O(1) access
type Store[T any] struct {
	data     sync.Map
	count    int64 // atomic counter
	maxItems int   // optional limit
}

// NewStore creates a new generic store with optional size limit
func NewStore[T any](maxItems ...int) *Store[T] {
	store := &Store[T]{}

	if len(maxItems) > 0 && maxItems[0] > 0 {
		store.maxItems = maxItems[0]
	}

	return store
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

		// Optional: enforce max items limit
		if s.maxItems > 0 && atomic.LoadInt64(&s.count) > int64(s.maxItems) {
			s.cleanup()
		}
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

// cleanup removes oldest items when maxItems is exceeded
func (s *Store[T]) cleanup() {
	// Simple cleanup: remove some items to get under limit
	// In a production system, you might implement LRU or other strategies
	currentCount := atomic.LoadInt64(&s.count)
	itemsToRemove := currentCount - int64(s.maxItems) + 10 // Remove extra for buffer

	if itemsToRemove <= 0 {
		return
	}

	removed := int64(0)
	s.data.Range(func(key, value interface{}) bool {
		if removed >= itemsToRemove {
			return false // Stop iteration
		}

		s.data.Delete(key)
		removed++
		return true
	})

	atomic.AddInt64(&s.count, -removed)
}

// === PAYLOAD STORE ===

// PayloadStore provides specialized storage for action payloads
type PayloadStore struct {
	store *Store[interface{}]
	mu    sync.RWMutex
}

// NewPayloadStore creates a new payload store
func NewPayloadStore(maxItems ...int) *PayloadStore {
	return &PayloadStore{
		store: NewStore[interface{}](maxItems...),
	}
}

// Set stores a payload for an action
func (ps *PayloadStore) Set(actionID string, payload interface{}) {
	ps.store.Set(actionID, payload)
}

// Get retrieves a payload for an action
func (ps *PayloadStore) Get(actionID string) (interface{}, bool) {
	return ps.store.Get(actionID)
}

// Forget removes a payload for an action
func (ps *PayloadStore) Forget(actionID string) {
	ps.store.Forget(actionID)
}

// Clear removes all payloads
func (ps *PayloadStore) Clear() {
	ps.store.Clear()
}

// Count returns the number of stored payloads
func (ps *PayloadStore) Count() int64 {
	return ps.store.Count()
}

// CompareAndSet compares current payload with new payload and sets if different
func (ps *PayloadStore) CompareAndSet(actionID string, newPayload interface{}) bool {
	currentPayload, exists := ps.Get(actionID)
	if !exists {
		ps.Set(actionID, newPayload)
		return true // New payload
	}

	// Compare payloads (deep comparison using JSON)
	if ps.payloadsEqual(currentPayload, newPayload) {
		return false // No change
	}

	ps.Set(actionID, newPayload)
	return true // Changed
}

// payloadsEqual compares two payloads for equality
func (ps *PayloadStore) payloadsEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Use JSON comparison for deep equality
	aJSON, err1 := json.Marshal(a)
	bJSON, err2 := json.Marshal(b)

	if err1 != nil || err2 != nil {
		// Fallback to simple comparison if JSON fails
		return a == b
	}

	return string(aJSON) == string(bJSON)
}

// === SUBSCRIBER TYPE ===

// Subscriber represents a function handler for an action
type Subscriber struct {
	ID        string      `json:"id"`
	ActionID  string      `json:"actionId"`
	Handler   HandlerFunc `json:"-"` // Function not serializable
	CreatedAt time.Time   `json:"createdAt"`
}

// HandlerFunc represents a function that handles action execution
type HandlerFunc func(payload interface{}) interface{}

// === ACTION METRICS TYPE ===

// ActionMetrics represents performance metrics for an action
type ActionMetrics struct {
	// Call metrics
	Calls      int64 `json:"calls"`
	Executions int64 `json:"executions"`
	Errors     int64 `json:"errors"`

	// Protection metrics
	Throttled int64 `json:"throttled"`
	Debounced int64 `json:"debounced"`
	Skipped   int64 `json:"skipped"`

	// Performance metrics
	TotalLatency      time.Duration `json:"totalLatency"`
	MinLatency        time.Duration `json:"minLatency"`
	MaxLatency        time.Duration `json:"maxLatency"`
	AverageLatency    time.Duration `json:"averageLatency"`
	LastExecutionTime int64         `json:"lastExecutionTime"`

	// Additional counters
	ExecutionCount int64 `json:"executionCount"`
	ErrorCount     int64 `json:"errorCount"`
}

// === TIMER TYPE ===

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

// === BRANCH STORE TYPE ===

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
