// context/state.go
// StateManager with IoStore rename and IO interface integration

package context

import (
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/neuralline/cyre-go/config"
	"github.com/neuralline/cyre-go/types"
)

/*
	StateManager - Enhanced state management

	Key Changes:
	- Renamed 'actions' field to 'ioStore'
	- IoStore now works with types.IO instead of ActionConfig
	- Maintains all existing functionality with proper naming
*/

// === CORE TYPES (unchanged) ===

// Subscriber represents a handler subscription
type Subscriber struct {
	ID        string      `json:"id"`
	ActionID  string      `json:"actionId"`
	Handler   HandlerFunc `json:"-"` // Not serializable
	CreatedAt time.Time   `json:"createdAt"`
}

// Timer represents a scheduled operation
type Timer struct {
	ID                   string                                     `json:"id"`
	ActionID             string                                     `json:"actionId"`
	Type                 string                                     `json:"type"`
	Interval             time.Duration                              `json:"interval"`
	Delay                time.Duration                              `json:"delay"`
	Repeat               int                                        `json:"repeat"` // -1 for infinite
	Executed             int                                        `json:"executed"`
	Status               string                                     `json:"status"` // "active", "paused", "completed"
	NextExecution        time.Time                                  `json:"nextExecution"`
	CreatedAt            time.Time                                  `json:"createdAt"`
	LastExecution        time.Time                                  `json:"lastExecution"`
	ExecuteFunc          func(actionID string, payload interface{}) `json:"-"`
	Payload              interface{}                                `json:"payload"`
	TimeoutID            interface{}                                `json:"timeoutId,omitempty"`
	RecuperationInterval interface{}                                `json:"recuperationInterval,omitempty"`
}

// BranchStore represents workflow branch information
type BranchStore struct {
	ID        string                 `json:"id"`
	ParentID  string                 `json:"parentId"`
	Children  []string               `json:"children"`
	Data      map[string]interface{} `json:"data"`
	CreatedAt time.Time              `json:"createdAt"`
	UpdatedAt time.Time              `json:"updatedAt"`
}

// HandlerFunc represents an action handler function
type HandlerFunc func(payload interface{}) interface{}

// ActionMetrics tracks performance metrics for actions
type ActionMetrics struct {
	LastExecutionTime int64         `json:"lastExecutionTime"`
	ExecutionCount    int64         `json:"executionCount"`
	Errors            []string      `json:"errors"`
	Calls             int64         `json:"calls"`
	Executions        int64         `json:"executions"`
	ErrorCount        int64         `json:"errorCount"`
	Throttled         int64         `json:"throttled"`
	Debounced         int64         `json:"debounced"`
	Skipped           int64         `json:"skipped"`
	AverageLatency    time.Duration `json:"averageLatency"`
	TotalLatency      time.Duration `json:"totalLatency"`
	MinLatency        time.Duration `json:"minLatency"`
	MaxLatency        time.Duration `json:"maxLatency"`
}

// === PAYLOAD STORE (unchanged) ===

type PayloadStore struct {
	data map[string]interface{}
	mu   sync.RWMutex
}

func NewPayloadStore() *PayloadStore {
	return &PayloadStore{
		data: make(map[string]interface{}),
	}
}

func (ps *PayloadStore) Set(key string, value interface{}) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.data[key] = value
}

func (ps *PayloadStore) Get(key string) (interface{}, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	value, ok := ps.data[key]
	return value, ok
}

func (ps *PayloadStore) Forget(key string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	delete(ps.data, key)
}

func (ps *PayloadStore) Clear() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.data = make(map[string]interface{})
}

func (ps *PayloadStore) Count() int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return len(ps.data)
}

func (ps *PayloadStore) GetAll() map[string]interface{} {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	result := make(map[string]interface{})
	for k, v := range ps.data {
		result[k] = v
	}
	return result
}

// === GENERIC STORE (unchanged) ===

type Store[T any] struct {
	data    sync.Map
	count   int64
	maxSize int64
}

func NewStore[T any](maxSize int64) *Store[T] {
	return &Store[T]{
		maxSize: maxSize,
	}
}

