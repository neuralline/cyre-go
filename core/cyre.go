// core/cyre.go - Enhanced with MetricState integration
// Updated Call() method and handler execution with intelligent system awareness

package core

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/neuralline/cyre-go/config"
	cyrecontext "github.com/neuralline/cyre-go/context"
	"github.com/neuralline/cyre-go/schema"
	"github.com/neuralline/cyre-go/timekeeper"
	"github.com/neuralline/cyre-go/types"
)

// Enhanced Cyre struct with MetricState integration
type Cyre struct {
	stateManager *cyrecontext.StateManager
	metricState  *cyrecontext.MetricState // System brain integration
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

// Enhanced Initialize with MetricState
func Initialize(configParams ...map[string]interface{}) InitResult {
	startTime := time.Now()

	cyreOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())

		// Initialize dependencies
		stateManager := cyrecontext.Initialize()
		metricState := cyrecontext.InitializeMetricState() // Initialize system brain
		timeKeeper := timekeeper.Initialize()

		// Create worker pool
		workerPoolSize := config.WorkerPoolSize

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
	})

	return InitResult{
		OK:      true,
		Payload: startTime.UnixNano(),
		Message: config.MSG["WELCOME"],
	}
}

// GetCyre returns the global Cyre instance
func GetCyre() *Cyre {
	if GlobalCyre == nil {
		Initialize()
	}
	return GlobalCyre
}

// Enhanced Call method with MetricState intelligence
func (c *Cyre) Call(actionID string, payload interface{}) <-chan CallResult {
	resultChan := make(chan CallResult, 1)

	// Update call metrics for rate tracking (like cyre.ts)
	c.metricState.UpdateCallMetrics()

	// System state checks using MetricState brain
	if !c.initialized {
		resultChan <- CallResult{
			OK:      false,
			Message: config.MSG["CALL_OFFLINE"],
			Error:   fmt.Errorf("system not initialized"),
		}
		close(resultChan)
		return resultChan
	}

	// Critical system flag checks (ultra-fast atomic reads)
	if c.metricState.IsShutdown() {
		resultChan <- CallResult{
			OK:      false,
			Message: "System is shutting down",
			Error:   fmt.Errorf("system shutdown in progress"),
		}
		close(resultChan)
		return resultChan
	}

	if c.metricState.IsLocked() {
		resultChan <- CallResult{
			OK:      false,
			Message: "System is locked for maintenance",
			Error:   fmt.Errorf("system maintenance in progress"),
		}
		close(resultChan)
		return resultChan
	}

	if c.metricState.IsHibernating() {
		resultChan <- CallResult{
			OK:      false,
			Message: "System is hibernating",
			Error:   fmt.Errorf("system in hibernation mode"),
		}
		close(resultChan)
		return resultChan
	}

	// Get action configuration
	ioConfig, exists := c.stateManager.IO().Get(actionID)
	if !exists {
		resultChan <- CallResult{
			OK:      false,
			Message: config.MSG["CALL_INVALID_ID"],
			Error:   fmt.Errorf("action %s not registered", actionID),
		}
		close(resultChan)
		return resultChan
	}

	// Breathing system intelligence (matches TypeScript cyre.ts pattern)
	priority := ioConfig.Priority
	if priority == "" {
		priority = "normal" // Default priority
	}

	// Smart blocking based on system stress (like TypeScript recuperation check)
	if c.metricState.IsRecuperating() && priority != "critical" {
		cyrecontext.SensorInfo("Action blocked - system recuperating").
			Location("core/cyre.go").
			ActionID(actionID).
			Metadata(map[string]interface{}{
				"priority":       priority,
				"stressLevel":    c.metricState.Get().Breathing.StressLevel,
				"isRecuperating": true,
			}).
			Log()

		resultChan <- CallResult{
			OK:      false,
			Message: "System is recuperating - only critical actions allowed",
			Error:   fmt.Errorf("system recovering from stress"),
		}
		close(resultChan)
		return resultChan
	}

	// Additional blocking based on priority and stress level
	if c.metricState.ShouldBlockNormal() && priority == "normal" {
		resultChan <- CallResult{
			OK:      false,
			Message: "System under stress - normal actions blocked",
		}
		close(resultChan)
		return resultChan
	}

	if c.metricState.ShouldBlockLow() && (priority == "low" || priority == "background") {
		resultChan <- CallResult{
			OK:      false,
			Message: "System under stress - low priority actions blocked",
		}
		close(resultChan)
		return resultChan
	}

	// Get intelligent worker limit from MetricState
	workerLimit := c.metricState.GetWorkerLimit()

	// Execute with intelligent concurrency control
	if ioConfig.HasFastPath {
		// Fast path execution with worker limit awareness
		go func() {
			defer close(resultChan)
			c.executeWithWorkerLimit(actionID, payload, resultChan, workerLimit)
		}()
	} else {
		// Pipeline execution with full intelligence
		go func() {
			defer close(resultChan)
			c.executeWithPipeline(actionID, payload, ioConfig, resultChan, workerLimit)
		}()
	}

	return resultChan
}

