// config/cyre-config.go
// Enhanced configuration matching TypeScript Cyre with timing, messages, and default state

package config

import "time"

/*
	C.Y.R.E - C.O.N.F.I.G

	Enhanced configuration system providing:
	- Performance tuning constants
	- Timing configurations (UI, animation, API)
	- System messages (British AI Assistant style)
	- Breathing system parameters
	- Default system state initialization
	- Complete TypeScript compatibility
*/

const (
	// === PERFORMANCE CONSTANTS ===

	// Core performance settings
	DefaultChannelBuffer = 1000  // Buffer size for action channels
	MaxConcurrentActions = 10000 // Maximum concurrent action executions
	WorkerPoolSize       = 100   // Default worker pool size
	MaxRetainedActions   = 50000 // Maximum actions in registry

	// Memory management
	MaxRetainedPayloads = 10000           // Maximum payloads to retain
	CleanupInterval     = 5 * time.Minute // State cleanup frequency
	GCTriggerThreshold  = 0.85            // Memory usage to trigger cleanup

	// === TIMING CONSTANTS ===

	// Protection timing defaults
	DefaultThrottleDuration = 100 * time.Millisecond // Default throttle window
	DefaultDebounceDuration = 50 * time.Millisecond  // Default debounce delay
	MinTimerResolution      = 1 * time.Millisecond   // Minimum timer precision
	MaxTimerDuration        = 24 * time.Hour         // Maximum timer duration

	// Execution timing
	DefaultCallTimeout     = 30 * time.Second     // Default call timeout
	HandlerTimeout         = 10 * time.Second     // Handler execution timeout
	ProtectionCheckTimeout = 1 * time.Millisecond // Protection check timeout

	// Interval and repeat settings
	MinInterval       = 1 * time.Millisecond // Minimum interval duration
	MaxInterval       = 1 * time.Hour        // Maximum interval duration
	DefaultMaxRepeats = 1000000              // Default max repeat count

	// === NEW: UI/ANIMATION TIMING (from TypeScript) ===

	// UI and animation timing
	AnimationTiming      = 16*time.Millisecond + 670*time.Microsecond // 16.67ms - 60fps
	UIUpdateTiming       = 100 * time.Millisecond                     // User interface updates
	InputDebounceTiming  = 150 * time.Millisecond                     // Input handling (typing, scrolling)
	APIPollingTiming     = 1000 * time.Millisecond                    // API polling/data refresh
	BackgroundTaskTiming = 5000 * time.Millisecond                    // Background operations

	// System timing constants
	RecuperationInterval = 60 * time.Second      // 1 minute - Recuperation check interval
	LongInterval         = 60 * 60 * time.Second // 1 hour - Long timer handling
	MaxTimeout           = 2147483647            // Max safe timeout value (2^31 - 1)

	// === BREATHING SYSTEM ===

	// Stress monitoring
	StressThreshold   = 0.80 // Stress level to trigger breathing
	CriticalThreshold = 0.95 // Critical stress level
	RecoveryThreshold = 0.30 // Stress level for recovery
	HealthyThreshold  = 0.10 // Healthy operation threshold

	// Breathing rates (matching TypeScript structure)
	BreathingMinRate      = 50 * time.Millisecond   // Minimum breathing rate
	BreathingBaseRate     = 200 * time.Millisecond  // Base breathing rate
	BreathingMaxRate      = 1000 * time.Millisecond // Maximum breathing rate
	BreathingRecoveryRate = 2000 * time.Millisecond // Recovery breathing rate

	// Breathing stress levels (matching TypeScript)
	BreathingStressLow      = 0.5  // Low stress threshold
	BreathingStressMedium   = 0.75 // Medium stress threshold
	BreathingStressHigh     = 0.9  // High stress threshold
	BreathingStressCritical = 0.95 // Critical stress threshold

	// === PROTECTION SYSTEM (Enhanced from TypeScript) ===

	// Protection thresholds
	CallThreshold    = 100  // Call threshold
	MinDebounce      = 50   // Minimum debounce (ms)
	MinThrottle      = 50   // Minimum throttle (ms)
	MaxDelay         = 2000 // Maximum delay (ms)
	ProtectionWindow = 1000 // Protection window (ms)
	InitialDelay     = 25   // Initial delay (ms)
	SystemLoadDelay  = 250  // System load delay (ms)

	// System protection limits
	CPUWarning        = 85   // CPU warning threshold (%)
	CPUCritical       = 95   // CPU critical threshold (%)
	MemoryWarning     = 85   // Memory warning threshold (%)
	MemoryCritical    = 95   // Memory critical threshold (%)
	EventLoopWarning  = 200  // Event loop warning (ms)
	EventLoopCritical = 1000 // Event loop critical (ms)
	OverloadThreshold = 4    // System overload threshold

	// === SYSTEM DEFAULTS ===

	// Action defaults
	DefaultActionType = "default"
	DefaultLogLevel   = "info"
	DefaultPriority   = "medium"

	// System behavior
	EnableAutoCleanup = true // Enable automatic cleanup
	EnableMetrics     = true // Enable metrics collection
	EnableLogging     = true // Enable logging
	EnableBreathing   = true // Enable breathing system

	// Development settings
	DevMode         = false // Development mode flag
	VerboseLogging  = false // Verbose logging in dev mode
	EnableProfiling = false // Enable performance profiling
)

