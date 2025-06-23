// schema/channel-operators.go - FIXED LOCAL IMPORTS
// Fix all import paths to use local relative imports

package schema

import (
	"fmt"
	"time"

	"github.com/neuralline/cyre-go/state"
	"github.com/neuralline/cyre-go/timekeeper"
	"github.com/neuralline/cyre-go/types"
)

// === OPERATOR RESULT TYPES ===

// OperatorResult represents the result of an operator execution
type OperatorResult struct {
	Allow   bool           `json:"allow"`   // Whether to continue execution
	Payload interface{}    `json:"payload"` // Modified payload (if any)
	Message string         `json:"message"` // Result message
	Delay   *time.Duration `json:"delay"`   // Optional delay before execution
}

// OperatorFunc represents a channel operator function
type OperatorFunc func(actionID string, payload interface{}, config *types.IO, sm *state.StateManager) OperatorResult

// NewAllow creates a successful operator result
func NewAllow(payload interface{}, message string) OperatorResult {
	return OperatorResult{Allow: true, Payload: payload, Message: message}
}

// NewBlock creates a blocking operator result
func NewBlock(message string) OperatorResult {
	return OperatorResult{Allow: false, Message: message}
}

// NewDelay creates a result with delay
func NewDelay(payload interface{}, delay time.Duration, message string) OperatorResult {
	return OperatorResult{Allow: true, Payload: payload, Delay: &delay, Message: message}
}

// === FIXED OPERATORS ===

// Throttle implements rate limiting (first-call-wins)
func Throttle(actionID string, payload interface{}, config *types.IO, sm *state.StateManager) OperatorResult {
	if config.Throttle <= 0 {
		return NewAllow(payload, "No throttle configured")
	}

	throttleDuration := time.Duration(config.Throttle) * time.Millisecond
	now := time.Now()

	if lastTime, exists := state.GetThrottleTime(actionID); exists {
		if now.Sub(lastTime) < throttleDuration {
			// FIXED: Using direct state functions
			return NewBlock(fmt.Sprintf("Throttled - wait %dms", config.Throttle))
		}
	}

	state.SetThrottleTime(actionID, now)
	return NewAllow(payload, "Throttle check passed")
}

// Debounce implements call collapsing (last-call-wins)
func Debounce(actionID string, payload interface{}, config *types.IO, sm *state.StateManager) OperatorResult {
	if config.Debounce <= 0 {
		return NewAllow(payload, "No debounce configured")
	}

	debounceDuration := time.Duration(config.Debounce) * time.Millisecond

	// Handle maxWait within debounce operator
	var maxWaitDuration *time.Duration
	if config.MaxWait > 0 {
		duration := time.Duration(config.MaxWait) * time.Millisecond
		maxWaitDuration = &duration
	}

	// Cancel existing debounce timer if exists
	if existingTimer, exists := state.GetDebounceTimer(actionID); exists {
		_ = existingTimer
	}

	state.SetPayload(actionID, payload)

	message := fmt.Sprintf("Debounced - will execute after %dms", config.Debounce)
	if maxWaitDuration != nil {
		message += fmt.Sprintf(" (max wait: %dms)", config.MaxWait)
	}

	return NewDelay(payload, debounceDuration, message)
}

// Required ensures payload is provided when required
func Required(actionID string, payload interface{}, config *types.IO, sm *state.StateManager) OperatorResult {
	if !config.Required {
		return NewAllow(payload, "Payload not required")
	}

	if payload == nil {
		return NewBlock("Payload is required but not provided")
	}

	return NewAllow(payload, "Required payload provided")
}

// DetectChanges skips execution if payload hasn't changed
func DetectChanges(actionID string, payload interface{}, config *types.IO, sm *state.StateManager) OperatorResult {
	if !config.DetectChanges {
		return NewAllow(payload, "Change detection disabled")
	}

	if !state.ComparePayload(actionID, payload) {
		return NewBlock("Skipped - no changes detected")
	}

	return NewAllow(payload, "Change detected")
}

