// simple_breathing_test.go
// Test the breathing system with proper timing

package main

import (
	"fmt"
	"time"

	cyre "github.com/neuralline/cyre-go"
)

func main() {
	fmt.Println("🫁 BREATHING SYSTEM TEST - Proper Timing")
	fmt.Println("=========================================")

	// Initialize Cyre
	result := cyre.Init()
	if !result.OK {
		fmt.Printf("❌ Failed to initialize: %v\n", result.Error)
		return
	}
	fmt.Println("✅ Cyre initialized")

	// Register a simple action to create some activity
	err := cyre.Action(cyre.IO{
		ID: "test-action",
	})
	if err != nil {
		fmt.Printf("❌ Failed to register action: %v\n", err)
		return
	}

	cyre.On("test-action", func(payload interface{}) interface{} {
		return fmt.Sprintf("Response: %v", payload)
	})

	fmt.Println("✅ Action registered")

	fmt.Println("\n⏱️  Waiting for breathing system to activate...")
	fmt.Println("   Expected timeline:")
	fmt.Println("   • 0-2s: Breathing goroutine delay")
	fmt.Println("   • 2-4s: Ticker initialization")
	fmt.Println("   • 4s+: First breathing tick!")

	// Wait and monitor for 8 seconds to see breathing in action
	for i := 1; i <= 8; i++ {
		fmt.Printf("\n⏰ Second %d/8:\n", i)

		// Make some calls to create system activity
		if i >= 3 {
			for j := 0; j < 5; j++ {
				result := <-cyre.Call("test-action", fmt.Sprintf("call-%d-%d", i, j))
				if !result.OK {
					fmt.Printf("   Call failed: %s\n", result.Message)
				}
			}
		}

		// Sleep for 1 second
		time.Sleep(1 * time.Second)

		// Check if we're past the breathing system startup time
		if i >= 5 {
			fmt.Println("   🔍 Breathing system should be active now...")
			fmt.Println("   Looking for debug output above...")
		}
	}

	fmt.Println("\n📊 Final system check:")

	// The key insight: by now we should have seen:
	// DEBUG: Breathing tick X
	// DEBUG: updateBreathingFromMetrics() called
	// DEBUG: Health readings - CPU:X%, Memory:Y%, Goroutines:Z

	fmt.Println("✅ If you see 'DEBUG: Breathing tick' messages above,")
	fmt.Println("   the breathing system is working correctly!")

	fmt.Println("\n🏁 Test complete")
	cyre.Shutdown()
}
