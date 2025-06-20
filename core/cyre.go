// core/cyre.go
// Main Cyre API implementation with protection mechanisms

package core

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/neuralline/cyre-go/sensor"
	"github.com/neuralline/cyre-go/state"
	"github.com/neuralline/cyre-go/timekeeper"
)

/*
	C.Y.R.E - C.O.R.E

	Main API implementation providing:
	- Cyre's exact API: initialize, action, on, call, get, forget
	- O(1) performance with concurrent safety
	- Protection mechanisms: throttle, debounce, detectChanges
	- Action chaining (IntraLinks)
	- Async execution with channels
	- Zero-error reliability
	- 18,602+ ops/sec target performance
*/

// === CORE TYPES ===

// ActionConfig represents action configuration
type ActionConfig struct {
	ID            string         `json:"id"`
	Type          string         `json:"type,omitempty"`
	Payload       interface{}    `json:"payload,omitempty"`
	Interval      *time.Duration `json:"interval,omitempty"`
	Repeat        *int           `json:"repeat,omitempty"`
	Throttle      *time.Duration `json:"throttle,omitempty"`
	Debounce      *time.Duration `json:"debounce,omitempty"`
	DetectChanges bool           `json:"detectChanges,omitempty"`
	Log           bool           `json:"log,omitempty"`
	Priority      string         `json:"priority,omitempty"`
}

// HandlerFunc represents an action handler function
type HandlerFunc func(payload interface{}) interface{}

// CallResult represents the result of a call operation
type CallResult struct {
	OK      bool        `json:"ok"`
	Payload interface{} `json:"payload"`
	Message string      `json:"message"`
	Error   error       `json:"error,omitempty"`
}

// InitResult represents initialization result
type InitResult struct {
	OK      bool   `json:"ok"`
	Payload int64  `json:"payload"` // timestamp
	Message string `json:"message"`
}

// SubscribeResult represents subscription result
type SubscribeResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// === PROTECTION SYSTEM ===

// ProtectionSystem handles throttle, debounce, and change detection
type ProtectionSystem struct {
	stateManager *state.StateManager
	sensor       *sensor.Sensor
	timeKeeper   *timekeeper.TimeKeeper
	mu           sync.RWMutex
}

// NewProtectionSystem creates a new protection system
func NewProtectionSystem(sm *state.StateManager, s *sensor.Sensor, tk *timekeeper.TimeKeeper) *ProtectionSystem {
	return &ProtectionSystem{
		stateManager: sm,
		sensor:       s,
		timeKeeper:   tk,
	}
}

// CheckProtections validates all protection mechanisms
func (ps *ProtectionSystem) CheckProtections(actionID string, payload interface{}, config *state.ActionConfig) (bool, string) {
	// 1. Throttle check (rate limiting)
	if config.Throttle != nil && *config.Throttle > 0 {
		if !ps.checkThrottle(actionID, *config.Throttle) {
			ps.sensor.RecordProtection(actionID, "throttle")
			return false, "throttled"
		}
	}

	// 2. Change detection (skip identical payloads)
	if config.DetectChanges {
		if !ps.stateManager.ComparePayload(actionID, payload) {
			ps.sensor.RecordProtection(actionID, "skip")
			return false, "no_change"
		}
	}

	// 3. Debounce check happens at timer level
	// (debounce creates/cancels timers rather than blocking)

	return true, "allowed"
}

// checkThrottle implements throttle protection
func (ps *ProtectionSystem) checkThrottle(actionID string, throttleDuration time.Duration) bool {
	now := time.Now()

	if lastTime, exists := ps.stateManager.GetThrottleTime(actionID); exists {
		if now.Sub(lastTime) < throttleDuration {
			return false // Still throttled
		}
	}

	// Update throttle time
	ps.stateManager.SetThrottleTime(actionID, now)
	return true
}

// === CYRE CORE ===

