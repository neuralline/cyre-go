// test/benchmark_test.go
// Comprehensive performance benchmark suite for Cyre Go

package cyre_test

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/neuralline/cyre-go"
)

/*
	CYRE GO PERFORMANCE BENCHMARK SUITE

	Targeting to beat:
	- TypeScript Cyre: ~400k ops/sec
	- Rust Cyre: ~1M ops/sec

	Goal: Achieve 1M+ ops/sec with Go's concurrency advantages
*/

// Benchmark configuration
const (
	WarmupOperations = 10000
	BenchmarkSeconds = 10
	MaxGoroutines    = 1000
)

// Global counters for tracking
var (
	totalOperations int64
	totalErrors     int64
	benchmarkStart  time.Time
)

// === PURE PERFORMANCE BENCHMARKS ===

// BenchmarkBasicActionCall - Raw action call performance
func BenchmarkBasicActionCall(b *testing.B) {
	setupBenchmark(b)

	// Register ultra-minimal action
	cyre.Action(cyre.ActionConfig{
		ID: "perf-test",
	})

	// Ultra-minimal handler
	cyre.On("perf-test", func(payload interface{}) interface{} {
		return payload
	})

	// Warmup
	for i := 0; i < WarmupOperations; i++ {
		<-cyre.Call("perf-test", i)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			result := <-cyre.Call("perf-test", "benchmark")
			if !result.OK {
				b.Error("Call failed")
			}
			atomic.AddInt64(&totalOperations, 1)
		}
	})

	reportResults(b, "Basic Action Call")
}

// BenchmarkConcurrentLoad - Maximum concurrent throughput
func BenchmarkConcurrentLoad(b *testing.B) {
	setupBenchmark(b)

	// Setup multiple actions for load distribution
	numActions := 10
	for i := 0; i < numActions; i++ {
		actionID := fmt.Sprintf("concurrent-%d", i)
		cyre.Action(cyre.ActionConfig{ID: actionID})
		cyre.On(actionID, func(payload interface{}) interface{} {
			return fmt.Sprintf("processed-%v", payload)
		})
	}

	b.ResetTimer()
	b.SetParallelism(MaxGoroutines / runtime.NumCPU())

	b.RunParallel(func(pb *testing.PB) {
		actionIndex := 0
		for pb.Next() {
			actionID := fmt.Sprintf("concurrent-%d", actionIndex%numActions)
			result := <-cyre.Call(actionID, actionIndex)
			if !result.OK {
				atomic.AddInt64(&totalErrors, 1)
			}
			atomic.AddInt64(&totalOperations, 1)
			actionIndex++
		}
	})

	reportResults(b, "Concurrent Load")
}

// BenchmarkMemoryEfficiency - Memory-conscious performance
func BenchmarkMemoryEfficiency(b *testing.B) {
	setupBenchmark(b)

	cyre.Action(cyre.ActionConfig{ID: "memory-test"})
	cyre.On("memory-test", func(payload interface{}) interface{} {
		// Minimal memory allocation
		return true
	})

	// Force GC before benchmark
	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		result := <-cyre.Call("memory-test", nil)
		if !result.OK {
			b.Error("Call failed")
		}
	}

	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	allocPerOp := (m2.TotalAlloc - m1.TotalAlloc) / uint64(b.N)
	b.ReportMetric(float64(allocPerOp), "bytes/op")

	reportResults(b, "Memory Efficiency")
}

// === PROTECTION MECHANISM BENCHMARKS ===

// BenchmarkThrottlePerformance - Throttle protection overhead
func BenchmarkThrottlePerformance(b *testing.B) {
	setupBenchmark(b)

	throttleDuration := 1 * time.Microsecond // Minimal throttle
	cyre.Action(cyre.ActionConfig{
		ID:       "throttle-test",
		Throttle: &throttleDuration,
	})

	cyre.On("throttle-test", func(payload interface{}) interface{} {
		return "throttled"
	})

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			<-cyre.Call("throttle-test", "test")
			atomic.AddInt64(&totalOperations, 1)
		}
	})

	reportResults(b, "Throttle Performance")
}

