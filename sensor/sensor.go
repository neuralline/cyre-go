// sensor/sensor.go
// Metrics and monitoring system for performance tracking

package sensor

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/neuralline/cyre-go/config"
)

/*
	C.Y.R.E - S.E.N.S.O.R

	Advanced metrics and monitoring system providing:
	- Real-time performance tracking
	- System health monitoring
	- Error rate analysis
	- Memory usage tracking
	- Breathing system integration
	- Statistical analysis
	- Zero-overhead collection in production
*/

// === CORE TYPES ===

// Event represents a system event for tracking
type Event struct {
	Type      string                 `json:"type"`
	ActionID  string                 `json:"actionId"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration,omitempty"`
	Success   bool                   `json:"success"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// SystemMetrics represents overall system performance
type SystemMetrics struct {
	// Counters (atomic for thread safety)
	TotalCalls      int64 `json:"totalCalls"`
	TotalExecutions int64 `json:"totalExecutions"`
	TotalErrors     int64 `json:"totalErrors"`
	TotalThrottled  int64 `json:"totalThrottled"`
	TotalDebounced  int64 `json:"totalDebounced"`
	TotalSkipped    int64 `json:"totalSkipped"`

	// Timing
	StartTime    time.Time     `json:"startTime"`
	Uptime       time.Duration `json:"uptime"`
	LastActivity time.Time     `json:"lastActivity"`

	// Performance
	CallRate       float64       `json:"callRate"`      // calls per second
	ExecutionRate  float64       `json:"executionRate"` // executions per second
	ErrorRate      float64       `json:"errorRate"`     // error percentage
	SuccessRate    float64       `json:"successRate"`   // success percentage
	AverageLatency time.Duration `json:"averageLatency"`

	// System resources
	MemoryUsage    MemoryStats `json:"memoryUsage"`
	GoroutineCount int         `json:"goroutineCount"`

	// Health indicators
	HealthScore float64 `json:"healthScore"` // 0-1 health score
	StressLevel float64 `json:"stressLevel"` // 0-1 stress level
	IsHealthy   bool    `json:"isHealthy"`
	IsBreathing bool    `json:"isBreathing"`
}

// MemoryStats tracks memory usage
type MemoryStats struct {
	Alloc        uint64  `json:"alloc"`        // Current allocation
	TotalAlloc   uint64  `json:"totalAlloc"`   // Total allocations
	Sys          uint64  `json:"sys"`          // System memory
	NumGC        uint32  `json:"numGC"`        // GC cycles
	HeapAlloc    uint64  `json:"heapAlloc"`    // Heap allocation
	HeapInuse    uint64  `json:"heapInuse"`    // Heap in use
	StackInuse   uint64  `json:"stackInuse"`   // Stack in use
	UsagePercent float64 `json:"usagePercent"` // Usage percentage
}

// ChannelMetrics tracks per-action performance
type ChannelMetrics struct {
	ActionID        string        `json:"actionId"`
	Type            string        `json:"type"`
	Calls           int64         `json:"calls"`
	Executions      int64         `json:"executions"`
	Errors          int64         `json:"errors"`
	Throttled       int64         `json:"throttled"`
	Debounced       int64         `json:"debounced"`
	Skipped         int64         `json:"skipped"`
	AverageLatency  time.Duration `json:"averageLatency"`
	MinLatency      time.Duration `json:"minLatency"`
	MaxLatency      time.Duration `json:"maxLatency"`
	LastExecution   time.Time     `json:"lastExecution"`
	TotalLatency    time.Duration `json:"totalLatency"`
	SuccessRate     float64       `json:"successRate"`
	ErrorRate       float64       `json:"errorRate"`
	ExecutionRatio  float64       `json:"executionRatio"`  // executions/calls
	ProtectionRatio float64       `json:"protectionRatio"` // protections/calls
}

// === SENSOR MANAGER ===

// Sensor manages metrics collection and analysis
type Sensor struct {
	// Core state
	enabled      bool
	startTime    time.Time
	lastUpdate   time.Time
	updateTicker *time.Ticker
	mu           sync.RWMutex

	// Event storage
	events     []Event
	maxEvents  int
	eventIndex int64

	// Metrics aggregation
	systemMetrics  SystemMetrics
	channelMetrics map[string]*ChannelMetrics

	// Rate calculation
	callWindow  []int64 // Sliding window for call rate
	windowSize  int
	windowIndex int

	// Breathing system integration
	stressThreshold   float64
	criticalThreshold float64
	recoveryThreshold float64
	breathingEnabled  bool

	// Cleanup
	cleanupTicker *time.Ticker
	stopChan      chan struct{}
	wg            sync.WaitGroup
}

// GlobalSensor is the singleton sensor instance
var GlobalSensor *Sensor
var sensorOnce sync.Once

// Initialize creates and starts the global sensor
func Initialize() *Sensor {
	sensorOnce.Do(func() {
		GlobalSensor = &Sensor{
			enabled:           config.EnableMetrics,
			startTime:         time.Now(),
			maxEvents:         config.MaxMetricsEvents,
			events:            make([]Event, config.MaxMetricsEvents),
			channelMetrics:    make(map[string]*ChannelMetrics),
			windowSize:        config.ThroughputWindowSize,
			callWindow:        make([]int64, config.ThroughputWindowSize),
			stressThreshold:   config.StressThreshold,
			criticalThreshold: config.CriticalThreshold,
			recoveryThreshold: config.RecoveryThreshold,
			breathingEnabled:  config.EnableBreathing,
			stopChan:          make(chan struct{}),
		}

		// Initialize system metrics
		GlobalSensor.systemMetrics = SystemMetrics{
			StartTime: GlobalSensor.startTime,
		}

		// Start background processes
		GlobalSensor.start()
	})
	return GlobalSensor
}

// GetSensor returns the global sensor instance
func GetSensor() *Sensor {
	if GlobalSensor == nil {
		return Initialize()
	}
	return GlobalSensor
}

// === CORE OPERATIONS ===

// start begins background collection and cleanup
func (s *Sensor) start() {
	if !s.enabled {
		return
	}

	// Start metrics collection ticker
	s.updateTicker = time.NewTicker(config.MetricsCollectionRate)
	s.cleanupTicker = time.NewTicker(config.CleanupInterval)

	s.wg.Add(2)

	// Metrics update goroutine
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-s.updateTicker.C:
				s.updateSystemMetrics()
			case <-s.stopChan:
				return
			}
		}
	}()

	// Cleanup goroutine
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-s.cleanupTicker.C:
				s.cleanup()
			case <-s.stopChan:
				return
			}
		}
	}()
}

// Stop shuts down the sensor
func (s *Sensor) Stop() {
	if !s.enabled {
		return
	}

	close(s.stopChan)
	s.wg.Wait()

	if s.updateTicker != nil {
		s.updateTicker.Stop()
	}
	if s.cleanupTicker != nil {
		s.cleanupTicker.Stop()
	}
}

// === EVENT RECORDING ===

// RecordEvent records a system event for analysis
func (s *Sensor) RecordEvent(eventType, actionID string, duration time.Duration, success bool, err error, metadata map[string]interface{}) {
	if !s.enabled {
		return
	}

	event := Event{
		Type:      eventType,
		ActionID:  actionID,
		Timestamp: time.Now(),
		Duration:  duration,
		Success:   success,
		Metadata:  metadata,
	}

	if err != nil {
		event.Error = err.Error()
	}

	// Store event in circular buffer
	index := atomic.AddInt64(&s.eventIndex, 1) % int64(s.maxEvents)
	s.events[index] = event

	// Update channel metrics
	s.updateChannelMetrics(actionID, eventType, duration, success)

	// Update system counters
	s.updateSystemCounters(eventType, success)
}

// RecordCall records an action call
func (s *Sensor) RecordCall(actionID string) {
	s.RecordEvent("call", actionID, 0, true, nil, nil)
}

// RecordExecution records an action execution
func (s *Sensor) RecordExecution(actionID string, duration time.Duration, success bool, err error) {
	s.RecordEvent("execution", actionID, duration, success, err, nil)
}

// RecordProtection records a protection event (throttle, debounce, etc.)
func (s *Sensor) RecordProtection(actionID, protectionType string) {
	s.RecordEvent(protectionType, actionID, 0, true, nil, map[string]interface{}{
		"protection": protectionType,
	})
}

// === METRICS UPDATES ===

// updateChannelMetrics updates per-action metrics
func (s *Sensor) updateChannelMetrics(actionID, eventType string, duration time.Duration, success bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	metrics, exists := s.channelMetrics[actionID]
	if !exists {
		metrics = &ChannelMetrics{
			ActionID:   actionID,
			MinLatency: time.Hour, // Initialize to high value
		}
		s.channelMetrics[actionID] = metrics
	}

	switch eventType {
	case "call":
		atomic.AddInt64(&metrics.Calls, 1)

	case "execution":
		atomic.AddInt64(&metrics.Executions, 1)
		metrics.LastExecution = time.Now()

		if duration > 0 {
			metrics.TotalLatency += duration
			if duration < metrics.MinLatency {
				metrics.MinLatency = duration
			}
			if duration > metrics.MaxLatency {
				metrics.MaxLatency = duration
			}

			// Calculate average latency
			executions := atomic.LoadInt64(&metrics.Executions)
			if executions > 0 {
				metrics.AverageLatency = time.Duration(int64(metrics.TotalLatency) / executions)
			}
		}

		if !success {
			atomic.AddInt64(&metrics.Errors, 1)
		}

	case "throttle":
		atomic.AddInt64(&metrics.Throttled, 1)
	case "debounce":
		atomic.AddInt64(&metrics.Debounced, 1)
	case "skip":
		atomic.AddInt64(&metrics.Skipped, 1)
	}

	// Calculate rates
	calls := atomic.LoadInt64(&metrics.Calls)
	executions := atomic.LoadInt64(&metrics.Executions)
	errors := atomic.LoadInt64(&metrics.Errors)

	if calls > 0 {
		metrics.ExecutionRatio = float64(executions) / float64(calls)
		metrics.ErrorRate = float64(errors) / float64(calls)
		metrics.SuccessRate = 1.0 - metrics.ErrorRate

		protections := atomic.LoadInt64(&metrics.Throttled) +
			atomic.LoadInt64(&metrics.Debounced) +
			atomic.LoadInt64(&metrics.Skipped)
		metrics.ProtectionRatio = float64(protections) / float64(calls)
	}
}

// updateSystemCounters updates global counters
func (s *Sensor) updateSystemCounters(eventType string, success bool) {
	switch eventType {
	case "call":
		atomic.AddInt64(&s.systemMetrics.TotalCalls, 1)

		// Update call rate sliding window
		s.mu.Lock()
		s.callWindow[s.windowIndex] = atomic.LoadInt64(&s.systemMetrics.TotalCalls)
		s.windowIndex = (s.windowIndex + 1) % s.windowSize
		s.mu.Unlock()

	case "execution":
		atomic.AddInt64(&s.systemMetrics.TotalExecutions, 1)
		if !success {
			atomic.AddInt64(&s.systemMetrics.TotalErrors, 1)
		}

	case "throttle":
		atomic.AddInt64(&s.systemMetrics.TotalThrottled, 1)
	case "debounce":
		atomic.AddInt64(&s.systemMetrics.TotalDebounced, 1)
	case "skip":
		atomic.AddInt64(&s.systemMetrics.TotalSkipped, 1)
	}

	s.systemMetrics.LastActivity = time.Now()
}

// updateSystemMetrics calculates system-wide metrics
func (s *Sensor) updateSystemMetrics() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.systemMetrics.Uptime = now.Sub(s.startTime)

	// Calculate rates
	totalCalls := atomic.LoadInt64(&s.systemMetrics.TotalCalls)
	totalExecutions := atomic.LoadInt64(&s.systemMetrics.TotalExecutions)
	totalErrors := atomic.LoadInt64(&s.systemMetrics.TotalErrors)

	uptimeSeconds := s.systemMetrics.Uptime.Seconds()
	if uptimeSeconds > 0 {
		s.systemMetrics.CallRate = float64(totalCalls) / uptimeSeconds
		s.systemMetrics.ExecutionRate = float64(totalExecutions) / uptimeSeconds
	}

	if totalExecutions > 0 {
		s.systemMetrics.ErrorRate = float64(totalErrors) / float64(totalExecutions)
		s.systemMetrics.SuccessRate = 1.0 - s.systemMetrics.ErrorRate
	}

	// Update memory stats
	s.updateMemoryStats()

	// Calculate health metrics
	s.calculateHealthMetrics()

	s.lastUpdate = now
}

// updateMemoryStats collects current memory statistics
func (s *Sensor) updateMemoryStats() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	s.systemMetrics.MemoryUsage = MemoryStats{
		Alloc:        m.Alloc,
		TotalAlloc:   m.TotalAlloc,
		Sys:          m.Sys,
		NumGC:        m.NumGC,
		HeapAlloc:    m.HeapAlloc,
		HeapInuse:    m.HeapInuse,
		StackInuse:   m.StackInuse,
		UsagePercent: float64(m.HeapInuse) / float64(m.Sys) * 100,
	}

	s.systemMetrics.GoroutineCount = runtime.NumGoroutine()
}

// calculateHealthMetrics determines system health
func (s *Sensor) calculateHealthMetrics() {
	// Calculate stress level based on multiple factors
	errorStress := s.systemMetrics.ErrorRate
	memoryStress := s.systemMetrics.MemoryUsage.UsagePercent / 100.0
	goroutineStress := float64(s.systemMetrics.GoroutineCount) / 1000.0 // Normalize

	// Weighted stress calculation
	s.systemMetrics.StressLevel = (errorStress*0.4 + memoryStress*0.4 + goroutineStress*0.2)

	// Clamp stress level
	if s.systemMetrics.StressLevel > 1.0 {
		s.systemMetrics.StressLevel = 1.0
	}

	// Calculate health score (inverse of stress)
	s.systemMetrics.HealthScore = 1.0 - s.systemMetrics.StressLevel

	// Determine health status
	s.systemMetrics.IsHealthy = s.systemMetrics.StressLevel < s.stressThreshold
	s.systemMetrics.IsBreathing = s.breathingEnabled && s.systemMetrics.StressLevel > s.stressThreshold
}

// === PUBLIC API ===

// GetSystemMetrics returns current system metrics
func (s *Sensor) GetSystemMetrics() SystemMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Update before returning
	s.updateSystemMetrics()

	return s.systemMetrics
}

// GetChannelMetrics returns metrics for a specific action
func (s *Sensor) GetChannelMetrics(actionID string) (*ChannelMetrics, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metrics, exists := s.channelMetrics[actionID]
	if !exists {
		return nil, false
	}

	// Return a copy to avoid race conditions
	copy := *metrics
	return &copy, true
}

// GetAllChannelMetrics returns all channel metrics
func (s *Sensor) GetAllChannelMetrics() map[string]*ChannelMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*ChannelMetrics)
	for k, v := range s.channelMetrics {
		copy := *v
		result[k] = &copy
	}
	return result
}

// GetStressLevel returns current system stress level
func (s *Sensor) GetStressLevel() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.systemMetrics.StressLevel
}

// IsHealthy returns true if system is healthy
func (s *Sensor) IsHealthy() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.systemMetrics.IsHealthy
}

// IsBreathing returns true if breathing system is active
func (s *Sensor) IsBreathing() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.systemMetrics.IsBreathing
}

// === CLEANUP ===

// cleanup performs periodic maintenance
func (s *Sensor) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove old channel metrics for actions that no longer exist
	cutoff := time.Now().Add(-config.MetricsRetentionTime)
	for actionID, metrics := range s.channelMetrics {
		if metrics.LastExecution.Before(cutoff) && metrics.LastExecution != (time.Time{}) {
			delete(s.channelMetrics, actionID)
		}
	}

	// Force garbage collection if memory usage is high
	if s.systemMetrics.MemoryUsage.UsagePercent > config.GCTriggerThreshold*100 {
		runtime.GC()
	}
}
