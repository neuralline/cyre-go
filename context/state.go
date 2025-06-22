// context/state.go
// Updated StateManager with MetricState integration
// Simplified to focus on data storage while MetricState handles intelligence

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
	StateManager - Enhanced state management with MetricState integration

	Key Changes:
	- Delegates intelligence to MetricState
	- Focuses on data storage and retrieval
	- Automatically updates MetricState when store counts change
	- Maintains all existing functionality for backward compatibility
*/

// === GLOBAL VARIABLES ===

var (
	GlobalState *StateManager
	initOnce    sync.Once
)

// StateManager remains the same but with MetricState integration
type StateManager struct {
	// Core stores with O(1) access
	ioStore     *Store[*types.IO]      // IoStore: Channel configurations
	subscribers *Store[*Subscriber]    // Subscriber store: Handler subscriptions
	payloads    *PayloadStore          // Payload store: Current payload data
	metrics     *Store[*ActionMetrics] // Metrics store: Performance metrics (user-facing)
	timeline    *Store[*Timer]         // Timeline store: Scheduled tasks
	branches    *Store[*BranchStore]   // Branch store: Workflow branches

	// Protection state
	throttleMap sync.Map
	debounceMap sync.Map

	// System state (now primarily for backward compatibility)
	systemState config.SystemState

	// MetricState integration
	metricState *MetricState // Reference to system brain

	// System metadata
	initialized bool
	startTime   time.Time
	mu          sync.RWMutex
}

// Enhanced initialization with MetricState integration
func Initialize() *StateManager {
	initOnce.Do(func() {
		defaultState := config.DefaultSystemState
		defaultState.Initialized = true // Use Initialized field instead of Init
		defaultState.LastUpdate = time.Now().UnixMilli()

		GlobalState = &StateManager{
			ioStore:     NewStore[*types.IO](config.MaxRetainedActions),
			subscribers: NewStore[*Subscriber](config.MaxRetainedActions),
			payloads:    NewPayloadStore(),
			metrics:     NewStore[*ActionMetrics](config.MaxRetainedActions),
			timeline:    NewStore[*Timer](config.MaxRetainedActions),
			branches:    NewStore[*BranchStore](config.MaxRetainedActions),
			systemState: defaultState,
			startTime:   time.Now(),
			initialized: true,
		}

		// Initialize MetricState after StateManager is ready
		GlobalState.metricState = InitializeMetricState()
	})
	return GlobalState
}

// GetState returns the global state manager instance
func GetState() *StateManager {
	if GlobalState == nil {
		return Initialize()
	}
	return GlobalState
}

// === ENHANCED IO OPERATIONS WITH METRICSTATE ===

// IOStore provides TypeScript-style IO operations with intelligence
type IOStore struct {
	sm *StateManager
}

// Set action configuration with MetricState update
func (io *IOStore) Set(action *types.IO) error {
	if action.ID == "" {
		return fmt.Errorf("IO state: Channel must have an id")
	}

	// Update timestamp
	channel := *action // Copy the IO struct
	channel.Timestamp = time.Now().UnixMilli()

	if channel.Type == "" {
		channel.Type = channel.ID // Default type to ID
	}

	// Store in IO store
	io.sm.ioStore.Set(action.ID, &channel)

	// Initialize metrics if not exists
	if _, exists := io.sm.metrics.Get(action.ID); !exists {
		io.sm.metrics.Set(action.ID, &ActionMetrics{
			MinLatency: time.Hour,
		})
	}

	// Update MetricState with new store counts
	io.sm.updateMetricStateStoreCounts()

	return nil
}

// Get action configuration (unchanged)
func (io *IOStore) Get(actionID string) (*types.IO, bool) {
	return io.sm.ioStore.Get(actionID)
}

// Forget action configuration with MetricState update
func (io *IOStore) Forget(actionID string) bool {
	// Remove payload
	io.sm.payloads.Forget(actionID)

	// Remove from IO store
	result := io.sm.ioStore.Forget(actionID)

	// Update MetricState with new store counts
	io.sm.updateMetricStateStoreCounts()

	return result
}

