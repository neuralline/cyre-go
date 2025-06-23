// benchmark/stress_test.go
// Stress test to trigger breathing system and intelligent worker adjustment
// Tests if MetricState brain actually adjusts workers and manages stress

package main

import (
	"fmt"
	"log"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/neuralline/cyre-go"
	"github.com/neuralline/cyre-go/types"
)

func main() {
	fmt.Println("🧠 CYRE GO - BREATHING SYSTEM & WORKER INTELLIGENCE TEST")
	fmt.Println("========================================================")
	fmt.Printf("🔧 System Info: %d CPU cores, %d goroutines initially\n", runtime.NumCPU(), runtime.NumGoroutine())

	// Initialize Cyre
	result := cyre.Init()
	if !result.OK {
		log.Fatal("❌ Failed to initialize Cyre:", result.Error)
	}
	fmt.Printf("✅ Cyre initialized: %s\n", result.Message)

	// Register different types of actions to create varied load
	actions := []struct {
		id       string
		workType string
		delay    time.Duration
	}{
		{"cpu-intensive", "CPU", 10 * time.Millisecond},
		{"memory-intensive", "Memory", 5 * time.Millisecond},
		{"io-intensive", "IO", 15 * time.Millisecond},
		{"quick-task", "Quick", 1 * time.Millisecond},
		{"slow-task", "Slow", 50 * time.Millisecond},
	}

	fmt.Println("\n📋 Registering varied workload actions...")
	for _, action := range actions {
		err := cyre.Action(types.IO{
			ID:       action.id,
			Type:     "stress-test",
			Priority: "medium", // Fixed: use valid priority
		})
		if err != nil {
			log.Printf("❌ Failed to register %s: %v", action.id, err)
			continue
		}

		// Subscribe with work simulation
		cyre.On(action.id, createWorkloadHandler(action.workType, action.delay))
		fmt.Printf("   ✅ %s (%s work)\n", action.id, action.workType)
	}

	// Create a high-priority action for emergency testing
	cyre.Action(types.IO{
		ID:       "emergency-action",
		Type:     "emergency",
		Priority: "critical",
	})
	cyre.On("emergency-action", func(payload interface{}) interface{} {
		return map[string]interface{}{
			"status":    "emergency_handled",
			"timestamp": time.Now().Unix(),
		}
	})
	fmt.Println("   🚨 emergency-action (critical priority)")

	// Initial system state
	fmt.Println("\n📊 Initial System State:")
	printSystemMetrics(cyre, "INITIAL")

	// Phase 1: Light Load - Should not trigger breathing
	fmt.Println("\n🟢 PHASE 1: Light Load (Baseline)")
	fmt.Println("================================")
	runLoadTest(cyre, actions, 100, 1, "Light Load")
	time.Sleep(2 * time.Second)
	printSystemMetrics(cyre, "LIGHT LOAD")

	// Phase 2: Medium Load - Should start showing stress
	fmt.Println("\n🟡 PHASE 2: Medium Load (Building Stress)")
	fmt.Println("=========================================")
	runLoadTest(cyre, actions, 500, 5, "Medium Load")
	time.Sleep(2 * time.Second)
	printSystemMetrics(cyre, "MEDIUM LOAD")

	// Phase 3: Heavy Load - Should trigger breathing system
	fmt.Println("\n🔴 PHASE 3: Heavy Load (Trigger Breathing)")
	fmt.Println("==========================================")
	runLoadTest(cyre, actions, 2000, 20, "Heavy Load")
	time.Sleep(3 * time.Second)
	printSystemMetrics(cyre, "HEAVY LOAD")

	// Phase 4: Extreme Load - Should trigger recuperation
	fmt.Println("\n💥 PHASE 4: Extreme Load (Force Recuperation)")
	fmt.Println("==============================================")
	runLoadTest(cyre, actions, 5000, 50, "Extreme Load")
	time.Sleep(5 * time.Second)
	printSystemMetrics(cyre, "EXTREME LOAD")

	// Phase 5: Test emergency action during stress
	fmt.Println("\n🚨 PHASE 5: Emergency Action During Stress")
	fmt.Println("==========================================")
	testEmergencyDuringStress(cyre)

	// Phase 6: Recovery - Let system cool down
	fmt.Println("\n🌊 PHASE 6: Recovery Period (System Breathing)")
	fmt.Println("==============================================")
	fmt.Println("   Letting system recover for 10 seconds...")
	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)
		if i%2 == 0 {
			metrics := cyre.GetMetrics()
			if flags, ok := metrics["flags"].(map[string]interface{}); ok {
				fmt.Printf("   Recovery %d/10: recuperating=%v, stress=%.3f\n",
					i+1, flags["isRecuperating"], getStressLevel(metrics))
			}
		}
	}
	printSystemMetrics(cyre, "RECOVERY")

	// Phase 7: Worker intelligence test
	fmt.Println("\n🤖 PHASE 7: Worker Intelligence Test")
	fmt.Println("=====================================")
	testWorkerIntelligence(cyre, actions)

	// Final analysis
	fmt.Println("\n📈 FINAL ANALYSIS: Breathing System Performance")
	fmt.Println("===============================================")
	analyzeBreathingSystem(cyre)

	// Cleanup
	fmt.Println("\n🧹 Cleanup and Shutdown")
	fmt.Println("=======================")
	cyre.Shutdown()
	fmt.Println("✅ System shutdown completed")
}

