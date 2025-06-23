// examples/protection.go
// Complete test suite to verify all protection mechanisms and timekeeper functionality

package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/neuralline/cyre-go"
)

// Test data structures
type TestResult struct {
	Name        string
	Expected    string
	Actual      string
	Success     bool
	Duration    time.Duration
	Description string
}

type TestSuite struct {
	results []TestResult
	mu      sync.Mutex
}

func (ts *TestSuite) AddResult(result TestResult) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.results = append(ts.results, result)
}

func (ts *TestSuite) PrintResults() {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	fmt.Println("\n🧪 === CYRE GO PROTECTION & TIMEKEEPER TEST RESULTS ===")
	fmt.Println("========================================================")

	passed := 0
	failed := 0

	for _, result := range ts.results {
		status := "✅ PASS"
		if !result.Success {
			status = "❌ FAIL"
			failed++
		} else {
			passed++
		}

		fmt.Printf("%s | %-30s | %s\n", status, result.Name, result.Description)
		if !result.Success {
			fmt.Printf("     Expected: %s\n", result.Expected)
			fmt.Printf("     Actual:   %s\n", result.Actual)
		}
		if result.Duration > 0 {
			fmt.Printf("     Duration: %v\n", result.Duration)
		}
	}

	fmt.Println("========================================================")
	fmt.Printf("📊 Summary: %d passed, %d failed, %d total\n", passed, failed, len(ts.results))

	if failed == 0 {
		fmt.Println("🎉 ALL TESTS PASSED! Protection & TimeKeeper working correctly!")
	} else {
		fmt.Printf("⚠️  %d tests failed - protection mechanisms need fixing\n", failed)
	}
}

func main() {
	fmt.Println("🔧 CYRE GO PROTECTION & TIMEKEEPER VERIFICATION")
	fmt.Println("===============================================")

	// Initialize Cyre
	result := cyre.Init()
	if !result.OK {
		fmt.Printf("❌ Failed to initialize Cyre: %s\n", result.Message)
		return
	}

	testSuite := &TestSuite{}

	// Run all tests
	testThrottleProtection(testSuite)
	testDebounceProtection(testSuite)
	testChangeDetection(testSuite)
	testDelayExecution(testSuite)
	testIntervalExecution(testSuite)
	testRepeatExecution(testSuite)
	testCombinedProtections(testSuite)
	testTimeKeeperPrecision(testSuite)

	// Wait a bit for any remaining async operations
	time.Sleep(100 * time.Millisecond)

	// Print final results
	testSuite.PrintResults()
}

// === THROTTLE PROTECTION TESTS ===

func testThrottleProtection(ts *TestSuite) {
	fmt.Println("\n🚦 Testing Throttle Protection...")

	// Setup throttled action (1 second throttle)
	err := cyre.Action(cyre.ActionConfig{
		ID:       "throttle-test",
		Throttle: 1000,
	})

	if err != nil {
		ts.AddResult(TestResult{
			Name:        "Throttle Setup",
			Expected:    "No error",
			Actual:      err.Error(),
			Success:     false,
			Description: "Should register throttled action",
		})
		return
	}

	executionCount := 0
	cyre.On("throttle-test", func(payload interface{}) interface{} {
		executionCount++
		fmt.Printf("  🔥 Throttled action executed #%d at %s\n", executionCount, time.Now().Format("15:04:05.000"))
		return map[string]interface{}{"executed": executionCount}
	})

	start := time.Now()

	// Test 1: First call should succeed
	result1 := <-cyre.Call("throttle-test", "call1")
	ts.AddResult(TestResult{
		Name:        "Throttle First Call",
		Expected:    "Success",
		Actual:      fmt.Sprintf("OK: %t", result1.OK),
		Success:     result1.OK,
		Description: "First call should not be throttled",
	})

	// Test 2: Immediate second call should be throttled
	result2 := <-cyre.Call("throttle-test", "call2")
	ts.AddResult(TestResult{
		Name:        "Throttle Immediate Call",
		Expected:    "Blocked/Failed",
		Actual:      fmt.Sprintf("OK: %t, Message: %s", result2.OK, result2.Message),
		Success:     !result2.OK, // Should fail due to throttling
		Description: "Immediate second call should be throttled",
	})

	// Test 3: Wait for throttle to reset, then call should succeed
	fmt.Println("  ⏳ Waiting for throttle to reset...")
	time.Sleep(1100 * time.Millisecond) // Wait longer than throttle duration

	result3 := <-cyre.Call("throttle-test", "call3")
	ts.AddResult(TestResult{
		Name:        "Throttle After Reset",
		Expected:    "Success",
		Actual:      fmt.Sprintf("OK: %t", result3.OK),
		Success:     result3.OK,
		Duration:    time.Since(start),
		Description: "Call after throttle reset should succeed",
	})

	cyre.Forget("throttle-test")
}