// Cyre is the main system instance
type Cyre struct {
	stateManager *state.StateManager
	sensor       *sensor.Sensor
	timeKeeper   *timekeeper.TimeKeeper
	protection   *ProtectionSystem

	// Worker pool for concurrent execution
	workerPool     chan struct{}
	workerPoolSize int

	// System state
	initialized bool
	startTime   time.Time
	mu          sync.RWMutex

	// Context for shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// GlobalCyre is the singleton instance
var GlobalCyre *Cyre
var cyreOnce sync.Once

// Initialize creates and initializes the global Cyre instance
func Initialize(config ...map[string]interface{}) InitResult {
	startTime := time.Now()

	cyreOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())

		// Initialize dependencies
		stateManager := state.Initialize()
		sensor := sensor.Initialize()
		timeKeeper := timekeeper.Initialize()

		// Create protection system
		protection := NewProtectionSystem(stateManager, sensor, timeKeeper)

		// Create worker pool
		workerPoolSize := config.WorkerPoolSize
		if workerPoolSize <= 0 {
			workerPoolSize = config.WorkerPoolSize
		}

		GlobalCyre = &Cyre{
			stateManager:   stateManager,
			sensor:         sensor,
			timeKeeper:     timeKeeper,
			protection:     protection,
			workerPool:     make(chan struct{}, workerPoolSize),
			workerPoolSize: workerPoolSize,
			initialized:    true,
			startTime:      startTime,
			ctx:            ctx,
			cancel:         cancel,
		}

		// Fill worker pool
		for i := 0; i < workerPoolSize; i++ {
			GlobalCyre.workerPool <- struct{}{}
		}
	})

	return InitResult{
		OK:      true,
		Payload: startTime.UnixNano(),
		Message: "CYRE initialized successfully",
	}
}

// GetCyre returns the global Cyre instance
func GetCyre() *Cyre {
	if GlobalCyre == nil {
		Initialize()
	}
	return GlobalCyre
}

// === MAIN API METHODS (Cyre's exact API) ===

// Action registers an action with the system
func (c *Cyre) Action(config ActionConfig) error {
	if !c.initialized {
		return fmt.Errorf("Cyre not initialized")
	}

	if config.ID == "" {
		return fmt.Errorf("action must have an ID")
	}

	// Convert to internal format
	stateConfig := &state.ActionConfig{
		ID:            config.ID,
		Type:          config.Type,
		Throttle:      config.Throttle,
		Debounce:      config.Debounce,
		DetectChanges: config.DetectChanges,
		Log:           config.Log,
		Priority:      config.Priority,
		Timestamp:     time.Now().UnixNano(),
	}

	// Handle timing parameters
	if config.Interval != nil {
		stateConfig.Interval = config.Interval
	}
	if config.Repeat != nil {
		stateConfig.Repeat = config.Repeat
	}

	// Store action configuration
	err := c.stateManager.SetAction(stateConfig)
	if err != nil {
		return err
	}

	// Store initial payload if provided
	if config.Payload != nil {
		c.stateManager.SetPayload(config.ID, config.Payload)
	}

	return nil
}

// On subscribes to an action by ID
func (c *Cyre) On(actionID string, handler HandlerFunc) SubscribeResult {
	if !c.initialized {
		return SubscribeResult{
			OK:      false,
			Message: "Cyre not initialized",
		}
	}

	if actionID == "" {
		return SubscribeResult{
			OK:      false,
			Message: "actionID cannot be empty",
		}
	}

	if handler == nil {
		return SubscribeResult{
			OK:      false,
			Message: "handler cannot be nil",
		}
	}

	// Store handler
	err := c.stateManager.SetHandler(actionID, handler)
	if err != nil {
		return SubscribeResult{
			OK:      false,
			Message: err.Error(),
		}
	}

	return SubscribeResult{
		OK:      true,
		Message: fmt.Sprintf("Subscribed to action: %s", actionID),
	}
}

// Call triggers an action by ID with payload
func (c *Cyre) Call(actionID string, payload interface{}) <-chan CallResult {
	resultChan := make(chan CallResult, 1)

	if !c.initialized {
		resultChan <- CallResult{
			OK:      false,
			Message: "Cyre not initialized",
			Error:   fmt.Errorf("system not initialized"),
		}
		close(resultChan)
		return resultChan
	}

	// Record the call
	c.sensor.RecordCall(actionID)

	// Get action configuration
	actionConfig, exists := c.stateManager.GetAction(actionID)
	if !exists {
		resultChan <- CallResult{
			OK:      false,
			Message: fmt.Sprintf("Action not found: %s", actionID),
			Error:   fmt.Errorf("action %s not registered", actionID),
		}
		close(resultChan)
		return resultChan
	}

	// Handle debounced actions
	if actionConfig.Debounce != nil && *actionConfig.Debounce > 0 {
		return c.handleDebouncedCall(actionID, payload, *actionConfig.Debounce)
	}

	// Handle interval actions
	if actionConfig.Interval != nil && *actionConfig.Interval > 0 {
		return c.handleIntervalCall(actionID, payload, actionConfig)
	}

	// Handle immediate execution
	go c.executeAction(actionID, payload, resultChan)

	return resultChan
}

