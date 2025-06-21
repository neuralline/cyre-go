// cmd/quick-bench/main.go
// Quick performance test to see Cyre Go's raw speed

package main

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/neuralline/cyre-go"
)

func main() {
	fmt.Println("🚀 CYRE GO QUICK PERFORMANCE TEST")
	fmt.Println("=================================")
	fmt.Printf("💻 System: %d CPU cores\n", runtime.NumCPU())
	fmt.Println()

	// Initialize Cyre
	result := cyre.Initialize()
	if !result.OK {
		fmt.Printf("❌ Failed to initialize: %s\n", result.Message)
		return
	}

	// Setup ultra-fast action
	cyre.Action(cyre.ActionConfig{ID: "speed-test"})
	cyre.On("speed-test", func(payload interface{}) interface{} {
		return payload // Fastest possible
	})

	// Test parameters
	duration := 5 * time.Second
	concurrency := runtime.NumCPU() * 4

	fmt.Printf("⏱️  Test Duration: %v\n", duration)
	fmt.Printf("🔀 Concurrency: %d goroutines\n", concurrency)
	fmt.Println()

	// Warmup
	fmt.Print("🔄 Warming up...")
	for i := 0; i < 10000; i++ {
		<-cyre.Call("speed-test", i)
	}
	fmt.Println(" Done")

	// Main test
	var operations int64
	var errors int64

	fmt.Print("⚡ Testing...")
	start := time.Now()
	end := start.Add(duration)

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			counter := workerID
			for time.Now().Before(end) {
				result := <-cyre.Call("speed-test", counter)
				if result.OK {
					atomic.AddInt64(&operations, 1)
				} else {
					atomic.AddInt64(&errors, 1)
				}
				counter += concurrency
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
				fmt.Printf("\r⚡ Testing... %.0f ops/sec", float64(ops)/elapsed)
			case <-time.After(duration + time.Second):
				return
			}
		}
	}()

	wg.Wait()
	elapsed := time.Since(start)
	totalOps := atomic.LoadInt64(&operations)
	totalErrors := atomic.LoadInt64(&errors)

	fmt.Println(" Done!")
	fmt.Println()

	// Results
	opsPerSec := float64(totalOps) / elapsed.Seconds()
	errorRate := float64(totalErrors) / float64(totalOps+totalErrors) * 100

	fmt.Println("📊 RESULTS")
	fmt.Println("==========")
	fmt.Printf("Total Operations: %d\n", totalOps)
	fmt.Printf("Total Errors:     %d\n", totalErrors)
	fmt.Printf("Duration:         %v\n", elapsed)
	fmt.Printf("Throughput:       %.0f ops/sec\n", opsPerSec)
	fmt.Printf("Error Rate:       %.3f%%\n", errorRate)
	fmt.Printf("Avg Latency:      %.2fμs\n", float64(elapsed.Nanoseconds())/float64(totalOps)/1000)
	fmt.Println()

	// Competition comparison
	fmt.Println("🏁 PERFORMANCE BATTLE")
	fmt.Println("====================")
	fmt.Printf("TypeScript Cyre:  ~400,000 ops/sec\n")
	fmt.Printf("Rust Cyre:        ~1,000,000 ops/sec\n")
	fmt.Printf("Go Cyre:          ~%.0f ops/sec\n", opsPerSec)
	fmt.Println()

	if opsPerSec > 1500000 {
		fmt.Println("🏆🏆🏆 LEGENDARY! Go Cyre CRUSHES the competition!")
		fmt.Printf("🚀 %.1fx faster than Rust!\n", opsPerSec/1000000)
	} else if opsPerSec > 1000000 {
		fmt.Println("🏆🏆 CHAMPION! Go Cyre beats Rust!")
		fmt.Printf("🚀 %.1fx faster than Rust!\n", opsPerSec/1000000)
	} else if opsPerSec > 800000 {
		fmt.Println("🏆 EXCELLENT! Very close to Rust performance!")
		fmt.Printf("🚀 %.1fx faster than TypeScript!\n", opsPerSec/400000)
	} else if opsPerSec > 400000 {
		fmt.Println("🥈 GOOD! Beats TypeScript!")
		fmt.Printf("🚀 %.1fx faster than TypeScript!\n", opsPerSec/400000)
	} else {
		fmt.Println("📈 Solid foundation with room for optimization!")
	}

	// System health check
	fmt.Println()
	fmt.Printf("🏥 System Healthy: %t\n", cyre.IsHealthy())
	fmt.Printf("📈 Goroutines: %d\n", runtime.NumGoroutine())

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Printf("💾 Memory: %.2f MB\n", float64(m.Alloc)/1024/1024)

	fmt.Println()
	fmt.Println("🎯 Ready for .action talents!")
}
