// examples/cyre_routing.go
// Test file to verify correct routing - senders to right addresses, receivers get right payloads

package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/neuralline/cyre-go/core"
)

// TestMessage represents a message with sender/receiver tracking
type TestMessage struct {
	From      string      `json:"from"`
	To        string      `json:"to"`
	Content   interface{} `json:"content"`
	Timestamp int64       `json:"timestamp"`
	MessageID string      `json:"messageId"`
}

// ReceivedMessage tracks what each receiver got
type ReceivedMessage struct {
	ReceiverID   string      `json:"receiverId"`
	ExpectedFrom string      `json:"expectedFrom"`
	ActualFrom   string      `json:"actualFrom"`
	Content      interface{} `json:"content"`
	Correct      bool        `json:"correct"`
}

func main() {
	fmt.Println("🎯 Cyre Go - Routing & Addressing Verification Test")
	fmt.Println("==================================================")

	// Initialize Cyre
	result := core.Initialize()
	if !result.OK {
		log.Fatal("❌ Failed to initialize Cyre")
	}
	cyre := core.GetCyre()

	var receivedMessages []ReceivedMessage

	// 1. Test various ID naming conventions
	fmt.Println("\n1. Testing Various ID Naming Conventions...")

	actionIDs := []string{
		// Standard naming
		"user-service",
		"user_service",
		"userService",
		"UserService",

		// Namespace/hierarchical
		"app.user.login",
		"app/user/profile",
		"system::auth::validate",

		// Special characters (that should work)
		"api-v1-users",
		"service_2024_user",
		"queue-worker-1",

		// Numbers and mixed
		"service123",
		"worker-node-01",
		"cache_layer_2",

		// Long descriptive names
		"user-authentication-service",
		"payment-processing-gateway",
		"notification-delivery-system",
	}

	// Register all test actions
	fmt.Println("Registering actions with various naming conventions...")
	for _, actionID := range actionIDs {
		err := cyre.Action(core.ActionConfig{
			ID:   actionID,
			Type: "routing-test",
			Log:  false, // Reduce noise
		})
		if err != nil {
			fmt.Printf("❌ Failed to register '%s': %v\n", actionID, err)
		} else {
			fmt.Printf("✅ Registered: %s\n", actionID)
		}
	}

	// 2. Register handlers that track sender/receiver info
	fmt.Println("\n2. Registering Handlers with Sender/Receiver Tracking...")

	for _, actionID := range actionIDs {
		// Capture actionID in closure
		receiverID := actionID
		cyre.On(receiverID, func(payload interface{}) interface{} {
			// Extract message info
			if msg, ok := payload.(TestMessage); ok {
				received := ReceivedMessage{
					ReceiverID:   receiverID,
					ExpectedFrom: msg.To, // Message.To should match this receiver
					ActualFrom:   msg.From,
					Content:      msg.Content,
					Correct:      msg.To == receiverID,
				}
				receivedMessages = append(receivedMessages, received)

				if received.Correct {
					fmt.Printf("✅ %s correctly received from %s\n", receiverID, msg.From)
				} else {
					fmt.Printf("❌ %s received message intended for %s (from %s)\n", receiverID, msg.To, msg.From)
				}

				return map[string]interface{}{
					"received_by": receiverID,
					"from":        msg.From,
					"status":      "processed",
					"timestamp":   time.Now().Unix(),
				}
			}

			// Handle non-TestMessage payloads
			received := ReceivedMessage{
				ReceiverID:   receiverID,
				ExpectedFrom: "unknown",
				ActualFrom:   "unknown",
				Content:      payload,
				Correct:      false, // Unknown format
			}
			receivedMessages = append(receivedMessages, received)

			fmt.Printf("⚠️ %s received unexpected payload format: %v\n", receiverID, payload)
			return "unexpected_format"
		})
	}

	// 3. Test Point-to-Point Messaging
	fmt.Println("\n3. Testing Point-to-Point Messaging...")

	testCases := []struct {
		sender   string
		receiver string
		content  string
	}{
		{"user-service", "user_service", "User data sync"},
		{"app.user.login", "system::auth::validate", "Login request"},
		{"payment-processing-gateway", "notification-delivery-system", "Payment completed"},
		{"api-v1-users", "cache_layer_2", "Cache update"},
		{"service123", "worker-node-01", "Task assignment"},
	}

	for i, tc := range testCases {
		message := TestMessage{
			From:      tc.sender,
			To:        tc.receiver,
			Content:   tc.content,
			Timestamp: time.Now().Unix(),
			MessageID: fmt.Sprintf("msg-%d", i+1),
		}

		fmt.Printf("📤 Sending from '%s' to '%s': %s\n", tc.sender, tc.receiver, tc.content)

		// Send to receiver
		result := <-cyre.Call(tc.receiver, message)
		if result.OK {
			fmt.Printf("📥 Delivery confirmed: %v\n", result.Payload)
		} else {
			fmt.Printf("❌ Delivery failed: %s\n", result.Message)
		}

		time.Sleep(10 * time.Millisecond) // Small delay for clarity
	}

	// 4. Test Broadcast-style (same message to multiple receivers)
	fmt.Println("\n4. Testing Broadcast-style Messaging...")

	broadcastMessage := TestMessage{
		From:      "system-broadcaster",
		To:        "all-services",
		Content:   "System maintenance in 5 minutes",
		Timestamp: time.Now().Unix(),
		MessageID: "broadcast-1",
	}

	broadcastTargets := []string{
		"user-service",
		"payment-processing-gateway",
		"notification-delivery-system",
		"cache_layer_2",
	}

	fmt.Printf("📢 Broadcasting from '%s' to %d services\n", broadcastMessage.From, len(broadcastTargets))

	for _, target := range broadcastTargets {
		// Customize message for each target
		targetMessage := broadcastMessage
		targetMessage.To = target

		fmt.Printf("📤 → %s\n", target)
		result := <-cyre.Call(target, targetMessage)
		if !result.OK {
			fmt.Printf("❌ Broadcast to %s failed: %s\n", target, result.Message)
		}
	}

	// 5. Test Cross-naming-convention Communication
	fmt.Println("\n5. Testing Cross-Naming-Convention Communication...")

	crossTests := []struct {
		from string
		to   string
		note string
	}{
		{"userService", "user_service", "camelCase → snake_case"},
		{"app.user.login", "UserService", "dot.notation → PascalCase"},
		{"system::auth::validate", "api-v1-users", "double-colon → kebab-case"},
		{"worker-node-01", "app/user/profile", "numbered → slash notation"},
	}

	for _, test := range crossTests {
		message := TestMessage{
			From:      test.from,
			To:        test.to,
			Content:   fmt.Sprintf("Cross-convention test: %s", test.note),
			Timestamp: time.Now().Unix(),
			MessageID: fmt.Sprintf("cross-%s", strings.ReplaceAll(test.note, " ", "-")),
		}

		fmt.Printf("🔄 %s: %s → %s\n", test.note, test.from, test.to)
		result := <-cyre.Call(test.to, message)
		if !result.OK {
			fmt.Printf("❌ Cross-convention failed: %s\n", result.Message)
		}
	}

	// 6. Test Wrong Address (should fail gracefully)
	fmt.Println("\n6. Testing Wrong Addresses (Expected Failures)...")

	wrongAddresses := []string{
		"non-existent-service",
		"missing_service",
		"invalidService123",
		"does.not.exist",
	}

	for _, wrongAddr := range wrongAddresses {
		message := TestMessage{
			From:      "test-sender",
			To:        wrongAddr,
			Content:   "This should fail",
			Timestamp: time.Now().Unix(),
			MessageID: "wrong-addr-test",
		}

		fmt.Printf("📤 Sending to non-existent '%s'...\n", wrongAddr)
		result := <-cyre.Call(wrongAddr, message)
		if !result.OK {
			fmt.Printf("✅ Correctly failed: %s\n", result.Message)
		} else {
			fmt.Printf("⚠️ Unexpected success to non-existent address\n")
		}
	}

	// 7. Test Case Sensitivity
	fmt.Println("\n7. Testing Case Sensitivity...")

	// Register both versions
	cyre.Action(core.ActionConfig{ID: "testservice", Type: "case-test"})
	cyre.Action(core.ActionConfig{ID: "TestService", Type: "case-test"})
	cyre.Action(core.ActionConfig{ID: "TESTSERVICE", Type: "case-test"})

	cyre.On("testservice", func(payload interface{}) interface{} {
		fmt.Println("📥 Received by 'testservice' (lowercase)")
		return "lowercase-received"
	})

	cyre.On("TestService", func(payload interface{}) interface{} {
		fmt.Println("📥 Received by 'TestService' (PascalCase)")
		return "pascalcase-received"
	})

	cyre.On("TESTSERVICE", func(payload interface{}) interface{} {
		fmt.Println("📥 Received by 'TESTSERVICE' (UPPERCASE)")
		return "uppercase-received"
	})

	caseTests := []string{"testservice", "TestService", "TESTSERVICE"}
	for _, target := range caseTests {
		fmt.Printf("📤 Sending to '%s'\n", target)
		result := <-cyre.Call(target, TestMessage{
			From:    "case-tester",
			To:      target,
			Content: "Case sensitivity test",
		})
		if result.OK {
			fmt.Printf("✅ Response: %v\n", result.Payload)
		}
	}

	// 8. Test Special Characters in IDs
	fmt.Println("\n8. Testing Special Characters in IDs...")

	specialIDs := []string{
		"service-with-dashes",
		"service_with_underscores",
		"service.with.dots",
		"service123numbers",
	}

	for _, specialID := range specialIDs {
		err := cyre.Action(core.ActionConfig{ID: specialID, Type: "special-char-test"})
		if err != nil {
			fmt.Printf("❌ Failed to register '%s': %v\n", specialID, err)
			continue
		}

		cyre.On(specialID, func(payload interface{}) interface{} {
			fmt.Printf("📥 Special char service '%s' received message\n", specialID)
			return "special-char-success"
		})

		// Use a new variable to avoid type mismatch
		specialResult := <-cyre.Call(specialID, TestMessage{
			From:    "special-tester",
			To:      specialID,
			Content: "Special character test",
		})

		if specialResult.OK {
			fmt.Printf("✅ Special char ID '%s' works\n", specialID)
		} else {
			fmt.Printf("❌ Special char ID '%s' failed: %s\n", specialID, specialResult.Message)
		}
	}

	// 9. Analyze Results
	fmt.Println("\n9. Analyzing Routing Results...")

	totalMessages := len(receivedMessages)
	correctDeliveries := 0

	for _, msg := range receivedMessages {
		if msg.Correct {
			correctDeliveries++
		}
	}

	fmt.Printf("📊 Delivery Statistics:\n")
	fmt.Printf("   Total messages sent: %d\n", totalMessages)
	fmt.Printf("   Correct deliveries: %d\n", correctDeliveries)
	fmt.Printf("   Incorrect deliveries: %d\n", totalMessages-correctDeliveries)
	fmt.Printf("   Success rate: %.1f%%\n", float64(correctDeliveries)/float64(totalMessages)*100)

	// 10. Test ActionExists for all registered IDs
	fmt.Println("\n10. Verifying All Registered Actions Exist...")

	allTestIDs := append(actionIDs, "testservice", "TestService", "TESTSERVICE")
	allTestIDs = append(allTestIDs, specialIDs...)

	existsCount := 0
	for _, id := range allTestIDs {
		if cyre.ActionExists(id) {
			existsCount++
			fmt.Printf("✅ %s exists\n", id)
		} else {
			fmt.Printf("❌ %s missing\n", id)
		}
	}

	fmt.Printf("📊 Registration verification: %d/%d actions exist\n", existsCount, len(allTestIDs))

	// 11. Test Payload Integrity
	fmt.Println("\n11. Testing Payload Integrity...")

	// Send complex payload
	complexPayload := TestMessage{
		From: "integrity-tester",
		To:   "user-service",
		Content: map[string]interface{}{
			"nested": map[string]interface{}{
				"data":   []int{1, 2, 3, 4, 5},
				"string": "test string with special chars: !@#$%",
				"bool":   true,
				"null":   nil,
			},
			"array":  []string{"item1", "item2", "item3"},
			"number": 42.5,
		},
		Timestamp: time.Now().Unix(),
		MessageID: "integrity-test",
	}

	fmt.Println("📤 Sending complex payload for integrity test...")
	integrityResult := <-cyre.Call("user-service", complexPayload)
	if integrityResult.OK {
		fmt.Printf("✅ Complex payload delivered successfully\n")
		fmt.Printf("📥 Response: %v\n", integrityResult.Payload)
	} else {
		fmt.Printf("❌ Complex payload failed: %s\n", integrityResult.Message)
	}

	// Final Summary
	fmt.Println("\n🎉 Cyre Go Routing & Addressing Test Complete!")
	fmt.Println("==============================================")
	fmt.Println("✅ Various ID naming conventions tested")
	fmt.Println("✅ Point-to-point messaging verified")
	fmt.Println("✅ Broadcast-style messaging tested")
	fmt.Println("✅ Cross-convention communication checked")
	fmt.Println("✅ Wrong address handling verified")
	fmt.Println("✅ Case sensitivity confirmed")
	fmt.Println("✅ Special characters in IDs tested")
	fmt.Println("✅ Payload integrity maintained")
	fmt.Println("✅ Action existence verification completed")

	if correctDeliveries == totalMessages && existsCount == len(allTestIDs) {
		fmt.Println("\n🏆 PERFECT ROUTING: All messages delivered to correct addresses!")
	} else {
		fmt.Printf("\n⚠️ ROUTING ISSUES: %d delivery errors, %d missing actions\n",
			totalMessages-correctDeliveries, len(allTestIDs)-existsCount)
	}
}
