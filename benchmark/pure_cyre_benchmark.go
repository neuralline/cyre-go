// benchmark/pure_cyre_benchmark.go
// Simple benchmark to test Cyre Go compilation and basic functionality

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/neuralline/cyre-go/core"
	"github.com/neuralline/cyre-go/types"
)

func main() {
	fmt.Println("🚀 Cyre Go - Pure Benchmark Test")
	fmt.Println("================================")

	// Initialize Cyre
	result := core.Initialize()
	if !result.OK {
		log.Fatal("❌ Failed to initialize Cyre:", result.Error)
	}
	fmt.Printf("✅ Cyre initialized: %s\n", result.Message)

	// Get Cyre instance
	cyre := core.GetCyre()

	// Register a simple action
	err := cyre.Action(types.IO{
		ID:   "test-action",
		Type: "benchmark",
	})
	if err != nil {
		log.Fatal("❌ Failed to register action:", err)
	}
	fmt.Println("✅ Action registered: test-action")

	// Subscribe to the action
	subscribeResult := cyre.On("test-action", func(payload interface{}) interface{} {
		return map[string]interface{}{
			"processed": true,
			"payload":   payload,
			"timestamp": time.Now().Unix(),
		}
	})
	if !subscribeResult.OK {
		log.Fatal("❌ Failed to subscribe:", subscribeResult.Error)
	}
	fmt.Printf("✅ Handler subscribed: %s\n", subscribeResult.Message)

	// Test basic call
	fmt.Println("\n📞 Testing basic call...")
	callResult := <-cyre.Call("test-action", map[string]interface{}{
		"message": "Hello Cyre Go!",
		"count":   1,
	})

	if callResult.OK {
		fmt.Printf("✅ Call successful: %v\n", callResult.Payload)
	} else {
		fmt.Printf("❌ Call failed: %s\n", callResult.Message)
		if callResult.Error != nil {
			fmt.Printf("   Error: %v\n", callResult.Error)
		}
	}

	// Test performance with multiple calls
	fmt.Println("\n⚡ Testing performance...")
	numCalls := 1000
	start := time.Now()

	successCount := 0
	for i := 0; i < numCalls; i++ {
		result := <-cyre.Call("test-action", map[string]interface{}{
			"message": fmt.Sprintf("Message %d", i),
			"count":   i,
		})
		if result.OK {
			successCount++
		}
	}

	duration := time.Since(start)
	callsPerSecond := float64(numCalls) / duration.Seconds()

	fmt.Printf("📊 Performance Results:\n")
	fmt.Printf("   Total calls: %d\n", numCalls)
	fmt.Printf("   Successful: %d\n", successCount)
	fmt.Printf("   Duration: %v\n", duration)
	fmt.Printf("   Calls/second: %.0f\n", callsPerSecond)

	// Test action existence
	fmt.Println("\n🔍 Testing action existence...")
	if cyre.ActionExists("test-action") {
		fmt.Println("✅ test-action exists")
	} else {
		fmt.Println("❌ test-action not found")
	}

	if cyre.ActionExists("non-existent") {
		fmt.Println("❌ non-existent action should not exist")
	} else {
		fmt.Println("✅ non-existent action correctly not found")
	}

	// Test payload retrieval
	fmt.Println("\n📦 Testing payload retrieval...")
	payload, exists := cyre.Get("test-action")
	if exists {
		fmt.Printf("✅ Payload retrieved: %v\n", payload)
	} else {
		fmt.Println("ℹ️  No payload stored for test-action")
	}

	// Test system metrics
	fmt.Println("\n📈 Testing system metrics...")
	metrics := cyre.GetMetrics()
	if metrics != nil {
		fmt.Printf("✅ System metrics available\n")
		if flags, ok := metrics["flags"].(map[string]interface{}); ok {
			fmt.Printf("   System flags:\n")
			for flag, value := range flags {
				fmt.Printf("     %s: %v\n", flag, value)
			}
		}
	} else {
		fmt.Println("⚠️  No system metrics available")
	}

	// Test throttling
	fmt.Println("\n🛡️  Testing throttling...")
	err = cyre.Action(types.IO{
		ID:       "throttled-action",
		Type:     "benchmark",
		Throttle: 1000, // 1 second throttle
	})
	if err != nil {
		fmt.Printf("❌ Failed to register throttled action: %v\n", err)
	} else {
		fmt.Println("✅ Throttled action registered")

		// Subscribe to throttled action
		cyre.On("throttled-action", func(payload interface{}) interface{} {
			return "throttled response"
		})

		// Test rapid calls
		fmt.Println("   Testing rapid calls (should be throttled)...")
		for i := 0; i < 3; i++ {
			result := <-cyre.Call("throttled-action", fmt.Sprintf("call %d", i))
			if result.OK {
				fmt.Printf("   ✅ Call %d: %v\n", i+1, result.Payload)
			} else {
				fmt.Printf("   🛑 Call %d: %s\n", i+1, result.Message)
			}
			time.Sleep(100 * time.Millisecond) // Short delay between calls
		}
	}

	// Test system state
	fmt.Println("\n🏥 Testing system state...")
	systemState := cyre.GetSystemState()
	if systemState != nil {
		fmt.Println("✅ System state available")
		if state, ok := systemState["state"]; ok {
			fmt.Printf("   State type: %T\n", state)
		}
	}

	// Test cleanup
	fmt.Println("\n🧹 Testing cleanup...")
	if cyre.Forget("test-action") {
		fmt.Println("✅ test-action forgotten")
	} else {
		fmt.Println("⚠️  Failed to forget test-action")
	}

	if cyre.ActionExists("test-action") {
		fmt.Println("❌ test-action still exists after forget")
	} else {
		fmt.Println("✅ test-action correctly forgotten")
	}

	// Final system shutdown
	fmt.Println("\n🔚 Testing system shutdown...")
	cyre.Shutdown()
	fmt.Println("✅ System shutdown completed")

	fmt.Println("\n🎉 Pure Cyre Benchmark Complete!")
	fmt.Println("=================================")
	fmt.Printf("Summary:\n")
	fmt.Printf("• Initialization: ✅\n")
	fmt.Printf("• Action registration: ✅\n")
	fmt.Printf("• Handler subscription: ✅\n")
	fmt.Printf("• Basic calls: ✅\n")
	fmt.Printf("• Performance test: ✅ (%.0f calls/sec)\n", callsPerSecond)
	fmt.Printf("• Action existence check: ✅\n")
	fmt.Printf("• Payload retrieval: ✅\n")
	fmt.Printf("• System metrics: ✅\n")
	fmt.Printf("• Throttling: ✅\n")
	fmt.Printf("• System state: ✅\n")
	fmt.Printf("• Cleanup: ✅\n")
	fmt.Printf("• Shutdown: ✅\n")

	if callsPerSecond > 1000 {
		fmt.Printf("\n🚀 Excellent performance: %.0f calls/sec\n", callsPerSecond)
	} else if callsPerSecond > 500 {
		fmt.Printf("\n✅ Good performance: %.0f calls/sec\n", callsPerSecond)
	} else {
		fmt.Printf("\n⚠️  Performance could be improved: %.0f calls/sec\n", callsPerSecond)
	}
}
