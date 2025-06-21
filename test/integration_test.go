// test/integration_test.go
// Integration test to verify all components work together

package cyre

import (
	"testing"
	"time"
)

func TestIntegrationBasic(t *testing.T) {
	// Test basic workflow
	result := Initialize()
	if !result.OK {
		t.Fatalf("Initialize failed: %s", result.Message)
	}

	// Register action
	err := Action(ActionConfig{
		ID:      "integration-test",
		Payload: "initial",
	})
	if err != nil {
		t.Fatalf("Action registration failed: %v", err)
	}

	// Subscribe
	var executed bool
	var receivedPayload interface{}

	subResult := On("integration-test", func(payload interface{}) interface{} {
		executed = true
		receivedPayload = payload
		return "success"
	})

	if !subResult.OK {
		t.Fatalf("Subscription failed: %s", subResult.Message)
	}

	// Call
	callResult := <-Call("integration-test", "test-data")

	if !callResult.OK {
		t.Fatalf("Call failed: %s", callResult.Message)
	}

	if !executed {
		t.Error("Handler was not executed")
	}

	if receivedPayload != "test-data" {
		t.Errorf("Expected 'test-data', got %v", receivedPayload)
	}

	if callResult.Payload != "success" {
		t.Errorf("Expected 'success', got %v", callResult.Payload)
	}

	// Test Get
	if payload, exists := Get("integration-test"); !exists || payload != "test-data" {
		t.Errorf("Get failed: payload=%v, exists=%t", payload, exists)
	}

	// Test health
	if !IsHealthy() {
		t.Error("System should be healthy")
	}

	// Cleanup
	if !Forget("integration-test") {
		t.Error("Forget failed")
	}
}

func TestIntegrationProtections(t *testing.T) {
	Initialize()

	// Test all protection mechanisms together
	throttleDuration := 50 * time.Millisecond

	err := Action(ActionConfig{
		ID:            "protected-action",
		Throttle:      &throttleDuration,
		DetectChanges: true,
	})
	if err != nil {
		t.Fatalf("Protected action registration failed: %v", err)
	}

	callCount := 0
	On("protected-action", func(payload interface{}) interface{} {
		callCount++
		return callCount
	})

	// Test throttle
	result1 := <-Call("protected-action", map[string]interface{}{"value": 1})
	if !result1.OK {
		t.Error("First call should succeed")
	}

	result2 := <-Call("protected-action", map[string]interface{}{"value": 2})
	if result2.OK {
		t.Error("Second call should be throttled")
	}

	// Wait for throttle reset
	time.Sleep(60 * time.Millisecond)

	// Test change detection with same payload
	samePayload := map[string]interface{}{"value": 3}
	result3 := <-Call("protected-action", samePayload)
	if !result3.OK {
		t.Error("Call with new payload should succeed")
	}

	result4 := <-Call("protected-action", samePayload)
	if result4.OK {
		t.Error("Call with same payload should be skipped")
	}

	if callCount != 2 {
		t.Errorf("Expected 2 executions, got %d", callCount)
	}

	Forget("protected-action")
}