// Clear all action configurations with MetricState update
func (io *IOStore) Clear() {
	io.sm.ioStore.Clear()
	io.sm.payloads.Clear()

	// Update MetricState
	io.sm.updateMetricStateStoreCounts()
}

// GetAll action configurations (unchanged)
func (io *IOStore) GetAll() []*types.IO {
	return io.sm.ioStore.GetAll()
}

// GetMetrics retrieves metrics for an action (unchanged)
func (io *IOStore) GetMetrics(actionID string) (*ActionMetrics, bool) {
	return io.sm.metrics.Get(actionID)
}

// === ENHANCED SUBSCRIBERS OPERATIONS ===

type SubscriberStore struct {
	sm *StateManager
}

// Add subscriber with MetricState update
func (sub *SubscriberStore) Add(subscriber *Subscriber) error {
	if subscriber.ID == "" || subscriber.Handler == nil {
		return fmt.Errorf("Invalid subscriber format")
	}

	sub.sm.subscribers.Set(subscriber.ID, subscriber)

	// Update MetricState with new store counts
	sub.sm.updateMetricStateStoreCounts()

	return nil
}

// Get subscriber (unchanged)
func (sub *SubscriberStore) Get(actionID string) (*Subscriber, bool) {
	return sub.sm.subscribers.Get(actionID)
}

// Forget subscriber with MetricState update
func (sub *SubscriberStore) Forget(actionID string) bool {
	result := sub.sm.subscribers.Forget(actionID)

	// Update MetricState with new store counts
	sub.sm.updateMetricStateStoreCounts()

	return result
}

// Clear all subscribers with MetricState update
func (sub *SubscriberStore) Clear() {
	sub.sm.subscribers.Clear()

	// Update MetricState
	sub.sm.updateMetricStateStoreCounts()
}

// GetAll subscribers (unchanged)
func (sub *SubscriberStore) GetAll() []*Subscriber {
	return sub.sm.subscribers.GetAll()
}

// === ENHANCED TIMELINE OPERATIONS ===

type TimelineStore struct {
	sm *StateManager
}

// Add timer with MetricState update
func (tl *TimelineStore) Add(timer *Timer) error {
	if timer.ID == "" {
		return fmt.Errorf("Timer must have an ID")
	}

	tl.sm.timeline.Set(timer.ID, timer)

	// Update MetricState with store counts and active formations
	tl.sm.updateMetricStateStoreCounts()
	activeCount := len(tl.GetActive())
	if tl.sm.metricState != nil {
		tl.sm.metricState.SetActiveFormations(activeCount)
	}

	return nil
}

// Get timer (unchanged)
func (tl *TimelineStore) Get(timerID string) (*Timer, bool) {
	return tl.sm.timeline.Get(timerID)
}

// Forget timer with MetricState update
func (tl *TimelineStore) Forget(timerID string) bool {
	if timer, exists := tl.sm.timeline.Get(timerID); exists {
		timer.Status = "cancelled"
	}

	result := tl.sm.timeline.Forget(timerID)

	// Update MetricState
	tl.sm.updateMetricStateStoreCounts()
	activeCount := len(tl.GetActive())
	if tl.sm.metricState != nil {
		tl.sm.metricState.SetActiveFormations(activeCount)
	}

	return result
}

// Clear all timers with MetricState update
func (tl *TimelineStore) Clear() {
	timers := tl.sm.timeline.GetAll()
	for _, timer := range timers {
		timer.Status = "cancelled"
	}

	tl.sm.timeline.Clear()

	// Update MetricState
	tl.sm.updateMetricStateStoreCounts()
	if tl.sm.metricState != nil {
		tl.sm.metricState.SetActiveFormations(0)
	}
}

// GetAll timers (unchanged)
func (tl *TimelineStore) GetAll() []*Timer {
	return tl.sm.timeline.GetAll()
}

// GetActive timers (unchanged)
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

// === ENHANCED BRANCH OPERATIONS ===

type BranchStoreManager struct {
	sm *StateManager
}