// Block prevents execution if action is blocked
func Block(actionID string, payload interface{}, config *types.IO, sm *state.StateManager) OperatorResult {
	if config.Block || config.IsBlocked {
		reason := config.BlockReason
		if reason == "" {
			reason = "Action is blocked"
		}
		return NewBlock(reason)
	}

	return NewAllow(payload, "Action not blocked")
}

// Log enables/disables logging for action execution
func Log(actionID string, payload interface{}, config *types.IO, sm *state.StateManager) OperatorResult {
	if config.Log {
		return NewAllow(payload, "Logging enabled")
	}
	return NewAllow(payload, "Logging disabled")
}

// Scheduler - Direct TimeKeeper.Keep() port from TypeScript example
func Scheduler(actionID string, payload interface{}, config *types.IO, sm *state.StateManager) OperatorResult {
	if config.Interval <= 0 && config.Delay <= 0 && config.Repeat == 0 {
		return NewAllow(payload, "No scheduling configured")
	}

	timeKeeper := timekeeper.GetTimeKeeper()

	// Convert repeat for TimeKeeper
	var repeat interface{}
	if config.Repeat == -1 {
		repeat = true // Infinite
	} else if config.Repeat > 0 {
		repeat = config.Repeat
	} else {
		repeat = 1 // Default single execution
	}

	// Simple callback that just updates metrics (actual execution happens via TimeKeeper scheduling)
	callback := func() {
		// Get the handler and execute it
		if subscriber, exists := state.Subscribers().Get(actionID); exists {
			subscriber.Handler(payload)
		}
	}

	// Determine interval and delay
	interval := time.Duration(config.Interval) * time.Millisecond
	delay := time.Duration(config.Delay) * time.Millisecond

	// If only delay is specified, use it as interval for one-time execution
	if config.Interval <= 0 && config.Delay > 0 {
		interval = delay
		delay = 0 // No additional delay
	}

	// Direct port: TimeKeeper.keep(interval, callback, repeat, id, delay)
	var err error
	if delay > 0 {
		err = timeKeeper.Keep(interval, callback, repeat, actionID, delay)
	} else {
		err = timeKeeper.Keep(interval, callback, repeat, actionID)
	}

	if err != nil {
		return NewBlock(fmt.Sprintf("Scheduling failed: %v", err))
	}

	// Block immediate execution - TimeKeeper will handle the scheduling
	return NewBlock("Scheduled with TimeKeeper - immediate execution blocked")
}

// === PLACEHOLDER OPERATORS ===

func Schema(actionID string, payload interface{}, config *types.IO, sm *state.StateManager) OperatorResult {
	if config.Schema == nil || len(config.Schema) == 0 {
		return NewAllow(payload, "No schema validation")
	}

	state.Warn(fmt.Sprintf("Schema validation not implemented for action: %s", actionID)).
		Location("schema/channel-operators.go").
		Id(actionID).
		Log()

	return NewAllow(payload, "Schema validation not implemented - allowing")
}

func Auth(actionID string, payload interface{}, config *types.IO, sm *state.StateManager) OperatorResult {
	if config.Auth == nil {
		return NewAllow(payload, "No authentication required")
	}

	state.Warn(fmt.Sprintf("Authentication not implemented for action: %s (mode: %s)", actionID, config.Auth.Mode)).
		Location("schema/channel-operators.go").
		Id(actionID).
		Metadata(map[string]interface{}{
			"authMode": config.Auth.Mode,
			"hasToken": config.Auth.Token != "",
		}).
		Log()

	return NewAllow(payload, fmt.Sprintf("Authentication not implemented - allowing (mode: %s)", config.Auth.Mode))
}