// === PAYLOAD CONFIGURATION ===

var PAYLOAD_CONFIG = struct {
	MaxHistoryPerChannel int
}{
	MaxHistoryPerChannel: 10,
}

// === TIMING CONFIGURATION (TypeScript Style) ===

var TIMING = struct {
	Animation      time.Duration // 60fps - For smooth animations
	UIUpdate       time.Duration // User interface updates
	InputDebounce  time.Duration // Input handling (typing, scrolling)
	APIPolling     time.Duration // API polling/data refresh
	BackgroundTask time.Duration // Background operations
	Recuperation   time.Duration // Recuperation check interval
	LongInterval   time.Duration // Long timer handling
	MaxTimeout     int64         // Max safe timeout value
}{
	Animation:      AnimationTiming,
	UIUpdate:       UIUpdateTiming,
	InputDebounce:  InputDebounceTiming,
	APIPolling:     APIPollingTiming,
	BackgroundTask: BackgroundTaskTiming,
	Recuperation:   RecuperationInterval,
	LongInterval:   LongInterval,
	MaxTimeout:     MaxTimeout,
}

// === SYSTEM MESSAGES (British AI Assistant Style) ===

var MSG = map[string]string{
	// System Status - British AI Assistant Style
	"OFFLINE":                   "Cyre offline - systems temporarily unavailable",
	"ONLINE":                    "Cyre online! at your service",
	"WELCOME":                   "Cyre ready! how may I assist you today?",
	"SYSTEM_LOCKED":             "System temporarily locked - please wait a moment while I reorganize",
	"SYSTEM_LOCKED_CHANNELS":    "Unable to create new channels at the moment - system is reorganizing",
	"SYSTEM_LOCKED_SUBSCRIBERS": "Unable to add new subscriptions currently - please try again shortly",

	// Performance Messages - Polite but Informative
	"SLOW_LISTENER_DETECTED":       "Performance notice - a task is taking longer than expected",
	"SLOW_ACTION_PIPELINE":         "Processing notice - workflow is running slower than usual",
	"HIGH_PIPELINE_OVERHEAD":       "Efficiency notice - system overhead detected, optimizing...",
	"INEFFICIENT_PIPELINE_RATIO":   "Performance advisory - task coordination could be improved",
	"PERFORMANCE_DEGRADATION":      "System notice - performance adjustment in progress",
	"AUTO_OPTIMIZATION_SUGGESTION": "Recommendation - this process could benefit from optimization",

	// Action Related - Professional & Clear
	"ACTION_PREPARE_FAILED": "Unable to prepare task - please check your configuration",
	"ACTION_EMIT_FAILED":    "Communication error - unable to send task",
	"ACTION_EXECUTE_FAILED": "Task execution failed - runtime error encountered",
	"ACTION_SKIPPED":        "Task skipped - no changes detected from previous request",
	"ACTION_ID_REQUIRED":    "Task identifier required - please provide a channel ID",

	// Channel Related - Helpful & Specific
	"CHANNEL_VALIDATION_FAILED":  "Channel setup declined - configuration requirements not met",
	"CHANNEL_CREATION_FAILED":    "Unable to create channel - please verify your configuration",
	"CHANNEL_UPDATE_FAILED":      "Channel update unsuccessful - validation requirements not satisfied",
	"CHANNEL_CREATED":            "Channel established - ready for operation",
	"CHANNEL_UPDATED":            "Channel configuration updated successfully",
	"CHANNEL_INVALID_DEFINITION": "Channel definition invalid - please review your setup",
	"CHANNEL_MISSING_ID":         "Channel identifier required - please provide a unique ID",
	"CHANNEL_MISSING_TYPE":       "Channel type specification required",
	"CHANNEL_INVALID_TYPE":       "Channel type not recognized - please specify a valid type",
	"CHANNEL_INVALID_PAYLOAD":    "Payload format not accepted - please check your data structure",
	"CHANNEL_INVALID_STRUCTURE":  "Channel structure invalid - please review configuration requirements",

	// Subscription Related - Courteous & Informative
	"SUBSCRIPTION_INVALID_PARAMS":   "Subscription parameters not accepted - please verify your settings",
	"SUBSCRIPTION_EXISTS":           "Subscription already exists - updating configuration as requested",
	"SUBSCRIPTION_SUCCESS_SINGLE":   "Successfully subscribed to channel",
	"SUBSCRIPTION_SUCCESS_MULTIPLE": "Successfully subscribed to multiple channels",
	"SUBSCRIPTION_INVALID_TYPE":     "Subscription type not recognized - please specify a valid type",
	"SUBSCRIPTION_INVALID_HANDLER":  "Handler function not accepted - please provide a valid function",
	"SUBSCRIPTION_FAILED":           "Subscription unsuccessful - please check your configuration",

	// Call Related - Clear Error Communication
	"CALL_OFFLINE":        "Call unsuccessful - system is currently offline",
	"CALL_INVALID_ID":     "Call failed - channel identifier not recognized",
	"CALL_NOT_RESPONDING": "Call timeout - channel is not responding",
	"CALL_NO_SUBSCRIBER":  "Call unsuccessful - no handler found for this channel",

	// Dispatch Related - Professional Error Handling
	"DISPATCH_NO_SUBSCRIBER": "Dispatch failed - no subscriber registered for channel",
	"TIMELINE_NO_SUBSCRIBER": "Timeline error - no handler registered for scheduled task",

	// Timing Related - Helpful Advisories
	"TIMING_WARNING":           "Timing advisory - duration below recommended UI update threshold",
	"TIMING_ANIMATION_WARNING": "Performance suggestion - consider requestAnimationFrame for smooth animations",
	"TIMING_INVALID":           "Timer duration not accepted - please specify a valid timeframe",
	"TIMING_RECUPERATION":      "System rest mode - conserving resources for optimal performance",

	// Additional British AI Assistant Messages
	"TASK_UNDERSTOOD":        "Task understood - proceeding with your request",
	"TASK_COMPLETED":         "Task completed successfully - anything else I can help with?",
	"CONFIGURATION_ACCEPTED": "Configuration accepted - settings applied",
	"OPERATION_SUCCESSFUL":   "Operation completed as requested",
	"REQUEST_ACKNOWLEDGED":   "Request acknowledged - processing now",
	"SYSTEM_READY":           "All systems ready - standing by for instructions",
	"MAINTENANCE_MODE":       "Maintenance mode active - optimizing system performance",
	"COORDINATION_ACTIVE":    "Task coordination active - managing your requests",
	"INTELLIGENCE_ENGAGED":   "Processing intelligence engaged - analyzing your requirements",

	// Polite Error Variations
	"UNABLE_TO_COMPLY":        "I'm unable to comply with that request - please check the requirements",
	"TEMPORARILY_UNAVAILABLE": "Service temporarily unavailable - please try again in a moment",
	"ACCESS_PERMISSIONS":      "Access permissions required - please verify your credentials",
	"RESOURCE_UNAVAILABLE":    "Requested resource currently unavailable - shall I suggest alternatives?",
	"VALIDATION_REQUIREMENTS": "Validation requirements not met - please review your input",

	// Success Confirmations
	"ACKNOWLEDGED_AND_PROCESSED": "Request acknowledged and processed successfully",
	"CONFIGURATION_APPLIED":      "Configuration applied - system updated as requested",
	"SUBSCRIPTION_ESTABLISHED":   "Subscription established - you'll receive updates as they occur",
	"CHANNEL_OPERATIONAL":        "Channel operational - ready to handle your requests",
	"SYSTEM_OPTIMIZED":           "System optimization complete - performance improved",

	// System Headers
	"QUANTUM_HEADER": "Q0.0U0.0A0.0N0.0T0.0U0.0M0 - I0.0N0.0C0.0E0.0P0.0T0.0I0.0O0.0N0.0S0-- ",
}

