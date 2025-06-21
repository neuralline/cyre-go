// test/cyre_test.go
// Basic functionality tests for Cyre Go

package cyre

import (
	"testing"
	"time"
)

// TestBasicFunctionality tests core Cyre operations
func TestBasicFunctionality(t *testing.T) {
	// Initialize Cyre
	result := Initialize()
	if !result.OK {
		t.Fatalf("Failed to initialize Cyre: %s", result.Message)
	}

	// Register action
	err := Action(ActionConfig{
		ID:      "test-action",
		Payload: "initial",
	})
	if err != nil {
		t.Fatalf("Failed to register action: %v", err)
	}

	// Subscribe to action
	var handlerCalled bool
	var receivedPayload interface{}

	subResult := On("test-action", func(payload interface{}) interface{} {
		handlerCalled = true
		receivedPayload = payload
		return "handler-response"
	})

	if !subResult.OK {
		t.Fatalf("Failed to subscribe: %s", subResult.Message)
	}

	// Call action
	resultChan := Call("test-action", "test-payload")
	callResult := <-resultChan

	if !callResult.OK {
		t.Fatalf("Call failed: %s", callResult.Message)
	}

	if !handlerCalled {
		t.Error("Handler was not called")
	}

	if receivedPayload != "test-payload" {
		t.Errorf("Expected payload 'test-payload', got %v", receivedPayload)
	}

	if callResult.Payload != "handler-response" {
		t.Errorf("Expected response 'handler-response', got %v", callResult.Payload)
	}

	// Test Get function
	if payload, exists := Get("test-action"); !exists || payload != "test-payload" {
		t.Errorf("Get failed: expected 'test-payload', got %v (exists: %t)", payload, exists)
	}

	// Test Forget function
	if !Forget("test-action") {
		t.Error("Forget failed")
	}

	// Verify action is gone
	if _, exists := Get("test-action"); exists {
		t.Error("Action still exists after forget")
	}
}

// TestThrottleProtection tests throttle mechanism
func TestThrottleProtection(t *testing.T) {
	Initialize()

	throttleDuration := 100 * time.Millisecond
	err := Action(ActionConfig{
		ID:       "throttled-action",
		Throttle: &throttleDuration,
	})
	if err != nil {
		t.Fatalf("Failed to register throttled action: %v", err)
	}

	var callCount int
	On("throttled-action", func(payload interface{}) interface{} {
		callCount++
		return callCount
	})

	// First call should succeed
	result1 := <-Call("throttled-action", "call1")
	if !result1.OK {
		t.Error("First call should succeed")
	}

	// Second immediate call should be throttled
	result2 := <-Call("throttled-action", "call2")
	if result2.OK {
		t.Error("Second call should be throttled")
	}

	// Wait for throttle to expire
	time.Sleep(150 * time.Millisecond)

	// Third call should succeed
	result3 := <-Call("throttled-action", "call3")
	if !result3.OK {
		t.Error("Third call should succeed after throttle expires")
	}

	// Should have only 2 successful executions
	if callCount != 2 {
		t.Errorf("Expected 2 handler calls, got %d", callCount)
	}

	Forget("throttled-action")
}

// TestDebounceProtection tests debounce mechanism
func TestDebounceProtection(t *testing.T) {
	Initialize()

	err := Action(ActionConfig{
		ID:       "debounced-action",
		Debounce: 1000,
	})
	if err != nil {
		t.Fatalf("Failed to register debounced action: %v", err)
	}

	var callCount int
	var lastPayload interface{}

	On("debounced-action", func(payload interface{}) interface{} {
		callCount++
		lastPayload = payload
		return payload
	})

	// Make rapid calls
	Call("debounced-action", "call1")
	Call("debounced-action", "call2")
	Call("debounced-action", "call3")

	// Wait for debounce to execute
	time.Sleep(200 * time.Millisecond)

	// Should only have executed once with the last payload
	if callCount != 1 {
		t.Errorf("Expected 1 debounced call, got %d", callCount)
	}

	if lastPayload != "call3" {
		t.Errorf("Expected last payload 'call3', got %v", lastPayload)
	}

	Forget("debounced-action")
}

