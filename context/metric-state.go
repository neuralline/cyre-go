// context/metric-state.go
// System Brain - Central intelligence engine for Cyre Go
// Handles system state, breathing control, and intelligent decisions

package context

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/neuralline/cyre-go/config"
)

/*
	MetricState - The System Brain

	This is the central nervous system that:
	- Tracks system performance and health metrics
	- Makes intelligent scaling and blocking decisions
	- Controls breathing system behavior
	- Provides ultra-fast flag checks for hot path
	- Manages store counts and call tracking

	Go-native optimizations:
	- Atomic operations for hot path flags
	- Read-write mutexes for efficient concurrent access
	- Memory-aligned struct fields for cache performance
	- Zero-allocation hot path methods
*/

type MetricState struct {
	// System state (protected by mutex)
	state *config.SystemState
	mu    sync.RWMutex

	// Performance tracking windows
	throughputWindow []float64
	latencyWindow    []float64
	windowSize       int

	// Intelligence tracking
	lastThroughput float64
	lastLatency    float64
	scaleAttempts  int

	// Atomic flags for ultra-fast hot path reads (~5ns each)
	isRecuperating int32 // 1=recuperating, 0=normal
	blockNormal    int32 // 1=block normal actions, 0=allow
	blockLow       int32 // 1=block low actions, 0=allow
	workerLimit    int32 // Current worker limit
	isLocked       int32 // 1=system locked, 0=unlocked
	isHibernating  int32 // 1=hibernating, 0=active
	isShutdown     int32 // 1=shutdown, 0=running

	// Call rate tracking (atomic for thread safety)
	callsTotal      int64 // Total calls since start
	lastCallTime    int64 // Last call timestamp (UnixMilli)
	callsThisSecond int64 // Calls in current second window
	currentSecond   int64 // Current second for rate calculation
}

// Global metric state instance
var globalMetricState *MetricState
var metricStateOnce sync.Once

// === INITIALIZATION ===

// InitializeMetricState creates and initializes the global metric state
// InitializeMetricState creates and initializes the global metric state
func InitializeMetricState() *MetricState {
	metricStateOnce.Do(func() {
		state := config.DefaultSystemState
		state.LastUpdate = time.Now().UnixMilli()

		globalMetricState = &MetricState{
			state:      &state,
			windowSize: 10, // Track last 10 measurements

			throughputWindow: make([]float64, 0, 10),
			latencyWindow:    make([]float64, 0, 10),
		}

		// Initialize atomic values
		atomic.StoreInt32(&globalMetricState.workerLimit, int32(runtime.NumCPU()))
		atomic.StoreInt64(&globalMetricState.currentSecond, time.Now().Unix())

		SensorSuccess("Metric State initialized successfully").
			Location("context/metric-state.go").
			Metadata(map[string]interface{}{
				"initialWorkers": runtime.NumCPU(),
				"windowSize":     10,
			}).
			Log()

		// Start background health monitoring after initialization
		globalMetricState.startHealthMonitoring()
	})
	return globalMetricState
}

func (ms *MetricState) startHealthMonitoring() {
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		for {
			cpu := ms.getCPUUsage()              // Goroutine-based CPU estimation
			memory := ms.getMemoryUsage()        // Real memory from runtime
			goroutines := runtime.NumGoroutine() // Live goroutine count
			gcPressure := ms.detectGCPressure()  // GC frequency analysis

			ms.UpdateSystemHealth(cpu, memory, goroutines, gcPressure)
			ms.UpdateThroughput(ms.measureThroughput())
		}
	}()
}

// GetMetricState returns the global metric state instance
func GetMetricState() *MetricState {
	if globalMetricState == nil {
		return InitializeMetricState()
	}
	return globalMetricState
}

// === SYSTEM FLAG MANAGEMENT (Like TypeScript examples) ===

// Lock the system (critical operation)
func (ms *MetricState) Lock() {
	atomic.StoreInt32(&ms.isLocked, 1)

	ms.mu.Lock()
	ms.state.Locked = true
	ms.state.LastUpdate = time.Now().UnixMilli()
	ms.mu.Unlock()

	SensorCritical("System locked for maintenance").
		Location("context/metric-state.go").
		Log()
}

// Unlock the system
func (ms *MetricState) Unlock() {
	atomic.StoreInt32(&ms.isLocked, 0)

	ms.mu.Lock()
	ms.state.Locked = false
	ms.state.LastUpdate = time.Now().UnixMilli()
	ms.mu.Unlock()

	SensorCritical("System unlocked - resuming operations").
		Location("context/metric-state.go").
		Log()
}

