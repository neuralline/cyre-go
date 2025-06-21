// context/state.go
// Central state management with O(1) lookups and concurrent safety

package state

import (
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/neuralline/cyre-go/config"
)

/*
	C.Y.R.E - S.T.A.T.E

	Central state management system providing:
	- O(1) lookups using sync.Map for maximum performance
	- Separated concerns: IO config vs payload data
	- Thread-safe concurrent access
	- Memory-efficient storage with cleanup
	- Metrics tracking integrated
	- Backward compatibility with Cyre API
*/

// === CORE TYPES ===

// ActionConfig represents action configuration (no payload data)
type ActionConfig struct {
	ID             string         `json:"id"`
	Type           string         `json:"type,omitempty"`
	Interval       *time.Duration `json:"interval,omitempty"`
	Repeat         *int           `json:"repeat,omitempty"`
	Throttle       *time.Duration `json:"throttle,omitempty"`
	Debounce       *time.Duration `json:"debounce,omitempty"`
	DetectChanges  bool           `json:"detectChanges,omitempty"`
	Log            bool           `json:"log,omitempty"`
	Priority       string         `json:"priority,omitempty"`
	Timestamp      int64          `json:"timestamp"`
	LastExecTime   int64          `json:"lastExecTime,omitempty"`
	ExecutionCount int64          `json:"executionCount,omitempty"`
	Errors         []string       `json:"errors,omitempty"`
}

// HandlerFunc represents an action handler function
type HandlerFunc func(payload interface{}) interface{}

// Subscriber represents a handler subscription
type Subscriber struct {
	ID        string      `json:"id"`
	ActionID  string      `json:"actionId"`
	Handler   HandlerFunc `json:"-"` // Not serializable
	CreatedAt time.Time   `json:"createdAt"`
}

// ActionMetrics tracks performance metrics for actions
type ActionMetrics struct {
	Calls          int64         `json:"calls"`
	Executions     int64         `json:"executions"`
	Errors         int64         `json:"errors"`
	Throttled      int64         `json:"throttled"`
	Debounced      int64         `json:"debounced"`
	Skipped        int64         `json:"skipped"`
	LastExecution  time.Time     `json:"lastExecution"`
	AverageLatency time.Duration `json:"averageLatency"`
	TotalLatency   time.Duration `json:"totalLatency"`
	MinLatency     time.Duration `json:"minLatency"`
	MaxLatency     time.Duration `json:"maxLatency"`
}

// === STATE STORES ===

// PayloadStore is a specialized, concurrent-safe store for payloads
type PayloadStore struct {
	data map[string]interface{}
	mu   sync.RWMutex
}

// NewPayloadStore creates a new payload store
func NewPayloadStore() *PayloadStore {
	return &PayloadStore{
		data: make(map[string]interface{}),
	}
}

// Set stores a payload
func (ps *PayloadStore) Set(key string, value interface{}) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.data[key] = value
}

// Get retrieves a payload
func (ps *PayloadStore) Get(key string) (interface{}, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	value, ok := ps.data[key]
	return value, ok
}

// Delete removes a payload
func (ps *PayloadStore) Delete(key string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	delete(ps.data, key)
}

// Clear removes all payloads
func (ps *PayloadStore) Clear() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.data = make(map[string]interface{})
}

// Count returns the number of items in the store
func (ps *PayloadStore) Count() int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return len(ps.data)
}

// Store provides concurrent-safe storage with O(1) operations
type Store[T any] struct {
	data    sync.Map
	count   int64
	maxSize int64
}

// NewStore creates a new concurrent store
func NewStore[T any](maxSize int64) *Store[T] {
	return &Store[T]{
		maxSize: maxSize,
	}
}

// Set stores a value with O(1) complexity
func (s *Store[T]) Set(key string, value T) {
	if _, loaded := s.data.LoadOrStore(key, value); !loaded {
		atomic.AddInt64(&s.count, 1)
	} else {
		s.data.Store(key, value)
	}
}

// Get retrieves a value with O(1) complexity
func (s *Store[T]) Get(key string) (T, bool) {
	if value, ok := s.data.Load(key); ok {
		return value.(T), true
	}
	var zero T
	return zero, false
}

// Delete removes a value with O(1) complexity
func (s *Store[T]) Delete(key string) bool {
	if _, loaded := s.data.LoadAndDelete(key); loaded {
		atomic.AddInt64(&s.count, -1)
		return true
	}
	return false
}

// Count returns the current number of items
func (s *Store[T]) Count() int64 {
	return atomic.LoadInt64(&s.count)
}

// GetAll returns all values (expensive operation)
func (s *Store[T]) GetAll() []T {
	var items []T
	s.data.Range(func(key, value interface{}) bool {
		items = append(items, value.(T))
		return true
	})
	return items
}

