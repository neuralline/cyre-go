// timekeeper/timekeeper.go
// Clean TimeKeeper with proper context and state package usage

package timekeeper

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/neuralline/cyre-go/state"
)

/*
	C.Y.R.E - T.I.M.E.K.E.E.P.E.R

	Simplified timing system providing:
	- Centralized quartz timing engine
	- Direct state integration (no StateManager references)
	- High-precision execution coordination
	- Clean API: keep, wait, forget, clear, hibernate, activate
	- Drift compensation for accuracy
	- Smart execution grouping for performance
*/

// === CORE TYPES ===

// TimerRepeat represents repeat configuration
type TimerRepeat interface{}

// ExecuteFunc represents a timer execution function
type ExecuteFunc func()

// QuartzEngine is the centralized timing coordinator
type QuartzEngine struct {
	// Core timing
	ticker       *time.Ticker
	tickInterval time.Duration
	isRunning    int32 // atomic
	lastTick     time.Time

	// Drift compensation
	driftTotal time.Duration

	// Execution coordination
	executionGroups map[int64][]string // Grouped by interval for efficiency

	// Background processing
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex
}

// TimeKeeper manages timing operations with centralized quartz engine
type TimeKeeper struct {
	quartz      *QuartzEngine
	hibernating int32 // atomic
	nextTimerID int64 // atomic
	// NO stateManager field - use direct state access
}

// GlobalTimeKeeper is the singleton instance
var GlobalTimeKeeper *TimeKeeper
var tkOnce sync.Once

// Init creates and starts the global TimeKeeper
func Init() *TimeKeeper {
	tkOnce.Do(func() {
		GlobalTimeKeeper = &TimeKeeper{}
		GlobalTimeKeeper.initQuartz()
	})
	return GlobalTimeKeeper
}

// GetTimeKeeper returns the global TimeKeeper instance
func GetTimeKeeper() *TimeKeeper {
	if GlobalTimeKeeper == nil {
		return Init()
	}
	return GlobalTimeKeeper
}

// === QUARTZ ENGINE IMPLEMENTATION ===

// initQuartz initializes the centralized quartz timing engine
func (tk *TimeKeeper) initQuartz() {
	ctx, cancel := context.WithCancel(context.Background())

	tk.quartz = &QuartzEngine{
		tickInterval:    10 * time.Millisecond, // 10ms precision like TypeScript
		executionGroups: make(map[int64][]string),
		ctx:             ctx,
		cancel:          cancel,
		lastTick:        time.Now(),
	}
}

// startQuartz starts the centralized timing engine
func (tk *TimeKeeper) startQuartz() {
	if !atomic.CompareAndSwapInt32(&tk.quartz.isRunning, 0, 1) {
		return // Already running
	}

	tk.quartz.ticker = time.NewTicker(tk.quartz.tickInterval)

	tk.quartz.wg.Add(1)
	go func() {
		defer tk.quartz.wg.Done()
		tk.quartzLoop()
	}()
}

// stopQuartz stops the centralized timing engine
func (tk *TimeKeeper) stopQuartz() {
	if !atomic.CompareAndSwapInt32(&tk.quartz.isRunning, 1, 0) {
		return // Already stopped
	}

	if tk.quartz.ticker != nil {
		tk.quartz.ticker.Stop()
	}

	tk.quartz.cancel()
	tk.quartz.wg.Wait()
}

// quartzLoop is the main timing loop (centralized like TypeScript version)
func (tk *TimeKeeper) quartzLoop() {
	for {
		select {
		case tickTime := <-tk.quartz.ticker.C:
			if atomic.LoadInt32(&tk.hibernating) == 1 {
				continue // Skip processing during hibernation
			}
			tk.processTick(tickTime)

		case <-tk.quartz.ctx.Done():
			return
		}
	}
}