// createWorkloadHandler creates handlers that simulate different types of work
func createWorkloadHandler(workType string, delay time.Duration) func(interface{}) interface{} {
	return func(payload interface{}) interface{} {
		start := time.Now()

		switch workType {
		case "CPU":
			// CPU-intensive work simulation
			sum := 0
			for i := 0; i < 100000; i++ {
				sum += i * i
			}
		case "Memory":
			// Memory allocation simulation
			data := make([]byte, 1024*100) // 100KB
			for i := range data {
				data[i] = byte(i % 256)
			}
		case "IO":
			// IO simulation with sleep
			time.Sleep(delay)
		case "Quick":
			// Minimal work
			time.Sleep(delay)
		case "Slow":
			// Longer work simulation
			time.Sleep(delay)
		}

		return map[string]interface{}{
			"workType":  workType,
			"duration":  time.Since(start).Milliseconds(),
			"timestamp": time.Now().Unix(),
		}
	}
}

// runLoadTest executes a load test with specified parameters
func runLoadTest(cyre *cyre.Cyre, actions []struct {
	id       string
	workType string
	delay    time.Duration
}, totalCalls int, concurrency int, phaseName string) {
	fmt.Printf("   🚀 Executing %d calls with concurrency %d\n", totalCalls, concurrency)

	start := time.Now()
	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64

	// Limit concurrent goroutines
	semaphore := make(chan struct{}, concurrency)

	for i := 0; i < totalCalls; i++ {
		wg.Add(1)
		go func(callNum int) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Pick random action
			action := actions[callNum%len(actions)]

			result := <-cyre.Call(action.id, map[string]interface{}{
				"call":  callNum,
				"phase": phaseName,
			})

			if result.OK {
				atomic.AddInt64(&successCount, 1)
			} else {
				atomic.AddInt64(&errorCount, 1)
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	successful := atomic.LoadInt64(&successCount)
	failed := atomic.LoadInt64(&errorCount)
	opsPerSec := float64(totalCalls) / duration.Seconds()

	fmt.Printf("   📊 Results: %d successful, %d failed, %.0f ops/sec\n",
		successful, failed, opsPerSec)
	fmt.Printf("   ⏱️  Duration: %v\n", duration.Round(time.Millisecond))
}

// testEmergencyDuringStress tests critical priority actions during system stress
func testEmergencyDuringStress(cyre *cyre.Cyre) {
	fmt.Println("   Testing emergency action during high stress...")

	// Create background stress
	go func() {
		for i := 0; i < 1000; i++ {
			cyre.Call("slow-task", map[string]interface{}{"background": i})
			time.Sleep(1 * time.Millisecond)
		}
	}()

	// Test emergency action
	start := time.Now()
	result := <-cyre.Call("emergency-action", map[string]interface{}{
		"emergency": "system_critical",
		"priority":  "maximum",
	})
	duration := time.Since(start)

	if result.OK {
		fmt.Printf("   ✅ Emergency action succeeded in %v\n", duration.Round(time.Microsecond))
	} else {
		fmt.Printf("   ❌ Emergency action failed: %s\n", result.Message)
	}
}

// testWorkerIntelligence tests if the system intelligently adjusts workers
func testWorkerIntelligence(cyre *cyre.Cyre, actions []struct {
	id       string
	workType string
	delay    time.Duration
}) {
	fmt.Println("   Testing worker intelligence and adaptation...")

	metrics := cyre.GetMetrics()
	if workers, ok := metrics["workers"].(map[string]interface{}); ok {
		fmt.Printf("   Initial workers: current=%v, optimal=%v, sweetSpot=%v\n",
			workers["current"], workers["optimal"], workers["sweetSpot"])
	}

	// Progressive load increase to trigger worker scaling
	loads := []int{100, 500, 1000, 2000}
	for _, load := range loads {
		fmt.Printf("   Testing with %d concurrent calls...\n", load)
		runLoadTest(cyre, actions, load, 10, fmt.Sprintf("Intelligence-%d", load))

		// Check if workers adapted
		time.Sleep(1 * time.Second)
		newMetrics := cyre.GetMetrics()
		if workers, ok := newMetrics["workers"].(map[string]interface{}); ok {
			fmt.Printf("     Workers now: current=%v, optimal=%v, sweetSpot=%v\n",
				workers["current"], workers["optimal"], workers["sweetSpot"])
		}
	}
}

// printSystemMetrics displays comprehensive system metrics
func printSystemMetrics(cyre *cyre.Cyre, phase string) {
	metrics := cyre.GetMetrics()

	fmt.Printf("📊 %s SYSTEM METRICS:\n", phase)
	fmt.Printf("──────────────────────────────────────\n")

	// Performance metrics
	if perf, ok := metrics["performance"].(map[string]interface{}); ok {
		fmt.Printf("🚀 Performance:\n")
		fmt.Printf("   Throughput: %.1f ops/sec\n", getFloat(perf, "throughputPerSec"))
		fmt.Printf("   Avg Latency: %.2f ms\n", getFloat(perf, "avgLatencyMs"))
		fmt.Printf("   Error Rate: %.3f%%\n", getFloat(perf, "errorRate")*100)
		fmt.Printf("   Queue Depth: %.0f\n", getFloat(perf, "queueDepth"))
		fmt.Printf("   Calls/sec: %.0f\n", getFloat(perf, "callsPerSecond"))
	}

	// Worker metrics
	if workers, ok := metrics["workers"].(map[string]interface{}); ok {
		fmt.Printf("👥 Workers:\n")
		fmt.Printf("   Current: %.0f\n", getFloat(workers, "current"))
		fmt.Printf("   Optimal: %.0f\n", getFloat(workers, "optimal"))
		fmt.Printf("   Sweet Spot Found: %v\n", workers["sweetSpot"])
	}

	// Health metrics
	if health, ok := metrics["health"].(map[string]interface{}); ok {
		fmt.Printf("🏥 Health:\n")
		fmt.Printf("   CPU: %.1f%%\n", getFloat(health, "cpuPercent"))
		fmt.Printf("   Memory: %.1f%%\n", getFloat(health, "memoryPercent"))
		fmt.Printf("   Goroutines: %.0f\n", getFloat(health, "goroutineCount"))
		fmt.Printf("   GC Pressure: %v\n", health["gcPressure"])
	}

	// Breathing system
	if breathing, ok := metrics["breathing"].(map[string]interface{}); ok {
		fmt.Printf("🫁 Breathing System:\n")
		fmt.Printf("   Stress Level: %.3f (%.1f%%)\n",
			getFloat(breathing, "stressLevel"), getFloat(breathing, "stressLevel")*100)
		fmt.Printf("   Phase: %v\n", breathing["phase"])
		fmt.Printf("   Recuperating: %v\n", breathing["isRecuperating"])
		fmt.Printf("   Block Normal: %v\n", breathing["blockNormal"])
		fmt.Printf("   Block Low: %v\n", breathing["blockLow"])
	}

	// System flags
	if flags, ok := metrics["flags"].(map[string]interface{}); ok {
		fmt.Printf("🏁 System Flags:\n")
		fmt.Printf("   Locked: %v\n", flags["isLocked"])
		fmt.Printf("   Hibernating: %v\n", flags["isHibernating"])
		fmt.Printf("   Shutdown: %v\n", flags["isShutdown"])
		fmt.Printf("   Recuperating: %v\n", flags["isRecuperating"])
	}

	fmt.Printf("   Runtime Goroutines: %d\n", runtime.NumGoroutine())
	fmt.Println()
}

// analyzeBreathingSystem provides analysis of breathing system effectiveness
func analyzeBreathingSystem(cyre *cyre.Cyre) {
	metrics := cyre.GetMetrics()

	stressLevel := getStressLevel(metrics)

	fmt.Printf("🧠 Breathing System Analysis:\n")
	fmt.Printf("──────────────────────────────\n")

	if stressLevel > 0.8 {
		fmt.Println("✅ System successfully detected high stress and triggered breathing")
	} else if stressLevel > 0.5 {
		fmt.Println("✅ System detected moderate stress and applied protections")
	} else {
		fmt.Println("ℹ️  System remained stable - breathing system on standby")
	}

	if flags, ok := metrics["flags"].(map[string]interface{}); ok {
		if flags["isRecuperating"].(bool) {
			fmt.Println("✅ Recuperation mode activated - system is intelligently recovering")
		}
	}

	if workers, ok := metrics["workers"].(map[string]interface{}); ok {
		if workers["sweetSpot"].(bool) {
			fmt.Printf("🎯 Optimal worker count discovered: %.0f workers\n", getFloat(workers, "optimal"))
		} else {
			fmt.Println("🔍 Still searching for optimal worker configuration")
		}
	}

	fmt.Printf("📊 Final stress level: %.3f (%.1f%%)\n", stressLevel, stressLevel*100)

	// Recommendations
	fmt.Printf("\n💡 System Intelligence Recommendations:\n")
	if stressLevel > 0.7 {
		fmt.Println("   • High stress detected - consider scaling infrastructure")
		fmt.Println("   • Breathing system actively protecting performance")
	} else if stressLevel > 0.3 {
		fmt.Println("   • Moderate load - system handling well")
		fmt.Println("   • Breathing system monitoring for optimization opportunities")
	} else {
		fmt.Println("   • Low stress - system running efficiently")
		fmt.Println("   • Breathing system in optimal monitoring mode")
	}
}

// Helper functions
func getFloat(data map[string]interface{}, key string) float64 {
	if val, ok := data[key]; ok {
		switch v := val.(type) {
		case float64:
			return v
		case int:
			return float64(v)
		case int64:
			return float64(v)
		}
	}
	return 0.0
}

func getStressLevel(metrics map[string]interface{}) float64 {
	if breathing, ok := metrics["breathing"].(map[string]interface{}); ok {
		return getFloat(breathing, "stressLevel")
	}
	return 0.0
}