// === DEBOUNCE PROTECTION TESTS ===

func testDebounceProtection(ts *TestSuite) {
	fmt.Println("\n🔄 Testing Debounce Protection...")

	// Setup debounced action (300ms debounce)
	err := cyre.Action(cyre.ActionConfig{
		ID:       "debounce-test",
		Debounce: 3000,
	})

	if err != nil {
		ts.AddResult(TestResult{
			Name:        "Debounce Setup",
			Expected:    "No error",
			Actual:      err.Error(),
			Success:     false,
			Description: "Should register debounced action",
		})
		return
	}

	executionCount := 0
	var lastPayload interface{}
	resultChan := make(chan bool, 1)

	cyre.On("debounce-test", func(payload interface{}) interface{} {
		executionCount++
		lastPayload = payload
		fmt.Printf("  🔄 Debounced action executed #%d with payload: %v at %s\n",
			executionCount, payload, time.Now().Format("15:04:05.000"))

		// Signal that execution happened
		select {
		case resultChan <- true:
		default:
		}

		return map[string]interface{}{"executed": executionCount, "payload": payload}
	})

	start := time.Now()

	// Test 1: Rapid calls should be debounced to last one
	fmt.Println("  📞 Making rapid calls...")
	cyre.Call("debounce-test", "rapid1")
	time.Sleep(50 * time.Millisecond)
	cyre.Call("debounce-test", "rapid2")
	time.Sleep(50 * time.Millisecond)
	cyre.Call("debounce-test", "rapid3")
	time.Sleep(50 * time.Millisecond)
	cyre.Call("debounce-test", "final")

	// Wait for debounce to execute
	select {
	case <-resultChan:
		// Execution happened
	case <-time.After(500 * time.Millisecond):
		// Timeout
	}

	ts.AddResult(TestResult{
		Name:        "Debounce Execution Count",
		Expected:    "1",
		Actual:      fmt.Sprintf("%d", executionCount),
		Success:     executionCount == 1,
		Description: "Rapid calls should be debounced to single execution",
	})

	ts.AddResult(TestResult{
		Name:        "Debounce Last Payload",
		Expected:    "final",
		Actual:      fmt.Sprintf("%v", lastPayload),
		Success:     lastPayload == "final",
		Duration:    time.Since(start),
		Description: "Should execute with last payload (final)",
	})

	cyre.Forget("debounce-test")
}

// === CHANGE DETECTION TESTS ===

