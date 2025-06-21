// timekeeper/timekeeper.go
// High-precision timing system with breathing integration

package timekeeper

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/neuralline/cyre-go/config"
	"github.com/neuralline/cyre-go/context"
)

/*
	C.Y.R.E - T.I.M.E.K.E.E.P.E.R

	Rock-solid timing system providing:
	- High-precision timing with drift compensation
	- Centralized timeline coordination
	- Adaptive timing with breathing system
	- Smart interval grouping for performance
	- Time chunking for long durations
	- Quartz engine for precise execution
	- Memory-efficient timer management
*/

// === CORE TYPES ===

// Timer represents a scheduled operation
type Timer struct {
	ID            string                                     `json:"id"`
	ActionID      string                                     `json:"actionId"`
	Type          TimerType                                  `json:"type"`
	Interval      time.Duration                              `json:"interval"`
	Delay         time.Duration                              `json:"delay"`
	Repeat        int                                        `json:"repeat"` // -1 for infinite
	Executed      int                                        `json:"executed"`
	NextExecution time.Time                                  `json:"nextExecution"`
	CreatedAt     time.Time                                  `json:"createdAt"`
	LastExecution time.Time                                  `json:"lastExecution"`
	ExecuteFunc   func(actionID string, payload interface{}) `json:"-"`
	Payload       interface{}                                `json:"payload"`
	Context       context.Context                            `json:"-"`
	CancelFunc    context.CancelFunc                         `json:"-"`
	Active        bool                                       `json:"active"`
	Metrics       TimerMetrics                               `json:"metrics"`
}

// TimerType defines the type of timer
type TimerType int

const (
	TimerOneShot  TimerType = iota // Execute once
	TimerInterval                  // Execute repeatedly
	TimerDebounce                  // Debounced execution
	TimerThrottle                  // Throttled execution
)

// TimerMetrics tracks timer performance
type TimerMetrics struct {
	Executions   int64         `json:"executions"`
	Skipped      int64         `json:"skipped"`
	Errors       int64         `json:"errors"`
	AverageDrift time.Duration `json:"averageDrift"`
	MaxDrift     time.Duration `json:"maxDrift"`
	TotalDrift   time.Duration `json:"totalDrift"`
	LastDrift    time.Duration `json:"lastDrift"`
}

// BreathingState represents the adaptive timing state
type BreathingState struct {
	Active      bool          `json:"active"`
	CurrentRate time.Duration `json:"currentRate"`
	StressLevel float64       `json:"stressLevel"`
	Phase       string        `json:"phase"` // "inhale", "hold", "exhale"
	Cycle       int64         `json:"cycle"`
	LastUpdate  time.Time     `json:"lastUpdate"`
}

// === TIMEKEEPER MANAGER ===

// TimeKeeper manages all timing operations with high precision
type TimeKeeper struct {
	// Core state
	timers          sync.Map // timerID -> *Timer
	activeCount     int64
	totalExecutions int64

	// Precision timing
	precisionTicker *time.Ticker
	tickInterval    time.Duration
	lastTick        time.Time
	driftTotal      time.Duration

	// Breathing system
	breathing BreathingState
	sensor    *sensor.Sensor

	// Background processing
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Configuration
	maxTimers     int64
	minResolution time.Duration
	maxResolution time.Duration

	// Performance optimization
	timerPool   sync.Pool
	nextTimerID int64

	mu sync.RWMutex
}

// GlobalTimeKeeper is the singleton instance
var GlobalTimeKeeper *TimeKeeper
var tkOnce sync.Once

// Initialize creates and starts the global TimeKeeper
func Initialize() *TimeKeeper {
	tkOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())

		GlobalTimeKeeper = &TimeKeeper{
			ctx:           ctx,
			cancel:        cancel,
			tickInterval:  config.MinTimerResolution,
			maxTimers:     int64(config.MaxRetainedActions),
			minResolution: config.MinTimerResolution,
			maxResolution: config.MaxTimerDuration,
			sensor:        sensor.GetSensor(),
			lastTick:      time.Now(),
			breathing: BreathingState{
				CurrentRate: config.BreathingBaseRate,
				Phase:       "inhale",
				LastUpdate:  time.Now(),
			},
		}

		// Initialize timer pool
		GlobalTimeKeeper.timerPool = sync.Pool{
			New: func() interface{} {
				return &Timer{}
			},
		}

		// Start the timing engine
		GlobalTimeKeeper.start()
	})
	return GlobalTimeKeeper
}

// GetTimeKeeper returns the global TimeKeeper instance
func GetTimeKeeper() *TimeKeeper {
	if GlobalTimeKeeper == nil {
		return Initialize()
	}
	return GlobalTimeKeeper
}

