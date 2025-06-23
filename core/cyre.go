// core/cyre.go - FIXED to integrate schema/execute.go pipeline
// Updated Call method to properly execute operator pipeline before handler

package cyre

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/neuralline/cyre-go/config"
	"github.com/neuralline/cyre-go/schema"
	"github.com/neuralline/cyre-go/sensor"
	"github.com/neuralline/cyre-go/state"
	"github.com/neuralline/cyre-go/timekeeper"
	"github.com/neuralline/cyre-go/types"
)

// Enhanced Cyre struct with MetricState integration
type Cyre struct {
	stateManager *state.StateManager
	metricState  *state.MetricState // System brain integration
	timeKeeper   *timekeeper.TimeKeeper

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

func Init(configParams ...map[string]interface{}) InitResult {
	startTime := time.Now()

	cyreOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())

		// Initialize dependencies

		// Check for accurate monitoring mode
		useAccurateMonitoring := false
		if len(configParams) > 0 && configParams[0] != nil {
			if accurate, ok := configParams[0]["accurateMonitoring"].(bool); ok {
				useAccurateMonitoring = accurate
			}
		}

		// Initialize MetricState with accurate monitoring
		var metricState *state.MetricState
		if useAccurateMonitoring {
			metricState = state.InitializeMetricStateAccurate()
		} else {
			metricState = state.InitializeMetricState()
		}

		// Get the global state manager (don't create new one)
		stateManager := state.GetStateManager()

		timeKeeper := timekeeper.Init()

		// Create worker pool - larger if using accurate monitoring
		workerPoolSize := config.WorkerPoolSize
		if useAccurateMonitoring {
			workerPoolSize = config.WorkerPoolSize * 2 // Double for accurate mode
		}

		// Allow override from initialization config
		if len(configParams) > 0 && configParams[0] != nil {
			if size, ok := configParams[0]["workerPoolSize"].(int); ok && size > 0 {
				workerPoolSize = size
			}
		}

		GlobalCyre = &Cyre{
			stateManager:   stateManager,
			metricState:    metricState,
			timeKeeper:     timeKeeper,
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

		// Initialize metric state
		metricState.Init()

		// Initialize breathing system (like TypeScript Cyre.init)

		initializeBreathing()

		if useAccurateMonitoring {
			sensor.Success("Cyre initialized with accurate system monitoring").
				Location("core/cyre.go").
				Metadata(map[string]interface{}{
					"workerPoolSize":     workerPoolSize,
					"accurateMonitoring": true,
					"breathingEnabled":   true,
				}).
				Log()
		}
	})

	return InitResult{
		OK:      true,
		Payload: startTime.UnixNano(),
		Message: config.MSG["WELCOME"],
	}
}

// initializeBreathing initializes the breathing system using TimeKeeper.Keep (matching TypeScript Cyre)
func initializeBreathing() {
	// Direct port from TypeScript cyre.ts initializeBreathing()
	err := GlobalCyre.timeKeeper.Keep(
		500*time.Millisecond, // reputation interval (500ms)
		func() { // callback on each reputation

			// Call MetricState breathing tick
			GlobalCyre.metricState.UpdateBreathingFromMetrics()
		},
		true,                 // repeat infinity (true = -1)
		"system-breathing",   // id for tracking progress and cancellation
		500*time.Millisecond, // start reputation after 2s delay (matching TypeScript)
	)

	if err != nil {
		sensor.Error(fmt.Sprintf("Failed to initialize breathing system: %v", err)).
			Location("core/cyre.go").
			Log()
		return
	}

	// Log success (matching TypeScript sensor.info)
	sensor.Info("Breathing system initialized with metrics integration").
		Location("core/cyre.go").
		EventType("system").
		Metadata(map[string]interface{}{
			"interval": "500ms",
			"delay":    "2000ms",
			"repeat":   "infinity",
			"id":       "system-breathing",
			"timing":   "TimeKeeper.Keep",
		}).
		Force().
		Log()
}

