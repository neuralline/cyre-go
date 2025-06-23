// examples/repeat/go

package main

import (
	"fmt"
	"time"

	"github.com/neuralline/cyre-go"
)

func main() {
	//cyre.Initialize()
	fmt.Println("🕐 SCHEDULER EXAMPLES - Repeat, Delay, Interval")
	fmt.Println("===============================================")

	// Initialize Cyre
	// result := cyre.Init()
	// if !result.OK {
	// 	panic("Failed to initialize Cyre")
	// }

	// Example 1: Interval only (every 2 seconds, infinite)
	fmt.Println("📋 Example 1: Health Check (every 2 seconds, infinite)")
	cyre.Action(cyre.ActionConfig{
		ID:       "health-check",
		Interval: 2000, // 2 seconds
		Repeat:   5,
		// No repeat specified = infinite
	})

	cyre.On("health-check", func(payload interface{}) interface{} {
		fmt.Printf("⚡ Health check at %s\n", time.Now().Format("15:04:05"))
		return "healthy"
	})

	// Example 2: Delay + Interval + Repeat (start after 3s, then every 1s, 5 times)
	fmt.Println("📋 Example 2: Delayed Notifications (3s delay, then every 1s, 5 times)")
	cyre.Action(cyre.ActionConfig{
		ID:       "delayed-notifications",
		Delay:    3000, // Start after 3 seconds
		Interval: 1000, // Then every 1 second
		Repeat:   5,    // Only 5 times total
	})

	cyre.On("delayed-notifications", func(payload interface{}) interface{} {
		fmt.Printf("🔔 Notification sent at %s\n", time.Now().Format("15:04:05"))
		return "sent"
	})

	// Example 3: Delay only (one-time execution after delay)
	fmt.Println("📋 Example 3: One-time Delayed Task (execute once after 5 seconds)")
	cyre.Action(cyre.ActionConfig{
		ID:    "delayed-task",
		Delay: 5000, // Execute once after 5 seconds
		// No interval = single execution
		// No repeat = single execution
	})

	cyre.On("delayed-task", func(payload interface{}) interface{} {
		fmt.Printf("🎯 Delayed task executed at %s\n", time.Now().Format("15:04:05"))
		return "completed"
	})

	// Example 4: Repeat with interval (every 500ms, exactly 10 times)
	fmt.Println("📋 Example 4: Burst Mode (every 500ms, exactly 10 times)")
	cyre.Action(cyre.ActionConfig{
		ID:       "burst-mode",
		Interval: 500, // Every 500ms
		Repeat:   10,  // Exactly 10 times
	})

	cyre.On("burst-mode", func(payload interface{}) interface{} {
		fmt.Printf("💥 Burst execution at %s\n", time.Now().Format("15:04:05"))
		return "burst"
	})

	// Example 5: Complex scheduling (2s delay, then every 3s, infinite)
	fmt.Println("📋 Example 5: Background Sync (2s delay, then every 3s, infinite)")
	cyre.Action(cyre.ActionConfig{
		ID:       "background-sync",
		Delay:    2000, // Start after 2 seconds
		Interval: 3000, // Then every 3 seconds
		Repeat:   10,   // Infinite (explicit)
	})

	cyre.On("background-sync", func(payload interface{}) interface{} {
		fmt.Printf("🔄 Background sync at %s\n", time.Now().Format("15:04:05"))
		return "synced"
	})

	// Start all scheduled actions by calling them once
	fmt.Println("\n🚀 Starting all scheduled actions...")

	cyre.Call("health-check", nil)
	cyre.Call("delayed-notifications", nil)
	cyre.Call("delayed-task", nil)
	cyre.Call("burst-mode", nil)
	cyre.Call("background-sync", nil)

	fmt.Println("\n⏰ Watching scheduled executions for 15 seconds...")
	fmt.Printf("Started at: %s\n", time.Now().Format("15:04:05"))

	// Let it run for 15 seconds to see the scheduling in action
	time.Sleep(15 * time.Second)

	fmt.Println("\n🛑 Stopping scheduled actions...")

	// Clean up
	cyre.Forget("health-check")
	cyre.Forget("delayed-notifications")
	cyre.Forget("delayed-task")
	cyre.Forget("burst-mode")
	cyre.Forget("background-sync")

	fmt.Println("✅ All scheduled actions stopped")
}