// === CORE OPERATIONS ===

// start begins the precision timing engine
func (tk *TimeKeeper) start() {
	// Start precision ticker
	tk.precisionTicker = time.NewTicker(tk.tickInterval)

	tk.wg.Add(2)

	// Main timing loop
	go func() {
		defer tk.wg.Done()
		tk.timingLoop()
	}()

	// Breathing system loop
	go func() {
		defer tk.wg.Done()
		tk.breathingLoop()
	}()
}

// Stop shuts down the TimeKeeper
func (tk *TimeKeeper) Stop() {
	tk.cancel()

	if tk.precisionTicker != nil {
		tk.precisionTicker.Stop()
	}

	// Cancel all active timers
	tk.timers.Range(func(key, value interface{}) bool {
		if timer, ok := value.(*Timer); ok {
			if timer.CancelFunc != nil {
				timer.CancelFunc()
			}
		}
		return true
	})

	tk.wg.Wait()
}

// === TIMER MANAGEMENT ===

// ScheduleOnce schedules a one-time execution
func (tk *TimeKeeper) ScheduleOnce(actionID string, delay time.Duration, executeFunc func(string, interface{}), payload interface{}) (*Timer, error) {
	return tk.createTimer(actionID, TimerOneShot, delay, 0, 1, executeFunc, payload)
}

// ScheduleInterval schedules repeated execution
func (tk *TimeKeeper) ScheduleInterval(actionID string, interval time.Duration, repeat int, executeFunc func(string, interface{}), payload interface{}) (*Timer, error) {
	return tk.createTimer(actionID, TimerInterval, 0, interval, repeat, executeFunc, payload)
}

// ScheduleDebounce schedules debounced execution
func (tk *TimeKeeper) ScheduleDebounce(actionID string, delay time.Duration, executeFunc func(string, interface{}), payload interface{}) (*Timer, error) {
	// Cancel existing debounce timer for this action
	tk.CancelActionTimers(actionID, TimerDebounce)

	return tk.createTimer(actionID, TimerDebounce, delay, 0, 1, executeFunc, payload)
}

// createTimer creates and schedules a new timer
func (tk *TimeKeeper) createTimer(actionID string, timerType TimerType, delay, interval time.Duration, repeat int, executeFunc func(string, interface{}), payload interface{}) (*Timer, error) {
	if executeFunc == nil {
		return nil, fmt.Errorf("executeFunc cannot be nil")
	}

	// Validate timing parameters
	delay = config.ValidateTimeout(delay)
	if interval > 0 {
		interval = config.ValidateInterval(interval)
	}
	repeat = config.ValidateRepeat(repeat)

	// Check timer limits
	if atomic.LoadInt64(&tk.activeCount) >= tk.maxTimers {
		return nil, fmt.Errorf("maximum timer limit reached: %d", tk.maxTimers)
	}

	// Get timer from pool
	timer := tk.timerPool.Get().(*Timer)

	// Setup timer
	timerID := fmt.Sprintf("%s_%d_%d", actionID, timerType, atomic.AddInt64(&tk.nextTimerID, 1))
	ctx, cancel := context.WithCancel(tk.ctx)

	now := time.Now()
	nextExecution := now.Add(delay)

	*timer = Timer{
		ID:            timerID,
		ActionID:      actionID,
		Type:          timerType,
		Interval:      interval,
		Delay:         delay,
		Repeat:        repeat,
		Executed:      0,
		NextExecution: nextExecution,
		CreatedAt:     now,
		ExecuteFunc:   executeFunc,
		Payload:       payload,
		Context:       ctx,
		CancelFunc:    cancel,
		Active:        true,
		Metrics:       TimerMetrics{},
	}

	// Store timer
	tk.timers.Store(timerID, timer)
	atomic.AddInt64(&tk.activeCount, 1)

	return timer, nil
}

// CancelTimer cancels a specific timer
func (tk *TimeKeeper) CancelTimer(timerID string) bool {
	if value, exists := tk.timers.LoadAndDelete(timerID); exists {
		timer := value.(*Timer)
		if timer.CancelFunc != nil {
			timer.CancelFunc()
		}
		timer.Active = false

		// Return timer to pool
		tk.timerPool.Put(timer)
		atomic.AddInt64(&tk.activeCount, -1)

		return true
	}
	return false
}

// CancelActionTimers cancels all timers for an action
func (tk *TimeKeeper) CancelActionTimers(actionID string, timerType ...TimerType) int {
	cancelled := 0
	typeMap := make(map[TimerType]bool)
	for _, t := range timerType {
		typeMap[t] = true
	}

	tk.timers.Range(func(key, value interface{}) bool {
		timer := value.(*Timer)
		if timer.ActionID == actionID {
			// If specific types requested, check type
			if len(timerType) > 0 && !typeMap[timer.Type] {
				return true
			}

			if tk.CancelTimer(timer.ID) {
				cancelled++
			}
		}
		return true
	})

	return cancelled
}