func testChangeDetection(ts *TestSuite) {
	fmt.Println("\n🔍 Testing Change Detection...")

	// Setup action with change detection
	err := cyre.Action(cyre.ActionConfig{
		ID:            "change-test",
		DetectChanges: true,
	})

	if err != nil {
		ts.AddResult(TestResult{
			Name:        "Change Detection Setup",
			Expected:    "No error",
			Actual:      err.Error(),
			Success:     false,
			Description: "Should register change detection action",
		})
		return
	}

	executionCount := 0
	cyre.On("change-test", func(payload interface{}) interface{} {
		executionCount++
		fmt.Printf("  🔍 Change detection action executed #%d with: %v\n", executionCount, payload)
		return map[string]interface{}{"executed": executionCount}
	})

	// Test 1: First call should execute
	samePayload := map[string]interface{}{"value": 42, "name": "test"}
	result1 := <-cyre.Call("change-test", samePayload)
	ts.AddResult(TestResult{
		Name:        "Change Detection First Call",
		Expected:    "Success",
		Actual:      fmt.Sprintf("OK: %t", result1.OK),
		Success:     result1.OK,
		Description: "First call should execute",
	})

	// Test 2: Same payload should be blocked
	result2 := <-cyre.Call("change-test", samePayload)
	ts.AddResult(TestResult{
		Name:        "Change Detection Duplicate",
		Expected:    "Blocked",
		Actual:      fmt.Sprintf("OK: %t, Message: %s", result2.OK, result2.Message),
		Success:     !result2.OK, // Should be blocked
		Description: "Duplicate payload should be blocked",
	})

	// Test 3: Different payload should execute
	differentPayload := map[string]interface{}{"value": 100, "name": "different"}
	result3 := <-cyre.Call("change-test", differentPayload)
	ts.AddResult(TestResult{
		Name:        "Change Detection Different",
		Expected:    "Success",
		Actual:      fmt.Sprintf("OK: %t", result3.OK),
		Success:     result3.OK,
		Description: "Different payload should execute",
	})

	ts.AddResult(TestResult{
		Name:        "Change Detection Execution Count",
		Expected:    "2",
		Actual:      fmt.Sprintf("%d", executionCount),
		Success:     executionCount == 2,
		Description: "Should execute twice (first + different)",
	})

	cyre.Forget("change-test")
}

// === DELAY EXECUTION TESTS ===

func testDelayExecution(ts *TestSuite) {
	fmt.Println("\n⏰ Testing Delay Execution...")

	// Setup delayed action (500ms delay)
	err := cyre.Action(cyre.ActionConfig{
		ID:    "delay-test",
		Delay: 500,
	})

	if err != nil {
		ts.AddResult(TestResult{
			Name:        "Delay Setup",
			Expected:    "No error",
			Actual:      err.Error(),
			Success:     false,
			Description: "Should register delayed action",
		})
		return
	}

	executed := false
	var executionTime time.Time

	cyre.On("delay-test", func(payload interface{}) interface{} {
		executed = true
		executionTime = time.Now()
		fmt.Printf("  ⏰ Delayed action executed at %s\n", executionTime.Format("15:04:05.000"))
		return map[string]interface{}{"delayed": true}
	})

	start := time.Now()
	result := <-cyre.Call("delay-test", "delayed-payload")

	elapsed := time.Since(start)

	ts.AddResult(TestResult{
		Name:        "Delay Call Response",
		Expected:    "Success",
		Actual:      fmt.Sprintf("OK: %t", result.OK),
		Success:     result.OK,
		Description: "Delay call should return success immediately",
	})

	// Wait for actual execution
	time.Sleep(600 * time.Millisecond)

	ts.AddResult(TestResult{
		Name:        "Delay Execution",
		Expected:    "true",
		Actual:      fmt.Sprintf("%t", executed),
		Success:     executed,
		Description: "Action should execute after delay",
	})

	actualDelay := executionTime.Sub(start)
	delayOK := actualDelay >= 500 && actualDelay < 500+100*time.Millisecond
	ts.AddResult(TestResult{
		Name:        "Delay Timing",
		Expected:    fmt.Sprintf("~%v", 500),
		Actual:      fmt.Sprintf("%v", actualDelay),
		Success:     delayOK,
		Duration:    elapsed,
		Description: "Should execute after specified delay",
	})

	cyre.Forget("delay-test")
}

// === INTERVAL EXECUTION TESTS ===