// Set branch with MetricState update
func (bs *BranchStoreManager) Set(branch *BranchStore) error {
	if branch.ID == "" {
		return fmt.Errorf("Branch must have an ID")
	}

	branch.UpdatedAt = time.Now()
	if branch.CreatedAt.IsZero() {
		branch.CreatedAt = time.Now()
	}

	bs.sm.branches.Set(branch.ID, branch)

	// Update MetricState
	bs.sm.updateMetricStateStoreCounts()

	return nil
}

// Get branch (unchanged)
func (bs *BranchStoreManager) Get(branchID string) (*BranchStore, bool) {
	return bs.sm.branches.Get(branchID)
}

// Forget branch with MetricState update
func (bs *BranchStoreManager) Forget(branchID string) bool {
	result := bs.sm.branches.Forget(branchID)

	// Update MetricState
	bs.sm.updateMetricStateStoreCounts()

	return result
}

// Clear all branches with MetricState update
func (bs *BranchStoreManager) Clear() {
	bs.sm.branches.Clear()

	// Update MetricState
	bs.sm.updateMetricStateStoreCounts()
}

// GetAll branches (unchanged)
func (bs *BranchStoreManager) GetAll() []*BranchStore {
	return bs.sm.branches.GetAll()
}

// === METRICSTATE INTEGRATION HELPER ===

// updateMetricStateStoreCounts updates MetricState with current store counts
func (sm *StateManager) updateMetricStateStoreCounts() {
	if sm.metricState == nil {
		return // MetricState not initialized yet
	}

	channels := int(sm.ioStore.Count())
	subscribers := int(sm.subscribers.Count())
	timeline := int(sm.timeline.Count())
	branches := int(sm.branches.Count())
	tasks := timeline // Use timeline count as tasks for now

	// Update MetricState with current counts
	sm.metricState.UpdateStoreCounts(channels, branches, tasks, subscribers, timeline)
}

// === ENHANCED SYSTEM STATE MANAGEMENT ===

// GetSystemState returns state from MetricState if available, fallback to local
func (sm *StateManager) GetSystemState() config.SystemState {
	if sm.metricState != nil {
		// Return intelligent state from MetricState
		return sm.metricState.Get()
	}

	// Fallback to local state
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.systemState
}

// UpdateSystemState delegates to MetricState if available
func (sm *StateManager) UpdateSystemState(updates map[string]interface{}) {
	// Update local state for backward compatibility
	sm.mu.Lock()
	sm.systemState.LastUpdate = time.Now().UnixMilli()

	if stress, ok := updates["stress"].(float64); ok {
		sm.systemState.Stress.Combined = stress
	}
	if activeFormations, ok := updates["activeFormations"].(int); ok {
		sm.systemState.ActiveFormations = activeFormations
		// Also update MetricState
		if sm.metricState != nil {
			sm.metricState.SetActiveFormations(activeFormations)
		}
	}
	sm.mu.Unlock()

	// Delegate intelligence updates to MetricState
	if sm.metricState != nil {
		// MetricState handles intelligent updates
		sm.updateMetricStateStoreCounts()
	}
}

// === SIMPLIFIED METRICS (Delegate to MetricState) ===

// UpdateMetrics updates both local metrics and MetricState
func (sm *StateManager) UpdateMetrics(actionID string, metricType string, duration time.Duration) {
	// Update local metrics for backward compatibility
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

		// Update MetricState with performance data
		if sm.metricState != nil {
			sm.metricState.UpdatePerformance(duration, true)
		}

	case "error":
		atomic.AddInt64(&metrics.ErrorCount, 1)

		// Update MetricState with error
		if sm.metricState != nil {
			sm.metricState.UpdatePerformance(duration, false)
		}

	case "throttled":
		atomic.AddInt64(&metrics.Throttled, 1)
	case "debounced":
		atomic.AddInt64(&metrics.Debounced, 1)
	case "skipped":
		atomic.AddInt64(&metrics.Skipped, 1)
	}

	sm.metrics.Set(actionID, metrics)
}

// === TYPESCRIPT-STYLE API ACCESS (Enhanced) ===

