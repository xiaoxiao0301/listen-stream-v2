// Package logger provides structured logging with context support.
//
// It wraps popular logging libraries and provides:
// - Structured logging (JSON format)
// - Log levels (Debug, Info, Warn, Error)
// - Context propagation (request ID, user ID, etc.)
// - Field-based logging
// - Performance optimization
package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"
)

// Level represents the severity of a log entry.
type Level int

const (
	// DebugLevel for debug messages.
	DebugLevel Level = iota
	// InfoLevel for informational messages.
	InfoLevel
	// WarnLevel for warning messages.
	WarnLevel
	// ErrorLevel for error messages.
	ErrorLevel
	// FatalLevel for fatal messages (calls os.Exit(1)).
	FatalLevel
)

// String returns the string representation of the log level.
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	case FatalLevel:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Field represents a key-value pair for structured logging.
type Field struct {
	Key   string
	Value interface{}
}

// Logger is the main logger interface.
type Logger interface {
	// Level management
	SetLevel(level Level)
	GetLevel() Level

	// Basic logging
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)

	// Context logging
	WithContext(ctx context.Context) Logger
	WithFields(fields ...Field) Logger

	// Writer interface
	Writer() io.Writer
}

// Entry represents a single log entry.
type Entry struct {
	Time    time.Time              `json:"time"`
	Level   string                 `json:"level"`
	Message string                 `json:"message"`
	Fields  map[string]interface{} `json:"fields,omitempty"`
	Caller  string                 `json:"caller,omitempty"`
}

// DefaultLogger is the default logger implementation.
type DefaultLogger struct {
	level      Level
	output     io.Writer
	mu         sync.Mutex
	fields     []Field
	timeFormat string
	caller     bool
}

// Config holds logger configuration.
type Config struct {
	Level      Level
	Output     io.Writer
	TimeFormat string
	Caller     bool // Include caller information
}

// DefaultConfig returns default logger configuration.
func DefaultConfig() *Config {
	return &Config{
		Level:      InfoLevel,
		Output:     os.Stdout,
		TimeFormat: time.RFC3339,
		Caller:     true,
	}
}

// New creates a new logger with the given configuration.
func New(cfg *Config) Logger {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}
	if cfg.TimeFormat == "" {
		cfg.TimeFormat = time.RFC3339
	}

	return &DefaultLogger{
		level:      cfg.Level,
		output:     cfg.Output,
		fields:     []Field{},
		timeFormat: cfg.TimeFormat,
		caller:     cfg.Caller,
	}
}

// Default returns a logger with default configuration.
func Default() Logger {
	return New(DefaultConfig())
}

// SetLevel sets the minimum log level.
func (l *DefaultLogger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// GetLevel returns the current log level.
func (l *DefaultLogger) GetLevel() Level {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.level
}

// Debug logs a debug message.
func (l *DefaultLogger) Debug(msg string, fields ...Field) {
	l.log(DebugLevel, msg, fields...)
}

// Info logs an info message.
func (l *DefaultLogger) Info(msg string, fields ...Field) {
	l.log(InfoLevel, msg, fields...)
}

// Warn logs a warning message.
func (l *DefaultLogger) Warn(msg string, fields ...Field) {
	l.log(WarnLevel, msg, fields...)
}

// Error logs an error message.
func (l *DefaultLogger) Error(msg string, fields ...Field) {
	l.log(ErrorLevel, msg, fields...)
}

// Fatal logs a fatal message and exits.
func (l *DefaultLogger) Fatal(msg string, fields ...Field) {
	l.log(FatalLevel, msg, fields...)
	os.Exit(1)
}

// WithContext returns a logger with context fields.
func (l *DefaultLogger) WithContext(ctx context.Context) Logger {
	fields := extractContextFields(ctx)
	return l.WithFields(fields...)
}

// WithFields returns a logger with additional fields.
func (l *DefaultLogger) WithFields(fields ...Field) Logger {
	newLogger := &DefaultLogger{
		level:      l.level,
		output:     l.output,
		fields:     make([]Field, len(l.fields)+len(fields)),
		timeFormat: l.timeFormat,
		caller:     l.caller,
	}
	copy(newLogger.fields, l.fields)
	copy(newLogger.fields[len(l.fields):], fields)
	return newLogger
}

// Writer returns the logger's output writer.
func (l *DefaultLogger) Writer() io.Writer {
	return l.output
}

// log is the internal logging function.
func (l *DefaultLogger) log(level Level, msg string, fields ...Field) {
	// Check if we should log at this level
	if level < l.GetLevel() {
		return
	}

	// Create entry
	entry := Entry{
		Time:    time.Now(),
		Level:   level.String(),
		Message: msg,
		Fields:  make(map[string]interface{}),
	}

	// Add logger's persistent fields
	for _, f := range l.fields {
		entry.Fields[f.Key] = f.Value
	}

	// Add message-specific fields
	for _, f := range fields {
		entry.Fields[f.Key] = f.Value
	}

	// Add caller information
	if l.caller {
		entry.Caller = getCaller(3)
	}

	// Marshal to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal log entry: %v\n", err)
		return
	}

	// Write to output
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output.Write(data)
	l.output.Write([]byte("\n"))
}

