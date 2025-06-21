// cyre.go
// Main package entry point for Cyre Go

package cyre

import (
	"fmt"
	"time"

	"github.com/neuralline/cyre-go/core"
)

/*
	C.Y.R.E - G.O

	Neural Line - Reactive Event Manager for Go
	Version 1.0.0

	High-performance reactive state/event management with:
	- 18,602+ ops/sec performance target
	- O(1) lookups and concurrent safety
	- Protection mechanisms: throttle, debounce, detectChanges
	- Action chaining (IntraLinks)
	- Adaptive timing with breathing system
	- Zero-error reliability
	- Cyre's exact API compatibility

	Basic Usage:
		cyre.Initialize()
		cyre.Action(cyre.ActionConfig{ID: "my-action"})
		cyre.On("my-action", func(payload interface{}) interface{} {
			return "processed"
		})
		result := <-cyre.Call("my-action", "data")
*/

// === EXPORTED TYPES ===

// ActionConfig represents action configuration
type ActionConfig = core.ActionConfig

// HandlerFunc represents an action handler function
type HandlerFunc = core.HandlerFunc

// CallResult represents the result of a call operation
type CallResult = core.CallResult

// InitResult represents initialization result
type InitResult = core.InitResult

// SubscribeResult represents subscription result
type SubscribeResult = core.SubscribeResult

// === MAIN API FUNCTIONS ===

// Initialize creates and initializes the Cyre system
// Returns: InitResult with timestamp and success status
//
// Example:
//
//	result := cyre.Initialize()
//	if result.OK {
//	    fmt.Println("Cyre initialized successfully")
//	}
func Initialize(config ...map[string]interface{}) InitResult {
	return core.Initialize(config...)
}

// Action registers an action with the system
// Parameters:
//   - config: ActionConfig with ID (required) and optional settings
//
// Example:
//
//	err := cyre.Action(cyre.ActionConfig{
//	    ID: "user-login",
//	    Throttle: &[]time.Duration{1 * time.Second}[0],
//	    DetectChanges: true,
//	})
func Action(config ActionConfig) error {
	return core.GetCyre().Action(config)
}

// On subscribes to an action by ID (NOT by type - this is critical!)
// Parameters:
//   - actionID: The exact action ID to subscribe to
//   - handler: Function to execute when action is called
//
// Example:
//
//	cyre.On("user-login", func(payload interface{}) interface{} {
//	    fmt.Printf("User logging in: %v\n", payload)
//	    return map[string]interface{}{"success": true}
//	})
func On(actionID string, handler HandlerFunc) SubscribeResult {
	return core.GetCyre().On(actionID, handler)
}

// Call triggers an action by ID with payload
// Parameters:
//   - actionID: The action ID to trigger
//   - payload: Data to pass to the action handler
//
// Returns: Channel that will receive the CallResult
//
// Example:
//
//	resultChan := cyre.Call("user-login", map[string]interface{}{
//	    "email": "user@example.com",
//	})
//	result := <-resultChan
//	if result.OK {
//	    fmt.Printf("Success: %v\n", result.Payload)
//	}
func Call(actionID string, payload interface{}) <-chan CallResult {
	return core.GetCyre().Call(actionID, payload)
}

// Get retrieves current payload for an action
// Parameters:
//   - actionID: The action ID to get payload for
//
// Returns: (payload, exists)
//
// Example:
//
//	if payload, exists := cyre.Get("user-login"); exists {
//	    fmt.Printf("Current payload: %v\n", payload)
//	}
func Get(actionID string) (interface{}, bool) {
	return core.GetCyre().Get(actionID)
}

// Forget removes an action and all associated state
// Parameters:
//   - actionID: The action ID to remove
//
// Returns: true if action was removed, false if not found
//
// Example:
//
//	if cyre.Forget("user-login") {
//	    fmt.Println("Action removed successfully")
//	}
func Forget(actionID string) bool {
	return core.GetCyre().Forget(actionID)
}

// Clear removes all actions and resets the system
// Use with caution - this clears ALL state
//
// Example:
//
//	cyre.Clear() // Reset entire system
func Clear() {
	core.GetCyre().Clear()
}

// === MONITORING & HEALTH ===

// IsHealthy returns true if the system is operating normally
// Checks sensor health, timekeeper health, and overall system state
//
// Example:
//
//	if cyre.IsHealthy() {
//	    fmt.Println("System is healthy")
//	}
func IsHealthy() bool {
	return core.GetCyre().IsHealthy()
}

// GetMetrics returns performance metrics
// Parameters:
//   - actionID: Optional action ID for specific metrics
//
// Returns: SystemMetrics if no actionID, ChannelMetrics if actionID provided
//
// Example:
//
//	systemMetrics := cyre.GetMetrics()
//	actionMetrics := cyre.GetMetrics("user-login")
func GetMetrics(actionID ...string) interface{} {
	return core.GetCyre().GetMetrics(actionID...)
}

// GetBreathingState returns current adaptive timing state
// Returns: BreathingState with stress level and timing adjustments
//
// Example:
//
//	breathing := cyre.GetBreathingState()
//	fmt.Printf("Stress level: %v\n", breathing)
func GetBreathingState() interface{} {
	return core.GetCyre().GetBreathingState()
}

// GetStats returns comprehensive system statistics
// Returns: map with system state, performance, and health data
//
// Example:
//
//	stats := cyre.GetStats()
//	fmt.Printf("Uptime: %v\n", stats["uptime"])
func GetStats() map[string]interface{} {
	return core.GetCyre().GetStats()
}

// === CONVENIENCE FUNCTIONS ===