// IO returns the enhanced IO store
func (sm *StateManager) IO() *IOStore {
	return &IOStore{sm: sm}
}

// Subscribers returns the enhanced subscriber store
func (sm *StateManager) Subscribers() *SubscriberStore {
	return &SubscriberStore{sm: sm}
}

// Timeline returns the enhanced timeline store
func (sm *StateManager) Timeline() *TimelineStore {
	return &TimelineStore{sm: sm}
}

// Branches returns the enhanced branch store
func (sm *StateManager) Branches() *BranchStoreManager {
	return &BranchStoreManager{sm: sm}
}

// === ENHANCED STATUS METHODS ===

// IsInitialized checks both StateManager and MetricState
func (sm *StateManager) IsInitialized() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	localInit := sm.initialized && sm.systemState.Initialized

	if sm.metricState != nil {
		metricInit := sm.metricState.Get().Initialized
		return localInit && metricInit
	}

	return localInit
}

// GetStats returns enhanced stats including MetricState intelligence
func (sm *StateManager) GetStats() map[string]interface{} {
	localStats := map[string]interface{}{
		"goroutines":  runtime.NumGoroutine(),
		"uptime_ns":   time.Since(sm.startTime).Nanoseconds(),
		"initialized": sm.initialized,
	}

	if sm.metricState != nil {
		// Get intelligent stats from MetricState
		intelligentState := sm.metricState.Get()

		return map[string]interface{}{
			"system":    intelligentState.Performance,
			"store":     intelligentState.Store,
			"workers":   intelligentState.Workers,
			"health":    intelligentState.Health,
			"breathing": intelligentState.Breathing,
			"local":     localStats,
			"flags": map[string]interface{}{
				"isRecuperating": sm.metricState.IsRecuperating(),
				"isLocked":       sm.metricState.IsLocked(),
				"isHibernating":  sm.metricState.IsHibernating(),
				"isShutdown":     sm.metricState.IsShutdown(),
			},
		}
	}

	// Fallback to local stats only
	return localStats
}

// === REMAINING METHODS (Mostly unchanged but with MetricState awareness) ===

// ForgetAction enhanced with MetricState update
func (sm *StateManager) ForgetAction(actionID string) bool {
	sm.IO().Forget(actionID)
	sm.Subscribers().Forget(actionID)
	sm.ForgetPayload(actionID)
	sm.metrics.Forget(actionID)
	sm.ClearDebounceTimer(actionID)
	sm.throttleMap.Delete(actionID)

	// Store counts automatically updated by individual Forget() calls
	return true
}

// Clear enhanced with MetricState update
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

	// Reset local system state
	sm.systemState = config.DefaultSystemState
	sm.systemState.Initialized = true
	sm.systemState.LastUpdate = time.Now().UnixMilli()

	// Update MetricState
	if sm.metricState != nil {
		sm.updateMetricStateStoreCounts()
	}
}

// Reset delegates to MetricState for intelligence, keeps local for compatibility
func (sm *StateManager) Reset() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Reset local state
	sm.systemState = config.DefaultSystemState
	sm.systemState.Initialized = true
	sm.systemState.LastUpdate = time.Now().UnixMilli()

	// MetricState maintains its own intelligence - don't reset it
}

// === PAYLOAD MANAGEMENT (Unchanged) ===

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

// === PROTECTION STATE (Unchanged) ===

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

func (sm *StateManager) GetMetrics(actionID string) (*ActionMetrics, bool) {
	return sm.metrics.Get(actionID)
}

// === STORES ACCESS (Updated with MetricState) ===

type Stores struct {
	IO          *Store[*types.IO]
	Subscribers *Store[*Subscriber]
	Timeline    *Store[*Timer]
	Branches    *Store[*BranchStore]
	Metrics     *Store[*ActionMetrics]
}

func (sm *StateManager) GetStores() Stores {
	return Stores{
		IO:          sm.ioStore,
		Subscribers: sm.subscribers,
		Timeline:    sm.timeline,
		Branches:    sm.branches,
		Metrics:     sm.metrics,
	}
}
