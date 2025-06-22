// test/pure_cyre_benchmark.go
// Test what Cyre actually does when users just use the API normally
// NO manual worker pools, NO manual channels, NO manual concurrency

package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/neuralline/cyre-go"
)

func main() {
	fmt.Println("🎯 Pure Cyre Benchmark - Testing Cyre's Intelligence")
	fmt.Println("===================================================")
	fmt.Printf("Testing what happens when users just call cyre.Call() normally\n")
	fmt.Printf("NO manual workers, NO manual channels, NO manual concurrency\n\n")

	// Initialize Cyre - let it set up its own optimal configuration
	result := cyre.Initialize()
	if !result.OK {
		log.Fatal("❌ Failed to initialize Cyre")
	}
	fmt.Printf("✅ Cyre initialized - using its own internal optimization\n\n")

	// Test 1: Single Action, High Volume
	fmt.Println("📊 TEST 1: Single Action Performance")
	fmt.Println("===================================")
	testSingleActionPerformance()

	// Test 2: Multiple Actions, Concurrent Load
	fmt.Println("\n📊 TEST 2: Multiple Actions Concurrent Load")
	fmt.Println("==========================================")
	testMultipleActionsPerformance()

	// Test 3: Mixed Workload (Fast + Protected Actions)
	fmt.Println("\n📊 TEST 3: Mixed Workload Performance")
	fmt.Println("====================================")
	testMixedWorkloadPerformance()

	// Test 4: Burst Load Test
	fmt.Println("\n📊 TEST 4: Burst Load Test")
	fmt.Println("=========================")
	testBurstLoadPerformance()

	// Test 5: Sustained Load Test
	fmt.Println("\n📊 TEST 5: Sustained Load Test")
	fmt.Println("=============================")
	testSustainedLoadPerformance()

	fmt.Println("\n🎉 Pure Cyre Benchmark Complete!")
	fmt.Println("================================")
	fmt.Println("This is what Cyre delivers out-of-the-box with zero manual optimization!")
}

// Test 1: Single action, let Cyre handle all the load
func testSingleActionPerformance() {
	// Just register a simple action - let Cyre optimize
	err := cyre.Action(cyre.IO{
		ID:   "simple-task",
		Name: "Simple processing task",
	})
	if err != nil {
		log.Printf("Failed to register action: %v", err)
		return
	}

	// Simple handler - no manual optimization
	var processedCount int64
	cyre.On("simple-task", func(payload interface{}) interface{} {
		processedCount++
		return map[string]interface{}{
			"processed": processedCount,
			"input":     payload,
		}
	})

	// Test: Just call Cyre normally, let it handle everything
	totalCalls := 100000
	fmt.Printf("Making %d calls to single action...\n", totalCalls)

	start := time.Now()

	// Sequential calls - test Cyre's internal handling
	for i := 0; i < totalCalls; i++ {
		result := <-cyre.Call("simple-task", map[string]interface{}{
			"id":   i,
			"data": fmt.Sprintf("task-%d", i),
		})

		if !result.OK {
			fmt.Printf("❌ Call %d failed: %s\n", i, result.Message)
		}

		// Progress indicator
		if i > 0 && i%10000 == 0 {
			fmt.Printf("   %d calls completed...\n", i)
		}
	}

	duration := time.Since(start)
	opsPerSec := float64(totalCalls) / duration.Seconds()

	fmt.Printf("📊 Single Action Results:\n")
	fmt.Printf("   Total calls: %d\n", totalCalls)
	fmt.Printf("   Duration: %v\n", duration)
	fmt.Printf("   Ops/second: %.0f\n", opsPerSec)
	fmt.Printf("   Avg latency: %v\n", duration/time.Duration(totalCalls))
	fmt.Printf("   Processed count: %d\n", processedCount)
}

