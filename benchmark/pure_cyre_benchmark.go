// Enhanced Pure Cyre Benchmark - Tests internal intelligence
package main

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	cyre "github.com/neuralline/cyre-go"
)

// benchmark/pure_cyre_benchmark.go - Enable accurate monitoring

// Add this at the beginning of main() function after the header:

func main() {
	fmt.Println("🚀 ENHANCED PURE CYRE BENCHMARK - ACCURATE MONITORING")
	fmt.Println("=====================================================")
	fmt.Printf("Testing Cyre's internal intelligence with REAL system monitoring\n")
	fmt.Printf("Go Version: %s | CPUs: %d\n", runtime.Version(), runtime.NumCPU())
	fmt.Println()

	// Initialize Cyre with accurate monitoring enabled
	result := cyre.Initialize(map[string]interface{}{
		"accurateMonitoring": true,
		"workerPoolSize":     200, // Double the worker pool for better concurrency
	})
	if !result.OK {
		fmt.Printf("❌ Failed to initialize: %v\n", result.Error)
		return
	}
	fmt.Println("✅ Cyre initialized with accurate system monitoring")

	// Wait for breathing system to start
	fmt.Println("⏱️  Waiting for accurate breathing system to initialize...")
	time.Sleep(4 * time.Second) // Slightly longer wait for accurate monitoring setup

	// Rest of the benchmark remains the same...
	// The key difference is now we'll see REAL CPU and memory measurements
	// in the debug output instead of the static 6.2% values

	// Test 1: Sequential Baseline
	fmt.Println("\n🔄 TEST 1: SEQUENTIAL BASELINE (Real Monitoring)")
	fmt.Println("===============================================")
	testSequentialPerformance()

	// Test 2: Concurrent Load (let Cyre manage concurrency)
	fmt.Println("\n⚡ TEST 2: CONCURRENT LOAD (Real Monitoring)")
	fmt.Println("===========================================")
	testConcurrentPerformance()

	// Test 3: Stress Test (trigger breathing system)
	fmt.Println("\n🔥 TEST 3: STRESS TEST (Real Monitoring)")
	fmt.Println("=======================================")
	testStressPerformance()

	// Test 4: Worker Intelligence
	fmt.Println("\n🧠 TEST 4: WORKER INTELLIGENCE (Real Monitoring)")
	fmt.Println("===============================================")
	testWorkerIntelligence()

	fmt.Println("\n🎉 Enhanced Pure Cyre Benchmark with Accurate Monitoring Complete!")
	cyre.Shutdown()
}

// Add environment variable option for easy testing:
// You can also run with: CYRE_ACCURATE=true go run benchmark/pure_cyre_benchmark.go

// Alternative initialization that checks environment:
func initializeWithAccurateMonitoring() {
	// Check environment variable
	useAccurate := os.Getenv("CYRE_ACCURATE") == "true"

	var result cyre.InitResult
	if useAccurate {
		result = cyre.Initialize(map[string]interface{}{
			"accurateMonitoring": true,
			"workerPoolSize":     200,
		})
		fmt.Println("✅ Cyre initialized with accurate monitoring (env var)")
	} else {
		result = cyre.Initialize()
		fmt.Println("✅ Cyre initialized with standard monitoring")
	}

	if !result.OK {
		panic(fmt.Sprintf("Failed to initialize: %v", result.Error))
	}
}

func testSequentialPerformance() {
	// Register simple action
	err := cyre.Action(cyre.IO{ID: "sequential-test"})
	if err != nil {
		fmt.Printf("❌ Failed to register action: %v\n", err)
		return
	}

	cyre.On("sequential-test", func(payload interface{}) interface{} {
		return "ok"
	})

	// Sequential calls (baseline)
	numCalls := 10000
	start := time.Now()

	for i := 0; i < numCalls; i++ {
		result := <-cyre.Call("sequential-test", i)
		if !result.OK {
			fmt.Printf("Call %d failed\n", i)
		}
	}

	duration := time.Since(start)
	opsPerSec := float64(numCalls) / duration.Seconds()

	fmt.Printf("📊 Sequential Results:\n")
	fmt.Printf("   Calls: %d\n", numCalls)
	fmt.Printf("   Duration: %v\n", duration)
	fmt.Printf("   Performance: %.0f ops/sec\n", opsPerSec)
}

func testConcurrentPerformance() {
	// Register action for concurrent test
	err := cyre.Action(cyre.IO{ID: "concurrent-test"})
	if err != nil {
		fmt.Printf("❌ Failed to register action: %v\n", err)
		return
	}

	cyre.On("concurrent-test", func(payload interface{}) interface{} {
		// Small amount of work to simulate real handler
		time.Sleep(10 * time.Microsecond)
		return payload
	})

	// Test different concurrency levels
	concurrencyLevels := []int{10, 50, 100, 200}

	for _, concurrency := range concurrencyLevels {
		fmt.Printf("\n🔄 Testing with %d concurrent goroutines:\n", concurrency)
		testConcurrencyLevel(concurrency)
		time.Sleep(2 * time.Second) // Let breathing system observe
	}
}