// getCaller returns the caller information.
func getCaller(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown"
	}

	// Extract just the filename (not full path)
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			file = file[i+1:]
			break
		}
	}

	return fmt.Sprintf("%s:%d", file, line)
}

// Context keys for logger fields.
type contextKey string

const (
	requestIDKey contextKey = "request_id"
	userIDKey    contextKey = "user_id"
	traceIDKey   contextKey = "trace_id"
)

// extractContextFields extracts logger fields from context.
func extractContextFields(ctx context.Context) []Field {
	var fields []Field

	if requestID, ok := ctx.Value(requestIDKey).(string); ok && requestID != "" {
		fields = append(fields, Field{Key: "request_id", Value: requestID})
	}

	if userID, ok := ctx.Value(userIDKey).(string); ok && userID != "" {
		fields = append(fields, Field{Key: "user_id", Value: userID})
	}

	if traceID, ok := ctx.Value(traceIDKey).(string); ok && traceID != "" {
		fields = append(fields, Field{Key: "trace_id", Value: traceID})
	}

	return fields
}

// WithRequestID adds request ID to context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// WithUserID adds user ID to context.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// WithTraceID adds trace ID to context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// Helper functions for creating fields

// String creates a string field.
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates an int field.
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int64 creates an int64 field.
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Float64 creates a float64 field.
func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

// Bool creates a bool field.
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Duration creates a duration field.
func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value.String()}
}

// Error creates an error field.
func Error(err error) Field {
	if err == nil {
		return Field{Key: "error", Value: nil}
	}
	return Field{Key: "error", Value: err.Error()}
}

// Any creates a field with any value.
func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// Global logger instance
var global Logger = Default()

// SetGlobalLogger sets the global logger instance.
func SetGlobalLogger(l Logger) {
	global = l
}

// Global logging functions

// Debug logs a debug message using the global logger.
func Debug(msg string, fields ...Field) {
	global.Debug(msg, fields...)
}

// Info logs an info message using the global logger.
func Info(msg string, fields ...Field) {
	global.Info(msg, fields...)
}

// Warn logs a warning message using the global logger.
func Warn(msg string, fields ...Field) {
	global.Warn(msg, fields...)
}

// ErrorLog logs an error message using the global logger.
func ErrorLog(msg string, fields ...Field) {
	global.Error(msg, fields...)
}

// Fatal logs a fatal message using the global logger.
func Fatal(msg string, fields ...Field) {
	global.Fatal(msg, fields...)
}

// WithContext returns a logger with context fields from the global logger.
func WithContext(ctx context.Context) Logger {
	return global.WithContext(ctx)
}

// WithFields returns a logger with additional fields from the global logger.
func WithFields(fields ...Field) Logger {
	return global.WithFields(fields...)
}