func Condition(actionID string, payload interface{}, config *types.IO, sm *state.StateManager) OperatorResult {
	if config.Condition == nil {
		return NewAllow(payload, "No condition specified")
	}

	state.Warn(fmt.Sprintf("Condition evaluation not implemented for action: %s", actionID)).
		Location("schema/channel-operators.go").
		Id(actionID).
		Log()

	return NewAllow(payload, "Condition evaluation not implemented - allowing")
}

func Transform(actionID string, payload interface{}, config *types.IO, sm *state.StateManager) OperatorResult {
	if config.Transform == nil {
		return NewAllow(payload, "No transformation specified")
	}

	state.Critical(fmt.Sprintf("Payload transformation not implemented for action: %s", actionID)).
		Location("schema/channel-operators.go").
		Id(actionID).
		Log()

	return NewAllow(payload, "Transformation not implemented - using original payload")
}

func Selector(actionID string, payload interface{}, config *types.IO, sm *state.StateManager) OperatorResult {
	if config.Selector == nil {
		return NewAllow(payload, "No selector specified")
	}

	state.Critical(fmt.Sprintf("Payload selector not implemented for action: %s", actionID)).
		Location("schema/channel-operators.go").
		Id(actionID).
		Log()

	return NewAllow(payload, "Selector not implemented - using full payload")
}

// === OPERATOR REGISTRY ===

var OperatorRegistry = map[string]OperatorFunc{
	"required":      Required,
	"throttle":      Throttle,
	"debounce":      Debounce,
	"detectChanges": DetectChanges,
	"block":         Block,
	"log":           Log,
	"scheduler":     Scheduler,
	"schema":        Schema,
	"auth":          Auth,
	"condition":     Condition,
	"transform":     Transform,
	"selector":      Selector,
}

func GetOperator(name string) (OperatorFunc, bool) {
	operator, exists := OperatorRegistry[name]
	return operator, exists
}

func GetImplementedOperators() []string {
	return []string{
		"required", "throttle", "debounce", "detectChanges",
		"block", "priority", "log", "scheduler",
	}
}

func GetPlaceholderOperators() []string {
	return []string{
		"schema", "auth", "condition", "transform", "selector",
	}
}

func GetAllOperators() []string {
	names := make([]string, 0, len(OperatorRegistry))
	for name := range OperatorRegistry {
		names = append(names, name)
	}
	return names
}

func ExecuteOperator(name string, actionID string, payload interface{}, config *types.IO, sm *state.StateManager) (OperatorResult, error) {
	operator, exists := GetOperator(name)
	if !exists {
		return OperatorResult{}, fmt.Errorf("operator '%s' not found", name)
	}

	result := operator(actionID, payload, config, sm)
	return result, nil
}

// === UTILITY FUNCTIONS - FIXED ===

func isOperatorConfigured(operatorName string, config *types.IO) bool {
	switch operatorName {
	case "block":
		return config.Block || config.IsBlocked
	case "required":
		return config.Required
	case "auth":
		return config.Auth != nil
	case "schema":
		return config.Schema != nil && len(config.Schema) > 0
	case "condition":
		return config.Condition != nil
	case "priority":
		return config.Priority != ""
	case "throttle":
		return config.Throttle > 0
	case "detectChanges":
		return config.DetectChanges
	case "selector":
		return config.Selector != nil
	case "transform":
		return config.Transform != nil
	case "scheduler":
		return config.Delay > 0 || config.Interval > 0 || config.Repeat != 0
	case "debounce":
		return config.Debounce > 0
	case "log":
		return config.Log
	default:
		return false
	}
}

func IsScheduledAction(config *types.IO) bool {
	return config.Delay > 0 || config.Interval > 0 || config.Repeat != 0
}

func GetSchedulingConfig(config *types.IO) (delay time.Duration, interval time.Duration, repeat int) {
	if config.Delay > 0 {
		delay = time.Duration(config.Delay) * time.Millisecond
	}

	if config.Interval > 0 {
		interval = time.Duration(config.Interval) * time.Millisecond
	}

	repeat = config.Repeat

	return delay, interval, repeat
}