// executeWithWorkerLimit handles fast path execution with worker limit
func (c *Cyre) executeWithWorkerLimit(actionID string, payload interface{}, resultChan chan CallResult, workerLimit int) {
	// Don't defer close here - let caller handle it

	// Try to get worker within limit
	select {
	case <-c.workerPool:
		// Got worker - execute
		defer func() {
			c.workerPool <- struct{}{}
		}()
		c.fastPathHandler(actionID, payload, resultChan)

	case <-time.After(1 * time.Millisecond):
		// Brief wait failed - check if we should execute anyway
		currentWorkers := c.workerPoolSize - len(c.workerPool)
		if currentWorkers < workerLimit {
			// Under limit - execute directly
			c.fastPathHandler(actionID, payload, resultChan)
		} else {
			// At limit - queue for later or reject
			resultChan <- CallResult{
				OK:      false,
				Message: "System at worker capacity",
			}
		}
	}
}

// executeWithPipeline handles full pipeline execution
func (c *Cyre) executeWithPipeline(actionID string, payload interface{}, ioConfig *types.IO, resultChan chan CallResult, workerLimit int) {
	// Don't defer close here - let caller handle it

	// Execute pipeline
	pipelineResult := schema.ExecutePipeline(actionID, payload, ioConfig, c.stateManager)

	// Handle pipeline errors
	if pipelineResult.Error != nil {
		c.metricState.UpdatePerformance(time.Since(time.Now()), false)
		resultChan <- CallResult{
			OK:      false,
			Message: "Pipeline execution failed",
			Error:   pipelineResult.Error,
		}
		return
	}

	// Handle pipeline blocking
	if !pipelineResult.Allow {
		resultChan <- CallResult{
			OK:      false,
			Message: pipelineResult.Message,
		}
		return
	}

	// Execute with worker management (no goroutine here - already in one)
	c.executeWithWorkerLimit(actionID, pipelineResult.Payload, resultChan, workerLimit)
}