// Initialize system
func (ms *MetricState) Init() {
	ms.mu.Lock()
	ms.state.Initialized = true
	ms.state.LastUpdate = time.Now().UnixMilli()
	ms.mu.Unlock()

	SensorCritical("System initialization completed").
		Location("context/metric-state.go").
		Log()
}

// Shutdown system
func (ms *MetricState) Shutdown() {
	atomic.StoreInt32(&ms.isShutdown, 1)

	ms.mu.Lock()
	ms.state.Shutdown = true
	ms.state.LastUpdate = time.Now().UnixMilli()
	ms.mu.Unlock()

	SensorCritical("System shutdown initiated").
		Location("context/metric-state.go").
		Log()
}

// SetHibernating controls hibernation state
func (ms *MetricState) SetHibernating(hibernating bool) {
	atomic.StoreInt32(&ms.isHibernating, boolToInt32(hibernating))

	ms.mu.Lock()
	ms.state.Hibernating = hibernating
	ms.state.LastUpdate = time.Now().UnixMilli()
	ms.mu.Unlock()

	if hibernating {
		SensorCritical("System entering hibernation mode").
			Location("context/metric-state.go").
			Log()
	} else {
		SensorCritical("System exiting hibernation mode").
			Location("context/metric-state.go").
			Log()
	}
}

// === HOT PATH FLAG CHECKS (Ultra-fast atomic reads) ===

// IsLocked returns true if system is locked (matches TypeScript isLocked())
func (ms *MetricState) IsLocked() bool {
	return atomic.LoadInt32(&ms.isLocked) == 1
}

// IsRecuperating returns true if system is in recovery mode
func (ms *MetricState) IsRecuperating() bool {
	return atomic.LoadInt32(&ms.isRecuperating) == 1
}

// IsHibernating returns true if system is hibernating
func (ms *MetricState) IsHibernating() bool {
	return atomic.LoadInt32(&ms.isHibernating) == 1
}

// IsShutdown returns true if system is shutting down
func (ms *MetricState) IsShutdown() bool {
	return atomic.LoadInt32(&ms.isShutdown) == 1
}

// ShouldBlockNormal returns true if normal actions should be blocked
func (ms *MetricState) ShouldBlockNormal() bool {
	return atomic.LoadInt32(&ms.blockNormal) == 1
}

// ShouldBlockLow returns true if low priority actions should be blocked
func (ms *MetricState) ShouldBlockLow() bool {
	return atomic.LoadInt32(&ms.blockLow) == 1
}

// GetWorkerLimit returns current worker limit
func (ms *MetricState) GetWorkerLimit() int {
	return int(atomic.LoadInt32(&ms.workerLimit))
}

// === STATE ACCESS (Like TypeScript get() method) ===

// Get returns read-only snapshot of current state (matches TypeScript get())
func (ms *MetricState) Get() config.SystemState {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	// Return copy of current state (read-only like TypeScript Object.freeze)
	stateCopy := *ms.state

	// Add atomic flag values
	stateCopy.InRecuperation = atomic.LoadInt32(&ms.isRecuperating) == 1
	stateCopy.Hibernating = atomic.LoadInt32(&ms.isHibernating) == 1
	stateCopy.Locked = atomic.LoadInt32(&ms.isLocked) == 1
	stateCopy.Shutdown = atomic.LoadInt32(&ms.isShutdown) == 1

	return stateCopy
}

// === CALL TRACKING (For rate limiting like cyre.ts) ===

// UpdateCallMetrics tracks call rate for intelligent rate limiting
func (ms *MetricState) UpdateCallMetrics() {
	now := time.Now()
	nowUnix := now.Unix()
	nowMilli := now.UnixMilli()

	// Increment total calls
	atomic.AddInt64(&ms.callsTotal, 1)

	// Update call rate tracking
	currentSec := atomic.LoadInt64(&ms.currentSecond)
	if nowUnix != currentSec {
		// New second - reset counter and update rate
		atomic.StoreInt64(&ms.callsThisSecond, 1)
		atomic.StoreInt64(&ms.currentSecond, nowUnix)

		// Update state with current rate
		ms.mu.Lock()
		ms.state.Performance.CallsPerSecond = atomic.LoadInt64(&ms.callsThisSecond)
		ms.state.Performance.CallsTotal = atomic.LoadInt64(&ms.callsTotal)
		ms.state.Performance.LastCallTimestamp = nowMilli
		ms.state.LastUpdate = nowMilli
		ms.mu.Unlock()
	} else {
		// Same second - increment counter
		atomic.AddInt64(&ms.callsThisSecond, 1)
	}

	// Store last call time for latency calculations
	atomic.StoreInt64(&ms.lastCallTime, nowMilli)
}

