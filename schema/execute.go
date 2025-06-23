// schema/execute.go
// Clean pipeline execution engine with proper delay/scheduling detection

package schema

import (
	"fmt"
	"time"

	"github.com/neuralline/cyre-go/state"
	"github.com/neuralline/cyre-go/types"
)

// === EXECUTION RESULT TYPES ===

// ExecutionResult represents the result of pipeline execution
type ExecutionResult struct {
	Allow           bool          `json:"allow"`           // Whether execution should proceed
	Payload         interface{}   `json:"payload"`         // Final processed payload
	Message         string        `json:"message"`         // Execution summary message
	Error           error         `json:"error"`           // Error if execution failed
	ExecutedOps     []string      `json:"executedOps"`     // Operators that were executed
	BlockedBy       string        `json:"blockedBy"`       // Operator that blocked execution (if any)
	Duration        time.Duration `json:"duration"`        // Total execution time
	Transformations int           `json:"transformations"` // Number of payload transformations

	// Scheduling detection
	Delay          *time.Duration         `json:"delay,omitempty"`          // Detected delay requirement
	RequiresDelay  bool                   `json:"requiresDelay"`            // If delay needed
	SchedulingInfo map[string]interface{} `json:"schedulingInfo,omitempty"` // Scheduling metadata
}

// === MAIN PIPELINE EXECUTION FUNCTION ===

// ExecutePipeline executes the operator pipeline defined in action._pipeline
func ExecutePipeline(actionID string, payload interface{}, action *types.IO, sm *state.StateManager) ExecutionResult {
	startTime := time.Now()

	// Fast path: no pipeline defined
	if action == nil {
		return ExecutionResult{
			Allow:    true,
			Payload:  payload,
			Message:  "No action configuration",
			Duration: time.Since(startTime),
		}
	}

	// Check if action has fast path (no operators)
	if action.HasFastPath {
		return ExecutionResult{
			Allow:    true,
			Payload:  payload,
			Message:  "Fast path - no operators",
			Duration: time.Since(startTime),
		}
	}

	// Get pipeline from action._pipeline field
	pipeline := getPipelineFromAction(action)
	if len(pipeline) == 0 {
		return ExecutionResult{
			Allow:    true,
			Payload:  payload,
			Message:  "No operators in pipeline",
			Duration: time.Since(startTime),
		}
	}

	// Execute pipeline steps
	return executePipelineSteps(actionID, payload, action, sm, pipeline, startTime)
}

// executePipelineSteps executes each operator in the pipeline sequence
func executePipelineSteps(actionID string, payload interface{}, action *types.IO, sm *state.StateManager, pipeline []string, startTime time.Time) ExecutionResult {
	currentPayload := payload
	executedOps := []string{}
	transformations := 0
	messages := []string{}
	var detectedDelay *time.Duration
	schedulingInfo := make(map[string]interface{})

	// Execute each operator in user-defined order
	for i, operatorName := range pipeline {
		// Get operator function from channel-operators.go
		operator, exists := GetOperator(operatorName)
		if !exists {
			// Log warning for unknown operator but continue
			state.Warn(fmt.Sprintf("Unknown operator '%s' in pipeline for action %s", operatorName, actionID)).
				Location("schema/execute.go").
				Id(actionID).
				Metadata(map[string]interface{}{
					"operator":     operatorName,
					"pipelineStep": i + 1,
				}).
				Log()
			continue
		}

		// Execute operator
		result := operator(actionID, currentPayload, action, sm)
		executedOps = append(executedOps, operatorName)

		// Add operator message to execution log
		if result.Message != "" {
			messages = append(messages, fmt.Sprintf("%s: %s", operatorName, result.Message))
		}

		// Check if operator blocked execution
		if !result.Allow {
			return ExecutionResult{
				Allow:           false,
				Payload:         currentPayload,
				Message:         result.Message,
				ExecutedOps:     executedOps,
				BlockedBy:       operatorName,
				Duration:        time.Since(startTime),
				Transformations: transformations,
				Delay:           detectedDelay,
			}
		}

		// Check if payload was transformed
		if result.Payload != nil && !isPayloadEqual(currentPayload, result.Payload) {
			currentPayload = result.Payload
			transformations++
		}

		// Handle detected delay (from debounce operator)
		if result.Delay != nil && *result.Delay > 0 {
			detectedDelay = result.Delay
			schedulingInfo["delaySource"] = operatorName
			schedulingInfo["delayDuration"] = result.Delay.Milliseconds()

			state.Info(fmt.Sprintf("Operator '%s' detected %v delay for action %s", operatorName, *result.Delay, actionID)).
				Location("schema/execute.go").
				Id(actionID).
				Metadata(map[string]interface{}{
					"operator": operatorName,
					"delay":    result.Delay.Milliseconds(),
				}).
				Log()
		}
	}

	// All operators executed successfully
	finalMessage := "Pipeline completed"
	if len(messages) > 0 {
		finalMessage = fmt.Sprintf("Pipeline completed: %s", joinMessages(messages))
	}

	return ExecutionResult{
		Allow:           true,
		Payload:         currentPayload,
		Message:         finalMessage,
		ExecutedOps:     executedOps,
		Duration:        time.Since(startTime),
		Transformations: transformations,
		Delay:           detectedDelay,
		RequiresDelay:   detectedDelay != nil,
		SchedulingInfo:  schedulingInfo,
	}
}

