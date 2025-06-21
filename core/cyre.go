// core/cyre.go
// Core API with only essential functions - clean implementation
// executeAction renamed to handler, removed all helper/utility functions

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

/*
	C.Y.R.E - C.O.R.E (Essential API Only)

	Core functions only:
	- Initialize() - system initialization
	- Action() - register actions
	- On() - subscribe to actions
	- Call() - trigger actions
	- Get() - retrieve payloads
	- Forget() - remove actions
	- Reset() - reset system state
	- Shutdown() - clean shutdown
*/

// === RESPONSE TYPES ===

// CyreResponse matches TypeScript CyreResponse interface exactly
type CyreResponse struct {
	OK        bool        `json:"ok"`
	Payload   interface{} `json:"payload"`
	Message   string      `json:"message"`
	Error     interface{} `json:"error,omitempty"`     // boolean or string
	Timestamp *int64      `json:"timestamp,omitempty"` // optional
	Metadata  *Metadata   `json:"metadata,omitempty"`  // optional
}

// Metadata matches TypeScript metadata structure
type Metadata struct {
	ExecutionTime    *int64        `json:"executionTime,omitempty"`    // milliseconds
	Scheduled        *bool         `json:"scheduled,omitempty"`        // if scheduled execution
	Delayed          *bool         `json:"delayed,omitempty"`          // if delayed
	Duration         *int64        `json:"duration,omitempty"`         // execution duration
	Interval         *int64        `json:"interval,omitempty"`         // interval timing
	Delay            *int64        `json:"delay,omitempty"`            // delay timing
	Repeat           interface{}   `json:"repeat,omitempty"`           // number or boolean
	IntraLink        *IntraLink    `json:"intraLink,omitempty"`        // action chaining
	ChainResult      *CyreResponse `json:"chainResult,omitempty"`      // chained result
	ValidationPassed *bool         `json:"validationPassed,omitempty"` // validation status
	ValidationErrors []string      `json:"validationErrors,omitempty"` // validation errors
	ConditionMet     *bool         `json:"conditionMet,omitempty"`     // condition result
	SelectorApplied  *bool         `json:"selectorApplied,omitempty"`  // selector applied
	TransformApplied *bool         `json:"transformApplied,omitempty"` // transform applied
}

// IntraLink represents action chaining
type IntraLink struct {
	ID      string      `json:"id"`
	Payload interface{} `json:"payload,omitempty"`
}

// Legacy result types for backward compatibility
type CallResult struct {
	OK      bool        `json:"ok"`
	Payload interface{} `json:"payload"`
	Message string      `json:"message"`
	Error   error       `json:"error,omitempty"`
}

type InitResult struct {
	OK      bool   `json:"ok"`
	Payload int64  `json:"payload"`
	Message string `json:"message"`
}

type SubscribeResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// HandlerFunc represents an action handler function
type HandlerFunc func(payload interface{}) interface{}

// === CYRE CORE ===

type Cyre struct {
	stateManager *cyrecontext.StateManager
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

// GlobalCyre is the singleton instance
var GlobalCyre *Cyre
var cyreOnce sync.Once

// === ESSENTIAL API FUNCTIONS ONLY ===

// Initialize creates and initializes the global Cyre instance
func Initialize(configParams ...map[string]interface{}) InitResult {
	startTime := time.Now()

	cyreOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())

		// Initialize dependencies
		stateManager := cyrecontext.Initialize()
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
	})

	return InitResult{
		OK:      true,
		Payload: startTime.UnixNano(),
		Message: config.MSG["WELCOME"],
	}
}

