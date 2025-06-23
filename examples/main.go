// examples/main.go
// Demonstration of Cyre Go core functionality

package main

import (
	"fmt"
	"log"
	"time"

	"github.com/neuralline/cyre-go"
)

func main() {
	fmt.Println("🚀 Cyre Go - Core Functionality Demo")
	fmt.Println("=====================================")

	// 1. Initialize Cyre
	fmt.Println("\n1. Initializing Cyre...")
	result := cyre.Init()
	if result.OK {
		fmt.Printf("✅ Cyre initialized successfully at %d\n", result.Payload)
	} else {
		log.Fatal("❌ Failed to initialize Cyre")
	}

	// 2. Register a simple action
	fmt.Println("\n2. Registering actions...")
	err := cyre.Action(cyre.ActionConfig{
		ID:      "user-login",
		Type:    "auth",
		Payload: map[string]interface{}{"status": "idle"},
	})
	if err != nil {
		log.Fatalf("❌ Failed to register action: %v", err)
	}
	fmt.Println("✅ Action 'user-login' registered")

	// 3. Subscribe to the action
	fmt.Println("\n3. Subscribing to action...")
	subResult := cyre.On("user-login", func(payload interface{}) interface{} {
		fmt.Printf("🔔 User login handler called with: %v\n", payload)

		// Simulate authentication logic
		if userMap, ok := payload.(map[string]interface{}); ok {
			if email, exists := userMap["email"]; exists {
				return map[string]interface{}{
					"success":   true,
					"userID":    "user123",
					"email":     email,
					"timestamp": time.Now().Unix(),
				}
			}
		}

		return map[string]interface{}{
			"success": false,
			"error":   "Invalid payload format",
		}
	})

	if subResult.OK {
		fmt.Printf("✅ %s\n", subResult.Message)
	} else {
		log.Fatalf("❌ Failed to subscribe: %s", subResult.Message)
	}

	// 4. Call the action
	fmt.Println("\n4. Calling action...")
	resultChan := cyre.Call("user-login", map[string]interface{}{
		"email":    "user@example.com",
		"password": "secret123",
	})

	// Wait for result
	callResult := <-resultChan
	if callResult.OK {
		fmt.Printf("✅ Action executed successfully: %v\n", callResult.Payload)
	} else {
		fmt.Printf("❌ Action failed: %s\n", callResult.Message)
	}

	// 5. Demonstrate throttled action
	fmt.Println("\n5. Testing throttle protection...")
	err = cyre.Action(cyre.ActionConfig{
		ID:       "api-call",
		Throttle: 1000,
	})
	if err != nil {
		log.Fatalf("❌ Failed to register throttled action: %v", err)
	}

	cyre.On("api-call", func(payload interface{}) interface{} {
		fmt.Printf("🌐 API call executed at %s\n", time.Now().Format("15:04:05.000"))
		return map[string]interface{}{"response": "API data", "timestamp": time.Now().Unix()}
	})

	// Make rapid calls to test throttling
	fmt.Println("Making rapid calls (should be throttled)...")
	for i := 0; i < 3; i++ {
		resultChan := cyre.Call("api-call", map[string]interface{}{"request": i})
		result := <-resultChan

		if result.OK {
			fmt.Printf("  Call %d: ✅ Executed\n", i+1)
		} else {
			fmt.Printf("  Call %d: ⛔ %s\n", i+1, result.Message)
		}

		time.Sleep(200 * time.Millisecond) // Rapid calls
	}

	// Wait for throttle to reset
	fmt.Println("Waiting for throttle to reset...")
	time.Sleep(1100 * time.Millisecond)

	resultChan = cyre.Call("api-call", map[string]interface{}{"request": "after-wait"})
	callResult = <-resultChan
	if callResult.OK {
		fmt.Println("  After wait: ✅ Executed (throttle reset)")
	}

	// 6. Demonstrate debounced action
	fmt.Println("\n6. Testing debounce protection...")
	err = cyre.Action(cyre.ActionConfig{
		ID:       "search-input",
		Debounce: 455,
	})

	cyre.On("search-input", func(payload interface{}) interface{} {
		if searchMap, ok := payload.(map[string]interface{}); ok {
			term := searchMap["term"]
			fmt.Printf("🔍 Search executed for: %v\n", term)
			return map[string]interface{}{
				"results": []string{"result1", "result2", "result3"},
				"term":    term,
			}
		}
		return nil
	})

	// Make rapid search calls (should be debounced)
	fmt.Println("Making rapid search calls (should be debounced to last one)...")
	searchTerms := []string{"a", "ab", "abc", "abcd"}

	for _, term := range searchTerms {
		cyre.Call("search-input", map[string]interface{}{"term": term})
		fmt.Printf("  Queued search for: %s\n", term)
		time.Sleep(50 * time.Millisecond) // Rapid typing
	}

	// Wait for debounce to execute
	time.Sleep(400 * time.Millisecond)

	// 7. Demonstrate change detection
	fmt.Println("\n7. Testing change detection...")
	err = cyre.Action(cyre.ActionConfig{
		ID:            "state-update",
		DetectChanges: true,
	})
	if err != nil {
		log.Fatalf("❌ Failed to register change detection action: %v", err)
	}

	cyre.On("state-update", func(payload interface{}) interface{} {
		fmt.Printf("📊 State updated: %v\n", payload)
		return map[string]interface{}{"updated": true, "timestamp": time.Now().Unix()}
	})

	// Test change detection
	fmt.Println("Testing identical payloads (should skip duplicates)...")

	testPayload := map[string]interface{}{"value": 42}

	for i := 0; i < 3; i++ {
		resultChan := cyre.Call("state-update", testPayload)
		result := <-resultChan

		if result.OK {
			fmt.Printf("  Call %d: ✅ Executed (payload changed)\n", i+1)
		} else {
			fmt.Printf("  Call %d: ⏭️  %s\n", i+1, result.Message)
		}
	}

	// Now change the payload
	fmt.Println("Changing payload...")
	newPayload := map[string]interface{}{"value": 100}
	resultChan = cyre.Call("state-update", newPayload)
	callResult = <-resultChan
	if callResult.OK {
		fmt.Println("  New payload: ✅ Executed (payload changed)")
	}

	// 8. Test action chaining (IntraLinks)
	fmt.Println("\n8. Testing action chaining...")

	// Register validation action
	err = cyre.Action(cyre.ActionConfig{ID: "validate-input"})
	if err != nil {
		log.Fatalf("❌ Failed to register validation action: %v", err)
	}

	// Register processing action
	err = cyre.Action(cyre.ActionConfig{ID: "process-input"})
	if err != nil {
		log.Fatalf("❌ Failed to register processing action: %v", err)
	}

	// Validation handler that chains to processing
	cyre.On("validate-input", func(payload interface{}) interface{} {
		fmt.Printf("🔍 Validating input: %v\n", payload)

		if inputMap, ok := payload.(map[string]interface{}); ok {
			if data, exists := inputMap["data"]; exists {
				// Return chain link to next action
				return map[string]interface{}{
					"id": "process-input",
					"payload": map[string]interface{}{
						"validatedData": data,
						"valid":         true,
						"timestamp":     time.Now().Unix(),
					},
				}
			}
		}

		return map[string]interface{}{"valid": false}
	})

	// Processing handler
	cyre.On("process-input", func(payload interface{}) interface{} {
		fmt.Printf("⚙️  Processing validated input: %v\n", payload)
		return map[string]interface{}{
			"processed": true,
			"result":    "Processing complete",
		}
	})

	// Trigger the chain
	fmt.Println("Triggering validation chain...")
	resultChan = cyre.Call("validate-input", map[string]interface{}{
		"data": "important user data",
	})

	chainResult := <-resultChan
	if chainResult.OK {
		fmt.Printf("✅ Chain completed: %v\n", chainResult.Payload)
	} else {
		fmt.Printf("❌ Chain failed: %s\n", chainResult.Message)
	}

	// 10. Retrieve stored payloads
	fmt.Println("\n10. Retrieving stored data...")
	if payload, exists := cyre.Get("user-login"); exists {
		fmt.Printf("Current user-login payload: %v\n", payload)
	}

	fmt.Println("\n🎉 Cyre Go Core Demo Complete!")
	fmt.Println("=====================================")
	fmt.Println("Core features demonstrated:")
	fmt.Println("✅ Action registration and execution")
	fmt.Println("✅ Handler subscription and calling")
	fmt.Println("✅ Throttle protection (rate limiting)")
	fmt.Println("✅ Debounce protection (call collapsing)")
	fmt.Println("✅ Change detection (skip identical)")
	fmt.Println("✅ Action chaining (IntraLinks)")
	fmt.Println("✅ System monitoring and health")
	fmt.Println("✅ Payload storage and retrieval")
}