// TestChangeDetection tests detectChanges mechanism
func TestChangeDetection(t *testing.T) {
	Initialize()

	err := Action(ActionConfig{
		ID:            "change-detection-action",
		DetectChanges: true,
	})
	if err != nil {
		t.Fatalf("Failed to register change detection action: %v", err)
	}

	var callCount int
	On("change-detection-action", func(payload interface{}) interface{} {
		callCount++
		return payload
	})

	testPayload := map[string]interface{}{"value": 42}

	// First call should execute
	result1 := <-Call("change-detection-action", testPayload)
	if !result1.OK {
		t.Error("First call should succeed")
	}

	// Second call with same payload should be skipped
	result2 := <-Call("change-detection-action", testPayload)
	if result2.OK {
		t.Error("Second call with identical payload should be skipped")
	}

	// Third call with different payload should execute
	newPayload := map[string]interface{}{"value": 100}
	result3 := <-Call("change-detection-action", newPayload)
	if !result3.OK {
		t.Error("Third call with different payload should succeed")
	}

	// Should have only 2 successful executions
	if callCount != 2 {
		t.Errorf("Expected 2 handler calls, got %d", callCount)
	}

	Forget("change-detection-action")
}

// TestActionChaining tests IntraLinks functionality
func TestActionChaining(t *testing.T) {
	Initialize()

	// Register first action
	err := Action(ActionConfig{ID: "chain-start"})
	if err != nil {
		t.Fatalf("Failed to register chain-start action: %v", err)
	}

	// Register second action
	err = Action(ActionConfig{ID: "chain-end"})
	if err != nil {
		t.Fatalf("Failed to register chain-end action: %v", err)
	}

	var chainStartCalled bool
	var chainEndCalled bool
	var finalResult interface{}

	// First handler chains to second
	On("chain-start", func(payload interface{}) interface{} {
		chainStartCalled = true
		return map[string]interface{}{
			"id":      "chain-end",
			"payload": "chained-data",
		}
	})

	// Second handler returns final result
	On("chain-end", func(payload interface{}) interface{} {
		chainEndCalled = true
		finalResult = payload
		return "chain-complete"
	})

	// Trigger the chain
	result := <-Call("chain-start", "initial-data")

	if !result.OK {
		t.Fatalf("Chain call failed: %s", result.Message)
	}

	if !chainStartCalled {
		t.Error("Chain start handler was not called")
	}

	if !chainEndCalled {
		t.Error("Chain end handler was not called")
	}

	if finalResult != "chained-data" {
		t.Errorf("Expected chained data 'chained-data', got %v", finalResult)
	}

	if result.Payload != "chain-complete" {
		t.Errorf("Expected final result 'chain-complete', got %v", result.Payload)
	}

	Forget("chain-start")
	Forget("chain-end")
}

// TestSystemHealth tests health monitoring
func TestSystemHealth(t *testing.T) {
	Initialize()

	if !IsHealthy() {
		t.Error("System should be healthy after initialization")
	}

	metrics := GetMetrics()
	if metrics == nil {
		t.Error("Metrics should be available")
	}

	stats := GetStats()
	if stats == nil {
		t.Error("Stats should be available")
	}

	if _, exists := stats["uptime"]; !exists {
		t.Error("Stats should include uptime")
	}
}

// TestConvenienceFunctions tests helper functions
func TestConvenienceFunctions(t *testing.T) {
	Initialize()

	err := Action(ActionConfig{
		ID:       "sync-test",
		Throttle: ThrottleDuration(50 * time.Millisecond),
		Debounce: DebounceDuration(25 * time.Millisecond),
		Repeat:   RepeatCount(3),
	})
	if err != nil {
		t.Fatalf("Failed to register action with helpers: %v", err)
	}

	On("sync-test", func(payload interface{}) interface{} {
		return "sync-response"
	})

	// Test CallSync
	result := CallSync("sync-test", "sync-data")
	if !result.OK {
		t.Errorf("CallSync failed: %s", result.Message)
	}

	if result.Payload != "sync-response" {
		t.Errorf("Expected 'sync-response', got %v", result.Payload)
	}

	// Test CallWithTimeout
	resultWithTimeout, err := CallWithTimeout("sync-test", "timeout-data", 1*time.Second)
	if err != nil {
		t.Errorf("CallWithTimeout failed: %v", err)
	}

	if !resultWithTimeout.OK {
		t.Errorf("CallWithTimeout should succeed: %s", resultWithTimeout.Message)
	}

	// Test ActionExists
	if !ActionExists("sync-test") {
		t.Error("ActionExists should return true for registered action")
	}

	if ActionExists("non-existent") {
		t.Error("ActionExists should return false for non-existent action")
	}

	Forget("sync-test")
}

