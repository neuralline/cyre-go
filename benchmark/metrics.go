// benchmark/metrics.go
// Direct diagnostic of MetricState functionality
// Checks if the system brain is actually working

package main

import (
	"fmt"
	"log"
	"runtime"
	"time"

	"github.com/neuralline/cyre-go"
	"github.com/neuralline/cyre-go/types"
)

func main() {
	fmt.Println("🔬 METRICSTATE DIAGNOSTIC - System Brain Health Check")
	fmt.Println("====================================================")

	// Initialize
	result := cyre.Init()
	if !result.OK {
		log.Fatal("Failed to initialize:", result.Error)
	}
	fmt.Println("✅ Cyre initialized")

	// Test 1: Check initial state
	fmt.Println("\n🧪 Test 1: Initial MetricState Check")
	fmt.Println("-----------------------------------")
	checkMetricState(cyre, "INITIAL")

	// Test 2: Register actions and check store counts
	fmt.Println("\n🧪 Test 2: Action Registration & Store Counts")
	fmt.Println("---------------------------------------------")

	for i := 0; i < 5; i++ {
		actionID := fmt.Sprintf("test-action-%d", i)
		err := cyre.Action(types.IO{
			ID:       actionID,
			Priority: "medium",
		})
		if err != nil {
			fmt.Printf("❌ Failed to register %s: %v\n", actionID, err)
		} else {
			fmt.Printf("✅ Registered %s\n", actionID)
		}

		// Add handler
		cyre.On(actionID, func(payload interface{}) interface{} {
			time.Sleep(10 * time.Millisecond) // Simulate work
			return "completed"
		})
	}

	checkMetricState(cyre, "AFTER REGISTRATION")

	// Test 3: Execute calls and check performance metrics
	fmt.Println("\n🧪 Test 3: Performance Metrics Update")
	fmt.Println("-------------------------------------")

	// Execute some calls
	for i := 0; i < 10; i++ {
		result := <-cyre.Call("test-action-0", map[string]interface{}{
			"test": i,
		})
		if result.OK {
			fmt.Printf("Call %d: OK\n", i)
		} else {
			fmt.Printf("Call %d: Failed - %s\n", i, result.Message)
		}
	}

	checkMetricState(cyre, "AFTER CALLS")

	// Test 4: Force memory allocation to trigger health metrics
	fmt.Println("\n🧪 Test 4: Memory Pressure Test")
	fmt.Println("-------------------------------")

	// Allocate memory
	var memoryChunks [][]byte
	for i := 0; i < 100; i++ {
		chunk := make([]byte, 1024*1024) // 1MB
		memoryChunks = append(memoryChunks, chunk)
	}

	// Force GC
	runtime.GC()
	runtime.GC()

	fmt.Printf("Allocated ~100MB, forced GC\n")
	checkMetricState(cyre, "MEMORY PRESSURE")

	// Test 5: Check if MetricState methods work directly
	fmt.Println("\n🧪 Test 5: Direct MetricState Access")
	fmt.Println("-----------------------------------")
	checkDirectAccess(cyre)

	// Test 6: Simulate high call rate
	fmt.Println("\n🧪 Test 6: High Call Rate Test")
	fmt.Println("------------------------------")

	start := time.Now()
	for i := 0; i < 100; i++ {
		cyre.Call("test-action-1", fmt.Sprintf("burst-%d", i))
	}
	fmt.Printf("Submitted 100 calls in %v\n", time.Since(start))

	time.Sleep(2 * time.Second) // Let them complete
	checkMetricState(cyre, "HIGH CALL RATE")

	// Final cleanup
	memoryChunks = nil
	runtime.GC()

	fmt.Println("\n🏁 Diagnostic Complete")
	cyre.Shutdown()
}

func checkMetricState(cyre *cyre.Cyre, phase string) {
	fmt.Printf("\n📊 METRICSTATE CHECK: %s\n", phase)

	// Get metrics
	metrics := cyre.GetMetrics()
	if metrics == nil {
		fmt.Println("❌ GetMetrics() returned nil")
		return
	}

	// Check each category
	categories := []string{"performance", "store", "workers", "health", "breathing", "flags"}
	for _, category := range categories {
		if data, exists := metrics[category]; exists {
			fmt.Printf("✅ %s: %+v\n", category, data)
		} else {
			fmt.Printf("❌ %s: missing\n", category)
		}
	}

	// Get system state
	systemState := cyre.GetSystemState()
	if systemState == nil {
		fmt.Println("❌ GetSystemState() returned nil")
	} else {
		fmt.Printf("✅ SystemState available: %T\n", systemState)
		if state, ok := systemState["state"]; ok {
			fmt.Printf("   State data: %+v\n", state)
		}
	}

	// Runtime info
	fmt.Printf("🏃 Goroutines: %d\n", runtime.NumGoroutine())

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	fmt.Printf("💾 Memory: Alloc=%dKB, Sys=%dKB, GC=%d\n",
		memStats.Alloc/1024, memStats.Sys/1024, memStats.NumGC)
}

func checkDirectAccess(cyre *cyre.Cyre) {
	// We can't access MetricState directly from here, but we can test
	// if the Cyre methods that depend on it work

	fmt.Println("Testing Cyre methods that use MetricState...")

	// Test Get
	if payload, exists := cyre.Get("test-action-0"); exists {
		fmt.Printf("✅ Get() works: %v\n", payload)
	} else {
		fmt.Println("❌ Get() returned no payload")
	}

	// Check if we get any error messages
	metrics := cyre.GetMetrics()
	if errorMsg, ok := metrics["error"]; ok {
		fmt.Printf("❌ MetricState error: %v\n", errorMsg)
	} else {
		fmt.Println("✅ No error messages from MetricState")
	}
}