// Test 2: Multiple actions, let Cyre balance the load
func testMultipleActionsPerformance() {
	actionCount := 5
	callsPerAction := 10000

	// Register multiple actions - let Cyre handle routing
	for i := 0; i < actionCount; i++ {
		actionID := fmt.Sprintf("task-%d", i)

		err := cyre.Action(cyre.IO{
			ID:   actionID,
			Name: fmt.Sprintf("Task %d", i),
		})
		if err != nil {
			log.Printf("Failed to register action %s: %v", actionID, err)
			continue
		}

		// Each action has its own counter
		actionIndex := i // Capture for closure
		cyre.On(actionID, func(payload interface{}) interface{} {
			return map[string]interface{}{
				"action": actionIndex,
				"result": "processed",
				"data":   payload,
			}
		})
	}

	fmt.Printf("Making %d calls across %d actions (%d each)...\n",
		actionCount*callsPerAction, actionCount, callsPerAction)

	start := time.Now()

	// Round-robin calls across actions - let Cyre handle load balancing
	totalCalls := 0
	for i := 0; i < callsPerAction; i++ {
		for j := 0; j < actionCount; j++ {
			actionID := fmt.Sprintf("task-%d", j)
			result := <-cyre.Call(actionID, map[string]interface{}{
				"round":  i,
				"action": j,
			})

			if !result.OK {
				fmt.Printf("❌ Call to %s failed: %s\n", actionID, result.Message)
			}
			totalCalls++
		}

		if i > 0 && i%1000 == 0 {
			fmt.Printf("   %d rounds completed...\n", i)
		}
	}

	duration := time.Since(start)
	opsPerSec := float64(totalCalls) / duration.Seconds()

	fmt.Printf("📊 Multiple Actions Results:\n")
	fmt.Printf("   Actions: %d\n", actionCount)
	fmt.Printf("   Total calls: %d\n", totalCalls)
	fmt.Printf("   Duration: %v\n", duration)
	fmt.Printf("   Ops/second: %.0f\n", opsPerSec)
	fmt.Printf("   Avg latency: %v\n", duration/time.Duration(totalCalls))
}

// Test 3: Mix of fast and protected actions
func testMixedWorkloadPerformance() {
	// Fast action - no protection
	err := cyre.Action(cyre.IO{
		ID:   "fast-action",
		Name: "Fast processing",
	})
	if err != nil {
		log.Printf("Failed to register fast action: %v", err)
		return
	}

	// Protected action - with throttling
	err = cyre.Action(cyre.IO{
		ID:       "protected-action",
		Name:     "Protected processing",
		Throttle: 10, // 10ms throttle
	})
	if err != nil {
		log.Printf("Failed to register protected action: %v", err)
		return
	}

	// Handlers
	var fastCount, protectedCount int64
	cyre.On("fast-action", func(payload interface{}) interface{} {
		fastCount++
		return "fast-result"
	})

	cyre.On("protected-action", func(payload interface{}) interface{} {
		protectedCount++
		return "protected-result"
	})

	// Mixed workload - 70% fast, 30% protected
	totalCalls := 50000
	fastCalls := int(float64(totalCalls) * 0.7)
	protectedCalls := totalCalls - fastCalls

	fmt.Printf("Mixed workload: %d fast + %d protected = %d total calls\n",
		fastCalls, protectedCalls, totalCalls)

	start := time.Now()

	callCount := 0
	// Interleave fast and protected calls
	for i := 0; i < totalCalls; i++ {
		if i%10 < 7 { // 70% fast
			result := <-cyre.Call("fast-action", map[string]interface{}{"id": i})
			if result.OK {
				callCount++
			}
		} else { // 30% protected
			result := <-cyre.Call("protected-action", map[string]interface{}{"id": i})
			if result.OK {
				callCount++
			}
		}

		if i > 0 && i%5000 == 0 {
			fmt.Printf("   %d mixed calls completed...\n", i)
		}
	}

	duration := time.Since(start)
	opsPerSec := float64(callCount) / duration.Seconds()

	fmt.Printf("📊 Mixed Workload Results:\n")
	fmt.Printf("   Total attempted: %d\n", totalCalls)
	fmt.Printf("   Successful calls: %d\n", callCount)
	fmt.Printf("   Fast processed: %d\n", fastCount)
	fmt.Printf("   Protected processed: %d\n", protectedCount)
	fmt.Printf("   Duration: %v\n", duration)
	fmt.Printf("   Ops/second: %.0f\n", opsPerSec)
}

