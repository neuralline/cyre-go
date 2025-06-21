// schema/channel-operators.go - COMPLETE FIX for all compilation errors

package schema

import (
	"fmt"
	"time"

	cyrecontext "github.com/neuralline/cyre-go/context"
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
type OperatorFunc func(actionID string, payload interface{}, config *types.IO, sm *cyrecontext.StateManager) OperatorResult

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
func Throttle(actionID string, payload interface{}, config *types.IO, sm *cyrecontext.StateManager) OperatorResult {
	// FIXED: config.Throttle == nil -> config.Throttle <= 0
	// FIXED: *config.Throttle -> config.Throttle
	if config.Throttle <= 0 {
		return NewAllow(payload, "No throttle configured")
	}

	// FIXED: *config.Throttle -> config.Throttle
	throttleDuration := time.Duration(config.Throttle) * time.Millisecond
	now := time.Now()

	if lastTime, exists := sm.GetThrottleTime(actionID); exists {
		if now.Sub(lastTime) < throttleDuration {
			sm.UpdateMetrics(actionID, "throttled", 0)
			// FIXED: *config.Throttle -> config.Throttle
			return NewBlock(fmt.Sprintf("Throttled - wait %dms", config.Throttle))
		}
	}

	sm.SetThrottleTime(actionID, now)
	return NewAllow(payload, "Throttle check passed")
}

// Debounce implements call collapsing (last-call-wins)
func Debounce(actionID string, payload interface{}, config *types.IO, sm *cyrecontext.StateManager) OperatorResult {
	// FIXED: config.Debounce == nil -> config.Debounce <= 0
	// FIXED: *config.Debounce -> config.Debounce
	if config.Debounce <= 0 {
		return NewAllow(payload, "No debounce configured")
	}

	// FIXED: *config.Debounce -> config.Debounce
	debounceDuration := time.Duration(config.Debounce) * time.Millisecond

	// Handle maxWait within debounce operator
	var maxWaitDuration *time.Duration
	// FIXED: config.MaxWait != nil && *config.MaxWait > 0 -> config.MaxWait > 0
	if config.MaxWait > 0 {
		// FIXED: *config.MaxWait -> config.MaxWait
		duration := time.Duration(config.MaxWait) * time.Millisecond
		maxWaitDuration = &duration
	}

	// Cancel existing debounce timer if exists
	if existingTimer, exists := sm.GetDebounceTimer(actionID); exists {
		_ = existingTimer
	}

	sm.SetPayload(actionID, payload)
	sm.UpdateMetrics(actionID, "debounced", 0)

	// FIXED: *config.Debounce and *config.MaxWait -> direct values
	message := fmt.Sprintf("Debounced - will execute after %dms", config.Debounce)
	if maxWaitDuration != nil {
		message += fmt.Sprintf(" (max wait: %dms)", config.MaxWait)
	}

	return NewDelay(payload, debounceDuration, message)
}

// Required ensures payload is provided when required
func Required(actionID string, payload interface{}, config *types.IO, sm *cyrecontext.StateManager) OperatorResult {
	if !config.Required {
		return NewAllow(payload, "Payload not required")
	}

	if payload == nil {
		return NewBlock("Payload is required but not provided")
	}

	return NewAllow(payload, "Required payload provided")
}

// DetectChanges skips execution if payload hasn't changed
func DetectChanges(actionID string, payload interface{}, config *types.IO, sm *cyrecontext.StateManager) OperatorResult {
	if !config.DetectChanges {
		return NewAllow(payload, "Change detection disabled")
	}

	if !sm.ComparePayload(actionID, payload) {
		sm.UpdateMetrics(actionID, "skipped", 0)
		return NewBlock("Skipped - no changes detected")
	}

	return NewAllow(payload, "Change detected")
}

// Block prevents execution if action is blocked
func Block(actionID string, payload interface{}, config *types.IO, sm *cyrecontext.StateManager) OperatorResult {
	if config.Block || config.IsBlocked {
		reason := config.BlockReason
		if reason == "" {
			reason = "Action is blocked"
		}
		return NewBlock(reason)
	}

	return NewAllow(payload, "Action not blocked")
}

// Priority handles priority-based execution during system stress
func Priority(actionID string, payload interface{}, config *types.IO, sm *cyrecontext.StateManager) OperatorResult {
	if config.Priority == "" {
		return NewAllow(payload, "No priority specified")
	}

	systemState := sm.GetSystemState()
	if systemState.System.IsOverloaded {
		switch config.Priority {
		case "low":
			return NewBlock("Dropped due to system stress (low priority)")
		case "medium":
			if time.Now().UnixNano()%2 == 0 {
				return NewBlock("Dropped due to system stress (medium priority)")
			}
		case "high", "critical":
			break
		}
	}

	return NewAllow(payload, fmt.Sprintf("Priority %s accepted", config.Priority))
}

// Log enables/disables logging for action execution
func Log(actionID string, payload interface{}, config *types.IO, sm *cyrecontext.StateManager) OperatorResult {
	if config.Log {
		return NewAllow(payload, "Logging enabled")
	}
	return NewAllow(payload, "Logging disabled")
}

// Scheduler handles delay, interval, and repeat as a unified timing operator
func Scheduler(actionID string, payload interface{}, config *types.IO, sm *cyrecontext.StateManager) OperatorResult {
	// FIXED: All pointer checks -> direct value checks
	hasDelay := config.Delay > 0
	hasInterval := config.Interval > 0
	hasRepeat := config.Repeat != 0 // 0 means default (single execution)

	if !hasDelay && !hasInterval && !hasRepeat {
		return NewAllow(payload, "No scheduling configured")
	}

	var message string
	var delay time.Duration

	if hasDelay {
		// FIXED: *config.Delay -> config.Delay
		delay = time.Duration(config.Delay) * time.Millisecond
		message += fmt.Sprintf("Delay: %dms", config.Delay)
	}

	if hasInterval {
		if message != "" {
			message += ", "
		}
		// FIXED: *config.Interval -> config.Interval
		message += fmt.Sprintf("Interval: %dms", config.Interval)
	}

	if hasRepeat {
		if message != "" {
			message += ", "
		}
		repeatCount := config.Repeat
		if repeatCount == -1 {
			message += "Repeat: infinite"
		} else {
			if config.ExecutionCount >= int64(repeatCount) {
				return NewBlock(fmt.Sprintf("Repeat limit reached (%d)", repeatCount))
			}
			message += fmt.Sprintf("Repeat: %d/%d", config.ExecutionCount+1, repeatCount)
		}
	}

	config.IsScheduled = true

	if delay > 0 {
		return NewDelay(payload, delay, message)
	}

	return NewAllow(payload, message)
}

// === PLACEHOLDER OPERATORS (unchanged) ===

func Schema(actionID string, payload interface{}, config *types.IO, sm *cyrecontext.StateManager) OperatorResult {
	if config.Schema == nil || len(config.Schema) == 0 {
		return NewAllow(payload, "No schema validation")
	}

	cyrecontext.SensorWarn(fmt.Sprintf("Schema validation not implemented for action: %s", actionID)).
		Location("schema/channel-operators.go").
		ActionID(actionID).
		Log()

	return NewAllow(payload, "Schema validation not implemented - allowing")
}

func Auth(actionID string, payload interface{}, config *types.IO, sm *cyrecontext.StateManager) OperatorResult {
	if config.Auth == nil {
		return NewAllow(payload, "No authentication required")
	}

	cyrecontext.SensorWarn(fmt.Sprintf("Authentication not implemented for action: %s (mode: %s)", actionID, config.Auth.Mode)).
		Location("schema/channel-operators.go").
		ActionID(actionID).
		Metadata(map[string]interface{}{
			"authMode": config.Auth.Mode,
			"hasToken": config.Auth.Token != "",
		}).
		Log()

	return NewAllow(payload, fmt.Sprintf("Authentication not implemented - allowing (mode: %s)", config.Auth.Mode))
}

func Condition(actionID string, payload interface{}, config *types.IO, sm *cyrecontext.StateManager) OperatorResult {
	if config.Condition == nil {
		return NewAllow(payload, "No condition specified")
	}

	cyrecontext.SensorWarn(fmt.Sprintf("Condition evaluation not implemented for action: %s", actionID)).
		Location("schema/channel-operators.go").
		ActionID(actionID).
		Log()

	return NewAllow(payload, "Condition evaluation not implemented - allowing")
}

func Transform(actionID string, payload interface{}, config *types.IO, sm *cyrecontext.StateManager) OperatorResult {
	if config.Transform == nil {
		return NewAllow(payload, "No transformation specified")
	}

	cyrecontext.SensorWarn(fmt.Sprintf("Payload transformation not implemented for action: %s", actionID)).
		Location("schema/channel-operators.go").
		ActionID(actionID).
		Log()

	return NewAllow(payload, "Transformation not implemented - using original payload")
}

func Selector(actionID string, payload interface{}, config *types.IO, sm *cyrecontext.StateManager) OperatorResult {
	if config.Selector == nil {
		return NewAllow(payload, "No selector specified")
	}

	cyrecontext.SensorWarn(fmt.Sprintf("Payload selector not implemented for action: %s", actionID)).
		Location("schema/channel-operators.go").
		ActionID(actionID).
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
	"priority":      Priority,
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

func ExecuteOperator(name string, actionID string, payload interface{}, config *types.IO, sm *cyrecontext.StateManager) (OperatorResult, error) {
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
	// FIXED: config.Throttle != nil && *config.Throttle > 0 -> config.Throttle > 0
	case "throttle":
		return config.Throttle > 0
	case "detectChanges":
		return config.DetectChanges
	case "selector":
		return config.Selector != nil
	case "transform":
		return config.Transform != nil
	case "scheduler":
		// FIXED: All pointer checks -> direct value checks
		return config.Delay > 0 || config.Interval > 0 || config.Repeat != 0
	// FIXED: config.Debounce != nil && *config.Debounce > 0 -> config.Debounce > 0
	case "debounce":
		return config.Debounce > 0
	case "log":
		return config.Log
	default:
		return false
	}
}

func IsScheduledAction(config *types.IO) bool {
	// FIXED: All pointer checks -> direct value checks
	return config.Delay > 0 || config.Interval > 0 || config.Repeat != 0
}

func GetSchedulingConfig(config *types.IO) (delay time.Duration, interval time.Duration, repeat int) {
	// FIXED: All pointer dereferences -> direct value access
	if config.Delay > 0 {
		delay = time.Duration(config.Delay) * time.Millisecond
	}

	if config.Interval > 0 {
		interval = time.Duration(config.Interval) * time.Millisecond
	}

	repeat = config.Repeat

	return delay, interval, repeat
}