// === PROTECTION CONFIGURATION ===

var PROTECTION = struct {
	CallThreshold   int
	MinDebounce     int
	MinThrottle     int
	MaxDelay        int
	Window          int
	InitialDelay    int
	SystemLoadDelay int
	System          struct {
		CPU struct {
			Warning  int
			Critical int
		}
		Memory struct {
			Warning  int
			Critical int
		}
		EventLoop struct {
			Warning  int
			Critical int
		}
		OverloadThreshold int
	}
}{
	CallThreshold:   CallThreshold,
	MinDebounce:     MinDebounce,
	MinThrottle:     MinThrottle,
	MaxDelay:        MaxDelay,
	Window:          ProtectionWindow,
	InitialDelay:    InitialDelay,
	SystemLoadDelay: SystemLoadDelay,
	System: struct {
		CPU struct {
			Warning  int
			Critical int
		}
		Memory struct {
			Warning  int
			Critical int
		}
		EventLoop struct {
			Warning  int
			Critical int
		}
		OverloadThreshold int
	}{
		CPU: struct {
			Warning  int
			Critical int
		}{
			Warning:  CPUWarning,
			Critical: CPUCritical,
		},
		Memory: struct {
			Warning  int
			Critical int
		}{
			Warning:  MemoryWarning,
			Critical: MemoryCritical,
		},
		EventLoop: struct {
			Warning  int
			Critical int
		}{
			Warning:  EventLoopWarning,
			Critical: EventLoopCritical,
		},
		OverloadThreshold: OverloadThreshold,
	},
}