// GetCallsPerSecond returns current call rate
func (ms *MetricState) GetCallsPerSecond() int64 {
	return atomic.LoadInt64(&ms.callsThisSecond)
}

// === PERFORMANCE TRACKING ===

// UpdatePerformance updates performance metrics from action execution
func (ms *MetricState) UpdatePerformance(latency time.Duration, success bool) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	// Add to sliding windows
	latencyMs := float64(latency.Nanoseconds()) / 1e6
	ms.addToWindow(&ms.latencyWindow, latencyMs)

	// Update error rate (simple approach)
	if !success {
		ms.state.Performance.ErrorRate = min(ms.state.Performance.ErrorRate+0.01, 1.0)
	} else {
		ms.state.Performance.ErrorRate = max(ms.state.Performance.ErrorRate-0.001, 0.0)
	}

	// Recalculate averages
	ms.state.Performance.AvgLatencyMs = ms.calculateAverage(ms.latencyWindow)
	ms.state.LastUpdate = time.Now().UnixMilli()
}

// UpdateThroughput measures throughput and triggers scaling decisions
func (ms *MetricState) UpdateThroughput(actionsPerSecond float64) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.addToWindow(&ms.throughputWindow, actionsPerSecond)
	ms.state.Performance.ThroughputPerSec = ms.calculateAverage(ms.throughputWindow)

	// Trigger intelligent scaling decision
	ms.makeScalingDecision()
}

// UpdateSystemHealth updates health metrics from breathing system
func (ms *MetricState) UpdateSystemHealth(cpu, memory float64, goroutines int, gcPressure bool) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.state.Health.CPUPercent = cpu
	ms.state.Health.MemoryPercent = memory
	ms.state.Health.GoroutineCount = goroutines
	ms.state.Health.GCPressure = gcPressure

	// Calculate overall stress level
	ms.calculateStressLevel()

	// Update breathing control flags
	ms.updateBreathingControl()

	ms.state.LastUpdate = time.Now().UnixMilli()
}

// UpdateStoreCounts tracks store sizes for system awareness
func (ms *MetricState) UpdateStoreCounts(channels, branches, tasks, subscribers, timeline int) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.state.Store.Channels = channels
	ms.state.Store.Branches = branches
	ms.state.Store.Tasks = tasks
	ms.state.Store.Subscribers = subscribers
	ms.state.Store.Timeline = timeline
	ms.state.LastUpdate = time.Now().UnixMilli()
}

// SetActiveFormations updates active scheduled formations count
func (ms *MetricState) SetActiveFormations(count int) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.state.ActiveFormations = count
	ms.state.LastUpdate = time.Now().UnixMilli()
}

// === INTELLIGENT SCALING LOGIC ===

// makeScalingDecision implements smart worker scaling
func (ms *MetricState) makeScalingDecision() {
	currentThroughput := ms.state.Performance.ThroughputPerSec
	currentLatency := ms.state.Performance.AvgLatencyMs

	// Don't scale if we found sweet spot
	if ms.state.Workers.SweetSpot {
		return
	}

	// Don't scale too frequently (minimum 5 seconds)
	if time.Now().UnixMilli()-ms.state.Workers.LastScaleUp < 5000 {
		return
	}

	// Don't scale if system is stressed
	if ms.state.Breathing.StressLevel > 0.7 {
		return
	}

	// Determine if we should scale up
	shouldScale := ms.shouldScaleUp(currentThroughput, currentLatency)

	if shouldScale {
		ms.scaleUpWorkers()
	} else if ms.scaleAttempts > 0 {
		// We tried scaling but it didn't help - found sweet spot
		ms.detectSweetSpot()
	}
}

// shouldScaleUp determines if adding workers would help performance
func (ms *MetricState) shouldScaleUp(throughput, latency float64) bool {
	// Don't exceed reasonable limits
	if ms.state.Workers.Current >= runtime.NumCPU()*4 {
		return false
	}

	// First attempt - just try it
	if ms.lastThroughput == 0 {
		return true
	}

	// Scale if throughput improved and latency didn't get much worse
	throughputImproved := throughput > ms.lastThroughput*1.05 // 5% improvement
	latencyAcceptable := latency <= ms.lastLatency*1.15       // Max 15% latency increase

	return throughputImproved && latencyAcceptable
}