// getPipelineFromAction extracts the operator pipeline from action configuration
func getPipelineFromAction(action *types.IO) []string {
	// Check if _pipeline field exists and has operators
	if len(action.Pipeline) > 0 {
		// Extract operator names from Pipeline field
		pipeline := make([]string, len(action.Pipeline))
		for i, op := range action.Pipeline {
			pipeline[i] = op.Type
		}
		return pipeline
	}

	// Fallback: build pipeline from individual operator fields
	return buildPipelineFromFields(action)
}

// buildPipelineFromFields builds a pipeline from individual operator fields
func buildPipelineFromFields(action *types.IO) []string {
	pipeline := []string{}

	// Standard operator order (protection -> processing -> scheduling)
	operatorChecks := []struct {
		name      string
		condition func(*types.IO) bool
	}{
		{"block", func(a *types.IO) bool { return a.Block || a.IsBlocked }},
		{"required", func(a *types.IO) bool { return a.Required }},
		{"auth", func(a *types.IO) bool { return a.Auth != nil }},
		{"schema", func(a *types.IO) bool { return a.Schema != nil && len(a.Schema) > 0 }},
		{"condition", func(a *types.IO) bool { return a.Condition != nil }},
		{"priority", func(a *types.IO) bool { return a.Priority != "" }},
		{"throttle", func(a *types.IO) bool { return a.Throttle > 0 }},
		{"detectChanges", func(a *types.IO) bool { return a.DetectChanges }},
		{"selector", func(a *types.IO) bool { return a.Selector != nil }},
		{"transform", func(a *types.IO) bool { return a.Transform != nil }},
		{"scheduler", func(a *types.IO) bool {
			return (a.Delay > 0) || (a.Interval > 0) || (a.Repeat != 0)
		}},
		{"debounce", func(a *types.IO) bool { return a.Debounce > 0 }},
		{"log", func(a *types.IO) bool { return a.Log }},
	}

	for _, check := range operatorChecks {
		if check.condition(action) {
			pipeline = append(pipeline, check.name)
		}
	}

	return pipeline
}

// isPayloadEqual checks if two payloads are equal
func isPayloadEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// joinMessages joins operator messages with appropriate formatting
func joinMessages(messages []string) string {
	if len(messages) == 0 {
		return ""
	}
	if len(messages) == 1 {
		return messages[0]
	}
	if len(messages) <= 3 {
		result := ""
		for i, msg := range messages {
			if i > 0 {
				result += "; "
			}
			result += msg
		}
		return result
	}
	return fmt.Sprintf("%s; %s; ... (%d total)", messages[0], messages[1], len(messages))
}