// Enhanced fast path handler with performance tracking
func (c *Cyre) fastPathHandler(actionID string, payload interface{}, resultChan chan CallResult) {
	start := time.Now()

	// Get handler
	subscriber, exists := c.stateManager.Subscribers().Get(actionID)
	if !exists {
		duration := time.Since(start)
		c.metricState.UpdatePerformance(duration, false)

		resultChan <- CallResult{
			OK:      false,
			Message: config.MSG["CALL_NO_SUBSCRIBER"],
			Error:   fmt.Errorf("no handler for action %s", actionID),
		}
		return
	}

	// Update state with payload
	c.stateManager.SetPayload(actionID, payload)

	// Execute handler with panic recovery
	var result interface{}
	var execError error

	func() {
		defer func() {
			if r := recover(); r != nil {
				execError = fmt.Errorf("handler panic: %v", r)
				cyrecontext.SensorError(fmt.Sprintf("Handler panic for action %s: %v", actionID, r)).
					Location("core/cyre.go").
					ActionID(actionID).
					Log()
			}
		}()

		result = subscriber.Handler(payload)
	}()

	duration := time.Since(start)
	success := execError == nil

	// Update MetricState with performance data
	c.metricState.UpdatePerformance(duration, success)

	// Update traditional metrics
	c.stateManager.UpdateMetrics(actionID, "execution", duration)

	if execError != nil {
		c.stateManager.UpdateMetrics(actionID, "error", 0)
		resultChan <- CallResult{
			OK:      false,
			Message: config.MSG["ACTION_EXECUTE_FAILED"],
			Error:   execError,
		}
		return
	}

	// Handle action chaining (IntraLinks)
	if chainResult, isChain := c.handleActionChain(result); isChain {
		resultChan <- chainResult
		return
	}

	// Update execution stats in IO config
	ioConfig, _ := c.stateManager.IO().Get(actionID)
	if ioConfig != nil {
		ioConfig.ExecutionCount++
		ioConfig.LastExecTime = time.Now().UnixMilli()
		ioConfig.ExecutionDuration = duration.Milliseconds()
		c.stateManager.IO().Set(ioConfig)
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
			cyrecontext.SensorError(fmt.Sprintf("Action compilation error for %s: %s", config.ID, err)).
				Location("core/cyre.go").
				ActionID(config.ID).
				Log()
		}
		return fmt.Errorf("action compilation failed: %s", strings.Join(compileResult.Errors, "; "))
	}

	// Log compilation warnings
	if len(compileResult.Warnings) > 0 {
		for _, warning := range compileResult.Warnings {
			cyrecontext.SensorWarn(fmt.Sprintf("Action compilation warning for %s: %s", config.ID, warning)).
				Location("core/cyre.go").
				ActionID(config.ID).
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

	// Store compiled action
	err := c.stateManager.IO().Set(compiledAction)
	if err != nil {
		return fmt.Errorf("failed to store compiled action: %w", err)
	}

	// Update store counts in MetricState
	c.updateStoreCounts()

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
	subscriber := &cyrecontext.Subscriber{
		ID:        actionID,
		ActionID:  actionID,
		Handler:   cyrecontext.HandlerFunc(handler),
		CreatedAt: time.Now(),
	}

	// Store handler
	err := c.stateManager.Subscribers().Add(subscriber)
	if err != nil {
		return SubscribeResult{
			OK:      false,
			Message: config.MSG["SUBSCRIPTION_FAILED"],
		}
	}

	// Update store counts in MetricState
	c.updateStoreCounts()

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
	return c.stateManager.GetPayload(actionID)
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

	// Remove from state
	success := c.stateManager.ForgetAction(actionID)

	// Update store counts in MetricState
	c.updateStoreCounts()

	return success
}

// Reset resets system state
func (c *Cyre) Reset() {
	if !c.initialized {
		return
	}
	c.stateManager.Reset()
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
	c.stateManager.Clear()

	// Cancel context
	c.cancel()

	// Mark as shutdown
	c.initialized = false
}

// updateStoreCounts updates MetricState with current store counts
func (c *Cyre) updateStoreCounts() {
	stores := c.stateManager.GetStores()

	channels := int(stores.IO.Count())
	subscribers := int(stores.Subscribers.Count())
	timeline := int(stores.Timeline.Count())
	branches := int(stores.Branches.Count())
	tasks := timeline // Use timeline as tasks for now

	c.metricState.UpdateStoreCounts(channels, branches, tasks, subscribers, timeline)
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

// GetSystemState returns current system state from MetricState
func (c *Cyre) GetSystemState() map[string]interface{} {
	if !c.initialized || c.metricState == nil {
		return map[string]interface{}{"error": "system not initialized"}
	}

	return map[string]interface{}{
		"state": c.metricState.Get(),
	}
}

// ActionExists checks if an action is registered
func (c *Cyre) ActionExists(actionID string) bool {
	if !c.initialized {
		return false
	}
	_, exists := c.stateManager.IO().Get(actionID)
	return exists
}
