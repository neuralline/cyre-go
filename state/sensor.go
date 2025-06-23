// state/sensor.go
// Cyre Go Sensor - TypeScript-style API for terminal logging
// Matches Cyre TypeScript sensor.error() pattern

package state

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// === LOG LEVELS ===

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	SUCCESS
	CRITICAL
	SYS
)

func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case SUCCESS:
		return "SUCCESS"
	case CRITICAL:
		return "CRITICAL"
	case SYS:
		return "SYS"
	default:
		return "UNKNOWN"
	}
}

// LogsByDefault returns true for levels that should log without force
func (l LogLevel) LogsByDefault() bool {
	return l == ERROR || l == CRITICAL
}

// === COLORS ===

const (
	Reset        = "\033[0m"
	Bold         = "\033[1m"
	Dim          = "\033[2m"
	Cyan         = "\033[36m"
	YellowBright = "\033[93m"
	RedBright    = "\033[91m"
	GreenBright  = "\033[92m"
	WhiteBright  = "\033[97m"
	White        = "\033[37m"
	BgRed        = "\033[41m"
	BgMagenta    = "\033[45m"
)

func (l LogLevel) ColorStyle() string {
	switch l {
	case DEBUG:
		return Dim
	case INFO:
		return Cyan
	case WARN:
		return YellowBright
	case ERROR:
		return RedBright
	case SUCCESS:
		return GreenBright
	case CRITICAL:
		return BgRed
	case SYS:
		return BgMagenta
	default:
		return Reset
	}
}

// === SENSOR ENTRY ===

