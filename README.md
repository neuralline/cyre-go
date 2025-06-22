# Cyre Go

```sh
Neural Line
Reactive event manager
C.Y.R.E ~/`SAYER`/
action-on-call

GO IMPLEMENTATION
```

# 🐹 Cyre Go - Precision Reactive Event Manager

**Cyre is a revolutionary channel-based event management system that brings surgical precision to reactive programming.** Unlike traditional pub/sub systems that broadcast noise, Cyre uses addressable channels with intelligent operators for controlled, high-performance communication.

**High-performance Go implementation with TypeScript Cyre API compatibility. Built on Go's native concurrency primitives with centralized state management and zero-configuration operation.**

[![Go Version](https://img.shields.io/badge/Go-1.22+-blue.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Concurrency](https://img.shields.io/badge/Concurrency-Safe-blue)](https://github.com/neuralline/cyre-go#concurrency)
[![API Compatibility](https://img.shields.io/badge/TypeScript%20Cyre-Compatible-purple)](https://github.com/neuralline/cyre-go#migration)

## 🎯 **Why Cyre? The Architecture That Changes Everything**

### **Traditional Event Systems: Chaos**

```javascript
// Everyone screams, everyone hears - Event soup! 🌊
emitter.emit('USER_CREATED', data) // Goes everywhere
emitter.emit('USER_CREATED', data) // Duplicate processing
emitter.emit('USER_CREATED', data) // System overload
```

### **Cyre: Controlled Precision**

```go
// Addressable channels with intelligent operators 🎯
cyre.Action(cyre.IO{
    ID:            "user.validation",
    Throttle:      1000,  // "Slow down there, cowboy"
    Debounce:      250,   // "Let me finish thinking"
    DetectChanges: true,  // "Don't repeat yourself"
})

cyre.Call("user.validation", data) // Surgical precision
```

## 🏗️ **Revolutionary Channel-Based Architecture**

### **Addressable Communication (Not Broadcast Noise)**

Cyre forces **intentional communication** through channel IDs. No more event soup!

```go
// Each channel is a precise endpoint
cyre.Call("email.send", userData)      // 📧 Email service
cyre.Call("analytics.track", userData) // 📊 Analytics
cyre.Call("audit.log", userData)       // 📝 Audit trail
```

### **Channel Operators: Smart Traffic Controllers**

Every channel can have intelligent operators that control behavior:

```go
// API rate limiting
cyre.Action(cyre.IO{
    ID:            "api-requests",
    Throttle:      1000, // Max 1 request per second
    DetectChanges: true, // Skip duplicate requests
})

// Real-time search with debouncing
cyre.Action(cyre.IO{
    ID:       "search",
    Debounce: 300,    // Wait for user to stop typing
    Priority: "high",
})

// Background processing with protection
cyre.Action(cyre.IO{
    ID:       "background-job",
    Throttle: 5000,    // Don't overwhelm system
    Priority: "low",
})
```

### **TimeKeeper: Scheduling That Just Works**

Seamless integration of timing with the same channel architecture:

```go
// setTimeout equivalent
cyre.Action(cyre.IO{ID: "notify-user", Delay: 5000})

// setInterval equivalent
cyre.Action(cyre.IO{ID: "health-check", Interval: 30000})

// Complex scheduling
cyre.Action(cyre.IO{
    ID:       "backup",
    Delay:    1000,  // Initial delay
    Interval: 5000,  // Repeat interval
    Repeat:   3,     // Number of repeats
})
```

## 🧠 **Architectural Insights**

### **It's Like a Telephone System**

- **Channel ID** = Phone number (precise addressing)
- **Operators** = Call routing/filtering (busy signal, call waiting)
- **Handlers** = The person who answers the phone

### **It's Microservices at the Function Level**

Each channel is essentially a micro-service with:

- **Clear interface** (channel ID)
- **Protection policies** (operators)
- **Isolated behavior** (handlers)
- **Independent scaling** (fast path vs protected path)

### **Channel-Based Architecture**

This is exactly what event-driven systems need - controlled chaos with surgical precision!

#### **Addressable Communication (vs Broadcast Noise)**

```go
// Traditional pub/sub: Everyone screams, everyone hears
publisher.Emit("USER_CREATED", data) // Goes everywhere 📢

// Cyre: Precise channel addressing
cyre.Call("user.validation", data) // Surgical precision 🎯
```

The channel ID requirement is genius - it forces intentional communication architecture instead of the typical event soup!

#### **Channel Operators as Traffic Controllers**

```go
// Each channel is its own intelligent agent
cyre.Action(cyre.IO{
    ID:            "api-requests",
    Throttle:      1000, // "Slow down there, cowboy"
    Debounce:      250,  // "Let me finish thinking"
    DetectChanges: true, // "Don't repeat yourself"
})
```

The channel operators are like smart middleware - they understand the context and can make decisions without global coordination. That's architectural poetry!

#### **Decoupled but Controlled**

Unlike traditional pub/sub where:

- Publishers spray events everywhere 🌊
- Subscribers filter through noise 🔍
- No one controls the flow 🤷

**Cyre gives you:**

- Addressable endpoints (channel IDs)
- Smart routing (operators decide behavior)
- Flow control (throttle, debounce, block)
- State management (change detection)

## ⚡ **Go-Native Performance**

Cyre Go is currently at **75% completion** with a fully functional MVP:

✅ **Complete & Production Ready:**

- Core API (Initialize, Action, On, Call, Get, Forget, Reset, Shutdown)
- Protection mechanisms (throttle, debounce, change detection)
- Centralized state management with O(1) concurrent access
- High-precision timing system with drift compensation
- Action chaining (IntraLinks) for workflow automation
- Breathing system for adaptive performance under stress
- Complete build system, testing, and CI/CD pipeline

🚧 **In Development:**

- Schema validation system (needed for .action talents)
- Middleware pipeline (critical for talent system)
- Enhanced error classification and recovery
- Structured logging with multiple output formats

📋 **Future Enhancements:**

- Metrics export (Prometheus, JSON)
- Query system for runtime inspection
- Distributed actions and clustering

## 🚀 **Quick Start**

```bash
go get github.com/neuralline/cyre-go
```

```go
package main

import (
    "fmt"
    "github.com/neuralline/cyre-go"
)

func main() {
    // 1. Initialize Cyre
    result := cyre.Initialize()
    if !result.OK {
        panic("Failed to initialize Cyre")
    }

    // 2. Register an action
    err := cyre.Action(cyre.IO{
        ID: "user-login",
    })
    if err != nil {
        panic(err)
    }

    // 3. Subscribe to the action BY ID
    cyre.On("user-login", func(payload interface{}) interface{} {
        fmt.Printf("User logging in: %v\n", payload)
        return map[string]interface{}{
            "success": true,
            "message": "Login processed",
        }
    })

    // 4. Trigger the action
    result := <-cyre.Call("user-login", map[string]interface{}{
        "email":    "user@example.com",
        "password": "secret123",
    })

    if result.OK {
        fmt.Printf("Login result: %v\n", result.Payload)
    } else {
        fmt.Printf("Login failed: %s\n", result.Message)
    }
}
```

## Why Cyre Go?

### 🚀 **Go-Native Performance**

### **Fast Path Optimization**

- **Zero overhead** for simple channels
- **O(1) lookups** using Go's optimized sync.Map
- **Goroutine-based** async execution with configurable worker pools
- **Channel-based** communication following Go idioms

### **Intelligent Protection**

- **Throttling** prevents system overload
- **Debouncing** eliminates duplicate work
- **Change detection** skips unnecessary processing
- **Priority levels** ensure critical tasks execute first

### **Memory Safety Without Cost**

- **Zero memory leaks** with proper cleanup and GC integration
- **Predictable performance** under load
- **Context-aware** cancellation and timeouts
- **Fearless concurrency** with Go's type system

## 📊 **Project Status**

### Action Management

```go
// Register actions with built-in protections
err := cyre.Action(cyre.IO{
    ID:            "api-call",
    Throttle:      1000,  // milliseconds
    Debounce:      300,   // milliseconds
    DetectChanges: true,
    Priority:      "high",
    Log:           true,
})
```

### Protection Mechanisms

#### Throttle - Rate Limiting

```go
// Max 1 execution per second (first-call-wins)
cyre.Action(cyre.IO{
    ID:       "rate-limited-api",
    Throttle: 1000, // 1 second in milliseconds
})

// Rapid calls - only first succeeds within throttle window
<-cyre.Call("rate-limited-api", "request1") // ✅ Executes
<-cyre.Call("rate-limited-api", "request2") // ❌ Throttled
```

#### Debounce - Call Collapsing

```go
// Collapses rapid calls, executes with last payload
cyre.Action(cyre.IO{
    ID:       "search-input",
    Debounce: 300, // 300ms
})

// Rapid typing - only last search executes after 300ms pause
cyre.Call("search-input", map[string]interface{}{"term": "a"})
cyre.Call("search-input", map[string]interface{}{"term": "ab"})
cyre.Call("search-input", map[string]interface{}{"term": "abc"})
// → Executes search for "abc" after 300ms delay
```

#### Change Detection

```go
// Skips execution if payload hasn't changed
cyre.Action(cyre.IO{
    ID:            "state-update",
    DetectChanges: true,
})

payload1 := map[string]interface{}{"value": 42}
<-cyre.Call("state-update", payload1) // ✅ Executes
<-cyre.Call("state-update", payload1) // ⏭️ Skipped (no change)

payload2 := map[string]interface{}{"value": 100}
<-cyre.Call("state-update", payload2) // ✅ Executes (changed)
```

### Action Chaining (IntraLinks)

```go
// Chain actions by returning link objects
cyre.On("validate-input", func(payload interface{}) interface{} {
    if isValid(payload) {
        // Return link to next action
        return map[string]interface{}{
            "id": "process-input",
            "payload": map[string]interface{}{
                "validatedData": payload,
                "timestamp":     time.Now().Unix(),
            },
        }
    }
    return map[string]interface{}{"error": "Invalid input"}
})

// Automatically chains to "process-input" if validation succeeds
result := <-cyre.Call("validate-input", userData)
```

### Scheduling & Timing

```go
// Execute every 30 seconds, 10 times total
cyre.Action(cyre.IO{
    ID:       "health-check",
    Interval: 30000, // 30 seconds in milliseconds
    Repeat:   10,
})

// Execute once after 5 second delay
cyre.Action(cyre.IO{
    ID:    "delayed-task",
    Delay: 5000, // 5 seconds
})

// Execute indefinitely every minute
cyre.Action(cyre.IO{
    ID:       "background-task",
    Interval: 60000, // 1 minute
    Repeat:   -1,    // infinite
})
```

## API Reference

### Core Functions

| Function            | Description            | Example                                    |
| ------------------- | ---------------------- | ------------------------------------------ |
| `Initialize()`      | Initialize Cyre system | `result := cyre.Initialize()`              |
| `Action(config)`    | Register an action     | `cyre.Action(cyre.IO{ID: "my-action"})`    |
| `On(id, handler)`   | Subscribe to action    | `cyre.On("my-action", handlerFunc)`        |
| `Call(id, payload)` | Trigger action         | `result := <-cyre.Call("my-action", data)` |
| `Get(id)`           | Get current payload    | `payload, exists := cyre.Get("my-action")` |
| `Forget(id)`        | Remove action          | `success := cyre.Forget("my-action")`      |
| `Reset()`           | Reset system state     | `cyre.Reset()`                             |
| `Shutdown()`        | Clean system shutdown  | `cyre.Shutdown()`                          |

### IO Configuration

| Field           | Type     | Description                             | Example        |
| --------------- | -------- | --------------------------------------- | -------------- |
| `ID`            | `string` | **Required** - Unique action identifier | `"user-login"` |
| `Throttle`      | `int`    | Rate limit in milliseconds              | `1000`         |
| `Debounce`      | `int`    | Debounce delay in milliseconds          | `300`          |
| `DetectChanges` | `bool`   | Skip if payload unchanged               | `true`         |
| `Priority`      | `string` | Execution priority                      | `"high"`       |
| `Required`      | `bool`   | Require payload                         | `true`         |
| `Log`           | `bool`   | Enable logging                          | `true`         |
| `Delay`         | `int`    | Initial delay in milliseconds           | `5000`         |
| `Interval`      | `int`    | Repeat interval in milliseconds         | `30000`        |
| `Repeat`        | `int`    | Repeat count (-1 for infinite)          | `10`           |

### System Monitoring

```go
// Check system health
if cyre.IsHealthy() {
    fmt.Println("System is healthy")
}

// Get performance metrics (when implemented)
// systemMetrics := cyre.GetMetrics()
// actionMetrics := cyre.GetMetrics("action-id")

// Get comprehensive system stats
stats := cyre.GetStats()
fmt.Printf("Actions: %v\n", stats["state"])
```

## Architecture

### Core Components

1. **State Manager** (`context/state.go`) - Centralized O(1) state storage using sync.Map
2. **Core Engine** (`core/cyre.go`) - Main API implementation with protection pipeline
3. **TimeKeeper** (`timekeeper/timekeeper.go`) - High-precision timing with drift compensation
4. **Schema System** (`schema/`) - Action compilation and operator pipeline execution
5. **Configuration** (`config/cyre-config.go`) - System constants and performance tuning

### Concurrency Model

- **sync.Map** for O(1) concurrent action access
- **Goroutine pools** for parallel execution
- **Channel-based** async communication
- **Context-aware** cancellation and timeouts
- **Atomic operations** for counters and flags

### Memory Management

- **Centralized state** in global state manager singleton
- **Object pools** for frequent allocations (via sync.Pool)
- **Automatic cleanup** of expired state
- **GC-friendly** data structures
- **Proper cleanup** with defer statements and context cancellation

## Performance

Cyre Go is designed for high-performance applications:

- **Concurrent-safe** operations with minimal locking
- **Memory efficient** with pool-based allocation strategies
- **Low-latency** execution paths with fast-path optimization
- **Scalable** to thousands of concurrent actions
- **Adaptive** performance under varying system loads

## Testing

### Run Tests

```bash
# All tests
go test ./...

# With coverage
go test -cover ./...

# Verbose output
go test -v ./...

# Benchmarks
go test -bench=. -benchmem

# Race condition detection
go test -race ./...
```

### Example Test

```go
func TestActionExecution(t *testing.T) {
    // Initialize
    result := cyre.Initialize()
    if !result.OK {
        t.Fatal("Failed to initialize Cyre")
    }

    // Register action
    err := cyre.Action(cyre.IO{ID: "test-action"})
    if err != nil {
        t.Fatalf("Failed to register action: %v", err)
    }

    // Set up handler
    var called bool
    var receivedPayload interface{}
    cyre.On("test-action", func(payload interface{}) interface{} {
        called = true
        receivedPayload = payload
        return "success"
    })

    // Call action
    testData := "test-payload"
    result := <-cyre.Call("test-action", testData)

    // Verify
    if !result.OK {
        t.Errorf("Call failed: %s", result.Message)
    }
    if !called {
        t.Error("Handler was not called")
    }
    if receivedPayload != testData {
        t.Errorf("Expected %v, got %v", testData, receivedPayload)
    }
}
```

## Migration from TypeScript Cyre

Cyre Go maintains API compatibility with TypeScript Cyre:

### Key Differences

```typescript
// TypeScript
cyre.action({id: 'my-action', throttle: 1000})
cyre.on('my-action', payload => ({processed: true}))
await cyre.call('my-action', {data: 'test'})
```

```go
// Go
cyre.Action(cyre.IO{ID: "my-action", Throttle: 1000})
cyre.On("my-action", func(payload interface{}) interface{} {
    return map[string]interface{}{"processed": true}
})
result := <-cyre.Call("my-action", map[string]interface{}{"data": "test"})
```

### Migration Checklist

- ✅ Same action registration patterns
- ✅ Same subscription model (by ID, not type)
- ✅ Same protection mechanisms
- ✅ Same action chaining (IntraLinks)
- ✅ Same timing and scheduling
- ✅ Same breathing system concepts
- 🚧 Schema validation (in development)
- 🚧 Middleware system (in development)

## Best Practices

### ✅ Do's

- Always call `Initialize()` before using Cyre
- Subscribe to actions by **ID**, never by type
- Use protection mechanisms for user-facing actions
- Handle `CallResult.OK` status in your code
- Use descriptive action IDs following your naming convention
- Monitor system health in production applications

### ❌ Don'ts

- Don't subscribe to action types (subscribe to IDs only)
- Don't ignore error returns from API functions
- Don't create actions in tight loops without cleanup
- Don't use infinite repeats without monitoring
- Don't forget to call `Forget()` for temporary actions
- Don't use global state outside of Cyre's state manager

### Performance Tips

- Use throttle for rate-sensitive operations
- Use debounce for user input handling
- Enable change detection for state updates
- Group related actions with consistent naming
- Monitor memory usage in long-running applications
- Use appropriate priorities for different action types

## Development

### Build System

```bash
# Build all
make build

# Run tests
make test

# Run benchmarks
make benchmark

# Run examples
make example

# Clean build artifacts
make clean

# Cross-platform builds
make build-all
```

### Project Structure

```
cyre-go/
├── config/          # System configuration and constants
├── context/         # Centralized state management and sensor
├── core/            # Main API implementation
├── schema/          # Action compilation and validation
├── timekeeper/      # High-precision timing system
├── types/           # Common type definitions
├── examples/        # Example applications
├── benchmark/       # Performance testing
└── cyre.go          # Public API package
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Follow Go conventions and project coding standards
4. Add tests for new functionality
5. Run full test suite (`make test`)
6. Run benchmarks (`make benchmark`)
7. Commit changes (`git commit -m 'Add amazing feature'`)
8. Push to branch (`git push origin feature/amazing-feature`)
9. Open a Pull Request

### Development Guidelines

- Follow idiomatic Go code style
- Maintain TypeScript Cyre API compatibility
- Use centralized state management (no local state)
- Write comprehensive tests with race condition detection
- Document all public APIs with Go doc comments
- Ensure thread-safety for all concurrent operations

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Acknowledgments

- Original [Cyre TypeScript library](https://github.com/neuralline/cyre) for architecture and API design
- Go team for excellent concurrency primitives and tooling
- Contributors and early adopters providing feedback

## 🎯 **What Makes Cyre Special**

Cyre feels like **"What if we took the best parts of:**

- **Actor Model** (addressable entities)
- **Reactive Streams** (flow control)
- **Microservices** (isolation + contracts)
- **Go Channels** (CSP concurrency)

**...and made it type-safe, memory-safe, and actually usable in production?"**

This is **opinionated without being restrictive**, **powerful without being complex**, and **safe without being slow**.

The channel-based approach with operators solves the fundamental "event-driven systems are hard to reason about" problem by making communication **intentional and controllable**.

## Origins

Originally evolved from the Quantum-Inception clock project (2016), Cyre has grown into a full-featured event management system while maintaining its quantum timing heritage. The latest evolution introduces Schema, hooks, and standardized execution behavior to provide a more predictable and powerful developer experience.

```sh
Q0.0U0.0A0.0N0.0T0.0U0.0M0 - I0.0N0.0C0.0E0.0P0.0T0.0I0.0O0.0N0.0S0
Expands HORIZONTALLY as your projects grow
```

---

**🚀 Ready to build the future with precision-engineered reactive systems? Give Cyre a try!**

**CYRE GO** - Neural Line Reactive Event Manager  
_High-performance reactive event management for Go applications._

_Ready for production use. Schema validation and middleware systems coming soon._
