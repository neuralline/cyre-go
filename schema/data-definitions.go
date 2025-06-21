// schema/data-definitions.go
// Action compilation with talent discovery and validation for cyre.action() inputs
// O(1) field lookups with user-defined order preservation

package schema

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/neuralline/cyre-go/types"
)

/*
	C.Y.R.E - D.A.T.A - D.E.F.I.N.I.T.I.O.N.S

	Action compilation system with Go-native optimizations:
	- O(1) field validation using map lookup
	- Preserves user declaration order
	- British AI assistant style error messages
	- Talent discovery and pipeline building
	- Fast path optimization
	- Mandatory field blocking
*/

// === VALIDATION RESULT TYPES ===

// DataDefResult represents the result of field validation
type DataDefResult struct {
	OK          bool        `json:"ok"`
	Data        interface{} `json:"data,omitempty"`
	Error       string      `json:"error,omitempty"`
	Blocking    bool        `json:"blocking,omitempty"`   // Stop compilation immediately
	TalentName  string      `json:"talentName,omitempty"` // Operator name for pipeline
	Suggestions []string    `json:"suggestions,omitempty"`
}

// CompileResult represents the result of action compilation
type CompileResult struct {
	CompiledAction *types.IO `json:"compiledAction"`
	Errors         []string  `json:"errors"`
	Warnings       []string  `json:"warnings"`
	HasFastPath    bool      `json:"hasFastPath"`
	TalentPipeline []string  `json:"talentPipeline"` // Operators in user-defined order
	HasProtections bool      `json:"hasProtections"`
	HasProcessing  bool      `json:"hasProcessing"`
	HasScheduling  bool      `json:"hasScheduling"`
}

// === HELPER FUNCTIONS ===

// describeValue describes the actual value received for error messages
func describeValue(value interface{}) string {
	if value == nil {
		return "null"
	}

	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.String:
		return fmt.Sprintf(`string "%s"`, value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("number %d", value)
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("number %g", value)
	case reflect.Bool:
		return fmt.Sprintf("boolean %t", value)
	case reflect.Slice, reflect.Array:
		return fmt.Sprintf("array with %d items", rv.Len())
	case reflect.Map, reflect.Struct:
		if rv.Kind() == reflect.Map {
			keys := make([]string, 0, rv.Len())
			for _, key := range rv.MapKeys() {
				keys = append(keys, fmt.Sprintf("%v", key.Interface()))
			}
			return fmt.Sprintf("object with keys: %s", strings.Join(keys, ", "))
		}
		return "object"
	case reflect.Func:
		return "function"
	case reflect.Ptr:
		if rv.IsNil() {
			return "null pointer"
		}
		return describeValue(rv.Elem().Interface())
	default:
		return fmt.Sprintf("%T: %v", value, value)
	}
}

// Fast type checking helpers
func isString(value interface{}) (string, bool) {
	if s, ok := value.(string); ok {
		return s, true
	}
	return "", false
}

func isInt(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case float64:
		if v == float64(int(v)) {
			return int(v), true
		}
	}
	return 0, false
}

func isFloat(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	}
	return 0, false
}

func isBool(value interface{}) (bool, bool) {
	if b, ok := value.(bool); ok {
		return b, true
	}
	return false, false
}

func isFunction(value interface{}) bool {
	return reflect.TypeOf(value).Kind() == reflect.Func
}

// === VALIDATION FUNCTIONS ===

// DataDefinitionFunc represents a field validation function
type DataDefinitionFunc func(value interface{}) DataDefResult

// newSuccess creates a successful validation result
func newSuccess(data interface{}, talentName ...string) DataDefResult {
	result := DataDefResult{OK: true, Data: data}
	if len(talentName) > 0 && talentName[0] != "" {
		result.TalentName = talentName[0]
	}
	return result
}

