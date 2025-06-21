// types/io.go
// IO interface - Direct port from TypeScript Cyre
// Enhanced action configuration with channel organization fields

package types

import (
	"time"
)

/*
	IO Interface - Complete port from TypeScript Cyre

	Represents channel configuration matching the TypeScript IO interface exactly.
	This is the data structure that users configure when creating channels.
*/

// === FUNCTION TYPES ===

// ConditionFunction evaluates whether action should execute
type ConditionFunction func(payload interface{}, context map[string]interface{}) bool

// SelectorFunction selects specific part of payload to watch for changes
type SelectorFunction func(payload interface{}) interface{}

// TransformFunction transforms payload before execution
type TransformFunction func(payload interface{}) interface{}

// === CONFIGURATION SUB-TYPES ===

// PriorityConfig defines execution priority during system stress
type PriorityConfig struct {
	Level        string `json:"level"`        // "low", "medium", "high", "critical"
	Weight       int    `json:"weight"`       // Numeric weight for ordering
	DropOnStress bool   `json:"dropOnStress"` // Drop if system stressed
}

// AuthConfig defines authentication and authorization
type AuthConfig struct {
	Mode           string   `json:"mode"`           // "token", "context", "group", "disabled"
	Token          string   `json:"token"`          // Authentication token
	AllowedCallers []string `json:"allowedCallers"` // Allowed caller IDs
	GroupPolicy    string   `json:"groupPolicy"`    // Group-based access policy
	SessionTimeout int      `json:"sessionTimeout"` // Session timeout in milliseconds
}

// Pipeline represents a compiled channel operator
type Pipeline struct {
	Type    string                 `json:"type"`    // "throttle", "debounce", "transform", etc.
	Config  map[string]interface{} `json:"config"`  // Operator-specific configuration
	Enabled bool                   `json:"enabled"` // Whether operator is active
	Order   int                    `json:"order"`   // Execution order in pipeline
}

// === MAIN IO INTERFACE ===

// IO represents complete channel configuration matching TypeScript IO interface
type IO struct {
	// === CORE IDENTIFICATION ===
	ID          string      `json:"id"`          // Unique identifier for this action (required)
	Name        string      `json:"name"`        // Human-readable name for the channel
	Type        string      `json:"type"`        // Channel category/classification (defaults to id)
	Path        string      `json:"path"`        // Hierarchical path for organization (e.g., "sensors/temperature/room1")
	Group       string      `json:"group"`       // Group membership for bulk operations
	Tags        []string    `json:"tags"`        // Tags for filtering and organization
	Description string      `json:"description"` // Description of what this channel does
	Version     string      `json:"version"`     // Version for channel evolution tracking
	Payload     interface{} `json:"payload"`     // Add this line

	// === CHANNEL OPERATORS (Action Talents) ===
	Required      bool   `json:"required"`      // Require payload to be provided - boolean for basic requirement
	Interval      int    `json:"interval"`      // Milliseconds between executions for repeated actions
	Repeat        int    `json:"repeat"`        // Number of times to repeat execution, -1 for infinite repeats
	Delay         int    `json:"delay"`         // Milliseconds to delay before first execution
	Throttle      int    `json:"throttle"`      // Minimum milliseconds between executions (rate limiting)
	Debounce      int    `json:"debounce"`      // Collapse rapid calls within this window (milliseconds)
	MaxWait       int    `json:"maxWait"`       // Maximum wait time for debounce
	DetectChanges bool   `json:"detectChanges"` // Only execute if payload has changed from previous execution
	Log           bool   `json:"log"`           // Enable logging for this action
	Priority      string `json:"priority"`      // Priority level for execution during system stress
	Block         bool   `json:"block"`         // Block this action from execution

	// === ADVANCED CONFIGURATION ===
	Schema map[string]interface{} `json:"schema"` // Schema validation for payload. only execute when these conditions are error free
	Auth   *AuthConfig            `json:"auth"`   // Authentication: experimental

	// === STATE REACTIVITY OPTIONS ===
	Condition ConditionFunction `json:"-"` // Only execute when this condition returns true
	Selector  SelectorFunction  `json:"-"` // Select specific part of payload to watch for changes
	Transform TransformFunction `json:"-"` // Transform payload before execution

	// === INTERNAL OPTIMIZATION FIELDS ===
	IsBlocked     bool       `json:"_isBlocked"`     // Pre-computed blocking state for instant use
	BlockReason   string     `json:"_blockReason"`   // Reason for blocking if _isBlocked is true
	HasFastPath   bool       `json:"_hasFastPath"`   // True if action has no protections and can use fast path
	Pipeline      []Pipeline `json:"_pipeline"`      // Save Pre-compiled channel operators list here
	DebounceTimer string     `json:"_debounceTimer"` // Active debounce timer ID
	IsScheduled   bool       `json:"_isScheduled"`   // Flag to indicate if this channel has schedule elements
	BranchID      string     `json:"_branchId"`      // Branch ID if this channel belongs to a branch

	// === COMPILED PIPELINES (Internal) ===
	HasProtections     bool     `json:"_hasProtections"`     // Has any protection mechanisms
	HasProcessing      bool     `json:"_hasProcessing"`      // Has processing elements
	HasScheduling      bool     `json:"_hasScheduling"`      // Has scheduling elements
	ProcessingTalents  []string `json:"_processingTalents"`  // List of processing talents
	HasChangeDetection bool     `json:"_hasChangeDetection"` // Has change detection enabled

	// === SYSTEM FIELDS ===
	Timestamp         int64 `json:"_timestamp"`         // Latest call to channel timestamp
	TimeOfCreation    int64 `json:"_timeOfCreation"`    // Time of channel creation timestamp
	LastExecTime      int64 `json:"_lastExecTime"`      // Last successful execution time timestamp
	ExecutionDuration int64 `json:"_executionDuration"` // How long last execution took (milliseconds)
	ExecutionCount    int64 `json:"_executionCount"`    // How many times it has been successfully executed
	ErrorCount        int64 `json:"_errorCount"`        // Error execution count

	// === EXTENSIBILITY ===
	Custom map[string]interface{} `json:"custom"` // Allow indexing with string keys for additional properties
}

// === CONSTRUCTION ===

// NewIO creates a new IO configuration with required ID and defaults
func NewIO(id string) *IO {
	now := time.Now().UnixMilli()

	return &IO{
		ID:             id,
		Type:           id, // Default type to ID if not specified
		TimeOfCreation: now,
		Timestamp:      now,
		Custom:         make(map[string]interface{}),

		// Initialize optimization flags (will be computed during compilation)
		IsBlocked:          false,
		HasFastPath:        true, // Assume fast path until protections added
		HasProtections:     false,
		HasProcessing:      false,
		HasScheduling:      false,
		HasChangeDetection: false,
		ProcessingTalents:  []string{},
		Pipeline:           []Pipeline{},
	}
}