// === BREATHING CONFIGURATION ===

var BREATHING = struct {
	Rates struct {
		Min      time.Duration
		Base     time.Duration
		Max      time.Duration
		Recovery time.Duration
	}
	Stress struct {
		Low      float64
		Medium   float64
		High     float64
		Critical float64
	}
	Recovery struct {
		BreathDebt  int
		CoolDown    float64
		MinRecovery time.Duration
		MaxRecovery time.Duration
	}
	Limits struct {
		MaxCPU       int
		MaxMemory    int
		MaxEventLoop int
		MaxCallRate  int
	}
	Patterns struct {
		Normal struct {
			InRatio   float64
			OutRatio  float64
			HoldRatio float64
		}
		Recovery struct {
			InRatio   float64
			OutRatio  float64
			HoldRatio float64
		}
	}
}{
	Rates: struct {
		Min      time.Duration
		Base     time.Duration
		Max      time.Duration
		Recovery time.Duration
	}{
		Min:      BreathingMinRate,
		Base:     BreathingBaseRate,
		Max:      BreathingMaxRate,
		Recovery: BreathingRecoveryRate,
	},
	Stress: struct {
		Low      float64
		Medium   float64
		High     float64
		Critical float64
	}{
		Low:      BreathingStressLow,
		Medium:   BreathingStressMedium,
		High:     BreathingStressHigh,
		Critical: BreathingStressCritical,
	},
	Recovery: struct {
		BreathDebt  int
		CoolDown    float64
		MinRecovery time.Duration
		MaxRecovery time.Duration
	}{
		BreathDebt:  15,
		CoolDown:    1.1,
		MinRecovery: 500 * time.Millisecond,
		MaxRecovery: 5000 * time.Millisecond,
	},
	Limits: struct {
		MaxCPU       int
		MaxMemory    int
		MaxEventLoop int
		MaxCallRate  int
	}{
		MaxCPU:       80,
		MaxMemory:    85,
		MaxEventLoop: 50,
		MaxCallRate:  1000,
	},
	Patterns: struct {
		Normal struct {
			InRatio   float64
			OutRatio  float64
			HoldRatio float64
		}
		Recovery struct {
			InRatio   float64
			OutRatio  float64
			HoldRatio float64
		}
	}{
		Normal: struct {
			InRatio   float64
			OutRatio  float64
			HoldRatio float64
		}{
			InRatio:   1,
			OutRatio:  1,
			HoldRatio: 0.5,
		},
		Recovery: struct {
			InRatio   float64
			OutRatio  float64
			HoldRatio float64
		}{
			InRatio:   2,
			OutRatio:  2,
			HoldRatio: 1,
		},
	},
}