// Get retrieves current payload for an action
func (c *Cyre) Get(actionID string) (interface{}, bool) {
	if !c.initialized {
		return nil, false
	}

	return c.stateManager.GetPayload(actionID)
}

// Forget removes an action and all associated state
func (c *Cyre) Forget(actionID string) bool {
	if !c.initialized {
		return false
	}

	// Cancel any active timers
	c.timeKeeper.CancelActionTimers(actionID)

	// Remove from state
	return c.stateManager.ForgetAction(actionID)
}

// Clear removes all actions and resets the system
func (c *Cyre) Clear() {
	if !c.initialized {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Stop all timers
	c.timeKeeper.Stop()

	// Clear all state
	c.stateManager.Clear()

	// Reinitialize timekeeper
	c.timeKeeper = timekeeper.Initialize()
}

// === EXECUTION ENGINE ===

// executeAction executes an action with protection mechanisms
func (c *Cyre) executeAction(actionID string, payload interface{}, resultChan chan CallResult) {
	defer close(resultChan)

	// Get worker from pool
	<-c.workerPool
	defer func() {
		c.workerPool <- struct{}{}
	}()

	start := time.Now()

	// Get action configuration
	actionConfig, exists := c.stateManager.GetAction(actionID)
	if !exists {
		resultChan <- CallResult{
			OK:      false,
			Message: "Action not found",
			Error:   fmt.Errorf("action %s not found", actionID),
		}
		return
	}

	// Check protection mechanisms
	allowed, reason := c.protection.CheckProtections(actionID, payload, actionConfig)
	if !allowed {
		resultChan <- CallResult{
			OK:      false,
			Message: fmt.Sprintf("Action blocked: %s", reason),
		}
		return
	}

	// Get handler
	handler, exists := c.stateManager.GetHandler(actionID)
	if !exists {
		resultChan <- CallResult{
			OK:      false,
			Message: "No handler registered",
			Error:   fmt.Errorf("no handler for action %s", actionID),
		}
		return
	}

	// Update payload
	c.stateManager.SetPayload(actionID, payload)

	// Execute handler with panic recovery
	var result interface{}
	var execError error

	func() {
		defer func() {
			if r := recover(); r != nil {
				execError = fmt.Errorf("handler panic: %v", r)
			}
		}()

		result = handler(payload)
	}()

	duration := time.Since(start)
	success := execError == nil

	// Record execution
	c.sensor.RecordExecution(actionID, duration, success, execError)

	if execError != nil {
		resultChan <- CallResult{
			OK:      false,
			Message: "Handler execution failed",
			Error:   execError,
		}
		return
	}

	// Handle action chaining (IntraLinks)
	if chainResult, isChain := c.handleActionChain(result); isChain {
		resultChan <- chainResult
		return
	}

	// Return successful result
	resultChan <- CallResult{
		OK:      true,
		Payload: result,
		Message: "Action executed successfully",
	}
}

// handleDebouncedCall handles debounced action execution
func (c *Cyre) handleDebouncedCall(actionID string, payload interface{}, debounceDuration time.Duration) <-chan CallResult {
	resultChan := make(chan CallResult, 1)

	// Create debounced timer
	_, err := c.timeKeeper.ScheduleDebounce(actionID, debounceDuration,
		func(id string, pl interface{}) {
			execChan := make(chan CallResult, 1)
			c.executeAction(id, pl, execChan)

			// Forward result
			result := <-execChan
			resultChan <- result
			close(resultChan)
		}, payload)

	if err != nil {
		resultChan <- CallResult{
			OK:      false,
			Message: "Failed to schedule debounced action",
			Error:   err,
		}
		close(resultChan)
	}

	return resultChan
}

// handleIntervalCall handles interval-based action execution
func (c *Cyre) handleIntervalCall(actionID string, payload interface{}, config *state.ActionConfig) <-chan CallResult {
	resultChan := make(chan CallResult, 1)

	repeat := -1 // Infinite by default
	if config.Repeat != nil {
		repeat = *config.Repeat
	}

	// Create interval timer
	_, err := c.timeKeeper.ScheduleInterval(actionID, *config.Interval, repeat,
		func(id string, pl interface{}) {
			execChan := make(chan CallResult, 1)
			c.executeAction(id, pl, execChan)
			<-execChan // Consume result but don't forward for intervals
		}, payload)

	if err != nil {
		resultChan <- CallResult{
			OK:      false,
			Message: "Failed to schedule interval action",
			Error:   err,
		}
	} else {
		resultChan <- CallResult{
			OK:      true,
			Message: "Interval action scheduled",
		}
	}

	close(resultChan)
	return resultChan
}

// handleActionChain handles action chaining (IntraLinks)
func (c *Cyre) handleActionChain(result interface{}) (CallResult, bool) {
	// Check if result is a chain link
	if chainMap, ok := result.(map[string]interface{}); ok {
		if nextID, hasID := chainMap["id"].(string); hasID {
			nextPayload := chainMap["payload"]

			// Execute chained action
			chainChan := c.Call(nextID, nextPayload)
			chainResult := <-chainChan

			return CallResult{
				OK:      chainResult.OK,
				Payload: chainResult.Payload,
				Message: fmt.Sprintf("Chained to action: %s", nextID),
				Error:   chainResult.Error,
			}, true
		}
	}

	return CallResult{}, false
}

// === SYSTEM STATUS & MONITORING ===

// IsHealthy returns true if system is healthy
func (c *Cyre) IsHealthy() bool {
	if !c.initialized {
		return false
	}

	return c.sensor.IsHealthy() && c.timeKeeper.IsHealthy()
}

// GetMetrics returns system metrics
func (c *Cyre) GetMetrics(actionID ...string) interface{} {
	if !c.initialized {
		return nil
	}

	if len(actionID) > 0 && actionID[0] != "" {
		// Return specific action metrics
		if metrics, exists := c.sensor.GetChannelMetrics(actionID[0]); exists {
			return metrics
		}
		return nil
	}

	// Return system metrics
	return c.sensor.GetSystemMetrics()
}

// GetBreathingState returns current breathing state
func (c *Cyre) GetBreathingState() interface{} {
	if !c.initialized {
		return nil
	}

	return c.timeKeeper.GetBreathingState()
}

// GetStats returns comprehensive system statistics
func (c *Cyre) GetStats() map[string]interface{} {
	if !c.initialized {
		return map[string]interface{}{
			"initialized": false,
		}
	}

	stateStats := c.stateManager.GetStats()
	sensorStats := c.sensor.GetSystemMetrics()
	timerStats := c.timeKeeper.GetStats()

	return map[string]interface{}{
		"initialized": c.initialized,
		"uptime":      time.Since(c.startTime),
		"workerPool":  c.workerPoolSize,
		"state":       stateStats,
		"sensor":      sensorStats,
		"timekeeper":  timerStats,
		"healthy":     c.IsHealthy(),
	}
}

// === PACKAGE-LEVEL API (Cyre's global functions) ===

// Action registers an action (global function)
func Action(config ActionConfig) error {
	return GetCyre().Action(config)
}

// On subscribes to an action (global function)
func On(actionID string, handler HandlerFunc) SubscribeResult {
	return GetCyre().On(actionID, handler)
}

// Call triggers an action (global function)
func Call(actionID string, payload interface{}) <-chan CallResult {
	return GetCyre().Call(actionID, payload)
}

// Get retrieves payload (global function)
func Get(actionID string) (interface{}, bool) {
	return GetCyre().Get(actionID)
}

// Forget removes an action (global function)
func Forget(actionID string) bool {
	return GetCyre().Forget(actionID)
}

// Clear resets the system (global function)
func Clear() {
	GetCyre().Clear()
}

// IsHealthy checks system health (global function)
func IsHealthy() bool {
	return GetCyre().IsHealthy()
}

// GetMetrics returns metrics (global function)
func GetMetrics(actionID ...string) interface{} {
	return GetCyre().GetMetrics(actionID...)
}