// TestBatchOperations tests batch functionality
func TestBatchOperations(t *testing.T) {
	Initialize()

	// Test ActionBatch
	configs := []ActionConfig{
		{ID: "batch1"},
		{ID: "batch2"},
		{ID: "batch3"},
	}

	errors := ActionBatch(configs)
	for i, err := range errors {
		if err != nil {
			t.Errorf("ActionBatch[%d] failed: %v", i, err)
		}
	}

	// Test OnBatch
	var callCounts = make(map[string]int)
	handlers := map[string]HandlerFunc{
		"batch1": func(p interface{}) interface{} {
			callCounts["batch1"]++
			return "response1"
		},
		"batch2": func(p interface{}) interface{} {
			callCounts["batch2"]++
			return "response2"
		},
		"batch3": func(p interface{}) interface{} {
			callCounts["batch3"]++
			return "response3"
		},
	}

	subResults := OnBatch(handlers)
	for actionID, result := range subResults {
		if !result.OK {
			t.Errorf("OnBatch[%s] failed: %s", actionID, result.Message)
		}
	}

	// Test calls
	for _, config := range configs {
		result := CallSync(config.ID, "test")
		if !result.OK {
			t.Errorf("Batch call[%s] failed: %s", config.ID, result.Message)
		}
	}

	// Verify all handlers were called
	for actionID := range handlers {
		if callCounts[actionID] != 1 {
			t.Errorf("Handler[%s] called %d times, expected 1", actionID, callCounts[actionID])
		}
	}

	// Test ForgetBatch
	actionIDs := []string{"batch1", "batch2", "batch3"}
	forgetResults := ForgetBatch(actionIDs)

	for _, actionID := range actionIDs {
		if !forgetResults[actionID] {
			t.Errorf("ForgetBatch[%s] should return true", actionID)
		}
	}
}

// TestErrorHandling tests error scenarios
func TestErrorHandling(t *testing.T) {
	Initialize()

	// Test calling non-existent action
	result := <-Call("non-existent", "data")
	if result.OK {
		t.Error("Call to non-existent action should fail")
	}

	// Test handler that panics
	err := Action(ActionConfig{ID: "panic-action"})
	if err != nil {
		t.Fatalf("Failed to register panic action: %v", err)
	}

	On("panic-action", func(payload interface{}) interface{} {
		panic("handler panic")
	})

	panicResult := <-Call("panic-action", "data")
	if panicResult.OK {
		t.Error("Panic action should return error")
	}

	if panicResult.Error == nil {
		t.Error("Panic action should include error")
	}

	Forget("panic-action")
}

// BenchmarkBasicCall benchmarks basic call performance
func BenchmarkBasicCall(b *testing.B) {
	Initialize()

	err := Action(ActionConfig{ID: "benchmark-action"})
	if err != nil {
		b.Fatalf("Failed to register benchmark action: %v", err)
	}

	On("benchmark-action", func(payload interface{}) interface{} {
		return payload
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			result := <-Call("benchmark-action", "benchmark-data")
			if !result.OK {
				b.Errorf("Benchmark call failed: %s", result.Message)
			}
		}
	})

	Forget("benchmark-action")
}

// BenchmarkThrottledCall benchmarks throttled calls
func BenchmarkThrottledCall(b *testing.B) {
	Initialize()

	throttleDuration := 1 * time.Millisecond
	err := Action(ActionConfig{
		ID:       "throttled-benchmark",
		Throttle: &throttleDuration,
	})
	if err != nil {
		b.Fatalf("Failed to register throttled benchmark action: %v", err)
	}

	On("throttled-benchmark", func(payload interface{}) interface{} {
		return payload
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		<-Call("throttled-benchmark", i)
	}

	Forget("throttled-benchmark")
}