func testConcurrencyLevel(concurrency int) {
	callsPerWorker := 1000
	totalCalls := concurrency * callsPerWorker

	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64

	start := time.Now()

	// Launch concurrent workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < callsPerWorker; j++ {
				result := <-cyre.Call("concurrent-test", fmt.Sprintf("worker-%d-call-%d", workerID, j))
				if result.OK {
					atomic.AddInt64(&successCount, 1)
				} else {
					atomic.AddInt64(&errorCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	successful := atomic.LoadInt64(&successCount)
	failed := atomic.LoadInt64(&errorCount)
	opsPerSec := float64(totalCalls) / duration.Seconds()

	fmt.Printf("   Total calls: %d\n", totalCalls)
	fmt.Printf("   Successful: %d\n", successful)
	fmt.Printf("   Failed: %d\n", failed)
	fmt.Printf("   Duration: %v\n", duration)
	fmt.Printf("   Performance: %.0f ops/sec\n", opsPerSec)
	fmt.Printf("   Goroutines: %d\n", runtime.NumGoroutine())
}

func testStressPerformance() {
	// Register stress test action
	err := cyre.Action(cyre.IO{ID: "stress-test"})
	if err != nil {
		fmt.Printf("❌ Failed to register action: %v\n", err)
		return
	}

	cyre.On("stress-test", func(payload interface{}) interface{} {
		// Simulate some CPU work
		sum := 0
		for i := 0; i < 1000; i++ {
			sum += i
		}
		return sum
	})

	// Heavy concurrent load to trigger breathing system
	fmt.Println("🔥 Applying heavy load to trigger breathing system...")

	concurrency := runtime.NumCPU() * 4 // 4x CPU cores
	callsPerWorker := 2000
	totalCalls := concurrency * callsPerWorker

	var wg sync.WaitGroup
	var successCount int64

	start := time.Now()

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < callsPerWorker; j++ {
				result := <-cyre.Call("stress-test", j)
				if result.OK {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}()
	}

	wg.Wait()
	duration := time.Since(start)

	successful := atomic.LoadInt64(&successCount)
	opsPerSec := float64(successful) / duration.Seconds()

	fmt.Printf("📊 Stress Test Results:\n")
	fmt.Printf("   Concurrency: %d workers\n", concurrency)
	fmt.Printf("   Total calls: %d\n", totalCalls)
	fmt.Printf("   Successful: %d\n", successful)
	fmt.Printf("   Duration: %v\n", duration)
	fmt.Printf("   Performance: %.0f ops/sec\n", opsPerSec)
	fmt.Printf("   Peak goroutines: %d\n", runtime.NumGoroutine())

	// Let breathing system recover
	fmt.Println("🫁 Letting breathing system recover...")
	time.Sleep(5 * time.Second)
	fmt.Printf("   Recovery goroutines: %d\n", runtime.NumGoroutine())
}

func testWorkerIntelligence() {
	// Register critical priority action
	err := cyre.Action(cyre.IO{
		ID:       "critical-test",
		Priority: "critical",
	})
	if err != nil {
		fmt.Printf("❌ Failed to register critical action: %v\n", err)
		return
	}

	cyre.On("critical-test", func(payload interface{}) interface{} {
		return "critical-response"
	})

	// Register normal priority action
	err = cyre.Action(cyre.IO{
		ID:       "normal-test",
		Priority: "medium",
	})
	if err != nil {
		fmt.Printf("❌ Failed to register normal action: %v\n", err)
		return
	}

	cyre.On("normal-test", func(payload interface{}) interface{} {
		return "normal-response"
	})

	fmt.Println("🧠 Testing worker intelligence and priority handling...")

	// Create background stress
	var wg sync.WaitGroup
	fmt.Println("   Creating background stress...")

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 500; j++ {
				<-cyre.Call("normal-test", fmt.Sprintf("background-%d", j))
			}
		}()
	}

	// Test critical actions during stress
	time.Sleep(1 * time.Second) // Let stress build up

	fmt.Println("   Testing critical actions during stress...")
	criticalStart := time.Now()

	for i := 0; i < 10; i++ {
		result := <-cyre.Call("critical-test", fmt.Sprintf("critical-%d", i))
		if result.OK {
			fmt.Printf("   ✅ Critical call %d: %v\n", i+1, result.Payload)
		} else {
			fmt.Printf("   ❌ Critical call %d failed: %s\n", i+1, result.Message)
		}
	}

	criticalDuration := time.Since(criticalStart)
	fmt.Printf("   Critical actions avg latency: %v\n", criticalDuration/10)

	wg.Wait()
	fmt.Println("   Background stress completed")
}