// === DEFAULT SYSTEM STATE (TypeScript defaultMetrics equivalent) ===

// SystemState represents the complete system state
type SystemState struct {
	System struct {
		CPU          float64 `json:"cpu"`
		Memory       float64 `json:"memory"`
		EventLoop    float64 `json:"eventLoop"`
		IsOverloaded bool    `json:"isOverloaded"`
	} `json:"system"`

	Breathing struct {
		BreathCount       int64         `json:"breathCount"`
		CurrentRate       time.Duration `json:"currentRate"`
		LastBreath        int64         `json:"lastBreath"`
		Stress            float64       `json:"stress"`
		IsRecuperating    bool          `json:"isRecuperating"`
		RecuperationDepth int           `json:"recuperationDepth"`
		Pattern           string        `json:"pattern"`
		NextBreathDue     int64         `json:"nextBreathDue"`
	} `json:"breathing"`

	Performance struct {
		CallsTotal        int64 `json:"callsTotal"`
		CallsPerSecond    int64 `json:"callsPerSecond"`
		LastCallTimestamp int64 `json:"lastCallTimestamp"`
		ActiveQueues      struct {
			Critical   int `json:"critical"`
			High       int `json:"high"`
			Medium     int `json:"medium"`
			Low        int `json:"low"`
			Background int `json:"background"`
		} `json:"activeQueues"`
		QueueDepth int `json:"queueDepth"`
	} `json:"performance"`

	Stress struct {
		CPU       float64 `json:"cpu"`
		Memory    float64 `json:"memory"`
		EventLoop float64 `json:"eventLoop"`
		CallRate  float64 `json:"callRate"`
		Combined  float64 `json:"combined"`
	} `json:"stress"`

	Store struct {
		Channels    int `json:"channels"`
		Branches    int `json:"branches"`
		Tasks       int `json:"tasks"`
		Subscribers int `json:"subscribers"`
		Timeline    int `json:"timeline"`
	} `json:"store"`

	LastUpdate       int64 `json:"lastUpdate"`
	InRecuperation   bool  `json:"inRecuperation"`
	Hibernating      bool  `json:"hibernating"`
	ActiveFormations int   `json:"activeFormations"`
	Locked           bool  `json:"_Locked"`
	Init             bool  `json:"_init"`
	Shutdown         bool  `json:"_shutdown"`
}