// Action registers an action with the system using schema compilation
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
		// Log compilation errors via sensor
		for _, err := range compileResult.Errors {
			cyrecontext.SensorError(fmt.Sprintf("Action compilation error for %s: %s", config.ID, err)).
				Location("core/cyre.go").
				ActionID(config.ID).
				Log()
		}
		return fmt.Errorf("action compilation failed: %s", strings.Join(compileResult.Errors, "; "))
	}

	// Log compilation warnings if any
	if len(compileResult.Warnings) > 0 {
		for _, warning := range compileResult.Warnings {
			cyrecontext.SensorWarn(fmt.Sprintf("Action compilation warning for %s: %s", config.ID, warning)).
				Location("core/cyre.go").
				ActionID(config.ID).
				Log()
		}
	}

	// Use compiled action with _pipeline included
	compiledAction := compileResult.CompiledAction
	if compiledAction == nil {
		return fmt.Errorf("compilation failed to produce valid action")
	}

	// Set creation timestamp if not set
	if compiledAction.TimeOfCreation == 0 {
		compiledAction.TimeOfCreation = time.Now().UnixMilli()
	}

	// Store compiled action with pipeline in IoStore
	err := c.stateManager.IO().Set(compiledAction)
	if err != nil {
		return fmt.Errorf("failed to store compiled action: %w", err)
	}

	return nil
}

// On subscribes to an action by ID
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

	return SubscribeResult{
		OK:      true,
		Message: config.MSG["SUBSCRIPTION_ESTABLISHED"],
	}
}

// Call triggers an action by ID with payload using clean pipeline execution
func (c *Cyre) Call(actionID string, payload interface{}) <-chan CallResult {
	resultChan := make(chan CallResult, 1)

	if !c.initialized {
		resultChan <- CallResult{
			OK:      false,
			Message: config.MSG["CALL_OFFLINE"],
			Error:   fmt.Errorf("system not initialized"),
		}
		close(resultChan)
		return resultChan
	}

	c.stateManager.UpdateMetrics(actionID, "call", 0)

	// Get compiled IO configuration from IoStore
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

	// Execute pipeline and handle result
	pipelineResult := schema.ExecutePipeline(actionID, payload, ioConfig, c.stateManager)

	// Handle pipeline errors
	if pipelineResult.Error != nil {
		resultChan <- CallResult{
			OK:      false,
			Message: "Pipeline execution failed",
			Error:   pipelineResult.Error,
		}
		close(resultChan)
		return resultChan
	}

	// Handle pipeline blocking
	if !pipelineResult.Allow {
		resultChan <- CallResult{
			OK:      false,
			Message: pipelineResult.Message,
		}
		close(resultChan)
		return resultChan
	}

	// Check if scheduling is needed
	if c.requiresScheduling(ioConfig, pipelineResult) {
		return c.scheduleExecution(actionID, pipelineResult.Payload, ioConfig, pipelineResult)
	}

	// Immediate execution with pipeline-processed payload
	go c.handler(actionID, pipelineResult.Payload, resultChan)
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
	c.timeKeeper.Forget(fmt.Sprintf("%s_debounce", actionID))
	c.timeKeeper.Forget(fmt.Sprintf("%s_interval", actionID))
	c.timeKeeper.Forget(fmt.Sprintf("%s_delay", actionID))

	// Remove from state
	success := c.stateManager.ForgetAction(actionID)

	return success
}

// Reset resets system to default state
func (c *Cyre) Reset() {
	if !c.initialized {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.stateManager.Reset()
}

// Shutdown performs clean system shutdown
func (c *Cyre) Shutdown() {
	if !c.initialized {
		return
	}
	c.mu.Lock()
	c.Reset()

	defer c.mu.Unlock()

	// Stop timekeeper
	c.timeKeeper.Stop()

	// Clear state
	c.stateManager.Clear()

	// Cancel context
	c.cancel()

	// Mark as shutdown
	c.initialized = false
}

// === EXECUTION ENGINE (RENAMED: executeAction → handler) ===

// handler executes an action with the processed payload from pipeline
func (c *Cyre) handler(actionID string, processedPayload interface{}, resultChan chan CallResult) {
	defer close(resultChan)

	// Get worker from pool
	<-c.workerPool
	defer func() {
		c.workerPool <- struct{}{}
	}()

	start := time.Now()

	// Get IO configuration
	ioConfig, exists := c.stateManager.IO().Get(actionID)
	if !exists {
		resultChan <- CallResult{
			OK:      false,
			Message: config.MSG["CALL_INVALID_ID"],
			Error:   fmt.Errorf("action %s not found", actionID),
		}
		return
	}

	// Get handler
	subscriber, exists := c.stateManager.Subscribers().Get(actionID)
	if !exists {
		resultChan <- CallResult{
			OK:      false,
			Message: config.MSG["CALL_NO_SUBSCRIBER"],
			Error:   fmt.Errorf("no handler for action %s", actionID),
		}
		return
	}

	// Update state with processed payload
	c.stateManager.SetPayload(actionID, processedPayload)

	// Execute handler with processed payload and panic recovery
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

		result = subscriber.Handler(processedPayload)
	}()

	duration := time.Since(start)
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
	ioConfig.ExecutionCount++
	ioConfig.LastExecTime = time.Now().UnixMilli()
	ioConfig.ExecutionDuration = duration.Milliseconds()

	// Save updated config back to store
	c.stateManager.IO().Set(ioConfig)

	resultChan <- CallResult{
		OK:      true,
		Payload: result,
		Message: config.MSG["ACKNOWLEDGED_AND_PROCESSED"],
	}
}

