package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestLevel_String(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{DebugLevel, "DEBUG"},
		{InfoLevel, "INFO"},
		{WarnLevel, "WARN"},
		{ErrorLevel, "ERROR"},
		{FatalLevel, "FATAL"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("Level.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNew(t *testing.T) {
	buf := &bytes.Buffer{}
	cfg := &Config{
		Level:  DebugLevel,
		Output: buf,
		Caller: true,
	}

	logger := New(cfg)
	if logger == nil {
		t.Fatal("New() returned nil")
	}

	dl, ok := logger.(*DefaultLogger)
	if !ok {
		t.Fatal("New() did not return *DefaultLogger")
	}

	if dl.level != DebugLevel {
		t.Errorf("level = %v, want %v", dl.level, DebugLevel)
	}

	if dl.output != buf {
		t.Error("output not set correctly")
	}
}

func TestDefaultLogger_SetLevel(t *testing.T) {
	logger := Default()

	logger.SetLevel(ErrorLevel)
	if logger.GetLevel() != ErrorLevel {
		t.Errorf("GetLevel() = %v, want %v", logger.GetLevel(), ErrorLevel)
	}
}

func TestDefaultLogger_Info(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(&Config{
		Level:  InfoLevel,
		Output: buf,
		Caller: false,
	})

	logger.Info("test message", String("key", "value"))

	var entry Entry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if entry.Level != "INFO" {
		t.Errorf("Level = %v, want INFO", entry.Level)
	}

	if entry.Message != "test message" {
		t.Errorf("Message = %v, want 'test message'", entry.Message)
	}

	if entry.Fields["key"] != "value" {
		t.Errorf("Field key = %v, want 'value'", entry.Fields["key"])
	}
}

func TestDefaultLogger_Debug_FilteredByLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(&Config{
		Level:  InfoLevel,
		Output: buf,
	})

	logger.Debug("debug message")

	if buf.Len() > 0 {
		t.Error("Debug message was logged when level was Info")
	}
}

func TestDefaultLogger_Error(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(&Config{
		Level:  ErrorLevel,
		Output: buf,
		Caller: false,
	})

	testErr := errors.New("test error")
	logger.Error("error occurred", Error(testErr))

	var entry Entry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if entry.Level != "ERROR" {
		t.Errorf("Level = %v, want ERROR", entry.Level)
	}

	if entry.Fields["error"] != "test error" {
		t.Errorf("Error field = %v, want 'test error'", entry.Fields["error"])
	}
}

func TestDefaultLogger_WithFields(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(&Config{
		Level:  InfoLevel,
		Output: buf,
		Caller: false,
	})

	childLogger := logger.WithFields(
		String("service", "api"),
		Int("port", 8080),
	)

	childLogger.Info("server started")

	var entry Entry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if entry.Fields["service"] != "api" {
		t.Errorf("service field = %v, want 'api'", entry.Fields["service"])
	}

	if entry.Fields["port"] != float64(8080) {
		t.Errorf("port field = %v, want 8080", entry.Fields["port"])
	}
}

func TestDefaultLogger_WithContext(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(&Config{
		Level:  InfoLevel,
		Output: buf,
		Caller: false,
	})

	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-123")
	ctx = WithUserID(ctx, "user-456")
	ctx = WithTraceID(ctx, "trace-789")

	contextLogger := logger.WithContext(ctx)
	contextLogger.Info("request processed")

	var entry Entry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if entry.Fields["request_id"] != "req-123" {
		t.Errorf("request_id = %v, want 'req-123'", entry.Fields["request_id"])
	}

	if entry.Fields["user_id"] != "user-456" {
		t.Errorf("user_id = %v, want 'user-456'", entry.Fields["user_id"])
	}

	if entry.Fields["trace_id"] != "trace-789" {
		t.Errorf("trace_id = %v, want 'trace-789'", entry.Fields["trace_id"])
	}
}

func TestDefaultLogger_Caller(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(&Config{
		Level:  InfoLevel,
		Output: buf,
		Caller: true,
	})

	logger.Info("test with caller")

	var entry Entry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if entry.Caller == "" {
		t.Error("Caller field is empty")
	}

	if !strings.Contains(entry.Caller, "logger_test.go") {
		t.Errorf("Caller = %v, does not contain 'logger_test.go'", entry.Caller)
	}
}

func TestFieldHelpers(t *testing.T) {
	tests := []struct {
		name     string
		field    Field
		expected interface{}
	}{
		{"String", String("key", "value"), "value"},
		{"Int", Int("key", 42), 42},
		{"Int64", Int64("key", int64(42)), int64(42)},
		{"Float64", Float64("key", 3.14), 3.14},
		{"Bool", Bool("key", true), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.field.Key != "key" {
				t.Errorf("Key = %v, want 'key'", tt.field.Key)
			}
			if tt.field.Value != tt.expected {
				t.Errorf("Value = %v, want %v", tt.field.Value, tt.expected)
			}
		})
	}
}