// scaleUpWorkers adds more workers intelligently
func (ms *MetricState) scaleUpWorkers() {
	ms.state.Workers.Current++
	ms.state.Workers.LastScaleUp = time.Now().UnixMilli()
	ms.scaleAttempts++

	// Update atomic worker limit for hot path
	atomic.StoreInt32(&ms.workerLimit, int32(ms.state.Workers.Current))

	// Store metrics for next comparison
	ms.lastThroughput = ms.state.Performance.ThroughputPerSec
	ms.lastLatency = ms.state.Performance.AvgLatencyMs

	ms.state.Breathing.Phase = "scaling"

	SensorInfo("Scaling up workers").
		Location("context/metric-state.go").
		Metadata(map[string]interface{}{
			"currentWorkers": ms.state.Workers.Current,
			"throughput":     ms.state.Performance.ThroughputPerSec,
			"latency":        ms.state.Performance.AvgLatencyMs,
		}).
		Log()
}

// detectSweetSpot identifies optimal worker count
func (ms *MetricState) detectSweetSpot() {
	ms.state.Workers.SweetSpot = true
	ms.state.Workers.Optimal = ms.state.Workers.Current - 1 // Previous was better
	ms.state.Workers.Current = ms.state.Workers.Optimal

	// Update atomic limit
	atomic.StoreInt32(&ms.workerLimit, int32(ms.state.Workers.Current))

	ms.state.Breathing.Phase = "optimized"

	SensorSuccess("Sweet spot found - optimal worker configuration detected").
		Location("context/metric-state.go").
		Metadata(map[string]interface{}{
			"optimalWorkers": ms.state.Workers.Optimal,
			"throughput":     ms.state.Performance.ThroughputPerSec,
			"latency":        ms.state.Performance.AvgLatencyMs,
		}).
		Log()
}

// calculateStressLevel determines overall system stress
func (ms *MetricState) calculateStressLevel() {
	// Multi-factor stress calculation
	cpuStress := ms.state.Health.CPUPercent / 100.0
	memoryStress := ms.state.Health.MemoryPercent / 100.0

	// Error rate stress (exponential - errors are serious)
	errorStress := ms.state.Performance.ErrorRate * 2.0
	if errorStress > 1.0 {
		errorStress = 1.0
	}

	// Latency stress (stress increases above 100ms)
	latencyStress := 0.0
	if ms.state.Performance.AvgLatencyMs > 100 {
		latencyStress = (ms.state.Performance.AvgLatencyMs - 100) / 400.0
		if latencyStress > 1.0 {
			latencyStress = 1.0
		}
	}

	// Weighted combination
	ms.state.Breathing.StressLevel = cpuStress*0.3 + memoryStress*0.2 + errorStress*0.3 + latencyStress*0.2

	if ms.state.Breathing.StressLevel > 1.0 {
		ms.state.Breathing.StressLevel = 1.0
	}
}

// updateBreathingControl sets breathing flags based on stress
func (ms *MetricState) updateBreathingControl() {
	stress := ms.state.Breathing.StressLevel

	// Determine breathing state
	isRecuperating := stress > 0.8
	blockNormal := stress > 0.7
	blockLow := stress > 0.5

	ms.state.Breathing.IsRecuperating = isRecuperating
	ms.state.Breathing.BlockNormal = blockNormal
	ms.state.Breathing.BlockLow = blockLow
	ms.state.InRecuperation = isRecuperating

	// Update atomic flags for ultra-fast hot path reads
	atomic.StoreInt32(&ms.isRecuperating, boolToInt32(isRecuperating))
	atomic.StoreInt32(&ms.blockNormal, boolToInt32(blockNormal))
	atomic.StoreInt32(&ms.blockLow, boolToInt32(blockLow))

	// Update phase
	if isRecuperating {
		ms.state.Breathing.Phase = "recovery"
	} else if blockNormal {
		ms.state.Breathing.Phase = "stressed"
	} else if !ms.state.Workers.SweetSpot {
		ms.state.Breathing.Phase = "scaling"
	} else {
		ms.state.Breathing.Phase = "normal"
	}
}

// === HELPER FUNCTIONS ===

func (ms *MetricState) addToWindow(window *[]float64, value float64) {
	*window = append(*window, value)
	if len(*window) > ms.windowSize {
		*window = (*window)[1:] // Remove oldest
	}
}

func (ms *MetricState) calculateAverage(window []float64) float64 {
	if len(window) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, v := range window {
		sum += v
	}
	return sum / float64(len(window))
}

func boolToInt32(b bool) int32 {
	if b {
		return 1
	}
	return 0
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