// processTick processes a timing tick and executes due timers
func (tk *TimeKeeper) processTick(tickTime time.Time) {
	// Calculate drift for compensation
	expectedTime := tk.quartz.lastTick.Add(tk.quartz.tickInterval)
	drift := tickTime.Sub(expectedTime)
	tk.quartz.driftTotal += drift
	tk.quartz.lastTick = tickTime

	currentTime := time.Now()

	// Get active timers using direct Timeline store access
	allTimers := state.Timeline().GetAll()
	var activeTimers []*state.Timer
	for _, timer := range allTimers {
		if timer.Status == "active" {
			activeTimers = append(activeTimers, timer)
		}
	}

	if len(activeTimers) == 0 {
		tk.stopQuartz() // No active timers, stop quartz
		return
	}

	// Process due timers
	var toExecute []*state.Timer
	for _, timer := range activeTimers {
		if timer.Status == "active" && currentTime.After(timer.NextExecution) {
			toExecute = append(toExecute, timer)
		}
	}

	// Execute timers concurrently
	for _, timer := range toExecute {
		go tk.executeTimer(timer, currentTime)
	}
}

// executeTimer executes a timer and handles scheduling
func (tk *TimeKeeper) executeTimer(timer *state.Timer, executionTime time.Time) {
	defer func() {
		if r := recover(); r != nil {
			// Handle panic gracefully - simple logging
			fmt.Printf("Timer execution panic for %s: %v\n", timer.ID, r)
		}
	}()

	// Execute the function
	if timer.ExecuteFunc != nil {
		timer.ExecuteFunc(timer.ActionID, timer.Payload)
	}

	// Update execution count
	timer.Executed++
	timer.LastExecution = executionTime

	// Determine if timer should continue
	shouldContinue := tk.shouldContinueTimer(timer)

	if shouldContinue {
		// Schedule next execution
		tk.scheduleNext(timer, executionTime)
	} else {
		// Timer completed, remove from timeline using direct store access
		timer.Status = "completed"
		state.Timeline().Forget(timer.ID)
		tk.removeFromExecutionGroups(timer)
	}
}

// shouldContinueTimer determines if a timer should continue executing
func (tk *TimeKeeper) shouldContinueTimer(timer *state.Timer) bool {
	if timer.Repeat == -1 {
		return true // Infinite repeat
	}
	if timer.Repeat > 0 {
		return timer.Executed < timer.Repeat
	}
	return false // Single execution
}

// scheduleNext schedules the next execution of a timer
func (tk *TimeKeeper) scheduleNext(timer *state.Timer, currentTime time.Time) {
	// Calculate next execution time
	nextTime := currentTime.Add(timer.Interval)
	timer.NextExecution = nextTime

	// Update timer using direct store access
	state.Timeline().Set(timer.ID, timer)

	// Update execution groups
	tk.addToExecutionGroups(timer)
}

// addToExecutionGroups adds timer to execution groups for efficiency
func (tk *TimeKeeper) addToExecutionGroups(timer *state.Timer) {
	tk.quartz.mu.Lock()
	defer tk.quartz.mu.Unlock()

	// Group by interval (rounded to nearest 10ms for efficiency)
	intervalKey := timer.Interval.Milliseconds() / 10 * 10
	tk.quartz.executionGroups[intervalKey] = append(tk.quartz.executionGroups[intervalKey], timer.ID)
}

// removeFromExecutionGroups removes timer from execution groups
func (tk *TimeKeeper) removeFromExecutionGroups(timer *state.Timer) {
	tk.quartz.mu.Lock()
	defer tk.quartz.mu.Unlock()

	intervalKey := timer.Interval.Milliseconds() / 10 * 10
	group := tk.quartz.executionGroups[intervalKey]

	for i, id := range group {
		if id == timer.ID {
			// Remove from slice
			tk.quartz.executionGroups[intervalKey] = append(group[:i], group[i+1:]...)
			break
		}
	}

	// Clean up empty groups
	if len(tk.quartz.executionGroups[intervalKey]) == 0 {
		delete(tk.quartz.executionGroups, intervalKey)
	}
}

// === PUBLIC API ===