// newError creates a validation error result
func newError(error string, suggestions ...string) DataDefResult {
	return DataDefResult{
		OK:          false,
		Error:       error,
		Suggestions: suggestions,
	}
}

// newBlocking creates a blocking validation error result
func newBlocking(error string, suggestions ...string) DataDefResult {
	return DataDefResult{
		OK:          false,
		Error:       error,
		Blocking:    true,
		Suggestions: suggestions,
	}
}

// === DATA DEFINITIONS MAP (O(1) lookup) ===

// DataDefinitions provides O(1) field validation lookup
var DataDefinitions = map[string]DataDefinitionFunc{
	// === MANDATORY FIELDS ===
	"id": func(value interface{}) DataDefResult {
		s, ok := isString(value)
		if !ok || len(s) == 0 {
			return newBlocking(
				fmt.Sprintf("Channel ID must be a non-empty text value, but received %s", describeValue(value)),
				"Provide a unique text identifier for this channel",
				`Example: "user-validator" or "sensor-IUG576"`,
				"Avoid spaces - use hyphens or underscores instead",
			)
		}
		return newSuccess(s)
	},

	// === ORGANIZATIONAL FIELDS ===
	"name": func(value interface{}) DataDefResult {
		if value == nil {
			return newSuccess(nil)
		}
		s, ok := isString(value)
		if !ok {
			return newError(
				fmt.Sprintf("Name must be text, but received %s", describeValue(value)),
				"Provide a human-readable name for this channel",
			)
		}
		return newSuccess(s)
	},

	"type": func(value interface{}) DataDefResult {
		if value == nil {
			return newSuccess(nil)
		}
		s, ok := isString(value)
		if !ok {
			return newError(
				fmt.Sprintf("Type must be text, but received %s", describeValue(value)),
				"Specify channel category like 'user-action' or 'sensor-data'",
			)
		}
		return newSuccess(s)
	},

	"path": func(value interface{}) DataDefResult {
		if value == nil {
			return newSuccess(nil)
		}
		s, ok := isString(value)
		if !ok {
			return newError(
				fmt.Sprintf("Path must be text, but received %s", describeValue(value)),
				"Use hierarchical format like 'app/users/profile'",
				"Separate levels with forward slashes",
			)
		}
		if len(s) == 0 {
			return newSuccess(nil)
		}
		// Validate path format
		pathRegex := regexp.MustCompile(`^[a-zA-Z0-9/_-]+$`)
		if !pathRegex.MatchString(s) {
			return newError(
				"Path contains invalid characters",
				"Use only letters, numbers, /, _, and -",
			)
		}
		return newSuccess(s)
	},

	"group": func(value interface{}) DataDefResult {
		if value == nil {
			return newSuccess(nil)
		}
		s, ok := isString(value)
		if !ok {
			return newError(
				fmt.Sprintf("Group must be text, but received %s", describeValue(value)),
				"Specify group name for bulk operations",
			)
		}
		return newSuccess(s)
	},

	"description": func(value interface{}) DataDefResult {
		if value == nil {
			return newSuccess(nil)
		}
		s, ok := isString(value)
		if !ok {
			return newError(
				fmt.Sprintf("Description must be text, but received %s", describeValue(value)),
				"Provide a description of what this channel does",
			)
		}
		return newSuccess(s)
	},

	"version": func(value interface{}) DataDefResult {
		if value == nil {
			return newSuccess(nil)
		}
		s, ok := isString(value)
		if !ok {
			return newError(
				fmt.Sprintf("Version must be text, but received %s", describeValue(value)),
				"Use semantic versioning like '1.0.0'",
			)
		}
		return newSuccess(s)
	},

	// === PROTECTION OPERATORS ===
	"required": func(value interface{}) DataDefResult {
		if value == nil {
			return newSuccess(nil)
		}
		b, ok := isBool(value)
		if !ok {
			return newError(
				fmt.Sprintf("Required must be true or false, but received %s", describeValue(value)),
				"Use true to require payload, false to make optional",
			)
		}
		return newSuccess(b, "required")
	},

	"block": func(value interface{}) DataDefResult {
		if value == nil {
			return newSuccess(nil)
		}
		b, ok := isBool(value)
		if !ok {
			return newError(
				fmt.Sprintf("Block must be true or false, but received %s", describeValue(value)),
				"Use true to block execution, false to allow",
			)
		}
		if b {
			return newBlocking("Service not available")
		}
		return newSuccess(b, "block")
	},

	"throttle": func(value interface{}) DataDefResult {
		if value == nil {
			return newSuccess(nil)
		}
		n, ok := isInt(value)
		if !ok || n < 0 {
			return newError(
				fmt.Sprintf("Throttle must be a positive number (milliseconds), but received %s", describeValue(value)),
				"Specify time in milliseconds to limit execution frequency",
				"Example: 1000 for maximum once per second",
				"Use 0 to disable throttling",
			)
		}
		return newSuccess(n, "throttle")
	},

	"debounce": func(value interface{}) DataDefResult {
		if value == nil {
			return newSuccess(nil)
		}
		n, ok := isInt(value)
		if !ok || n < 0 {
			return newError(
				fmt.Sprintf("Debounce must be a positive number (milliseconds), but received %s", describeValue(value)),
				"Specify delay in milliseconds to wait for rapid calls to settle",
				"Example: 300 to wait 300ms after last call before executing",
				"Use 0 to disable debouncing",
			)
		}
		return newSuccess(n, "debounce")
	},

	"maxWait": func(value interface{}) DataDefResult {
		if value == nil {
			return newSuccess(nil)
		}
		n, ok := isInt(value)
		if !ok || n < 0 {
			return newError(
				fmt.Sprintf("MaxWait must be a positive number (milliseconds), but received %s", describeValue(value)),
				"Specify maximum wait time for debounce",
				"Forces execution even if calls keep coming",
			)
		}
		return newSuccess(n) // No separate operator - handled by debounce
	},

	"detectChanges": func(value interface{}) DataDefResult {
		if value == nil {
			return newSuccess(nil)
		}
		b, ok := isBool(value)
		if !ok {
			return newError(
				fmt.Sprintf("DetectChanges must be true or false, but received %s", describeValue(value)),
				"Use true to only execute when data changes from previous call",
				"Use false to execute every time regardless of changes",
			)
		}
		return newSuccess(b, "detectChanges")
	},

	"priority": func(value interface{}) DataDefResult {
		if value == nil {
			return newSuccess(nil)
		}
		s, ok := isString(value)
		if !ok {
			return newError(
				fmt.Sprintf("Priority must be text, but received %s", describeValue(value)),
				"Use 'low', 'medium', 'high', or 'critical'",
			)
		}
		validPriorities := []string{"low", "medium", "high", "critical"}
		for _, valid := range validPriorities {
			if s == valid {
				return newSuccess(s, "priority")
			}
		}
		return newError(
			fmt.Sprintf("Priority must be one of: %s", strings.Join(validPriorities, ", ")),
			"Use 'low', 'medium', 'high', or 'critical'",
		)
	},

	"log": func(value interface{}) DataDefResult {
		if value == nil {
			return newSuccess(nil)
		}
		b, ok := isBool(value)
		if !ok {
			return newError(
				fmt.Sprintf("Log must be true or false, but received %s", describeValue(value)),
				"Use true to enable logging, false to disable",
			)
		}
		return newSuccess(b, "log")
	},

	// === SCHEDULING OPERATORS ===
	"delay": func(value interface{}) DataDefResult {
		if value == nil {
			return newSuccess(nil)
		}
		n, ok := isInt(value)
		if !ok || n < 0 {
			return newError(
				fmt.Sprintf("Delay must be a positive number (milliseconds), but received %s", describeValue(value)),
				"Specify initial delay in milliseconds before first execution",
				"Example: 1000 to wait 1 second before executing",
				"Use 0 for immediate execution",
			)
		}
		return newSuccess(n, "scheduler")
	},

	"interval": func(value interface{}) DataDefResult {
		if value == nil {
			return newSuccess(nil)
		}
		n, ok := isInt(value)
		if !ok || n <= 0 {
			return newError(
				fmt.Sprintf("Interval must be a positive number (milliseconds), but received %s", describeValue(value)),
				"Specify time in milliseconds between repeated executions",
				"Example: 5000 to execute every 5 seconds",
				"Must be greater than 0 for repeated execution",
			)
		}
		return newSuccess(n, "scheduler")
	},

	"repeat": func(value interface{}) DataDefResult {
		if value == nil {
			return newSuccess(nil)
		}

		// Handle boolean values
		if b, ok := isBool(value); ok {
			if b {
				return newSuccess(-1, "scheduler") // true = infinite
			}
			return newSuccess(1, "scheduler") // false = single execution
		}

		// Handle numeric values
		if n, ok := isInt(value); ok {
			if n < 0 {
				return newError(
					fmt.Sprintf("Repeat count must be positive, but received %d", n),
					"Use positive integers: 1, 2, 3, etc.",
					"Use true for infinite repetitions",
				)
			}
			return newSuccess(n, "scheduler")
		}

		return newError(
			fmt.Sprintf("Repeat must be a number, true, or false, but received %s", describeValue(value)),
			"Use a number to specify exact repetitions (e.g., 5)",
			"Use true for infinite repetitions",
			"Use false or omit to execute only once",
		)
	},

	// === PROCESSING OPERATORS (Placeholders with function validation) ===
	"schema": func(value interface{}) DataDefResult {
		if value == nil {
			return newSuccess(nil)
		}
		if !isFunction(value) && reflect.TypeOf(value).Kind() != reflect.Map {
			return newError(
				fmt.Sprintf("Schema must be a validation function or object, but received %s", describeValue(value)),
				"Use schema builders or provide validation function",
				"Function should return { ok: boolean, data?: any, errors?: string[] }",
			)
		}
		return newSuccess(value, "schema")
	},

	"condition": func(value interface{}) DataDefResult {
		if value == nil {
			return newSuccess(nil)
		}
		if !isFunction(value) {
			return newError(
				fmt.Sprintf("Condition must be a function that returns true or false, but received %s", describeValue(value)),
				"Function should return boolean: (payload) => boolean",
				"Return true to allow execution, false to skip",
				"Example: (payload) => payload.status === 'active'",
			)
		}
		return newSuccess(value, "condition")
	},

	"selector": func(value interface{}) DataDefResult {
		if value == nil {
			return newSuccess(nil)
		}
		if !isFunction(value) {
			return newError(
				fmt.Sprintf("Selector must be a function that extracts data, but received %s", describeValue(value)),
				"Function should extract part of your data: (payload) => any",
				"Return the specific data you want to use",
				"Example: (payload) => payload.user.email",
			)
		}
		return newSuccess(value, "selector")
	},

	"transform": func(value interface{}) DataDefResult {
		if value == nil {
			return newSuccess(nil)
		}
		if !isFunction(value) {
			return newError(
				fmt.Sprintf("Transform must be a function that modifies data, but received %s", describeValue(value)),
				"Function should return modified data: (payload) => any",
				"Transform and return your data as needed",
				"Example: (payload) => ({ ...payload, processed: true })",
			)
		}
		return newSuccess(value, "transform")
	},

	"auth": func(value interface{}) DataDefResult {
		if value == nil {
			return newSuccess(nil)
		}
		// Accept map/struct for auth configuration
		rv := reflect.ValueOf(value)
		if rv.Kind() != reflect.Map && rv.Kind() != reflect.Struct {
			return newError(
				fmt.Sprintf("Auth must be an object, but received %s", describeValue(value)),
				"Provide auth configuration object",
				"Example: {mode: 'token', token: 'abc123'}",
			)
		}
		return newSuccess(value, "auth")
	},

	// === TAGS (Special handling for slice) ===
	"tags": func(value interface{}) DataDefResult {
		if value == nil {
			return newSuccess(nil)
		}
		rv := reflect.ValueOf(value)
		if rv.Kind() != reflect.Slice {
			return newError(
				fmt.Sprintf("Tags must be an array, but received %s", describeValue(value)),
				"Provide array of tag strings",
				"Example: ['user', 'authentication', 'critical']",
			)
		}
		// Convert to []string
		tags := make([]string, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			if s, ok := rv.Index(i).Interface().(string); ok {
				tags[i] = s
			} else {
				return newError(
					fmt.Sprintf("Tag at index %d must be text, but received %s", i, describeValue(rv.Index(i).Interface())),
					"All tags must be text values",
				)
			}
		}
		return newSuccess(tags)
	},

	// === INTERNAL FIELDS (Pass-through) ===
	"custom":              func(value interface{}) DataDefResult { return newSuccess(value) },
	"_isBlocked":          func(value interface{}) DataDefResult { return newSuccess(value) },
	"_blockReason":        func(value interface{}) DataDefResult { return newSuccess(value) },
	"_hasFastPath":        func(value interface{}) DataDefResult { return newSuccess(value) },
	"_pipeline":           func(value interface{}) DataDefResult { return newSuccess(value) },
	"_debounceTimer":      func(value interface{}) DataDefResult { return newSuccess(value) },
	"_isScheduled":        func(value interface{}) DataDefResult { return newSuccess(value) },
	"_branchId":           func(value interface{}) DataDefResult { return newSuccess(value) },
	"_hasProtections":     func(value interface{}) DataDefResult { return newSuccess(value) },
	"_hasProcessing":      func(value interface{}) DataDefResult { return newSuccess(value) },
	"_hasScheduling":      func(value interface{}) DataDefResult { return newSuccess(value) },
	"_processingTalents":  func(value interface{}) DataDefResult { return newSuccess(value) },
	"_hasChangeDetection": func(value interface{}) DataDefResult { return newSuccess(value) },
	"_timestamp":          func(value interface{}) DataDefResult { return newSuccess(value) },
	"_timeOfCreation":     func(value interface{}) DataDefResult { return newSuccess(value) },
	"_lastExecTime":       func(value interface{}) DataDefResult { return newSuccess(value) },
	"_executionDuration":  func(value interface{}) DataDefResult { return newSuccess(value) },
	"_executionCount":     func(value interface{}) DataDefResult { return newSuccess(value) },
	"_errorCount":         func(value interface{}) DataDefResult { return newSuccess(value) },
}