func (s *Store[T]) Set(key string, value T) {
	if _, loaded := s.data.LoadOrStore(key, value); !loaded {
		atomic.AddInt64(&s.count, 1)
	} else {
		s.data.Store(key, value)
	}
}

func (s *Store[T]) Get(key string) (T, bool) {
	if value, ok := s.data.Load(key); ok {
		return value.(T), true
	}
	var zero T
	return zero, false
}

func (s *Store[T]) Forget(key string) bool {
	if _, loaded := s.data.LoadAndDelete(key); loaded {
		atomic.AddInt64(&s.count, -1)
		return true
	}
	return false
}

func (s *Store[T]) Count() int64 {
	return atomic.LoadInt64(&s.count)
}

func (s *Store[T]) GetAll() []T {
	var items []T
	s.data.Range(func(key, value interface{}) bool {
		items = append(items, value.(T))
		return true
	})
	return items
}

func (s *Store[T]) Clear() {
	s.data.Range(func(key, value interface{}) bool {
		s.data.Delete(key)
		return true
	})
	atomic.StoreInt64(&s.count, 0)
}

// === CENTRAL STATE MANAGER (with IoStore rename) ===

type StateManager struct {
	// Core stores with O(1) access - RENAMED: actions -> ioStore
	ioStore     *Store[*types.IO]      // IoStore: Channel configurations (was 'actions')
	subscribers *Store[*Subscriber]    // Subscriber store: Handler subscriptions
	payloads    *PayloadStore          // Payload store: Current payload data
	metrics     *Store[*ActionMetrics] // Metrics store: Performance metrics
	timeline    *Store[*Timer]         // Timeline store: Scheduled tasks
	branches    *Store[*BranchStore]   // Branch store: Workflow branches

	// Protection state
	throttleMap sync.Map
	debounceMap sync.Map

	// System state
	systemState config.SystemState

	// System metadata
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
		defaultState := config.DefaultSystemState
		defaultState.Init = true
		defaultState.LastUpdate = time.Now().UnixMilli()

		GlobalState = &StateManager{
			ioStore:     NewStore[*types.IO](config.MaxRetainedActions), // RENAMED: actions -> ioStore
			subscribers: NewStore[*Subscriber](config.MaxRetainedActions),
			payloads:    NewPayloadStore(),
			metrics:     NewStore[*ActionMetrics](config.MaxRetainedActions),
			timeline:    NewStore[*Timer](config.MaxRetainedActions),
			branches:    NewStore[*BranchStore](config.MaxRetainedActions),
			systemState: defaultState,
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

// === IO OPERATIONS (Updated to use types.IO) ===

// IOStore provides TypeScript-style IO operations
type IOStore struct {
	sm *StateManager
}

// Set action configuration (matching TypeScript io.set)
func (io *IOStore) Set(action *types.IO) error {
	if action.ID == "" {
		return fmt.Errorf("IO state: Channel must have an id")
	}

	// Update timestamp like TypeScript version
	channel := *action // Copy the IO struct
	channel.Timestamp = time.Now().UnixMilli()

	if channel.Type == "" {
		channel.Type = channel.ID // Default type to ID like TypeScript
	}

	io.sm.ioStore.Set(action.ID, &channel) // Use ioStore instead of actions

	// Initialize metrics if not exists
	if _, exists := io.sm.metrics.Get(action.ID); !exists {
		io.sm.metrics.Set(action.ID, &ActionMetrics{
			MinLatency: time.Hour,
		})
	}

	// Update store count in system state
	io.sm.updateStoreCount("channels", int(io.sm.ioStore.Count()))

	return nil
}

// Get action configuration (matching TypeScript io.get)
func (io *IOStore) Get(actionID string) (*types.IO, bool) {
	return io.sm.ioStore.Get(actionID) // Use ioStore instead of actions
}

// Forget action configuration (matching TypeScript io.forget)
func (io *IOStore) Forget(actionID string) bool {
	// Also remove payload
	io.sm.payloads.Forget(actionID)
	result := io.sm.ioStore.Forget(actionID) // Use ioStore instead of actions

	// Update store count
	io.sm.updateStoreCount("channels", int(io.sm.ioStore.Count()))

	return result
}

// Clear all action configurations (matching TypeScript io.clear)
func (io *IOStore) Clear() {
	io.sm.ioStore.Clear() // Use ioStore instead of actions
	io.sm.payloads.Clear()
	io.sm.updateStoreCount("channels", 0)
}

// GetAll action configurations (matching TypeScript io.getAll)
func (io *IOStore) GetAll() []*types.IO {
	return io.sm.ioStore.GetAll() // Use ioStore instead of actions
}

// GetMetrics retrieves metrics for an action (matching TypeScript io.getMetrics)
func (io *IOStore) GetMetrics(actionID string) (*ActionMetrics, bool) {
	return io.sm.metrics.Get(actionID)
}

// === SUBSCRIBERS OPERATIONS (unchanged) ===

type SubscriberStore struct {
	sm *StateManager
}

func (sub *SubscriberStore) Add(subscriber *Subscriber) error {
	if subscriber.ID == "" || subscriber.Handler == nil {
		return fmt.Errorf("Invalid subscriber format")
	}

	sub.sm.subscribers.Set(subscriber.ID, subscriber)
	sub.sm.updateStoreCount("subscribers", int(sub.sm.subscribers.Count()))

	return nil
}

func (sub *SubscriberStore) Get(actionID string) (*Subscriber, bool) {
	return sub.sm.subscribers.Get(actionID)
}

func (sub *SubscriberStore) Forget(actionID string) bool {
	result := sub.sm.subscribers.Forget(actionID)
	sub.sm.updateStoreCount("subscribers", int(sub.sm.subscribers.Count()))
	return result
}

func (sub *SubscriberStore) Clear() {
	sub.sm.subscribers.Clear()
	sub.sm.updateStoreCount("subscribers", 0)
}

func (sub *SubscriberStore) GetAll() []*Subscriber {
	return sub.sm.subscribers.GetAll()
}

// === TIMELINE OPERATIONS (unchanged) ===

type TimelineStore struct {
	sm *StateManager
}

func (tl *TimelineStore) Add(timer *Timer) error {
	if timer.ID == "" {
		return fmt.Errorf("Timer must have an ID")
	}

	tl.sm.timeline.Set(timer.ID, timer)

	activeCount := len(tl.GetActive())
	tl.sm.systemState.ActiveFormations = activeCount
	tl.sm.updateStoreCount("timeline", int(tl.sm.timeline.Count()))

	return nil
}

func (tl *TimelineStore) Get(timerID string) (*Timer, bool) {
	return tl.sm.timeline.Get(timerID)
}

func (tl *TimelineStore) Forget(timerID string) bool {
	if timer, exists := tl.sm.timeline.Get(timerID); exists {
		timer.Status = "cancelled"
	}

	result := tl.sm.timeline.Forget(timerID)

	activeCount := len(tl.GetActive())
	tl.sm.systemState.ActiveFormations = activeCount
	tl.sm.updateStoreCount("timeline", int(tl.sm.timeline.Count()))

	return result
}

func (tl *TimelineStore) Clear() {
	timers := tl.sm.timeline.GetAll()
	for _, timer := range timers {
		timer.Status = "cancelled"
	}

	tl.sm.timeline.Clear()
	tl.sm.systemState.ActiveFormations = 0
	tl.sm.updateStoreCount("timeline", 0)
}

func (tl *TimelineStore) GetAll() []*Timer {
	return tl.sm.timeline.GetAll()
}

func (tl *TimelineStore) GetActive() []*Timer {
	timers := tl.sm.timeline.GetAll()
	var active []*Timer
	for _, timer := range timers {
		if timer.Status == "active" {
			active = append(active, timer)
		}
	}
	return active
}

// === BRANCH OPERATIONS (unchanged) ===

type BranchStoreManager struct {
	sm *StateManager
}

func (bs *BranchStoreManager) Set(branch *BranchStore) error {
	if branch.ID == "" {
		return fmt.Errorf("Branch must have an ID")
	}

	branch.UpdatedAt = time.Now()
	if branch.CreatedAt.IsZero() {
		branch.CreatedAt = time.Now()
	}

	bs.sm.branches.Set(branch.ID, branch)
	bs.sm.updateStoreCount("branches", int(bs.sm.branches.Count()))

	return nil
}

func (bs *BranchStoreManager) Get(branchID string) (*BranchStore, bool) {
	return bs.sm.branches.Get(branchID)
}

func (bs *BranchStoreManager) Forget(branchID string) bool {
	result := bs.sm.branches.Forget(branchID)
	bs.sm.updateStoreCount("branches", int(bs.sm.branches.Count()))
	return result
}

func (bs *BranchStoreManager) Clear() {
	bs.sm.branches.Clear()
	bs.sm.updateStoreCount("branches", 0)
}

func (bs *BranchStoreManager) GetAll() []*BranchStore {
	return bs.sm.branches.GetAll()
}

// === TYPESCRIPT-STYLE API ACCESS ===

// IO returns the IO store (matching TypeScript export const io)
func (sm *StateManager) IO() *IOStore {
	return &IOStore{sm: sm}
}

// Subscribers returns the subscriber store
func (sm *StateManager) Subscribers() *SubscriberStore {
	return &SubscriberStore{sm: sm}
}

// Timeline returns the timeline store
func (sm *StateManager) Timeline() *TimelineStore {
	return &TimelineStore{sm: sm}
}

// Branches returns the branch store
func (sm *StateManager) Branches() *BranchStoreManager {
	return &BranchStoreManager{sm: sm}
}

// === PAYLOAD MANAGEMENT (unchanged) ===

func (sm *StateManager) SetPayload(actionID string, payload interface{}) {
	sm.payloads.Set(actionID, payload)
}

func (sm *StateManager) GetPayload(actionID string) (interface{}, bool) {
	return sm.payloads.Get(actionID)
}

func (sm *StateManager) ForgetPayload(actionID string) {
	sm.payloads.Forget(actionID)
}

func (sm *StateManager) ComparePayload(actionID string, newPayload interface{}) bool {
	currentPayload, exists := sm.GetPayload(actionID)
	if !exists {
		return true
	}

	if newPayload == nil && currentPayload == nil {
		return false
	}
	if newPayload == nil || currentPayload == nil {
		return true
	}

	currentJSON, err1 := json.Marshal(currentPayload)
	newJSON, err2 := json.Marshal(newPayload)

	if err1 != nil || err2 != nil {
		return true
	}

	return string(currentJSON) != string(newJSON)
}

// === REMAINING METHODS (unchanged but update references) ===

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
		metrics.LastExecutionTime = now.UnixMilli()

		metrics.TotalLatency += duration
		if duration < metrics.MinLatency {
			metrics.MinLatency = duration
		}
		if duration > metrics.MaxLatency {
			metrics.MaxLatency = duration
		}

		executions := atomic.LoadInt64(&metrics.Executions)
		if executions > 0 {
			metrics.AverageLatency = time.Duration(int64(metrics.TotalLatency) / executions)
		}

		atomic.AddInt64(&metrics.ExecutionCount, 1)

	case "error":
		atomic.AddInt64(&metrics.ErrorCount, 1)
	case "throttled":
		atomic.AddInt64(&metrics.Throttled, 1)
	case "debounced":
		atomic.AddInt64(&metrics.Debounced, 1)
	case "skipped":
		atomic.AddInt64(&metrics.Skipped, 1)
	}

	sm.metrics.Set(actionID, metrics)
}