func TestDurationField(t *testing.T) {
	field := Duration("elapsed", 100*time.Millisecond)

	if field.Key != "elapsed" {
		t.Errorf("Key = %v, want 'elapsed'", field.Key)
	}

	if field.Value != "100ms" {
		t.Errorf("Value = %v, want '100ms'", field.Value)
	}
}

func TestErrorField(t *testing.T) {
	err := errors.New("test error")
	field := Error(err)

	if field.Key != "error" {
		t.Errorf("Key = %v, want 'error'", field.Key)
	}

	if field.Value != "test error" {
		t.Errorf("Value = %v, want 'test error'", field.Value)
	}

	field = Error(nil)
	if field.Value != nil {
		t.Errorf("Value = %v, want nil", field.Value)
	}
}

func TestAnyField(t *testing.T) {
	type custom struct {
		Name string
		Age  int
	}

	obj := custom{Name: "Alice", Age: 30}
	field := Any("user", obj)

	if field.Key != "user" {
		t.Errorf("Key = %v, want 'user'", field.Key)
	}

	result, ok := field.Value.(custom)
	if !ok {
		t.Fatal("Value is not of type custom")
	}

	if result.Name != "Alice" || result.Age != 30 {
		t.Errorf("Value = %+v, want {Name:Alice Age:30}", result)
	}
}

func TestGlobalLogger(t *testing.T) {
	buf := &bytes.Buffer{}
	customLogger := New(&Config{
		Level:  InfoLevel,
		Output: buf,
		Caller: false,
	})

	SetGlobalLogger(customLogger)

	Info("global logger test", String("source", "global"))

	var entry Entry
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if entry.Message != "global logger test" {
		t.Errorf("Message = %v, want 'global logger test'", entry.Message)
	}

	if entry.Fields["source"] != "global" {
		t.Errorf("source field = %v, want 'global'", entry.Fields["source"])
	}
}

func TestLogLevels(t *testing.T) {
	tests := []struct {
		name      string
		logLevel  Level
		logFunc   func(Logger, string)
		message   string
		shouldLog bool
	}{
		{"Debug at Debug level", DebugLevel, func(l Logger, m string) { l.Debug(m) }, "debug msg", true},
		{"Debug at Info level", InfoLevel, func(l Logger, m string) { l.Debug(m) }, "debug msg", false},
		{"Info at Info level", InfoLevel, func(l Logger, m string) { l.Info(m) }, "info msg", true},
		{"Info at Error level", ErrorLevel, func(l Logger, m string) { l.Info(m) }, "info msg", false},
		{"Error at Info level", InfoLevel, func(l Logger, m string) { l.Error(m) }, "error msg", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := New(&Config{
				Level:  tt.logLevel,
				Output: buf,
			})

			tt.logFunc(logger, tt.message)

			logged := buf.Len() > 0
			if logged != tt.shouldLog {
				t.Errorf("Message logged = %v, want %v", logged, tt.shouldLog)
			}

			if logged {
				var entry Entry
				if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
					t.Fatalf("Failed to parse log output: %v", err)
				}

				if entry.Message != tt.message {
					t.Errorf("Message = %v, want %v", entry.Message, tt.message)
				}
			}
		})
	}
}

func TestConcurrentLogging(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := New(&Config{
		Level:  InfoLevel,
		Output: buf,
		Caller: false,
	})

	done := make(chan bool, 100)

	for i := 0; i < 100; i++ {
		go func(id int) {
			logger.Info("concurrent message", Int("id", id))
			done <- true
		}(i)
	}

	for i := 0; i < 100; i++ {
		<-done
	}

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 100 {
		t.Errorf("Got %d log lines, want 100", len(lines))
	}
}

func BenchmarkLogger_Info(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := New(&Config{
		Level:  InfoLevel,
		Output: buf,
		Caller: false,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message")
	}
}

func BenchmarkLogger_InfoWithFields(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := New(&Config{
		Level:  InfoLevel,
		Output: buf,
		Caller: false,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message",
			String("key1", "value1"),
			Int("key2", 42),
			Bool("key3", true),
		)
	}
}

func BenchmarkLogger_WithFields(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := New(&Config{
		Level:  InfoLevel,
		Output: buf,
		Caller: false,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		childLogger := logger.WithFields(
			String("service", "api"),
			Int("port", 8080),
		)
		childLogger.Info("benchmark message")
	}
}

func BenchmarkLogger_WithContext(b *testing.B) {
	buf := &bytes.Buffer{}
	logger := New(&Config{
		Level:  InfoLevel,
		Output: buf,
		Caller: false,
	})

	ctx := context.Background()
	ctx = WithRequestID(ctx, "req-123")
	ctx = WithUserID(ctx, "user-456")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.WithContext(ctx).Info("benchmark message")
	}
}