// DefaultSystemState provides initial state matching TypeScript defaultMetrics
var DefaultSystemState = SystemState{
	System: struct {
		CPU          float64 `json:"cpu"`
		Memory       float64 `json:"memory"`
		EventLoop    float64 `json:"eventLoop"`
		IsOverloaded bool    `json:"isOverloaded"`
	}{
		CPU:          0,
		Memory:       0,
		EventLoop:    0,
		IsOverloaded: false,
	},

	Breathing: struct {
		BreathCount       int64         `json:"breathCount"`
		CurrentRate       time.Duration `json:"currentRate"`
		LastBreath        int64         `json:"lastBreath"`
		Stress            float64       `json:"stress"`
		IsRecuperating    bool          `json:"isRecuperating"`
		RecuperationDepth int           `json:"recuperationDepth"`
		Pattern           string        `json:"pattern"`
		NextBreathDue     int64         `json:"nextBreathDue"`
	}{
		BreathCount:       0,
		CurrentRate:       BreathingBaseRate,
		LastBreath:        time.Now().UnixMilli(),
		Stress:            0,
		IsRecuperating:    false,
		RecuperationDepth: 0,
		Pattern:           "NORMAL",
		NextBreathDue:     time.Now().UnixMilli() + int64(BreathingBaseRate.Milliseconds()),
	},

	Performance: struct {
		CallsTotal        int64 `json:"callsTotal"`
		CallsPerSecond    int64 `json:"callsPerSecond"`
		LastCallTimestamp int64 `json:"lastCallTimestamp"`
		ActiveQueues      struct {
			Critical   int `json:"critical"`
			High       int `json:"high"`
			Medium     int `json:"medium"`
			Low        int `json:"low"`
			Background int `json:"background"`
		} `json:"activeQueues"`
		QueueDepth int `json:"queueDepth"`
	}{
		CallsTotal:        0,
		CallsPerSecond:    0,
		LastCallTimestamp: time.Now().UnixMilli(),
		ActiveQueues: struct {
			Critical   int `json:"critical"`
			High       int `json:"high"`
			Medium     int `json:"medium"`
			Low        int `json:"low"`
			Background int `json:"background"`
		}{
			Critical:   0,
			High:       0,
			Medium:     0,
			Low:        0,
			Background: 0,
		},
		QueueDepth: 0,
	},

	Stress: struct {
		CPU       float64 `json:"cpu"`
		Memory    float64 `json:"memory"`
		EventLoop float64 `json:"eventLoop"`
		CallRate  float64 `json:"callRate"`
		Combined  float64 `json:"combined"`
	}{
		CPU:       0,
		Memory:    0,
		EventLoop: 0,
		CallRate:  0,
		Combined:  0,
	},

	Store: struct {
		Channels    int `json:"channels"`
		Branches    int `json:"branches"`
		Tasks       int `json:"tasks"`
		Subscribers int `json:"subscribers"`
		Timeline    int `json:"timeline"`
	}{
		Channels:    0,
		Branches:    0,
		Tasks:       0,
		Subscribers: 0,
		Timeline:    0,
	},

	LastUpdate:       time.Now().UnixMilli(),
	InRecuperation:   false,
	Hibernating:      false,
	ActiveFormations: 0,
	Locked:           false,
	Init:             false,
	Shutdown:         false,
}

// === HELPER FUNCTIONS ===

