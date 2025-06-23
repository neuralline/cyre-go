// /cyre.go
// Main package entry point for Cyre Go - Essential API Only
// Removed all helper/utility functions, keeping only core API

package cyre

import (
	"github.com/neuralline/cyre-go/core"
	"github.com/neuralline/cyre-go/types"
)

/*
	C.Y.R.E - G.O (Essential API)

	Neural Line - Reactive Event Manager for Go
	Version 1.0.0

	Essential API functions only:
	- Initialize() - system initialization
	- Action() - register actions
	- On() - subscribe to actions
	- Call() - trigger actions
	- Get() - retrieve payloads
	- Forget() - remove actions
	- Reset() - reset system state
	- Shutdown() - clean shutdown
*/

// === EXPORTED TYPES ===

// ActionConfig represents action configuration (backward compatibility alias)
type ActionConfig = types.IO

// IO represents action configuration (new preferred name)
type IO = types.IO

// HandlerFunc represents an action handler function
type HandlerFunc = cyre.HandlerFunc

// CallResult represents the result of a call operation
type CallResult = cyre.CallResult

// InitResult represents initialization result
type InitResult = cyre.InitResult

// SubscribeResult represents subscription result
type SubscribeResult = cyre.SubscribeResult

// CyreResponse represents TypeScript-compatible response
type CyreResponse = cyre.CyreResponse

// Metadata represents response metadata
type Metadata = cyre.Metadata

// === ESSENTIAL API FUNCTIONS ONLY ===

// Init creates and initializes the Cyre system
// Returns: InitResult with timestamp and success status
//
// Example:
//
//	result := cyre.Init()
//	if result.OK {
//	    fmt.Println("Cyre initialized successfully")
//	}
func Init(config ...map[string]interface{}) InitResult {
	return cyre.Init(config...)
}

// Action registers an action with the system
// Parameters:
//   - config: ActionConfig/IO with ID (required) and optional settings
//
// Example:
//
//	err := cyre.Action(cyre.IO{
//	    ID: "user-login",
//	    Throttle: 1000, // milliseconds
//	    DetectChanges: true,
//	})
func Action(config ActionConfig) error {
	return cyre.GetCyre().Action(types.IO(config))
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
	return cyre.GetCyre().On(actionID, handler)
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
	return cyre.GetCyre().Call(actionID, payload)
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
	return cyre.GetCyre().Get(actionID)
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
	return cyre.GetCyre().Forget(actionID)
}

// Reset resets system to default state
// Use with caution - this resets system state but keeps actions
//
// Example:
//
//	cyre.Reset() // Reset system state
func Reset() {
	cyre.GetCyre().Reset()
}

// Shutdown performs clean system shutdown
// Use for graceful application shutdown
//
// Example:
//
//	cyre.Shutdown() // Clean shutdown
func Shutdown() {
	cyre.GetCyre().Shutdown()
}