// BenchmarkChangeDetectionPerformance - Change detection overhead
func BenchmarkChangeDetectionPerformance(b *testing.B) {
	setupBenchmark(b)

	cyre.Action(cyre.ActionConfig{
		ID:            "change-test",
		DetectChanges: true,
	})

	cyre.On("change-test", func(payload interface{}) interface{} {
		return "detected"
	})

	b.ResetTimer()

	// Alternate between same and different payloads
	payloads := []interface{}{"payload1", "payload2"}

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			payload := payloads[i%2]
			<-cyre.Call("change-test", payload)
			atomic.AddInt64(&totalOperations, 1)
			i++
		}
	})

	reportResults(b, "Change Detection Performance")
}

// === REAL-WORLD SCENARIO BENCHMARKS ===

// BenchmarkRealisticsWorkload - Real application patterns
func BenchmarkRealisticWorkload(b *testing.B) {
	setupBenchmark(b)

	// Setup realistic actions
	setupRealisticActions()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		scenarios := []func(){
			userLoginScenario,
			dataProcessingScenario,
			apiCallScenario,
			eventHandlingScenario,
		}

		i := 0
		for pb.Next() {
			scenario := scenarios[i%len(scenarios)]
			scenario()
			atomic.AddInt64(&totalOperations, 1)
			i++
		}
	})

	reportResults(b, "Realistic Workload")
}

// BenchmarkActionChaining - IntraLink performance
func BenchmarkActionChaining(b *testing.B) {
	setupBenchmark(b)

	// Setup 3-step chain
	cyre.Action(cyre.ActionConfig{ID: "chain-1"})
	cyre.Action(cyre.ActionConfig{ID: "chain-2"})
	cyre.Action(cyre.ActionConfig{ID: "chain-3"})

	cyre.On("chain-1", func(payload interface{}) interface{} {
		return map[string]interface{}{
			"id":      "chain-2",
			"payload": fmt.Sprintf("step1-%v", payload),
		}
	})

	cyre.On("chain-2", func(payload interface{}) interface{} {
		return map[string]interface{}{
			"id":      "chain-3",
			"payload": fmt.Sprintf("step2-%v", payload),
		}
	})

	cyre.On("chain-3", func(payload interface{}) interface{} {
		return fmt.Sprintf("final-%v", payload)
	})

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		result := <-cyre.Call("chain-1", i)
		if !result.OK {
			b.Error("Chain failed")
		}
	}

	reportResults(b, "Action Chaining")
}

// === STRESS TESTS ===

// BenchmarkExtremeConcurrency - Push concurrency limits
func BenchmarkExtremeConcurrency(b *testing.B) {
	setupBenchmark(b)

	cyre.Action(cyre.ActionConfig{ID: "extreme-test"})
	cyre.On("extreme-test", func(payload interface{}) interface{} {
		// Simulate minimal work
		time.Sleep(1 * time.Microsecond)
		return payload
	})

	// Unleash the goroutines!
	concurrency := runtime.NumCPU() * 100

	b.ResetTimer()

	var wg sync.WaitGroup
	operationsPerGoroutine := b.N / concurrency

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				result := <-cyre.Call("extreme-test", fmt.Sprintf("%d-%d", id, j))
				if !result.OK {
					atomic.AddInt64(&totalErrors, 1)
				}
				atomic.AddInt64(&totalOperations, 1)
			}
		}(i)
	}

	wg.Wait()
	reportResults(b, "Extreme Concurrency")
}

// === THROUGHPUT MEASUREMENT ===

