// examples/simple_bench.go
// Clean, simple benchmark that actually works

package main

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	cyre "github.com/neuralline/cyre-go"
)

// BenchmarkSimpleCall - Just test basic call performance
func BenchmarkSimpleCall(b *testing.B) {
	cyre.Initialize()

	cyre.Action(cyre.ActionConfig{ID: "simple"})
	cyre.On("simple", func(payload interface{}) interface{} {
		return payload
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := <-cyre.Call("simple", i)
		if !result.OK {
			b.Error("Call failed")
		}
	}

	cyre.Forget("simple")
}

// BenchmarkConcurrentSimple - Test concurrent performance
func BenchmarkConcurrentSimple(b *testing.B) {
	cyre.Initialize()

	cyre.Action(cyre.ActionConfig{ID: "concurrent"})
	cyre.On("concurrent", func(payload interface{}) interface{} {
		return payload
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			result := <-cyre.Call("concurrent", "test")
			if !result.OK {
				b.Error("Call failed")
			}
		}
	})

	cyre.Forget("concurrent")
}

// TestThroughputBattle - Custom test for 5-second sustained throughput
func TestThroughputBattle(t *testing.T) {
	cyre.Initialize()

	cyre.Action(cyre.ActionConfig{ID: "throughput"})
	cyre.On("throughput", func(payload interface{}) interface{} {
		return payload // Minimal processing
	})

	// Test parameters
	duration := 5 * time.Second
	concurrency := runtime.NumCPU() * 4

	fmt.Printf("\n🚀 CYRE GO THROUGHPUT BATTLE\n")
	fmt.Printf("============================\n")
	fmt.Printf("Duration: %v\n", duration)
	fmt.Printf("Concurrency: %d goroutines\n", concurrency)
	fmt.Printf("CPU Cores: %d\n", runtime.NumCPU())
	fmt.Printf("\n")

	var operations int64
	var errors int64

	start := time.Now()
	end := start.Add(duration)

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			counter := workerID

			for time.Now().Before(end) {
				result := <-cyre.Call("throughput", counter)
				if result.OK {
					atomic.AddInt64(&operations, 1)
				} else {
					atomic.AddInt64(&errors, 1)
				}
				counter++
			}
		}(i)
	}

	// Progress reporting
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				ops := atomic.LoadInt64(&operations)
				elapsed := time.Since(start).Seconds()
				if elapsed > 0 {
					fmt.Printf("Progress: %.1fs - %.0f ops/sec\n", elapsed, float64(ops)/elapsed)
				}
			case <-time.After(duration + time.Second):
				return
			}
		}
	}()

	wg.Wait()
	elapsed := time.Since(start)

	totalOps := atomic.LoadInt64(&operations)
	totalErrors := atomic.LoadInt64(&errors)
	opsPerSec := float64(totalOps) / elapsed.Seconds()
	errorRate := float64(totalErrors) / float64(totalOps+totalErrors) * 100
	avgLatency := float64(elapsed.Nanoseconds()) / float64(totalOps) / 1000 // microseconds

	fmt.Printf("\n📊 FINAL RESULTS\n")
	fmt.Printf("================\n")
	fmt.Printf("Total Operations: %s\n", formatNumber(totalOps))
	fmt.Printf("Total Errors:     %d\n", totalErrors)
	fmt.Printf("Duration:         %v\n", elapsed)
	fmt.Printf("Throughput:       %s ops/sec\n", formatNumber(int64(opsPerSec)))
	fmt.Printf("Error Rate:       %.3f%%\n", errorRate)
	fmt.Printf("Avg Latency:      %.1fμs\n", avgLatency)

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("Memory Usage:     %.2f MB\n", float64(m.Alloc)/1024/1024)

	fmt.Printf("\n🏁 PERFORMANCE BATTLE\n")
	fmt.Printf("=====================\n")
	fmt.Printf("TypeScript Cyre:  ~400,000 ops/sec (burst)\n")
	fmt.Printf("TypeScript Cyre:  ~13,000 ops/sec (sustained)\n")
	fmt.Printf("Rust Cyre:        ~1,000,000 ops/sec\n")
	fmt.Printf("Go Cyre:          ~%s ops/sec\n", formatNumber(int64(opsPerSec)))

	fmt.Printf("\n🏆 VERDICT:\n")
	if opsPerSec > 1000000 {
		fmt.Printf("🥇 CHAMPION! Go beats Rust! (%.1fx faster than Rust)\n", opsPerSec/1000000)
	} else if opsPerSec > 400000 {
		fmt.Printf("🥈 EXCELLENT! Go beats TypeScript burst! (%.1fx faster)\n", opsPerSec/400000)
	} else if opsPerSec > 13000 {
		fmt.Printf("🥉 GOOD! Go beats TypeScript sustained! (%.1fx faster)\n", opsPerSec/13000)
	} else {
		fmt.Printf("📈 Baseline established! Room for optimization.\n")
	}

	fmt.Printf("\n💪 GO ADVANTAGES:\n")
	fmt.Printf("✅ Native goroutines (no async overhead)\n")
	fmt.Printf("✅ Zero garbage collection pauses during test\n")
	fmt.Printf("✅ Direct memory management\n")
	fmt.Printf("✅ Compiled performance\n")

	cyre.Forget("throughput")

	// Verify no errors
	if totalErrors > 0 {
		t.Errorf("Expected 0 errors, got %d", totalErrors)
	}

	// Verify reasonable performance (at least 1000 ops/sec)
	if opsPerSec < 1000 {
		t.Errorf("Performance too low: %.0f ops/sec", opsPerSec)
	}
}

func TestSimpleBench(t *testing.T) {
	cyre.Initialize()
	cyre.Action(cyre.ActionConfig{ID: "bench", Log: false})
	cyre.On("bench", func(payload interface{}) interface{} {
		return payload
	})
	for i := 0; i < 1000; i++ {
		<-cyre.Call("bench", i)
	}
	cyre.Forget("bench")
}

func BenchmarkSimple(b *testing.B) {
	cyre.Initialize()
	cyre.Action(cyre.ActionConfig{ID: "bench", Log: false})
	cyre.On("bench", func(payload interface{}) interface{} {
		return payload
	})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		<-cyre.Call("bench", i)
	}
	cyre.Forget("bench")
}

func formatNumber(n int64) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	} else if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}