// Clear removes all items
func (s *Store[T]) Clear() {
	s.data.Range(func(key, value interface{}) bool {
		s.data.Delete(key)
		return true
	})
	atomic.StoreInt64(&s.count, 0)
}

// === CENTRAL STATE MANAGER ===

// StateManager manages all Cyre state with separated concerns
type StateManager struct {
	// Core stores with O(1) access
	actions  *Store[*ActionConfig]  // Action configurations
	handlers *Store[*Subscriber]    // Handler subscriptions
	payloads *PayloadStore          // Specialized payload store
	metrics  *Store[*ActionMetrics] // Performance metrics

	// Protection state
	throttleMap sync.Map // actionID -> lastExecution time
	debounceMap sync.Map // actionID -> timer

	// System state
	initialized bool
	startTime   time.Time
	mu          sync.RWMutex
}

// GlobalState is the singleton state manager
var GlobalState *StateManager
var initOnce sync.Once

// Initialize creates and initializes the global state manager
func Initialize() *StateManager {
	initOnce.Do(func() {
		GlobalState = &StateManager{
			actions:     NewStore[*ActionConfig](config.MaxRetainedActions),
			handlers:    NewStore[*Subscriber](config.MaxRetainedActions),
			payloads:    NewPayloadStore(),
			metrics:     NewStore[*ActionMetrics](config.MaxRetainedActions),
			startTime:   time.Now(),
			initialized: true,
		}
	})
	return GlobalState
}

// GetState returns the global state manager
func GetState() *StateManager {
	if GlobalState == nil {
		return Initialize()
	}
	return GlobalState
}

// === ACTION MANAGEMENT ===

// SetAction stores action configuration (Cyre's io.set equivalent)
func (sm *StateManager) SetAction(action *ActionConfig) error {
	if action.ID == "" {
		return fmt.Errorf("action must have an ID")
	}

	action.Timestamp = time.Now().UnixNano()
	if action.Type == "" {
		action.Type = action.ID // Default type to ID like Cyre
	}

	sm.actions.Set(action.ID, action)

	// Initialize metrics if not exists
	if _, exists := sm.metrics.Get(action.ID); !exists {
		sm.metrics.Set(action.ID, &ActionMetrics{
			MinLatency: time.Hour, // Initialize to high value
		})
	}

	return nil
}

// GetAction retrieves action configuration
func (sm *StateManager) GetAction(actionID string) (*ActionConfig, bool) {
	return sm.actions.Get(actionID)
}

// === HANDLER MANAGEMENT ===

// SetHandler stores handler subscription (Cyre's subscriber equivalent)
func (sm *StateManager) SetHandler(actionID string, handler func(payload interface{}) interface{}) error {
	if actionID == "" {
		return fmt.Errorf("actionID cannot be empty")
	}
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	subscriber := &Subscriber{
		ID:        fmt.Sprintf("%s_handler", actionID),
		ActionID:  actionID,
		Handler:   HandlerFunc(handler),
		CreatedAt: time.Now(),
	}

	sm.handlers.Set(actionID, subscriber)
	return nil
}

// GetHandler retrieves handler for action
func (sm *StateManager) GetHandler(actionID string) (func(payload interface{}) interface{}, bool) {
	if subscriber, exists := sm.handlers.Get(actionID); exists {
		return func(payload interface{}) interface{} {
			return subscriber.Handler(payload)
		}, true
	}
	return nil, false
}

// === PAYLOAD MANAGEMENT ===

// SetPayload stores current payload for action
func (sm *StateManager) SetPayload(actionID string, payload interface{}) {
	sm.payloads.Set(actionID, payload)
}

// GetPayload retrieves current payload for action
func (sm *StateManager) GetPayload(actionID string) (interface{}, bool) {
	return sm.payloads.Get(actionID)
}

// ComparePayload checks if payload has changed (for detectChanges)
func (sm *StateManager) ComparePayload(actionID string, newPayload interface{}) bool {
	currentPayload, exists := sm.GetPayload(actionID)
	if !exists {
		return true // No previous payload, so it's changed
	}

	// Handle nil payloads explicitly
	if newPayload == nil && currentPayload == nil {
		return false // Both are nil, no change
	}
	if newPayload == nil || currentPayload == nil {
		return true // One is nil, the other isn't, so it's changed
	}

	// Deep comparison using JSON marshaling (fast enough for most cases)
	currentJSON, err1 := json.Marshal(currentPayload)
	newJSON, err2 := json.Marshal(newPayload)

	if err1 != nil || err2 != nil {
		return true // If we can't compare, assume changed
	}

	return string(currentJSON) != string(newJSON)
}

// === METRICS MANAGEMENT ===