func testIntervalExecution(ts *TestSuite) {
	fmt.Println("\n🔁 Testing Interval Execution...")

	// Setup interval action (200ms interval, 3 times)
	intervalDuration := 200 * time.Millisecond
	repeatCount := 3
	err := cyre.Action(cyre.ActionConfig{
		ID:       "interval-test",
		Interval: 200,
		Repeat:   3,
	})

	if err != nil {
		ts.AddResult(TestResult{
			Name:        "Interval Setup",
			Expected:    "No error",
			Actual:      err.Error(),
			Success:     false,
			Description: "Should register interval action",
		})
		return
	}

	executionCount := 0
	var executionTimes []time.Time

	cyre.On("interval-test", func(payload interface{}) interface{} {
		executionCount++
		execTime := time.Now()
		executionTimes = append(executionTimes, execTime)
		fmt.Printf("  🔁 Interval action executed #%d at %s\n", executionCount, execTime.Format("15:04:05.000"))
		return map[string]interface{}{"execution": executionCount}
	})

	start := time.Now()
	result := <-cyre.Call("interval-test", "interval-payload")

	ts.AddResult(TestResult{
		Name:        "Interval Call Response",
		Expected:    "Success",
		Actual:      fmt.Sprintf("OK: %t", result.OK),
		Success:     result.OK,
		Description: "Interval call should return success",
	})

	// Wait for all executions to complete
	time.Sleep(time.Duration(repeatCount+1)*intervalDuration + 100*time.Millisecond)

	ts.AddResult(TestResult{
		Name:        "Interval Execution Count",
		Expected:    fmt.Sprintf("%d", repeatCount),
		Actual:      fmt.Sprintf("%d", executionCount),
		Success:     executionCount == repeatCount,
		Description: "Should execute specified number of times",
	})

	// Check timing between executions
	if len(executionTimes) >= 2 {
		interval := executionTimes[1].Sub(executionTimes[0])
		intervalOK := interval >= intervalDuration && interval < intervalDuration+50*time.Millisecond
		ts.AddResult(TestResult{
			Name:        "Interval Timing",
			Expected:    fmt.Sprintf("~%v", intervalDuration),
			Actual:      fmt.Sprintf("%v", interval),
			Success:     intervalOK,
			Duration:    time.Since(start),
			Description: "Executions should be spaced by interval",
		})
	}

	cyre.Forget("interval-test")
}

// === REPEAT EXECUTION TESTS ===

func testRepeatExecution(ts *TestSuite) {
	fmt.Println("\n🔂 Testing Repeat Execution...")

	// Setup repeat action (immediate, 4 times)
	repeatCount := 4
	err := cyre.Action(cyre.ActionConfig{
		ID:     "repeat-test",
		Repeat: 4,
	})

	if err != nil {
		ts.AddResult(TestResult{
			Name:        "Repeat Setup",
			Expected:    "No error",
			Actual:      err.Error(),
			Success:     false,
			Description: "Should register repeat action",
		})
		return
	}

	executionCount := 0
	cyre.On("repeat-test", func(payload interface{}) interface{} {
		executionCount++
		fmt.Printf("  🔂 Repeat action executed #%d\n", executionCount)
		return map[string]interface{}{"execution": executionCount}
	})

	start := time.Now()
	result := <-cyre.Call("repeat-test", "repeat-payload")

	// Wait a bit for repeats to complete
	time.Sleep(100 * time.Millisecond)

	ts.AddResult(TestResult{
		Name:        "Repeat Call Response",
		Expected:    "Success",
		Actual:      fmt.Sprintf("OK: %t", result.OK),
		Success:     result.OK,
		Description: "Repeat call should return success",
	})

	ts.AddResult(TestResult{
		Name:        "Repeat Execution Count",
		Expected:    fmt.Sprintf("%d", repeatCount),
		Actual:      fmt.Sprintf("%d", executionCount),
		Success:     executionCount == repeatCount,
		Duration:    time.Since(start),
		Description: "Should execute specified number of times",
	})

	cyre.Forget("repeat-test")
}

// === COMBINED PROTECTIONS TEST ===