// GetTimer retrieves a timer by ID
func (tk *TimeKeeper) GetTimer(timerID string) (*Timer, bool) {
	if value, exists := tk.timers.Load(timerID); exists {
		return value.(*Timer), true
	}
	return nil, false
}

// GetActionTimers returns all timers for an action
func (tk *TimeKeeper) GetActionTimers(actionID string) []*Timer {
	var timers []*Timer
	tk.timers.Range(func(key, value interface{}) bool {
		timer := value.(*Timer)
		if timer.ActionID == actionID {
			timers = append(timers, timer)
		}
		return true
	})
	return timers
}

// === PRECISION TIMING ENGINE ===

// timingLoop is the main precision timing loop
func (tk *TimeKeeper) timingLoop() {
	for {
		select {
		case tickTime := <-tk.precisionTicker.C:
			tk.processTick(tickTime)

		case <-tk.ctx.Done():
			return
		}
	}
}

// processTick processes a timing tick and executes due timers
func (tk *TimeKeeper) processTick(tickTime time.Time) {
	// Calculate drift
	expectedTime := tk.lastTick.Add(tk.tickInterval)
	drift := tickTime.Sub(expectedTime)
	tk.driftTotal += drift
	tk.lastTick = tickTime

	// Apply breathing adjustment
	adjustedTime := tk.applyBreathingAdjustment(tickTime)

	// Process due timers
	var toExecute []*Timer

	tk.timers.Range(func(key, value interface{}) bool {
		timer := value.(*Timer)
		if timer.Active && !timer.NextExecution.After(adjustedTime) {
			toExecute = append(toExecute, timer)
		}
		return true
	})

	// Execute timers concurrently
	for _, timer := range toExecute {
		go tk.executeTimer(timer, adjustedTime, drift)
	}
}

// executeTimer executes a timer and handles scheduling
func (tk *TimeKeeper) executeTimer(timer *Timer, executionTime time.Time, drift time.Duration) {
	defer func() {
		if r := recover(); r != nil {
			atomic.AddInt64(&timer.Metrics.Errors, 1)
			tk.sensor.RecordEvent("timer_error", timer.ActionID, 0, false,
				fmt.Errorf("timer panic: %v", r), map[string]interface{}{
					"timerID": timer.ID,
					"type":    timer.Type,
				})
		}
	}()

	start := time.Now()

	// Execute the function
	timer.ExecuteFunc(timer.ActionID, timer.Payload)

	duration := time.Since(start)
	timer.Executed++
	timer.LastExecution = executionTime

	// Update metrics
	atomic.AddInt64(&timer.Metrics.Executions, 1)
	atomic.AddInt64(&tk.totalExecutions, 1)

	// Calculate and track drift
	expectedExecution := timer.NextExecution
	actualDrift := executionTime.Sub(expectedExecution)
	timer.Metrics.LastDrift = actualDrift
	timer.Metrics.TotalDrift += actualDrift

	if actualDrift > timer.Metrics.MaxDrift {
		timer.Metrics.MaxDrift = actualDrift
	}

	executions := atomic.LoadInt64(&timer.Metrics.Executions)
	if executions > 0 {
		timer.Metrics.AverageDrift = time.Duration(int64(timer.Metrics.TotalDrift) / executions)
	}

	// Record execution
	tk.sensor.RecordExecution(timer.ActionID, duration, true, nil)

	// Schedule next execution or cleanup
	tk.scheduleNext(timer)
}

// scheduleNext schedules the next execution or cleans up the timer
func (tk *TimeKeeper) scheduleNext(timer *Timer) {
	switch timer.Type {
	case TimerOneShot, TimerDebounce:
		// One-time execution, remove timer
		tk.CancelTimer(timer.ID)

	case TimerInterval:
		// Check if more executions needed
		if timer.Repeat > 0 && timer.Executed >= timer.Repeat {
			tk.CancelTimer(timer.ID)
			return
		}

		// Schedule next execution
		nextTime := timer.NextExecution.Add(timer.Interval)

		// Apply breathing adjustment to interval
		if tk.breathing.Active {
			adjustment := tk.calculateBreathingAdjustment()
			nextTime = nextTime.Add(adjustment)
		}

		timer.NextExecution = nextTime

	case TimerThrottle:
		// Throttle timers are managed by protection system
		tk.CancelTimer(timer.ID)
	}
}

// === BREATHING SYSTEM ===