// GetMessage returns a system message by key
func GetMessage(key string) string {
	if msg, exists := MSG[key]; exists {
		return msg
	}
	return "Unknown system message"
}

// === EXISTING HELPER FUNCTIONS (unchanged) ===

// Environment holds runtime environment information
type Environment struct {
	IsTest        bool
	IsProduction  bool
	IsDevelopment bool
	HasProfile    bool
	NumCPU        int
	GOMAXPROCS    int
}

// RuntimeLimits defines system resource limits
type RuntimeLimits struct {
	MaxMemoryMB        int64
	MaxGoroutines      int
	MaxFileDescriptors int
	MaxConnections     int
}

// PerformanceProfile defines performance tuning profiles
type PerformanceProfile struct {
	Name             string
	WorkerPoolSize   int
	ChannelBuffer    int
	CleanupInterval  time.Duration
	MetricsEnabled   bool
	BreathingEnabled bool
}

// Predefined performance profiles (unchanged)
var (
	HighThroughputProfile = PerformanceProfile{
		Name:             "high-throughput",
		WorkerPoolSize:   200,
		ChannelBuffer:    2000,
		CleanupInterval:  1 * time.Minute,
		MetricsEnabled:   true,
		BreathingEnabled: true,
	}

	LowLatencyProfile = PerformanceProfile{
		Name:             "low-latency",
		WorkerPoolSize:   50,
		ChannelBuffer:    100,
		CleanupInterval:  10 * time.Second,
		MetricsEnabled:   false,
		BreathingEnabled: false,
	}

	BalancedProfile = PerformanceProfile{
		Name:             "balanced",
		WorkerPoolSize:   WorkerPoolSize,
		ChannelBuffer:    DefaultChannelBuffer,
		CleanupInterval:  CleanupInterval,
		MetricsEnabled:   true,
		BreathingEnabled: true,
	}

	TestProfile = PerformanceProfile{
		Name:             "test",
		WorkerPoolSize:   10,
		ChannelBuffer:    100,
		CleanupInterval:  1 * time.Second,
		MetricsEnabled:   true,
		BreathingEnabled: false,
	}
)

// GetProfile returns a performance profile by name
func GetProfile(name string) PerformanceProfile {
	switch name {
	case "high-throughput":
		return HighThroughputProfile
	case "low-latency":
		return LowLatencyProfile
	case "test":
		return TestProfile
	default:
		return BalancedProfile
	}
}

// === TIMING HELPERS ===

// TimingCategory represents different timing requirements
type TimingCategory int

const (
	TimingImmediate TimingCategory = iota // < 1ms
	TimingFast                            // 1-10ms
	TimingNormal                          // 10-100ms
	TimingSlow                            // 100ms-1s
	TimingBatch                           // > 1s
)

// GetTimingCategory determines timing category for a duration
func GetTimingCategory(duration time.Duration) TimingCategory {
	switch {
	case duration < 1*time.Millisecond:
		return TimingImmediate
	case duration < 10*time.Millisecond:
		return TimingFast
	case duration < 100*time.Millisecond:
		return TimingNormal
	case duration < 1*time.Second:
		return TimingSlow
	default:
		return TimingBatch
	}
}

// === VALIDATION HELPERS ===

// ValidateInterval ensures interval is within acceptable bounds
func ValidateInterval(interval time.Duration) time.Duration {
	if interval < MinInterval {
		return MinInterval
	}
	if interval > MaxInterval {
		return MaxInterval
	}
	return interval
}

// ValidateTimeout ensures timeout is within acceptable bounds
func ValidateTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return DefaultCallTimeout
	}
	if timeout > MaxTimerDuration {
		return MaxTimerDuration
	}
	return timeout
}

// ValidateRepeat ensures repeat count is within bounds
func ValidateRepeat(repeat int) int {
	if repeat < 0 {
		return 0
	}
	if repeat > DefaultMaxRepeats {
		return DefaultMaxRepeats
	}
	return repeat
}