func (sm *StateManager) GetMetrics(actionID string) (*ActionMetrics, bool) {
	return sm.metrics.Get(actionID)
}

// === PROTECTION STATE (unchanged) ===

func (sm *StateManager) SetThrottleTime(actionID string, execTime time.Time) {
	sm.throttleMap.Store(actionID, execTime)
}

func (sm *StateManager) GetThrottleTime(actionID string) (time.Time, bool) {
	if value, exists := sm.throttleMap.Load(actionID); exists {
		return value.(time.Time), true
	}
	return time.Time{}, false
}

func (sm *StateManager) SetDebounceTimer(actionID string, timer interface{}) {
	sm.debounceMap.Store(actionID, timer)
}

func (sm *StateManager) GetDebounceTimer(actionID string) (interface{}, bool) {
	if value, exists := sm.debounceMap.Load(actionID); exists {
		return value, true
	}
	return nil, false
}

func (sm *StateManager) ClearDebounceTimer(actionID string) {
	sm.debounceMap.Delete(actionID)
}

// === SYSTEM STATE MANAGEMENT (unchanged) ===

func (sm *StateManager) GetSystemState() config.SystemState {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.systemState
}

func (sm *StateManager) UpdateSystemState(updates map[string]interface{}) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.systemState.LastUpdate = time.Now().UnixMilli()

	if stress, ok := updates["stress"].(float64); ok {
		sm.systemState.Stress.Combined = stress
	}
	if activeFormations, ok := updates["activeFormations"].(int); ok {
		sm.systemState.ActiveFormations = activeFormations
	}
}