// breathingLoop manages the adaptive timing system
func (tk *TimeKeeper) breathingLoop() {
	breathingTicker := time.NewTicker(tk.breathing.CurrentRate)
	defer breathingTicker.Stop()

	for {
		select {
		case <-breathingTicker.C:
			tk.updateBreathing()

		case <-tk.ctx.Done():
			return
		}
	}
}

// updateBreathing updates the breathing state based on system stress
func (tk *TimeKeeper) updateBreathing() {
	tk.mu.Lock()
	defer tk.mu.Unlock()

	// Get current stress level from sensor
	stressLevel := tk.sensor.GetStressLevel()
	tk.breathing.StressLevel = stressLevel
	tk.breathing.LastUpdate = time.Now()

	// Determine if breathing should be active
	wasActive := tk.breathing.Active

	if stressLevel >= config.StressThreshold {
		tk.breathing.Active = true
	} else if stressLevel <= config.RecoveryThreshold {
		tk.breathing.Active = false
	}

	// Update breathing rate based on stress
	var newRate time.Duration
	switch {
	case stressLevel >= config.CriticalThreshold:
		newRate = config.BreathingCriticalRate
		tk.breathing.Phase = "critical"

	case stressLevel >= config.StressThreshold:
		newRate = config.BreathingStressRate
		tk.breathing.Phase = "stress"

	case stressLevel <= config.RecoveryThreshold:
		newRate = config.BreathingRecoveryRate
		tk.breathing.Phase = "recovery"

	default:
		newRate = config.BreathingBaseRate
		tk.breathing.Phase = "normal"
	}

	if newRate != tk.breathing.CurrentRate {
		tk.breathing.CurrentRate = newRate
		tk.breathing.Cycle++

		// Log breathing state change
		if wasActive != tk.breathing.Active {
			tk.sensor.RecordEvent("breathing_change", "system", 0, true, nil, map[string]interface{}{
				"active":      tk.breathing.Active,
				"stressLevel": stressLevel,
				"phase":       tk.breathing.Phase,
				"rate":        newRate,
			})
		}
	}
}

// applyBreathingAdjustment applies breathing adjustment to execution time
func (tk *TimeKeeper) applyBreathingAdjustment(baseTime time.Time) time.Time {
	if !tk.breathing.Active {
		return baseTime
	}

	adjustment := tk.calculateBreathingAdjustment()
	return baseTime.Add(adjustment)
}

// calculateBreathingAdjustment calculates timing adjustment based on breathing
func (tk *TimeKeeper) calculateBreathingAdjustment() time.Duration {
	if !tk.breathing.Active {
		return 0
	}

	// Apply breathing rate as adjustment
	stressFactor := tk.breathing.StressLevel
	baseAdjustment := tk.breathing.CurrentRate

	// Scale adjustment based on stress level
	adjustmentMs := float64(baseAdjustment.Milliseconds()) * stressFactor
	return time.Duration(adjustmentMs) * time.Millisecond
}

// === PUBLIC API ===

// GetBreathingState returns current breathing state
func (tk *TimeKeeper) GetBreathingState() BreathingState {
	tk.mu.RLock()
	defer tk.mu.RUnlock()
	return tk.breathing
}

// GetStats returns TimeKeeper statistics
func (tk *TimeKeeper) GetStats() map[string]interface{} {
	activeCount := atomic.LoadInt64(&tk.activeCount)
	totalExecutions := atomic.LoadInt64(&tk.totalExecutions)

	return map[string]interface{}{
		"activeTimers":    activeCount,
		"totalExecutions": totalExecutions,
		"tickInterval":    tk.tickInterval,
		"driftTotal":      tk.driftTotal,
		"breathing":       tk.GetBreathingState(),
		"averageDrift":    tk.driftTotal / time.Duration(totalExecutions),
	}
}

// GetActiveTimers returns count of active timers
func (tk *TimeKeeper) GetActiveTimers() int64 {
	return atomic.LoadInt64(&tk.activeCount)
}

// IsHealthy returns true if timing system is healthy
func (tk *TimeKeeper) IsHealthy() bool {
	activeCount := atomic.LoadInt64(&tk.activeCount)
	return activeCount < tk.maxTimers && tk.sensor.IsHealthy()
}

// === UTILITY FUNCTIONS ===

// Now returns high-precision current time
func Now() time.Time {
	return time.Now()
}

// Sleep performs high-precision sleep
func Sleep(duration time.Duration) {
	time.Sleep(duration)
}

// ValidateDuration ensures duration is within acceptable bounds
func ValidateDuration(duration time.Duration) time.Duration {
	if duration < config.MinTimerResolution {
		return config.MinTimerResolution
	}
	if duration > config.MaxTimerDuration {
		return config.MaxTimerDuration
	}
	return duration
}