// CallSync calls an action synchronously and returns the result
// This is a convenience wrapper around Call() for simpler usage
//
// Example:
//
//	result := cyre.CallSync("user-login", userData)
//	if result.OK {
//	    fmt.Printf("Login successful: %v\n", result.Payload)
//	}
func CallSync(actionID string, payload interface{}) CallResult {
	resultChan := Call(actionID, payload)
	return <-resultChan
}

// CallWithTimeout calls an action with a timeout
// Returns error if action doesn't complete within timeout
//
// Example:
//
//	result, err := cyre.CallWithTimeout("slow-action", data, 5*time.Second)
//	if err != nil {
//	    fmt.Printf("Action timed out: %v\n", err)
//	}
func CallWithTimeout(actionID string, payload interface{}, timeout time.Duration) (CallResult, error) {
	resultChan := Call(actionID, payload)

	select {
	case result := <-resultChan:
		return result, nil
	case <-time.After(timeout):
		return CallResult{
			OK:      false,
			Message: "Action timed out",
		}, fmt.Errorf("action %s timed out after %v", actionID, timeout)
	}
}

// ActionExists checks if an action is registered
// Returns: true if action exists, false otherwise
//
// Example:
//
//	if cyre.ActionExists("user-login") {
//	    fmt.Println("Action is registered")
//	}
func ActionExists(actionID string) bool {
	_, exists := Get(actionID)
	return exists
}

// === PROTECTION HELPERS ===

// ThrottleDuration creates a pointer to time.Duration for throttle config
// Helper function to make throttle configuration easier
//
// Example:
//
//	cyre.Action(cyre.ActionConfig{
//	    ID: "api-call",
//	    Throttle: cyre.ThrottleDuration(1 * time.Second),
//	})
func ThrottleDuration(d time.Duration) *time.Duration {
	return &d
}

// DebounceDuration creates a pointer to time.Duration for debounce config
// Helper function to make debounce configuration easier
//
// Example:
//
//	cyre.Action(cyre.ActionConfig{
//	    ID: "search-input",
//	    Debounce: cyre.DebounceDuration(300 * time.Millisecond),
//	})
func DebounceDuration(d time.Duration) *time.Duration {
	return &d
}

// IntervalDuration creates a pointer to time.Duration for interval config
// Helper function to make interval configuration easier
//
// Example:
//
//	cyre.Action(cyre.ActionConfig{
//	    ID: "health-check",
//	    Interval: cyre.IntervalDuration(30 * time.Second),
//	    Repeat: cyre.RepeatCount(10),
//	})
func IntervalDuration(d time.Duration) *time.Duration {
	return &d
}

// RepeatCount creates a pointer to int for repeat config
// Helper function to make repeat configuration easier
//
// Example:
//
//	cyre.Action(cyre.ActionConfig{
//	    ID: "retry-task",
//	    Repeat: cyre.RepeatCount(3),
//	})
func RepeatCount(count int) *int {
	return &count
}

// RepeatInfinite returns a pointer to -1 for infinite repeats
// Helper function for infinite repeat configuration
//
// Example:
//
//	cyre.Action(cyre.ActionConfig{
//	    ID: "background-task",
//	    Interval: cyre.IntervalDuration(1 * time.Minute),
//	    Repeat: cyre.RepeatInfinite(),
//	})
func RepeatInfinite() *int {
	infinite := -1
	return &infinite
}

// === BATCH OPERATIONS ===

// ActionBatch allows registering multiple actions at once
// Parameters:
//   - configs: Slice of ActionConfig to register
//
// Returns: slice of errors (nil for successful registrations)
//
// Example:
//
//	errors := cyre.ActionBatch([]cyre.ActionConfig{
//	    {ID: "action1"},
//	    {ID: "action2", Throttle: cyre.ThrottleDuration(1*time.Second)},
//	})
func ActionBatch(configs []ActionConfig) []error {
	errors := make([]error, len(configs))
	for i, config := range configs {
		errors[i] = Action(config)
	}
	return errors
}

// OnBatch allows subscribing to multiple actions at once
// Parameters:
//   - subscriptions: Map of actionID -> handler
//
// Returns: map of actionID -> SubscribeResult
//
// Example:
//
//	results := cyre.OnBatch(map[string]cyre.HandlerFunc{
//	    "action1": func(p interface{}) interface{} { return "result1" },
//	    "action2": func(p interface{}) interface{} { return "result2" },
//	})
func OnBatch(subscriptions map[string]HandlerFunc) map[string]SubscribeResult {
	results := make(map[string]SubscribeResult)
	for actionID, handler := range subscriptions {
		results[actionID] = On(actionID, handler)
	}
	return results
}

// ForgetBatch removes multiple actions at once
// Parameters:
//   - actionIDs: Slice of action IDs to remove
//
// Returns: map of actionID -> success status
//
// Example:
//
//	results := cyre.ForgetBatch([]string{"action1", "action2"})
func ForgetBatch(actionIDs []string) map[string]bool {
	results := make(map[string]bool)
	for _, actionID := range actionIDs {
		results[actionID] = Forget(actionID)
	}
	return results
}

// === ADVANCED FEATURES ===

// GetCyre returns the underlying Cyre instance for advanced operations
// Use this to access internal APIs not exposed at package level
//
// Example:
//
//	cyreInstance := cyre.GetCyre()
//	advancedStats := cyreInstance.GetStats()
func GetCyre() *core.Cyre {
	return core.GetCyre()
}

// Version returns the current Cyre Go version
func Version() string {
	return "1.0.0"
}

// Build info
var (
	BuildTime   = "unknown"
	BuildCommit = "unknown"
	GoVersion   = "unknown"
)

// BuildInfo returns build information
func BuildInfo() map[string]string {
	return map[string]string{
		"version":   Version(),
		"buildTime": BuildTime,
		"commit":    BuildCommit,
		"goVersion": GoVersion,
	}
}