// === INTERNAL FUNCTIONS (NOT EXPORTED) ===

// GetCyre returns the global Cyre instance
func GetCyre() *Cyre {
	if GlobalCyre == nil {
		Initialize()
	}
	return GlobalCyre
}

// requiresScheduling determines if action needs TimeKeeper scheduling
func (c *Cyre) requiresScheduling(ioConfig *types.IO, pipelineResult schema.ExecutionResult) bool {
	// Check for delay requirement from pipeline
	if pipelineResult.Delay != nil && *pipelineResult.Delay > 0 {
		return true
	}

	// Check action configuration for scheduling
	return schema.IsScheduledAction(ioConfig)
}

// scheduleExecution handles all scheduling using TimeKeeper
func (c *Cyre) scheduleExecution(actionID string, processedPayload interface{}, ioConfig *types.IO, pipelineResult schema.ExecutionResult) <-chan CallResult {
	resultChan := make(chan CallResult, 1)

	// Handle pipeline-detected delay (from debounce operator)
	if pipelineResult.Delay != nil && *pipelineResult.Delay > 0 {
		err := c.timeKeeper.Wait(*pipelineResult.Delay, func() {
			execChan := make(chan CallResult, 1)
			c.handler(actionID, processedPayload, execChan)
			result := <-execChan
			resultChan <- result
			close(resultChan)
		})

		if err != nil {
			resultChan <- CallResult{
				OK:      false,
				Message: "Failed to schedule delayed execution",
				Error:   err,
			}
			close(resultChan)
		}

		return resultChan
	}

	// Get scheduling configuration from action
	delay, interval, repeat := schema.GetSchedulingConfig(ioConfig)

	// Handle interval actions
	if interval > 0 {
		var repeatValue interface{} = true // Default infinite
		if repeat > 0 {
			repeatValue = repeat
		} else if repeat == 0 {
			repeatValue = 1 // Single execution
		}

		err := c.timeKeeper.Keep(
			interval,
			func() {
				execChan := make(chan CallResult, 1)
				c.handler(actionID, processedPayload, execChan)
				<-execChan // Consume result but don't forward for intervals
			},
			repeatValue,
			fmt.Sprintf("%s_interval", actionID),
			delay, // Optional delay before first execution
		)

		if err != nil {
			resultChan <- CallResult{
				OK:      false,
				Message: "Failed to schedule interval action",
				Error:   err,
			}
		} else {
			resultChan <- CallResult{
				OK:      true,
				Message: "Interval action scheduled successfully",
			}
		}

		close(resultChan)
		return resultChan
	}

	// Handle delayed actions only
	if delay > 0 {
		err := c.timeKeeper.Wait(delay, func() {
			execChan := make(chan CallResult, 1)
			c.handler(actionID, processedPayload, execChan)
			result := <-execChan
			resultChan <- result
			close(resultChan)
		})

		if err != nil {
			resultChan <- CallResult{
				OK:      false,
				Message: "Failed to schedule delayed action",
				Error:   err,
			}
			close(resultChan)
		}

		return resultChan
	}

	// Fallback to immediate execution
	go c.handler(actionID, processedPayload, resultChan)
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