// Test 4: Burst load - simulate real application spikes
func testBurstLoadPerformance() {
	err := cyre.Action(cyre.IO{
		ID:   "burst-handler",
		Name: "Burst load handler",
	})
	if err != nil {
		log.Printf("Failed to register burst action: %v", err)
		return
	}

	var burstCount int64
	cyre.On("burst-handler", func(payload interface{}) interface{} {
		burstCount++
		// Simulate some work
		time.Sleep(100 * time.Microsecond)
		return "burst-processed"
	})

	// Simulate traffic bursts
	bursts := []int{1000, 2000, 5000, 10000, 15000}
	fmt.Printf("Testing burst loads: %v\n", bursts)

	var totalCalls int
	var totalDuration time.Duration

	for i, burstSize := range bursts {
		fmt.Printf("   Burst %d: %d calls...\n", i+1, burstSize)

		start := time.Now()

		// Burst all calls at once - let Cyre handle the load
		var wg sync.WaitGroup
		successCount := int64(0)

		// Use limited concurrent goroutines to simulate realistic load
		concurrentCallers := 10
		callsPerCaller := burstSize / concurrentCallers

		for j := 0; j < concurrentCallers; j++ {
			wg.Add(1)
			go func(callerID int) {
				defer wg.Done()
				for k := 0; k < callsPerCaller; k++ {
					result := <-cyre.Call("burst-handler", map[string]interface{}{
						"caller": callerID,
						"call":   k,
					})
					if result.OK {
						successCount++
					}
				}
			}(j)
		}

		wg.Wait()
		burstDuration := time.Since(start)
		burstOpsPerSec := float64(successCount) / burstDuration.Seconds()

		fmt.Printf("      Duration: %v, Ops/sec: %.0f\n", burstDuration, burstOpsPerSec)

		totalCalls += int(successCount)
		totalDuration += burstDuration

		// Cool down between bursts
		time.Sleep(100 * time.Millisecond)
	}

	avgOpsPerSec := float64(totalCalls) / totalDuration.Seconds()

	fmt.Printf("📊 Burst Load Results:\n")
	fmt.Printf("   Total calls: %d\n", totalCalls)
	fmt.Printf("   Total duration: %v\n", totalDuration)
	fmt.Printf("   Average ops/second: %.0f\n", avgOpsPerSec)
	fmt.Printf("   Processed count: %d\n", burstCount)
}

// Test 5: Sustained load over time
func testSustainedLoadPerformance() {
	err := cyre.Action(cyre.IO{
		ID:   "sustained-task",
		Name: "Sustained load task",
	})
	if err != nil {
		log.Printf("Failed to register sustained action: %v", err)
		return
	}

	var sustainedCount int64
	cyre.On("sustained-task", func(payload interface{}) interface{} {
		sustainedCount++
		return "sustained-result"
	})

	// Run for 10 seconds with consistent load
	duration := 10 * time.Second
	targetRate := 10000 // Target 10K ops/sec
	interval := time.Second / time.Duration(targetRate)

	fmt.Printf("Sustained load: %d ops/sec for %v\n", targetRate, duration)

	start := time.Now()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	callCount := 0

	for {
		select {
		case <-ticker.C:
			go func(id int) {
				result := <-cyre.Call("sustained-task", map[string]interface{}{"id": id})
				if result.OK {
					callCount++
				}
			}(callCount)

		case <-time.After(duration):
			goto done
		}
	}

done:
	actualDuration := time.Since(start)

	// Wait a bit for final calls to complete
	time.Sleep(100 * time.Millisecond)

	actualOpsPerSec := float64(sustainedCount) / actualDuration.Seconds()

	fmt.Printf("📊 Sustained Load Results:\n")
	fmt.Printf("   Target rate: %d ops/sec\n", targetRate)
	fmt.Printf("   Actual rate: %.0f ops/sec\n", actualOpsPerSec)
	fmt.Printf("   Duration: %v\n", actualDuration)
	fmt.Printf("   Total processed: %d\n", sustainedCount)
	fmt.Printf("   Efficiency: %.1f%%\n", (actualOpsPerSec/float64(targetRate))*100)
}
