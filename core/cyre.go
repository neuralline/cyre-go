// core/cyre.go - Enhanced with MetricState integration
// Updated Call() method and handler execution with intelligent system awareness

package cyre

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

func Init(configParams ...map[string]interface{}) InitResult {
	startTime := time.Now()

	cyreOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())

		// Initialize dependencies
		stateManager := cyrecontext.Init()

		// Check for accurate monitoring mode
		useAccurateMonitoring := false
		if len(configParams) > 0 && configParams[0] != nil {
			if accurate, ok := configParams[0]["accurateMonitoring"].(bool); ok {
				useAccurateMonitoring = accurate
			}
		}

		// Initialize MetricState with accurate monitoring
		var metricState *cyrecontext.MetricState
		if useAccurateMonitoring {
			metricState = cyrecontext.InitializeMetricStateAccurate()
		} else {
			metricState = cyrecontext.InitializeMetricState()
		}

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

		if useAccurateMonitoring {
			cyrecontext.SensorSuccess("Cyre initialized with accurate system monitoring").
				Location("core/cyre.go").
				Metadata(map[string]interface{}{
					"workerPoolSize":     workerPoolSize,
					"accurateMonitoring": true,
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

// GetCyre returns the global Cyre instance
func GetCyre() *Cyre {
	if GlobalCyre == nil {
		Init()
	}
	return GlobalCyre
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

	// Get action configuration
	ioConfig, exists := c.stateManager.IO().Get(actionID)
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

	// Debug worker state
	// if currentWorkers == 0 {
	// 	fmt.Printf("DEBUG: Worker pool exhausted - Current: %d, Max: %d, Pool size: %d\n",
	// 		currentWorkers, maxWorkers, c.workerPoolSize)
	// }

	// ENHANCED: Try multiple strategies for worker allocation
	select {
	case <-c.workerPool:
		// Got worker from pool - execute
		go func() {
			defer func() {
				c.workerPool <- struct{}{} // Return worker
				close(resultChan)
			}()
			c.handler(actionID, payload, ioConfig, resultChan)
		}()

	case <-time.After(5 * time.Millisecond): // Increased timeout from 1ms to 5ms
		// No worker available quickly - check if we can exceed pool
		if currentWorkers < maxWorkers {
			// Execute without pool constraint if under intelligent limit
			go func() {
				defer close(resultChan)
				c.handler(actionID, payload, ioConfig, resultChan)
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

// handler with enhanced performance tracking
func (c *Cyre) handler(actionID string, payload interface{}, ioConfig *types.IO, resultChan chan CallResult) {
	start := time.Now()

	// Get handler
	subscriber, exists := c.stateManager.Subscribers().Get(actionID)
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

	// Update state with payload
	c.stateManager.SetPayload(actionID, payload)

	// Execute handler with panic recovery
	var result interface{}

	func() {
		defer func() {
			if r := recover(); r != nil {
				cyrecontext.SensorError(fmt.Sprintf("Handler panic for action %s: %v", actionID, r)).
					Location("core/cyre.go").
					ActionID(actionID).
					Log()
			}
		}()

		result = subscriber.Handler(payload)
	}()

	duration := time.Since(start)

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
