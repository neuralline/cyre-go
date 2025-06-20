// config/cyre-config.go
// Core configuration constants and settings for Cyre Go

package config

import "time"

/*
	C.Y.R.E - C.O.N.F.I.G

	Core configuration system providing:
	- Performance tuning constants
	- Timing and protection defaults
	- Memory management settings
	- Environment detection
	- Breathing system parameters
	- Zero-config smart defaults
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

	// === BREATHING SYSTEM ===

	// Stress monitoring
	StressThreshold   = 0.80 // Stress level to trigger breathing
	CriticalThreshold = 0.95 // Critical stress level
	RecoveryThreshold = 0.30 // Stress level for recovery
	HealthyThreshold  = 0.10 // Healthy operation threshold

	// Breathing rates (based on Cyre's 2:2:1 ratio)
	BreathingBaseRate     = 200 * time.Millisecond  // Base breathing rate
	BreathingStressRate   = 500 * time.Millisecond  // Stressed breathing rate
	BreathingCriticalRate = 1000 * time.Millisecond // Critical breathing rate
	BreathingRecoveryRate = 100 * time.Millisecond  // Recovery breathing rate

	// === MONITORING & METRICS ===

	// Sensor settings
	MetricsRetentionTime  = 1 * time.Hour          // How long to retain metrics
	MetricsCollectionRate = 100 * time.Millisecond // Metrics collection frequency
	MaxMetricsEvents      = 100000                 // Maximum events to store

	// Performance monitoring
	LatencyHistorySize   = 1000 // Number of latency samples to keep
	ThroughputWindowSize = 10   // Seconds for throughput calculation
	ErrorRateWindowSize  = 60   // Seconds for error rate calculation

	// === PROTECTION SYSTEM ===

	// Change detection
	PayloadComparisionDepth = 10      // Maximum depth for payload comparison
	MaxPayloadSize          = 1024000 // Maximum payload size (1MB)

	// Error handling
	MaxErrorsPerAction = 100             // Maximum errors before action suspension
	ErrorRetryDelay    = 1 * time.Second // Delay between error retries
	MaxRetryAttempts   = 3               // Maximum retry attempts

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

// === ENVIRONMENT DETECTION ===

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

// Predefined performance profiles
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