// Keep schedules a timer with centralized quartz execution
func (tk *TimeKeeper) Keep(interval time.Duration, callback func(), repeat TimerRepeat, id string, delay ...time.Duration) error {
	if callback == nil {
		return fmt.Errorf("callback cannot be nil")
	}

	if id == "" {
		id = fmt.Sprintf("timer_%d", atomic.AddInt64(&tk.nextTimerID, 1))
	}

	// Remove existing timer with same ID
	tk.Forget(id)

	// Determine delay
	var startDelay time.Duration
	if len(delay) > 0 {
		startDelay = delay[0]
	}

	// Convert repeat to internal format
	var repeatCount int
	switch r := repeat.(type) {
	case bool:
		if r {
			repeatCount = -1 // Infinite
		} else {
			repeatCount = 1 // Single execution
		}
	case int:
		repeatCount = r
	default:
		repeatCount = 1
	}

	// Create timer for timeline store
	now := time.Now()
	nextExecution := now.Add(startDelay)
	if startDelay == 0 {
		nextExecution = now.Add(interval)
	}

	timer := &state.Timer{
		ID:            id,
		ActionID:      id,
		Type:          "timer",
		Interval:      interval,
		Delay:         startDelay,
		Repeat:        repeatCount,
		Executed:      0,
		Status:        "active",
		NextExecution: nextExecution,
		CreatedAt:     now,
		ExecuteFunc: func(actionID string, payload interface{}) {
			callback()
		},
		Payload: nil,
	}

	// Add to timeline using direct store access
	state.Timeline().Set(timer.ID, timer)

	// Add to execution groups
	tk.addToExecutionGroups(timer)

	// Start quartz if not running
	if atomic.LoadInt32(&tk.quartz.isRunning) == 0 {
		tk.startQuartz()
	}

	return nil
}

// Wait creates a delayed execution (like setTimeout)
func (tk *TimeKeeper) Wait(duration time.Duration, callback func()) error {
	id := fmt.Sprintf("wait_%d", atomic.AddInt64(&tk.nextTimerID, 1))
	return tk.Keep(duration, func() {
		callback()
		tk.Forget(id) // Auto-cleanup
	}, 1, id, duration) // Execute once after delay
}

// Forget removes a timer by ID
func (tk *TimeKeeper) Forget(id string) bool {
	// Get timer from direct store access
	timer, exists := state.Timeline().Get(id)
	if !exists {
		return false
	}

	// Remove from execution groups
	tk.removeFromExecutionGroups(timer)

	// Remove using direct store access
	success := state.Timeline().Forget(id)

	// Stop quartz if no active timers
	allTimers := state.Timeline().GetAll()
	activeCount := 0
	for _, t := range allTimers {
		if t.Status == "active" {
			activeCount++
		}
	}
	if activeCount == 0 {
		tk.stopQuartz()
	}

	return success
}

// Clear removes all timers (forget all timeline tasks)
func (tk *TimeKeeper) Clear() {
	// Stop quartz
	tk.stopQuartz()

	// Clear execution groups
	tk.quartz.mu.Lock()
	tk.quartz.executionGroups = make(map[int64][]string)
	tk.quartz.mu.Unlock()

	// Clear timeline using direct store access
	state.Timeline().Clear()
}

// Hibernate pauses all timer processing
func (tk *TimeKeeper) Hibernate() {
	atomic.StoreInt32(&tk.hibernating, 1)
	tk.stopQuartz()
}

// Activate controls timer activation state
func (tk *TimeKeeper) Activate(id string, active bool) bool {
	timer, exists := state.Timeline().Get(id)
	if !exists {
		return false
	}

	if active {
		timer.Status = "active"
		tk.addToExecutionGroups(timer)

		// Wake up from hibernation if needed
		if atomic.LoadInt32(&tk.hibernating) == 1 {
			atomic.StoreInt32(&tk.hibernating, 0)
		}

		// Start quartz if not running
		if atomic.LoadInt32(&tk.quartz.isRunning) == 0 {
			tk.startQuartz()
		}
	} else {
		timer.Status = "paused"
		tk.removeFromExecutionGroups(timer)
	}

	// Update timer using direct store access
	state.Timeline().Set(timer.ID, timer)

	return true
}

// === STATUS AND UTILITY ===

// GetStats returns simplified timing statistics
func (tk *TimeKeeper) GetStats() map[string]interface{} {
	allTimers := state.Timeline().GetAll()
	activeCount := 0
	for _, timer := range allTimers {
		if timer.Status == "active" {
			activeCount++
		}
	}

	tk.quartz.mu.RLock()
	groupCount := len(tk.quartz.executionGroups)
	tk.quartz.mu.RUnlock()

	return map[string]interface{}{
		"activeTimers":    activeCount,
		"totalTimers":     len(allTimers),
		"quartzRunning":   atomic.LoadInt32(&tk.quartz.isRunning) == 1,
		"hibernating":     atomic.LoadInt32(&tk.hibernating) == 1,
		"executionGroups": groupCount,
		"driftTotal":      tk.quartz.driftTotal,
		"tickInterval":    tk.quartz.tickInterval,
	}
}
