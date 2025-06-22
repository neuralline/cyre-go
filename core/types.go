// core/types.go
// Core type definitions for Cyre Go
// Essential types used throughout the core system

package core

import (
	"fmt"
	"sync"
	"time"
)

/*
	Core Types for Cyre Go

	Defines essential types used by the core system:
	- Result types for API responses
	- Handler function types
	- Response metadata structures
	- Global variables and synchronization
*/

// === GLOBAL VARIABLES ===

var (
	GlobalCyre *Cyre     // Global Cyre instance
	cyreOnce   sync.Once // Ensure single initialization
)

// === HANDLER TYPES ===

// HandlerFunc represents a function that handles action execution
type HandlerFunc func(payload interface{}) interface{}

// === RESULT TYPES ===

// InitResult represents the result of system initialization
type InitResult struct {
	OK      bool        `json:"ok"`
	Payload interface{} `json:"payload,omitempty"`
	Message string      `json:"message,omitempty"`
	Error   error       `json:"error,omitempty"`
}

// CallResult represents the result of an action call
type CallResult struct {
	OK      bool        `json:"ok"`
	Payload interface{} `json:"payload,omitempty"`
	Message string      `json:"message,omitempty"`
	Error   error       `json:"error,omitempty"`

	// Performance metadata
	Duration   time.Duration `json:"duration,omitempty"`
	ExecutedAt time.Time     `json:"executedAt,omitempty"`
	ActionID   string        `json:"actionId,omitempty"`

	// Execution context
	HandlerUsed bool   `json:"handlerUsed,omitempty"`
	WasChained  bool   `json:"wasChained,omitempty"`
	ChainedTo   string `json:"chainedTo,omitempty"`
}

// SubscribeResult represents the result of a subscription operation
type SubscribeResult struct {
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	Error   error  `json:"error,omitempty"`

	// Subscription metadata
	ActionID    string    `json:"actionId,omitempty"`
	HandlerType string    `json:"handlerType,omitempty"`
	CreatedAt   time.Time `json:"createdAt,omitempty"`
}

// === TYPESCRIPT-COMPATIBLE RESPONSE TYPES ===

// CyreResponse represents a TypeScript-compatible response
type CyreResponse struct {
	OK       bool                   `json:"ok"`
	Data     interface{}            `json:"data,omitempty"`
	Error    string                 `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Timing information
	Timestamp time.Time     `json:"timestamp"`
	Duration  time.Duration `json:"duration,omitempty"`
}

// Metadata represents response metadata
type Metadata struct {
	ActionID    string                 `json:"actionId,omitempty"`
	HandlerType string                 `json:"handlerType,omitempty"`
	ExecutionID string                 `json:"executionId,omitempty"`
	Performance map[string]interface{} `json:"performance,omitempty"`
	System      map[string]interface{} `json:"system,omitempty"`
	Custom      map[string]interface{} `json:"custom,omitempty"`
}

// === HELPER FUNCTIONS ===

// NewCallResult creates a successful call result
func NewCallResult(payload interface{}, message string) CallResult {
	return CallResult{
		OK:         true,
		Payload:    payload,
		Message:    message,
		ExecutedAt: time.Now(),
	}
}

// NewCallError creates an error call result
func NewCallError(message string, err error) CallResult {
	return CallResult{
		OK:         false,
		Message:    message,
		Error:      err,
		ExecutedAt: time.Now(),
	}
}

// NewInitResult creates a successful init result
func NewInitResult(payload interface{}, message string) InitResult {
	return InitResult{
		OK:      true,
		Payload: payload,
		Message: message,
	}
}

// NewInitError creates an error init result
func NewInitError(message string, err error) InitResult {
	return InitResult{
		OK:      false,
		Message: message,
		Error:   err,
	}
}

// NewSubscribeResult creates a successful subscribe result
func NewSubscribeResult(actionID, message string) SubscribeResult {
	return SubscribeResult{
		OK:        true,
		Message:   message,
		ActionID:  actionID,
		CreatedAt: time.Now(),
	}
}

// NewSubscribeError creates an error subscribe result
func NewSubscribeError(message string, err error) SubscribeResult {
	return SubscribeResult{
		OK:      false,
		Message: message,
		Error:   err,
	}
}

// === CONVENIENCE TYPES ===

// ActionConfig is an alias for types.IO for backward compatibility
type ActionConfig interface{}

// ExecutionContext provides context for action execution
type ExecutionContext struct {
	ActionID   string                 `json:"actionId"`
	StartTime  time.Time              `json:"startTime"`
	Timeout    time.Duration          `json:"timeout"`
	Priority   string                 `json:"priority"`
	Metadata   map[string]interface{} `json:"metadata"`
	CancelFunc func()                 `json:"-"`
}

// SystemStatus represents current system status
type SystemStatus struct {
	Online       bool                   `json:"online"`
	Initialized  bool                   `json:"initialized"`
	Locked       bool                   `json:"locked"`
	Hibernating  bool                   `json:"hibernating"`
	Recuperating bool                   `json:"recuperating"`
	Stats        map[string]interface{} `json:"stats"`
	Health       map[string]interface{} `json:"health"`
}

// === CONFIGURATION TYPES ===

// CyreConfig represents system configuration
type CyreConfig struct {
	WorkerPoolSize  int           `json:"workerPoolSize"`
	DefaultTimeout  time.Duration `json:"defaultTimeout"`
	EnableMetrics   bool          `json:"enableMetrics"`
	EnableBreathing bool          `json:"enableBreathing"`
	EnableProfiling bool          `json:"enableProfiling"`
	MaxConcurrency  int           `json:"maxConcurrency"`
	PerformanceMode string        `json:"performanceMode"`
}

// DefaultCyreConfig provides sensible defaults
var DefaultCyreConfig = CyreConfig{
	WorkerPoolSize:  100,
	DefaultTimeout:  30 * time.Second,
	EnableMetrics:   true,
	EnableBreathing: true,
	EnableProfiling: false,
	MaxConcurrency:  10000,
	PerformanceMode: "balanced",
}

// === ERROR TYPES ===

// CyreError represents a Cyre-specific error
type CyreError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Context string `json:"context,omitempty"`
	Cause   error  `json:"-"`
}

func (e *CyreError) Error() string {
	if e.Context != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Context)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *CyreError) Unwrap() error {
	return e.Cause
}

// Common error codes
const (
	ErrCodeNotInitialized    = "NOT_INITIALIZED"
	ErrCodeInvalidAction     = "INVALID_ACTION"
	ErrCodeNoHandler         = "NO_HANDLER"
	ErrCodeTimeout           = "TIMEOUT"
	ErrCodeSystemLocked      = "SYSTEM_LOCKED"
	ErrCodeSystemHibernating = "SYSTEM_HIBERNATING"
	ErrCodeOverloaded        = "SYSTEM_OVERLOADED"
)

// NewCyreError creates a new Cyre error
func NewCyreError(code, message string, cause error) *CyreError {
	return &CyreError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}