// Call method with enhanced worker scaling based on accurate metrics
func (c *Cyre) Call(actionID string, payload interface{}) <-chan CallResult {
	resultChan := make(chan CallResult, 1)

	// Basic system check
	if !c.initialized {
		resultChan <- CallResult{
			OK:      false,
			Message: "System not initialized",
		}
		close(resultChan)
		return resultChan
	}

	// FIXED: Use direct state.IO() function instead of c.stateManager.IO()
	ioConfig, exists := state.IO().Get(actionID)
	if !exists {
		resultChan <- CallResult{
			OK:      false,
			Message: "Action not found",
		}
		close(resultChan)
		return resultChan
	}

	// Breathing system protection with accurate metrics
	if c.metricState != nil && c.metricState.IsRecuperating() {
		priority := ioConfig.Priority
		if priority == "" {
			priority = "normal"
		}

		if priority != "critical" {
			resultChan <- CallResult{
				OK:      false,
				Message: "System recuperating - only critical actions allowed",
			}
			close(resultChan)
			return resultChan
		}
	}

	// ENHANCED: More aggressive worker management
	maxWorkers := c.workerPoolSize
	if c.metricState != nil {
		maxWorkers = c.metricState.GetWorkerLimit()
	}

	currentWorkers := c.workerPoolSize - len(c.workerPool)

	// ENHANCED: Try multiple strategies for worker allocation
	select {
	case <-c.workerPool:
		// Got worker from pool - execute with pipeline
		go func() {
			defer func() {
				c.workerPool <- struct{}{} // Return worker
				close(resultChan)
			}()
			c.handlerWithPipeline(actionID, payload, ioConfig, resultChan)
		}()

	case <-time.After(5 * time.Millisecond): // Increased timeout from 1ms to 5ms
		// No worker available quickly - check if we can exceed pool
		if currentWorkers < maxWorkers {
			// Execute without pool constraint if under intelligent limit
			go func() {
				defer close(resultChan)
				c.handlerWithPipeline(actionID, payload, ioConfig, resultChan)
			}()
		} else {
			// At intelligent capacity - reject
			resultChan <- CallResult{
				OK:      false,
				Message: fmt.Sprintf("System at capacity (workers: %d/%d)", currentWorkers, maxWorkers),
			}
			close(resultChan)
		}
	}

	return resultChan
}

// GetCyre returns the global Cyre instance
func GetCyre() *Cyre {
	if GlobalCyre == nil {
		Init()
	}
	return GlobalCyre
}

// FIXED: New handler method that integrates pipeline execution
func (c *Cyre) handlerWithPipeline(actionID string, payload interface{}, ioConfig *types.IO, resultChan chan CallResult) {
	start := time.Now()

	// Update call metrics first
	if c.metricState != nil {
		c.metricState.UpdateCallMetrics()
	}

	// CRITICAL FIX: Execute pipeline before handler
	pipelineResult := schema.ExecutePipeline(actionID, payload, ioConfig, c.stateManager)

	// Check if pipeline blocked execution
	if !pipelineResult.Allow {
		duration := time.Since(start)
		if c.metricState != nil {
			c.metricState.UpdatePerformance(duration, false)
		}

		resultChan <- CallResult{
			OK:      false,
			Message: pipelineResult.Message,
			Error:   pipelineResult.Error,
		}
		return
	}

	// Check if pipeline detected delay requirement (debounce)
	if pipelineResult.RequiresDelay && pipelineResult.Delay != nil {
		// Schedule delayed execution with TimeKeeper
		c.timeKeeper.Wait(*pipelineResult.Delay, func() {
			// Execute the actual handler after delay
			c.executeHandler(actionID, pipelineResult.Payload, ioConfig, resultChan, start)
		})
		return
	}

	// No delay required - execute handler immediately with processed payload
	c.executeHandler(actionID, pipelineResult.Payload, ioConfig, resultChan, start)
}

// FIXED: Separated handler execution logic
func (c *Cyre) executeHandler(actionID string, processedPayload interface{}, ioConfig *types.IO, resultChan chan CallResult, start time.Time) {
	// FIXED: Use direct state.Subscribers() function instead of c.stateManager.Subscribers()
	subscriber, exists := state.Subscribers().Get(actionID)
	if !exists {
		duration := time.Since(start)
		if c.metricState != nil {
			c.metricState.UpdatePerformance(duration, false)
		}

		resultChan <- CallResult{
			OK:      false,
			Message: config.MSG["CALL_NO_SUBSCRIBER"],
			Error:   fmt.Errorf("no handler for action %s", actionID),
		}
		return
	}

	// FIXED: Use direct state.SetPayload() function instead of c.stateManager.SetPayload()
	state.SetPayload(actionID, processedPayload)

	// Execute handler with panic recovery
	var result interface{}

	func() {
		defer func() {
			if r := recover(); r != nil {
				sensor.Error(fmt.Sprintf("Handler panic for action %s: %v", actionID, r)).
					Location("core/cyre.go").
					Id(actionID).
					Log()
			}
		}()

		result = subscriber.Handler(processedPayload)
	}()

	duration := time.Since(start)

	// Update performance metrics
	if c.metricState != nil {
		c.metricState.UpdatePerformance(duration, true)
	}

	// Handle action chaining (IntraLinks)
	if chainResult, isChain := c.handleActionChain(result); isChain {
		resultChan <- chainResult
		return
	}

	// Update execution stats in IO config
	if ioConfig != nil {
		ioConfig.ExecutionCount++
		ioConfig.LastExecTime = time.Now().UnixMilli()
		ioConfig.ExecutionDuration = duration.Milliseconds()
		// FIXED: Use direct state.IO().Set() instead of c.stateManager.IO().Set()
		state.IO().Set(ioConfig.ID, ioConfig)
	}

	resultChan <- CallResult{
		OK:      true,
		Payload: result,
		Message: config.MSG["ACKNOWLEDGED_AND_PROCESSED"],
	}
}