// UpdateMetrics updates performance metrics for an action
func (sm *StateManager) UpdateMetrics(actionID string, metricType string, duration time.Duration) {
	metrics, exists := sm.metrics.Get(actionID)
	if !exists {
		metrics = &ActionMetrics{
			MinLatency: time.Hour,
		}
	}

	now := time.Now()

	switch metricType {
	case "call":
		atomic.AddInt64(&metrics.Calls, 1)
	case "execution":
		atomic.AddInt64(&metrics.Executions, 1)
		metrics.LastExecution = now

		// Update latency metrics
		metrics.TotalLatency += duration
		if duration < metrics.MinLatency {
			metrics.MinLatency = duration
		}
		if duration > metrics.MaxLatency {
			metrics.MaxLatency = duration
		}

		// Calculate average latency
		executions := atomic.LoadInt64(&metrics.Executions)
		if executions > 0 {
			metrics.AverageLatency = time.Duration(int64(metrics.TotalLatency) / executions)
		}

	case "error":
		atomic.AddInt64(&metrics.Errors, 1)
	case "throttled":
		atomic.AddInt64(&metrics.Throttled, 1)
	case "debounced":
		atomic.AddInt64(&metrics.Debounced, 1)
	case "skipped":
		atomic.AddInt64(&metrics.Skipped, 1)
	}

	sm.metrics.Set(actionID, metrics)
}

// GetMetrics retrieves metrics for an action
func (sm *StateManager) GetMetrics(actionID string) (*ActionMetrics, bool) {
	return sm.metrics.Get(actionID)
}

// === PROTECTION STATE ===

// SetThrottleTime records last execution time for throttling
func (sm *StateManager) SetThrottleTime(actionID string, execTime time.Time) {
	sm.throttleMap.Store(actionID, execTime)
}

// GetThrottleTime retrieves last execution time for throttling
func (sm *StateManager) GetThrottleTime(actionID string) (time.Time, bool) {
	if value, exists := sm.throttleMap.Load(actionID); exists {
		return value.(time.Time), true
	}
	return time.Time{}, false
}

// SetDebounceTimer stores debounce timer for action
func (sm *StateManager) SetDebounceTimer(actionID string, timer *time.Timer) {
	sm.debounceMap.Store(actionID, timer)
}

// GetDebounceTimer retrieves debounce timer for action
func (sm *StateManager) GetDebounceTimer(actionID string) (*time.Timer, bool) {
	if value, exists := sm.debounceMap.Load(actionID); exists {
		return value.(*time.Timer), true
	}
	return nil, false
}

// ClearDebounceTimer removes debounce timer for action
func (sm *StateManager) ClearDebounceTimer(actionID string) {
	if timer, exists := sm.GetDebounceTimer(actionID); exists {
		timer.Stop()
		sm.debounceMap.Delete(actionID)
	}
}

// === CLEANUP OPERATIONS ===

// ForgetAction removes action and all associated state (Cyre's forget equivalent)
func (sm *StateManager) ForgetAction(actionID string) bool {
	// Clean up all related state
	sm.actions.Delete(actionID)
	sm.handlers.Delete(actionID)
	sm.payloads.Delete(actionID)
	sm.metrics.Delete(actionID)
	sm.ClearDebounceTimer(actionID)
	sm.throttleMap.Delete(actionID)
	return true
}

// Clear removes all state from all stores
func (sm *StateManager) Clear() {
	sm.actions.Clear()
	sm.handlers.Clear()
	sm.payloads.Clear()
	sm.metrics.Clear()

	sm.throttleMap.Range(func(key, value interface{}) bool {
		sm.throttleMap.Delete(key)
		return true
	})

	sm.debounceMap.Range(func(key, value interface{}) bool {
		if timer, ok := value.(*time.Timer); ok {
			timer.Stop()
		}
		sm.debounceMap.Delete(key)
		return true
	})
}

// === SYSTEM STATUS ===

// IsInitialized checks if state manager is initialized
func (sm *StateManager) IsInitialized() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.initialized
}

// GetStats returns system statistics
func (sm *StateManager) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"actions":       sm.actions.Count(),
		"handlers":      sm.handlers.Count(),
		"payloads":      sm.payloads.Count(),
		"metrics":       sm.metrics.Count(),
		"goroutines":    runtime.NumGoroutine(),
		"uptime_ns":     time.Since(sm.startTime).Nanoseconds(),
		"initialized":   sm.initialized,
		"throttle_keys": sm.countMapKeys(&sm.throttleMap),
		"debounce_keys": sm.countMapKeys(&sm.debounceMap),
	}
}

func (sm *StateManager) countMapKeys(m *sync.Map) int {
	count := 0
	m.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}
