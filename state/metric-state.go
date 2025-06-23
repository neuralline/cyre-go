// state/metric-state.go - FIXED TYPE DEFINITIONS
// Fix all MetricState definition and import issues

package state

import (
	"fmt"
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

// === CPU MONITORING TYPES ===

type CPUStats struct {
	User    uint64
	Nice    uint64
	System  uint64
	Idle    uint64
	IOWait  uint64
	IRQ     uint64
	SoftIRQ uint64
	Total   uint64
}

type CPUMonitor struct {
	lastStats CPUStats
	lastTime  time.Time
	mu        sync.RWMutex
}

// === MAIN METRICSTATE TYPE ===

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

	// Health monitoring state
	lastGCStats runtime.MemStats
	lastGCTime  time.Time
}

// Global instances
var globalMetricState *MetricState
var metricStateOnce sync.Once
var globalCPUMonitor = &CPUMonitor{}

// === INITIALIZATION ===

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
			lastGCTime:       time.Now(),
		}

		// Initialize GC stats
		runtime.ReadMemStats(&globalMetricState.lastGCStats)

		// Initialize atomic values
		atomic.StoreInt32(&globalMetricState.workerLimit, int32(runtime.NumCPU()))
		atomic.StoreInt64(&globalMetricState.currentSecond, time.Now().Unix())

		Critical("Metric State initialized successfully").
			Location("context/metric-state.go").
			Metadata(map[string]interface{}{
				"initialWorkers": runtime.NumCPU(),
				"windowSize":     10,
			}).
			Log()

		// Start health monitoring using TimeKeeper.keep (like TypeScript Cyre)
		globalMetricState.initializeBreathing()
	})
	return globalMetricState
}

// InitializeMetricStateAccurate creates metric state with accurate monitoring
func InitializeMetricStateAccurate() *MetricState {
	metricStateOnce.Do(func() {
		state := config.DefaultSystemState
		state.LastUpdate = time.Now().UnixMilli()

		globalMetricState = &MetricState{
			state:      &state,
			windowSize: 10, // Track last 10 measurements

			throughputWindow: make([]float64, 0, 10),
			latencyWindow:    make([]float64, 0, 10),
			lastGCTime:       time.Now(),
		}

		// Initialize GC stats for accurate monitoring
		runtime.ReadMemStats(&globalMetricState.lastGCStats)

		// Initialize CPU monitoring
		globalCPUMonitor.lastTime = time.Time{}

		// Initialize atomic values with more realistic defaults
		initialWorkers := runtime.NumCPU() * 2 // Start with 2x CPU cores
		atomic.StoreInt32(&globalMetricState.workerLimit, int32(initialWorkers))
		atomic.StoreInt64(&globalMetricState.currentSecond, time.Now().Unix())

		// Update state with initial worker count
		globalMetricState.state.Workers.Current = initialWorkers
		globalMetricState.state.Workers.Optimal = initialWorkers

		Critical("Accurate Metric State initialized successfully").
			Location("context/metric-state.go").
			Metadata(map[string]interface{}{
				"initialWorkers": initialWorkers,
				"mode":           "accurate",
				"windowSize":     10,
				"cpuCores":       runtime.NumCPU(),
			}).
			Log()

		// Start breathing system using accurate measurements
		globalMetricState.initializeBreathingAccurate()
	})
	return globalMetricState
}

// GetMetricState returns the global metric state instance
func GetMetricState() *MetricState {
	if globalMetricState == nil {
		return InitializeMetricState()
	}
	return globalMetricState
}

// === SYSTEM FLAG MANAGEMENT ===

// Lock the system (critical operation)
func (ms *MetricState) Lock() {
	atomic.StoreInt32(&ms.isLocked, 1)

	ms.mu.Lock()
	ms.state.Locked = true
	ms.state.LastUpdate = time.Now().UnixMilli()
	ms.mu.Unlock()

	Critical("System locked for maintenance").
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

	Critical("System unlocked - resuming operations").
		Location("context/metric-state.go").
		Log()
}

// Initialize system
func (ms *MetricState) Init() {
	ms.mu.Lock()
	ms.state.Initialized = true
	ms.state.LastUpdate = time.Now().UnixMilli()
	ms.mu.Unlock()

	Warn("System initialization completed").
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

	Warn("System shutdown initiated").
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
		Critical("System entering hibernation mode").
			Location("context/metric-state.go").
			Log()
	} else {
		Critical("System exiting hibernation mode").
			Location("context/metric-state.go").
			Log()
	}
}

// === HOT PATH FLAG CHECKS (Ultra-fast atomic reads) ===

// IsLocked returns true if system is locked
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

// === STATE ACCESS ===

// Get returns read-only snapshot of current state
func (ms *MetricState) Get() config.SystemState {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	// Return copy of current state
	stateCopy := *ms.state

	// Add atomic flag values
	stateCopy.InRecuperation = atomic.LoadInt32(&ms.isRecuperating) == 1
	stateCopy.Hibernating = atomic.LoadInt32(&ms.isHibernating) == 1
	stateCopy.Locked = atomic.LoadInt32(&ms.isLocked) == 1
	stateCopy.Shutdown = atomic.LoadInt32(&ms.isShutdown) == 1

	return stateCopy
}

// === CALL TRACKING ===

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
		// New second - reset counter
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

	// Store last call time
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

	// Update error rate
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

// UpdateSystemHealth updates health metrics
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

// === BREATHING SYSTEM PLACEHOLDER METHODS ===
// These will be implemented in the next artifact

func (ms *MetricState) initializeBreathing() {
	// Basic breathing system initialization
	fmt.Printf("DEBUG: initializeBreathing() called\n")

	go func() {
		time.Sleep(2 * time.Second)

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		tickCount := 0
		for {
			select {
			case <-ticker.C:
				tickCount++
				ms.updateBreathingFromMetrics()
			}
		}
	}()

}