// BenchmarkThroughputMeasurement - Sustained throughput over time
func BenchmarkThroughputMeasurement(b *testing.B) {
	setupBenchmark(b)

	cyre.Action(cyre.ActionConfig{ID: "throughput-test"})
	cyre.On("throughput-test", func(payload interface{}) interface{} {
		return payload
	})

	// Run for fixed time period
	duration := time.Duration(BenchmarkSeconds) * time.Second
	start := time.Now()
	operations := int64(0)

	b.ResetTimer()

	// Launch multiple goroutines for sustained load
	concurrency := runtime.NumCPU() * 10
	var wg sync.WaitGroup

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter := 0
			for time.Since(start) < duration {
				result := <-cyre.Call("throughput-test", counter)
				if result.OK {
					atomic.AddInt64(&operations, 1)
				}
				counter++
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)

	opsPerSec := float64(operations) / elapsed.Seconds()
	b.ReportMetric(opsPerSec, "ops/sec")

	fmt.Printf("\n🚀 SUSTAINED THROUGHPUT: %.0f ops/sec over %v\n", opsPerSec, elapsed)
	fmt.Printf("📊 Total Operations: %d\n", operations)
	fmt.Printf("⚡ Peak Performance: %.0f ops/sec\n", opsPerSec)

	// Compare with other implementations
	fmt.Printf("\n📈 PERFORMANCE COMPARISON:\n")
	fmt.Printf("   TypeScript Cyre: ~400,000 ops/sec\n")
	fmt.Printf("   Rust Cyre:       ~1,000,000 ops/sec\n")
	fmt.Printf("   Go Cyre:         ~%.0f ops/sec\n", opsPerSec)

	if opsPerSec > 1000000 {
		fmt.Printf("🏆 WINNER! Go Cyre beats Rust Cyre!\n")
	} else if opsPerSec > 400000 {
		fmt.Printf("🥈 Great! Go Cyre beats TypeScript Cyre!\n")
	} else {
		fmt.Printf("📈 Room for optimization!\n")
	}
}

// === HELPER FUNCTIONS ===

func setupBenchmark(b *testing.B) {
	// Reset counters
	atomic.StoreInt64(&totalOperations, 0)
	atomic.StoreInt64(&totalErrors, 0)
	benchmarkStart = time.Now()

	// Initialize Cyre fresh for each benchmark
	cyre.Clear()
	result := cyre.Initialize()
	if !result.OK {
		b.Fatal("Failed to initialize Cyre")
	}

	// Ensure system is ready
	if !cyre.IsHealthy() {
		b.Fatal("System not healthy")
	}

	runtime.GC() // Start with clean memory
}

func reportResults(b *testing.B, testName string) {
	ops := atomic.LoadInt64(&totalOperations)
	errors := atomic.LoadInt64(&totalErrors)
	elapsed := time.Since(benchmarkStart)

	if elapsed > 0 {
		opsPerSec := float64(ops) / elapsed.Seconds()
		b.ReportMetric(opsPerSec, "ops/sec")

		errorRate := float64(errors) / float64(ops) * 100
		b.ReportMetric(errorRate, "error_%")

		if ops > 0 {
			avgLatency := elapsed / time.Duration(ops)
			b.ReportMetric(float64(avgLatency.Nanoseconds()), "ns/op")
		}
	}
}

// === REALISTIC SCENARIO HELPERS ===

func setupRealisticActions() {
	// User authentication
	cyre.Action(cyre.ActionConfig{
		ID:       "user-auth",
		Throttle: cyre.ThrottleDuration(10 * time.Millisecond),
	})
	cyre.On("user-auth", func(payload interface{}) interface{} {
		return map[string]interface{}{"authenticated": true, "userId": "user123"}
	})

	// Data processing
	cyre.Action(cyre.ActionConfig{
		ID:            "data-process",
		DetectChanges: true,
	})
	cyre.On("data-process", func(payload interface{}) interface{} {
		return map[string]interface{}{"processed": true, "data": payload}
	})

	// API call simulation
	cyre.Action(cyre.ActionConfig{
		ID:       "api-call",
		Throttle: cyre.ThrottleDuration(5 * time.Millisecond),
	})
	cyre.On("api-call", func(payload interface{}) interface{} {
		return map[string]interface{}{"response": "OK", "data": payload}
	})

	// Event handling
	cyre.Action(cyre.ActionConfig{ID: "event-handle"})
	cyre.On("event-handle", func(payload interface{}) interface{} {
		return map[string]interface{}{"handled": true, "event": payload}
	})
}

func userLoginScenario() {
	<-cyre.Call("user-auth", map[string]interface{}{
		"username": "testuser",
		"password": "password123",
	})
}

func dataProcessingScenario() {
	<-cyre.Call("data-process", []int{1, 2, 3, 4, 5})
}

func apiCallScenario() {
	<-cyre.Call("api-call", map[string]interface{}{
		"endpoint": "/api/data",
		"method":   "GET",
	})
}

func eventHandlingScenario() {
	<-cyre.Call("event-handle", map[string]interface{}{
		"type":      "user_action",
		"timestamp": time.Now().Unix(),
	})
}