func TestIntegrationChaining(t *testing.T) {
	Initialize()

	// Setup chain of actions
	err := Action(ActionConfig{ID: "step1"})
	if err != nil {
		t.Fatalf("Step1 registration failed: %v", err)
	}

	err = Action(ActionConfig{ID: "step2"})
	if err != nil {
		t.Fatalf("Step2 registration failed: %v", err)
	}

	err = Action(ActionConfig{ID: "step3"})
	if err != nil {
		t.Fatalf("Step3 registration failed: %v", err)
	}

	var step1Called, step2Called, step3Called bool
	var finalResult interface{}

	// Chain: step1 -> step2 -> step3
	On("step1", func(payload interface{}) interface{} {
		step1Called = true
		return map[string]interface{}{
			"id":      "step2",
			"payload": "from-step1",
		}
	})

	On("step2", func(payload interface{}) interface{} {
		step2Called = true
		return map[string]interface{}{
			"id":      "step3",
			"payload": "from-step2",
		}
	})

	On("step3", func(payload interface{}) interface{} {
		step3Called = true
		finalResult = payload
		return "chain-complete"
	})

	// Trigger the chain
	result := <-Call("step1", "initial")

	if !result.OK {
		t.Fatalf("Chain execution failed: %s", result.Message)
	}

	if !step1Called || !step2Called || !step3Called {
		t.Errorf("Chain steps not all called: step1=%t, step2=%t, step3=%t",
			step1Called, step2Called, step3Called)
	}

	if finalResult != "from-step2" {
		t.Errorf("Expected 'from-step2', got %v", finalResult)
	}

	if result.Payload != "chain-complete" {
		t.Errorf("Expected 'chain-complete', got %v", result.Payload)
	}

	// Cleanup
	Forget("step1")
	Forget("step2")
	Forget("step3")
}

func TestIntegrationMetrics(t *testing.T) {
	Initialize()

	// Register action
	err := Action(ActionConfig{ID: "metrics-test"})
	if err != nil {
		t.Fatalf("Metrics action registration failed: %v", err)
	}

	On("metrics-test", func(payload interface{}) interface{} {
		return "metrics-response"
	})

	// Make some calls
	for i := 0; i < 5; i++ {
		result := <-Call("metrics-test", i)
		if !result.OK {
			t.Errorf("Call %d failed: %s", i, result.Message)
		}
	}

	// Check system metrics
	systemMetrics := GetMetrics()
	if systemMetrics == nil {
		t.Error("System metrics should be available")
	}

	// Check action metrics
	actionMetrics := GetMetrics("metrics-test")
	if actionMetrics == nil {
		t.Error("Action metrics should be available")
	}

	// Check stats
	stats := GetStats()
	if stats == nil {
		t.Error("Stats should be available")
	}

	uptime, exists := stats["uptime"]
	if !exists {
		t.Error("Uptime should be in stats")
	}

	if duration, ok := uptime.(time.Duration); !ok || duration <= 0 {
		t.Error("Uptime should be a positive duration")
	}

	Forget("metrics-test")
}

func TestIntegrationConcurrency(t *testing.T) {
	Initialize()

	err := Action(ActionConfig{ID: "concurrent-test"})
	if err != nil {
		t.Fatalf("Concurrent action registration failed: %v", err)
	}

	execCount := 0
	On("concurrent-test", func(payload interface{}) interface{} {
		execCount++
		time.Sleep(1 * time.Millisecond) // Simulate work
		return execCount
	})

	// Make concurrent calls
	const numCalls = 10
	results := make([]<-chan CallResult, numCalls)

	for i := 0; i < numCalls; i++ {
		results[i] = Call("concurrent-test", i)
	}

	// Collect all results
	successCount := 0
	for i := 0; i < numCalls; i++ {
		result := <-results[i]
		if result.OK {
			successCount++
		}
	}

	if successCount != numCalls {
		t.Errorf("Expected %d successful calls, got %d", numCalls, successCount)
	}

	if execCount != numCalls {
		t.Errorf("Expected %d executions, got %d", numCalls, execCount)
	}

	Forget("concurrent-test")
}

// Benchmark integration performance
func BenchmarkIntegrationThroughput(b *testing.B) {
	Initialize()

	err := Action(ActionConfig{ID: "benchmark-integration"})
	if err != nil {
		b.Fatalf("Benchmark action registration failed: %v", err)
	}

	On("benchmark-integration", func(payload interface{}) interface{} {
		return payload
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			result := <-Call("benchmark-integration", i)
			if !result.OK {
				b.Errorf("Call failed: %s", result.Message)
			}
			i++
		}
	})

	Forget("benchmark-integration")
}