func testCombinedProtections(ts *TestSuite) {
	fmt.Println("\n🛡️ Testing Combined Protections...")

	// Setup action with multiple protections

	err := cyre.Action(cyre.ActionConfig{
		ID:            "combined-test",
		Throttle:      5000,
		Debounce:      2000,
		DetectChanges: true,
	})

	if err != nil {
		ts.AddResult(TestResult{
			Name:        "Combined Setup",
			Expected:    "No error",
			Actual:      err.Error(),
			Success:     false,
			Description: "Should register action with combined protections",
		})
		return
	}

	executionCount := 0
	cyre.On("combined-test", func(payload interface{}) interface{} {
		executionCount++
		fmt.Printf("  🛡️ Combined protection action executed #%d\n", executionCount)
		return map[string]interface{}{"protected": true}
	})

	// Test combined protections work together
	start := time.Now()
	result := <-cyre.Call("combined-test", map[string]interface{}{"test": "combined"})

	ts.AddResult(TestResult{
		Name:        "Combined Protections",
		Expected:    "Success",
		Actual:      fmt.Sprintf("OK: %t", result.OK),
		Success:     result.OK,
		Duration:    time.Since(start),
		Description: "Combined protections should work together",
	})

	cyre.Forget("combined-test")
}

// === TIMEKEEPER PRECISION TESTS ===

func testTimeKeeperPrecision(ts *TestSuite) {
	fmt.Println("\n⏱️ Testing TimeKeeper Precision...")

	// Test precise timing with short intervals
	intervalDuration := 100 * time.Millisecond
	repeatCount := 5
	err := cyre.Action(cyre.ActionConfig{
		ID:       "precision-test",
		Interval: 500,
		Repeat:   2000,
	})

	if err != nil {
		ts.AddResult(TestResult{
			Name:        "Precision Setup",
			Expected:    "No error",
			Actual:      err.Error(),
			Success:     false,
			Description: "Should register precision timing action",
		})
		return
	}

	var executionTimes []time.Time
	cyre.On("precision-test", func(payload interface{}) interface{} {
		executionTimes = append(executionTimes, time.Now())
		fmt.Printf("  ⏱️ Precision execution #%d at %s\n", len(executionTimes), time.Now().Format("15:04:05.000"))
		return map[string]interface{}{"precise": true}
	})

	start := time.Now()
	result := <-cyre.Call("precision-test", "precision-payload")

	// Wait for all executions
	time.Sleep(time.Duration(repeatCount+1)*intervalDuration + 100*time.Millisecond)

	ts.AddResult(TestResult{
		Name:        "Precision Call",
		Expected:    "Success",
		Actual:      fmt.Sprintf("OK: %t", result.OK),
		Success:     result.OK,
		Description: "Precision timing call should succeed",
	})

	// Check timing precision
	if len(executionTimes) >= 2 {
		var intervals []time.Duration
		for i := 1; i < len(executionTimes); i++ {
			intervals = append(intervals, executionTimes[i].Sub(executionTimes[i-1]))
		}

		var totalDrift time.Duration
		for _, interval := range intervals {
			drift := interval - intervalDuration
			if drift < 0 {
				drift = -drift
			}
			totalDrift += drift
		}

		avgDrift := totalDrift / time.Duration(len(intervals))
		precisionOK := avgDrift < 10*time.Millisecond // Allow 10ms average drift

		ts.AddResult(TestResult{
			Name:        "TimeKeeper Precision",
			Expected:    "<10ms average drift",
			Actual:      fmt.Sprintf("%v average drift", avgDrift),
			Success:     precisionOK,
			Duration:    time.Since(start),
			Description: "TimeKeeper should maintain precise intervals",
		})
	}

	cyre.Forget("precision-test")
}

// === HELPER FUNCTION TO CHECK IF ACTION EXISTS ===

func actionExists(actionID string) bool {
	// Try to get the action's current payload
	_, exists := cyre.Get(actionID)
	return exists
}