// === TALENT CATEGORIES ===

var (
	ProtectionTalents = map[string]bool{
		"block": true, "throttle": true, "debounce": true, "detectChanges": true,
		"required": true, "priority": true, "log": true,
	}

	ProcessingTalents = map[string]bool{
		"schema": true, "condition": true, "selector": true, "transform": true, "auth": true,
	}

	SchedulingTalents = map[string]bool{
		"delay": true, "interval": true, "repeat": true,
	}
)

// === MAIN COMPILATION FUNCTION ===

// CompileAction validates and compiles an action configuration
// Preserves user declaration order and builds talent pipeline
func CompileAction(action types.IO) CompileResult {
	errors := []string{}
	warnings := []string{}
	talentPipeline := []string{}

	// Track talent categories
	hasProtections := false
	hasProcessing := false
	hasScheduling := false

	// Create compiled action starting with input
	compiledAction := action

	// Get field names in declaration order using reflection
	actionValue := reflect.ValueOf(action)
	actionType := reflect.TypeOf(action)

	// Process fields in struct declaration order (preserves user intent)
	for i := 0; i < actionType.NumField(); i++ {
		field := actionType.Field(i)
		fieldValue := actionValue.Field(i)

		// Skip unexported fields
		if !fieldValue.CanInterface() {
			continue
		}

		// Get field name from JSON tag or struct name
		fieldName := field.Tag.Get("json")
		if fieldName == "" || fieldName == "-" {
			fieldName = strings.ToLower(field.Name)
		}
		// Remove omitempty suffix
		if commaIdx := strings.Index(fieldName, ","); commaIdx != -1 {
			fieldName = fieldName[:commaIdx]
		}

		// Skip zero values (unset fields)
		if fieldValue.IsZero() {
			continue
		}

		// Get validation function (O(1) lookup)
		validator, exists := DataDefinitions[fieldName]
		if !exists {
			// Unknown field - pass through with warning
			warnings = append(warnings, fmt.Sprintf("Unknown field: %s", fieldName))
			continue
		}

		// Validate field
		result := validator(fieldValue.Interface())

		if !result.OK {
			if result.Blocking {
				// Immediate blocking failure - return early
				compiledAction.Block = true
				compiledAction.IsBlocked = true
				compiledAction.BlockReason = result.Error
				return CompileResult{
					CompiledAction: &compiledAction,
					Errors:         []string{result.Error},
					Warnings:       warnings,
					HasFastPath:    false,
				}
			}
			// Non-blocking error
			errors = append(errors, result.Error)
		} else {
			// Successful validation - update compiled action
			// Use reflection to set the validated value
			compiledField := reflect.ValueOf(&compiledAction).Elem().FieldByName(field.Name)
			if compiledField.CanSet() && result.Data != nil {
				dataValue := reflect.ValueOf(result.Data)
				if dataValue.Type().AssignableTo(compiledField.Type()) {
					compiledField.Set(dataValue)
				}
			}

			// Track talent and add to pipeline if it has an operator
			if result.TalentName != "" {
				// Add to pipeline in user declaration order
				talentPipeline = append(talentPipeline, result.TalentName)

				// Track talent categories
				if ProtectionTalents[result.TalentName] {
					hasProtections = true
				}
				if ProcessingTalents[result.TalentName] {
					hasProcessing = true
				}
				if SchedulingTalents[result.TalentName] || result.TalentName == "scheduler" {
					hasScheduling = true
				}
			}
		}
	}

	// Determine fast path eligibility
	hasFastPath := !hasProtections && !hasProcessing && !hasScheduling

	// Set compilation flags
	compiledAction.HasFastPath = hasFastPath
	compiledAction.HasProtections = hasProtections
	compiledAction.HasProcessing = hasProcessing
	compiledAction.HasScheduling = hasScheduling
	compiledAction.IsScheduled = hasScheduling

	return CompileResult{
		CompiledAction: &compiledAction,
		Errors:         errors,
		Warnings:       warnings,
		HasFastPath:    hasFastPath,
		TalentPipeline: talentPipeline,
		HasProtections: hasProtections,
		HasProcessing:  hasProcessing,
		HasScheduling:  hasScheduling,
	}
}

// === UTILITY FUNCTIONS ===

// ValidateField validates a single field
func ValidateField(fieldName string, value interface{}) DataDefResult {
	if validator, exists := DataDefinitions[fieldName]; exists {
		return validator(value)
	}
	return newError(fmt.Sprintf("Unknown field: %s", fieldName))
}

// GetTalentName returns the talent name for a field, if any
func GetTalentName(fieldName string, value interface{}) string {
	result := ValidateField(fieldName, value)
	return result.TalentName
}

// IsValidField checks if a field name is recognized
func IsValidField(fieldName string) bool {
	_, exists := DataDefinitions[fieldName]
	return exists
}

// GetAllFields returns all recognized field names
func GetAllFields() []string {
	fields := make([]string, 0, len(DataDefinitions))
	for field := range DataDefinitions {
		fields = append(fields, field)
	}
	return fields
}