// Enhanced Action registration with store count tracking
func (c *Cyre) Action(config types.IO) error {
	if !c.initialized {
		return fmt.Errorf("Cyre not initialized")
	}

	if config.ID == "" {
		return fmt.Errorf("action must have an ID")
	}

	// Compile and validate action using schema system
	compileResult := schema.CompileAction(config)

	// Check for compilation errors
	if len(compileResult.Errors) > 0 {
		for _, err := range compileResult.Errors {
			sensor.Error(fmt.Sprintf("Action compilation error for %s: %s", config.ID, err)).
				Location("core/cyre.go").
				Id(config.ID).
				Log()
		}
		return fmt.Errorf("action compilation failed: %s", strings.Join(compileResult.Errors, "; "))
	}

	// Log compilation warnings
	if len(compileResult.Warnings) > 0 {
		for _, warning := range compileResult.Warnings {
			sensor.Warn(fmt.Sprintf("Action compilation warning for %s: %s", config.ID, warning)).
				Location("core/cyre.go").
				Id(config.ID).
				Log()
		}
	}

	// Use compiled action
	compiledAction := compileResult.CompiledAction
	if compiledAction == nil {
		return fmt.Errorf("compilation failed to produce valid action")
	}

	// Set creation timestamp
	if compiledAction.TimeOfCreation == 0 {
		compiledAction.TimeOfCreation = time.Now().UnixMilli()
	}

	// FIXED: Use direct state.ActionSet() function instead of c.stateManager.IO().Set()
	err := state.ActionSet(compiledAction)
	if err != nil {
		return fmt.Errorf("failed to store compiled action: %w", err)
	}

	return nil
}

// Enhanced On method with store count tracking
func (c *Cyre) On(actionID string, handler HandlerFunc) SubscribeResult {
	if !c.initialized {
		return SubscribeResult{
			OK:      false,
			Message: config.MSG["OFFLINE"],
		}
	}

	if actionID == "" {
		return SubscribeResult{
			OK:      false,
			Message: config.MSG["ACTION_ID_REQUIRED"],
		}
	}

	if handler == nil {
		return SubscribeResult{
			OK:      false,
			Message: config.MSG["SUBSCRIPTION_INVALID_HANDLER"],
		}
	}

	// Create subscriber
	subscriber := &state.Subscriber{
		ID:        actionID,
		ChannelID: actionID,
		Handler:   state.HandlerFunc(handler),
		CreatedAt: time.Now(),
	}

	// FIXED: Use direct state.SubscriberAdd() function instead of c.stateManager.Subscribers().Add()
	err := state.SubscriberAdd(subscriber)
	if err != nil {
		return SubscribeResult{
			OK:      false,
			Message: config.MSG["SUBSCRIPTION_FAILED"],
		}
	}

	return SubscribeResult{
		OK:      true,
		Message: config.MSG["SUBSCRIPTION_ESTABLISHED"],
	}
}

// Get retrieves current payload for an action
func (c *Cyre) Get(actionID string) (interface{}, bool) {
	if !c.initialized {
		return nil, false
	}
	// FIXED: Use direct state.GetPayload() function instead of c.stateManager.GetPayload()
	return state.GetPayload(actionID)
}

// Enhanced Forget method with store count tracking
func (c *Cyre) Forget(actionID string) bool {
	if !c.initialized {
		return false
	}

	// Cancel any active timers
	c.timeKeeper.Forget(fmt.Sprintf("%s_debounce", actionID))
	c.timeKeeper.Forget(fmt.Sprintf("%s_interval", actionID))
	c.timeKeeper.Forget(fmt.Sprintf("%s_delay", actionID))

	// FIXED: Use direct state.ActionForget() function instead of c.stateManager.IO().Forget()
	success := state.ActionForget(actionID)

	return success
}

// Reset resets system state
func (c *Cyre) Reset() {
	if !c.initialized {
		return
	}

	// FIXED: Use direct state.Clear() function
	state.Clear()
}

// Enhanced Shutdown with MetricState integration
func (c *Cyre) Shutdown() {
	if !c.initialized {
		return
	}

	// Signal shutdown to MetricState
	c.metricState.Shutdown()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Stop timekeeper
	c.timeKeeper.Hibernate()

	// Clear state
	state.Clear()

	// Cancel context
	c.cancel()

	// Mark as shutdown
	c.initialized = false
}

// handleActionChain handles action chaining (IntraLinks) - unchanged
func (c *Cyre) handleActionChain(result interface{}) (CallResult, bool) {
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

// GetMetrics returns current system metrics from MetricState
func (c *Cyre) GetMetrics() map[string]interface{} {
	if !c.initialized || c.metricState == nil {
		return map[string]interface{}{"error": "system not initialized"}
	}

	currentState := c.metricState.Get()

	return map[string]interface{}{
		"performance": currentState.Performance,
		"store":       currentState.Store,
		"workers":     currentState.Workers,
		"health":      currentState.Health,
		"breathing":   currentState.Breathing,
		"flags": map[string]interface{}{
			"isRecuperating": c.metricState.IsRecuperating(),
			"isLocked":       c.metricState.IsLocked(),
			"isHibernating":  c.metricState.IsHibernating(),
			"isShutdown":     c.metricState.IsShutdown(),
		},
	}
}