type SensorEntry struct {
	Message   string                 `json:"message"`
	Location  string                 `json:"location,omitempty"`
	ActionID  string                 `json:"actionId,omitempty"`
	EventType string                 `json:"eventType,omitempty"`
	Level     LogLevel               `json:"level"`
	Force     bool                   `json:"force,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// === SENSOR BUILDER (for chaining) ===

type SensorBuilder struct {
	entry SensorEntry
}

func newSensorBuilder(message string, level LogLevel) *SensorBuilder {
	return &SensorBuilder{
		entry: SensorEntry{
			Message:   message,
			Level:     level,
			Timestamp: time.Now(),
		},
	}
}

func (b *SensorBuilder) Location(location string) *SensorBuilder {
	b.entry.Location = location
	return b
}

func (b *SensorBuilder) Id(actionID string) *SensorBuilder {
	b.entry.ActionID = actionID
	return b
}

func (b *SensorBuilder) EventType(eventType string) *SensorBuilder {
	b.entry.EventType = eventType
	return b
}

func (b *SensorBuilder) Force() *SensorBuilder {
	b.entry.Force = true
	return b
}

func (b *SensorBuilder) Metadata(metadata map[string]interface{}) *SensorBuilder {
	b.entry.Metadata = metadata
	return b
}

func (b *SensorBuilder) Log() {
	defaultSensor.writeEntry(b.entry)
}

// === MAIN SENSOR STRUCT ===

type Sensor struct {
	enabled bool
}

var defaultSensor = &Sensor{enabled: true}

// === TYPESCRIPT-STYLE API ===

// Critical logs a critical message (always logs)
// sensor.Critical("Database connection lost!")
func Critical(message string) *SensorBuilder {
	builder := newSensorBuilder(message, CRITICAL)
	builder.Log() // Auto-log for critical
	return builder
}

// Error logs an error message (always logs)
// sensor.Error("Failed to process request")
func Error(message string) *SensorBuilder {
	builder := newSensorBuilder(message, ERROR)
	builder.Log() // Auto-log for errors
	return builder
}

// Warn logs a warning message (requires .Force() to actually log)
// sensor.Warn("High memory usage").Force().Log()
func Warn(message string) *SensorBuilder {
	return newSensorBuilder(message, WARN)
}

// Info logs an info message (requires .Force() to actually log)
// sensor.Info("User logged in").Force().Log()
func Info(message string) *SensorBuilder {
	return newSensorBuilder(message, INFO)
}

// Success logs a success message (requires .Force() to actually log)
// sensor.Success("Operation completed").Force().Log()
func Success(message string) *SensorBuilder {
	return newSensorBuilder(message, SUCCESS)
}

// Debug logs a debug message (requires .Force() to actually log)
// sensor.Debug("Cache hit for key: user_123").Force().Log()
func Debug(message string) *SensorBuilder {
	return newSensorBuilder(message, DEBUG)
}

// Sys logs a system message (requires .Force() to actually log)
// sensor.Sys("Quantum Breathing System activated").Force().Log()
func Sys(message string) *SensorBuilder {
	return newSensorBuilder(message, SYS)
}

// === PACKAGE-LEVEL CONVENIENCE (matches TypeScript sensor) ===

// These match the TypeScript sensor.error() pattern exactly

var sensor = struct {
	Critical func(string) *SensorBuilder
	Error    func(string) *SensorBuilder
	Warn     func(string) *SensorBuilder
	Info     func(string) *SensorBuilder
	Success  func(string) *SensorBuilder
	Debug    func(string) *SensorBuilder
	Sys      func(string) *SensorBuilder
}{
	Critical: Critical,
	Error:    Error,
	Warn:     Warn,
	Info:     Info,
	Success:  Success,
	Debug:    Debug,
	Sys:      Sys,
}

// GetSensor returns the sensor object for TypeScript-style usage
// Usage: sensor := context.GetSensor()
//
//	sensor.Error("Something went wrong")
func GetSensor() *struct {
	Critical func(string) *SensorBuilder
	Error    func(string) *SensorBuilder
	Warn     func(string) *SensorBuilder
	Info     func(string) *SensorBuilder
	Success  func(string) *SensorBuilder
	Debug    func(string) *SensorBuilder
	Sys      func(string) *SensorBuilder
} {
	return &sensor
}

// === INTERNAL IMPLEMENTATION ===

func (s *Sensor) writeEntry(entry SensorEntry) {
	// Check if we should log this entry
	shouldLog := entry.Force || entry.Level.LogsByDefault()

	if shouldLog {
		s.writeToTerminal(entry)
	} else {
		// In debug mode, show suppressed logs
		if isDebugMode() {
			fmt.Printf("%s%s[SUPPRESSED] %s %s: %s%s\n",
				Dim, Cyan, formatTimestamp(entry.Timestamp),
				entry.Level, entry.Message, Reset)
		}
	}
}

func (s *Sensor) writeToTerminal(entry SensorEntry) {
	timestamp := formatTimestamp(entry.Timestamp)
	colorStyle := entry.Level.ColorStyle()

	// Build log line: [timestamp] LEVEL: message
	var logLine strings.Builder
	logLine.WriteString(colorStyle)
	logLine.WriteString(fmt.Sprintf("[%s] %s: %s", timestamp, entry.Level, entry.Message))

	// Add optional enrichment in priority order: location, actionId, eventType, metadata
	if entry.Location != "" {
		logLine.WriteString(fmt.Sprintf(" @%s", entry.Location))
	}
	if entry.ActionID != "" {
		logLine.WriteString(fmt.Sprintf(" (%s)", entry.ActionID))
	}
	if entry.EventType != "" {
		logLine.WriteString(fmt.Sprintf(" [%s]", entry.EventType))
	}
	if entry.Metadata != nil && len(entry.Metadata) > 0 {
		logLine.WriteString(fmt.Sprintf(" %s", formatMetadata(entry.Metadata)))
	}

	// Reset colors
	logLine.WriteString(Reset)

	fmt.Println(logLine.String())
}

// === HELPER FUNCTIONS ===

func formatTimestamp(t time.Time) string {
	// Format as seconds.milliseconds within the day (like Rust version)
	secs := t.Unix() % 86400 // Seconds within the day
	millis := t.Nanosecond() / 1000000
	return fmt.Sprintf("%d.%03dZ", secs, millis)
}

func formatMetadata(metadata map[string]interface{}) string {
	if metadata == nil || len(metadata) == 0 {
		return ""
	}

	var parts []string
	for k, v := range metadata {
		parts = append(parts, fmt.Sprintf("%s:%v", k, v))
	}
	return fmt.Sprintf("{%s}", strings.Join(parts, " "))
}

func isDebugMode() bool {
	return os.Getenv("CYRE_DEBUG") == "true" || os.Getenv("DEBUG") == "true"
}