func (sm *StateManager) updateStoreCount(storeType string, count int) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	switch storeType {
	case "channels":
		sm.systemState.Store.Channels = count
	case "subscribers":
		sm.systemState.Store.Subscribers = count
	case "timeline":
		sm.systemState.Store.Timeline = count
	case "branches":
		sm.systemState.Store.Branches = count
	}

	sm.systemState.LastUpdate = time.Now().UnixMilli()
}

// === CLEANUP OPERATIONS (updated to use ioStore) ===

// ForgetAction removes action and all associated state (TypeScript-style forget)
func (sm *StateManager) ForgetAction(actionID string) bool {
	// Clean up all related state using TypeScript naming
	sm.IO().Forget(actionID)
	sm.Subscribers().Forget(actionID)
	sm.ForgetPayload(actionID)
	sm.metrics.Forget(actionID)
	sm.ClearDebounceTimer(actionID)
	sm.throttleMap.Delete(actionID)
	return true
}

// Clear removes all state from all stores (TypeScript-style clear)
func (sm *StateManager) Clear() {
	sm.IO().Clear()
	sm.Subscribers().Clear()
	sm.Timeline().Clear()
	sm.Branches().Clear()
	sm.payloads.Clear()
	sm.metrics.Clear()

	// Clear protection state
	sm.throttleMap.Range(func(key, value interface{}) bool {
		sm.throttleMap.Delete(key)
		return true
	})

	sm.debounceMap.Range(func(key, value interface{}) bool {
		sm.debounceMap.Delete(key)
		return true
	})

	// Reset system state to default
	sm.systemState = config.DefaultSystemState
	sm.systemState.Init = true
	sm.systemState.LastUpdate = time.Now().UnixMilli()
}