func (ms *MetricState) initializeBreathingAccurate() {
	// Accurate breathing system initialization
	ms.initializeBreathing() // Use basic for now, will enhance
}

func (ms *MetricState) updateBreathingFromMetrics() {

	cpu := ms.getCPUUsage()
	memory := ms.getMemoryUsage()
	goroutines := runtime.NumGoroutine()
	gcPressure := ms.detectGCPressure()

	ms.UpdateSystemHealth(cpu, memory, goroutines, gcPressure)

	throughput := ms.measureThroughput()
	ms.UpdateThroughput(throughput)
}

func (ms *MetricState) getCPUUsage() float64 {
	goroutines := float64(runtime.NumGoroutine())
	numCPU := float64(runtime.NumCPU())

	cpuPercent := (goroutines / (numCPU * 4.0)) * 100.0
	if cpuPercent > 100.0 {
		cpuPercent = 100.0
	}

	return cpuPercent
}

func (ms *MetricState) getMemoryUsage() float64 {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	if memStats.Sys == 0 {
		return 0.0
	}

	memoryPercent := (float64(memStats.Alloc) / float64(memStats.Sys)) * 100.0
	if memoryPercent > 100.0 {
		memoryPercent = 100.0
	}

	return memoryPercent
}

func (ms *MetricState) detectGCPressure() bool {
	var currentStats runtime.MemStats
	runtime.ReadMemStats(&currentStats)
	now := time.Now()

	gcDelta := currentStats.NumGC - ms.lastGCStats.NumGC
	timeDelta := now.Sub(ms.lastGCTime)

	ms.lastGCStats = currentStats
	ms.lastGCTime = now

	if timeDelta > 0 && gcDelta > 0 {
		gcPerSecond := float64(gcDelta) / timeDelta.Seconds()
		return gcPerSecond > 2.0
	}

	return false
}

func (ms *MetricState) measureThroughput() float64 {
	return float64(atomic.LoadInt64(&ms.callsThisSecond))
}

func (ms *MetricState) calculateStressLevel() {
	cpuStress := ms.state.Health.CPUPercent / 100.0
	memoryStress := ms.state.Health.MemoryPercent / 100.0

	errorStress := ms.state.Performance.ErrorRate * 2.0
	if errorStress > 1.0 {
		errorStress = 1.0
	}

	latencyStress := 0.0
	if ms.state.Performance.AvgLatencyMs > 100 {
		latencyStress = (ms.state.Performance.AvgLatencyMs - 100) / 400.0
		if latencyStress > 1.0 {
			latencyStress = 1.0
		}
	}

	ms.state.Breathing.StressLevel = cpuStress*0.3 + memoryStress*0.2 + errorStress*0.3 + latencyStress*0.2

	if ms.state.Breathing.StressLevel > 1.0 {
		ms.state.Breathing.StressLevel = 1.0
	}
}

func (ms *MetricState) updateBreathingControl() {
	stress := ms.state.Breathing.StressLevel

	isRecuperating := stress > 0.8
	blockNormal := stress > 0.7
	blockLow := stress > 0.5

	ms.state.Breathing.IsRecuperating = isRecuperating
	ms.state.Breathing.BlockNormal = blockNormal
	ms.state.Breathing.BlockLow = blockLow
	ms.state.InRecuperation = isRecuperating

	atomic.StoreInt32(&ms.isRecuperating, boolToInt32(isRecuperating))
	atomic.StoreInt32(&ms.blockNormal, boolToInt32(blockNormal))
	atomic.StoreInt32(&ms.blockLow, boolToInt32(blockLow))

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

func (ms *MetricState) makeScalingDecision() {
	// Basic scaling logic placeholder
	currentThroughput := ms.state.Performance.ThroughputPerSec
	currentLatency := ms.state.Performance.AvgLatencyMs

	if ms.state.Workers.SweetSpot {
		return
	}

	if time.Now().UnixMilli()-ms.state.Workers.LastScaleUp < 5000 {
		return
	}

	if ms.state.Breathing.StressLevel > 0.7 {
		return
	}

	shouldScale := ms.shouldScaleUp(currentThroughput, currentLatency)

	if shouldScale {
		ms.scaleUpWorkers()
	} else if ms.scaleAttempts > 0 {
		ms.detectSweetSpot()
	}
}

func (ms *MetricState) shouldScaleUp(throughput, latency float64) bool {
	if ms.state.Workers.Current >= runtime.NumCPU()*4 {
		return false
	}

	if ms.lastThroughput == 0 {
		return true
	}

	throughputImproved := throughput > ms.lastThroughput*1.05
	latencyAcceptable := latency <= ms.lastLatency*1.15

	return throughputImproved && latencyAcceptable
}

func (ms *MetricState) scaleUpWorkers() {
	ms.state.Workers.Current++
	ms.state.Workers.LastScaleUp = time.Now().UnixMilli()
	ms.scaleAttempts++

	atomic.StoreInt32(&ms.workerLimit, int32(ms.state.Workers.Current))

	ms.lastThroughput = ms.state.Performance.ThroughputPerSec
	ms.lastLatency = ms.state.Performance.AvgLatencyMs

	ms.state.Breathing.Phase = "scaling"
}

func (ms *MetricState) detectSweetSpot() {
	ms.state.Workers.SweetSpot = true
	ms.state.Workers.Optimal = ms.state.Workers.Current - 1
	ms.state.Workers.Current = ms.state.Workers.Optimal

	atomic.StoreInt32(&ms.workerLimit, int32(ms.state.Workers.Current))

	ms.state.Breathing.Phase = "optimized"
}