// Reset resets system state to defaults (TypeScript metricsState.reset equivalent)
func (sm *StateManager) Reset() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.systemState = config.DefaultSystemState
	sm.systemState.Init = true
	sm.systemState.LastUpdate = time.Now().UnixMilli()
}

// === SYSTEM STATUS ===

func (sm *StateManager) IsInitialized() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.initialized && sm.systemState.Init
}

func (sm *StateManager) GetStats() map[string]interface{} {
	systemState := sm.GetSystemState()

	return map[string]interface{}{
		"system":           systemState.System,
		"breathing":        systemState.Breathing,
		"performance":      systemState.Performance,
		"stress":           systemState.Stress,
		"store":            systemState.Store,
		"goroutines":       runtime.NumGoroutine(),
		"uptime_ns":        time.Since(sm.startTime).Nanoseconds(),
		"initialized":      sm.initialized,
		"lastUpdate":       systemState.LastUpdate,
		"activeFormations": systemState.ActiveFormations,
		"locked":           systemState.Locked,
		"hibernating":      systemState.Hibernating,
	}
}

// === STORES ACCESS (matching TypeScript exports) ===

// Stores provides read-only access to stores (matching TypeScript export const stores)
type Stores struct {
	IO          *Store[*types.IO] // RENAMED: was actions, now ioStore
	Subscribers *Store[*Subscriber]
	Timeline    *Store[*Timer]
	Branches    *Store[*BranchStore]
	Metrics     *Store[*ActionMetrics]
}

// GetStores returns readonly access to stores
func (sm *StateManager) GetStores() Stores {
	return Stores{
		IO:          sm.ioStore, // RENAMED: was sm.actions, now sm.ioStore
		Subscribers: sm.subscribers,
		Timeline:    sm.timeline,
		Branches:    sm.branches,
		Metrics:     sm.metrics,
	}
}
