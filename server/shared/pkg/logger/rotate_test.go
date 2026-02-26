package logger

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultRotateConfig(t *testing.T) {
	filename := "test.log"
	cfg := DefaultRotateConfig(filename)

	if cfg.Filename != filename {
		t.Errorf("expected filename %s, got %s", filename, cfg.Filename)
	}

	if cfg.MaxSize != 100*1024*1024 {
		t.Errorf("expected maxSize 104857600, got %d", cfg.MaxSize)
	}

	if cfg.MaxBackups != 10 {
		t.Errorf("expected maxBackups 10, got %d", cfg.MaxBackups)
	}

	if cfg.MaxAge != 30 {
		t.Errorf("expected maxAge 30, got %d", cfg.MaxAge)
	}
}

func TestNewRotateWriter_NilConfig(t *testing.T) {
	_, err := NewRotateWriter(nil)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestNewRotateWriter_EmptyFilename(t *testing.T) {
	cfg := &RotateConfig{
		Filename: "",
	}
	_, err := NewRotateWriter(cfg)
	if err == nil {
		t.Error("expected error for empty filename")
	}
}

func TestNewRotateWriter_DefaultValues(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.log")

	cfg := &RotateConfig{
		Filename:   filename,
		MaxSize:    -1, // Should default to 100MB
		MaxBackups: -1, // Should default to 0
		MaxAge:     -1, // Should default to 0
	}

	w, err := NewRotateWriter(cfg)
	if err != nil {
		t.Fatalf("failed to create rotate writer: %v", err)
	}
	defer w.Close()

	if w.maxSize != 100*1024*1024 {
		t.Errorf("expected default maxSize 104857600, got %d", w.maxSize)
	}

	if w.maxBackups != 0 {
		t.Errorf("expected default maxBackups 0, got %d", w.maxBackups)
	}

	if w.maxAge != 0 {
		t.Errorf("expected default maxAge 0, got %d", w.maxAge)
	}
}

func TestRotateWriter_Write(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.log")

	cfg := &RotateConfig{
		Filename:   filename,
		MaxSize:    1024 * 1024, // 1MB
		MaxBackups: 5,
		MaxAge:     7,
	}

	w, err := NewRotateWriter(cfg)
	if err != nil {
		t.Fatalf("failed to create rotate writer: %v", err)
	}
	defer w.Close()

	// Write some data
	data := []byte("test log entry\n")
	n, err := w.Write(data)
	if err != nil {
		t.Errorf("write failed: %v", err)
	}

	if n != len(data) {
		t.Errorf("expected to write %d bytes, wrote %d", len(data), n)
	}

	// Close and verify file contents
	w.Close()

	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	if string(content) != string(data) {
		t.Errorf("expected content %q, got %q", string(data), string(content))
	}
}

func TestRotateWriter_Rotation(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.log")

	// Set small max size to trigger rotation
	cfg := &RotateConfig{
		Filename:   filename,
		MaxSize:    100, // 100 bytes
		MaxBackups: 3,
		MaxAge:     7,
	}

	w, err := NewRotateWriter(cfg)
	if err != nil {
		t.Fatalf("failed to create rotate writer: %v", err)
	}
	defer w.Close()

	// Write data that exceeds maxSize
	data := []byte(strings.Repeat("x", 150))
	_, err = w.Write(data)
	if err != nil {
		t.Errorf("write failed: %v", err)
	}

	// Give it a moment to complete rotation
	time.Sleep(100 * time.Millisecond)

	// Check that backup file was created
	matches, err := filepath.Glob(filepath.Join(tmpDir, "test-*.log"))
	if err != nil {
		t.Fatalf("failed to glob backup files: %v", err)
	}

	if len(matches) != 1 {
		t.Errorf("expected 1 backup file, found %d", len(matches))
	}
}

func TestRotateWriter_MaxBackups(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.log")

	cfg := &RotateConfig{
		Filename:   filename,
		MaxSize:    50, // 50 bytes
		MaxBackups: 2,  // Keep only 2 backups
		MaxAge:     0,
	}

	w, err := NewRotateWriter(cfg)
	if err != nil {
		t.Fatalf("failed to create rotate writer: %v", err)
	}
	defer w.Close()

	// Write multiple times to create multiple rotations
	for i := 0; i < 5; i++ {
		data := []byte(strings.Repeat(fmt.Sprintf("%d", i), 60))
		_, err = w.Write(data)
		if err != nil {
			t.Errorf("write %d failed: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	w.Close()

	// Check number of backup files
	matches, err := filepath.Glob(filepath.Join(tmpDir, "test-*.log"))
	if err != nil {
		t.Fatalf("failed to glob backup files: %v", err)
	}

	// Should have at most maxBackups files
	if len(matches) > cfg.MaxBackups {
		t.Errorf("expected at most %d backup files, found %d", cfg.MaxBackups, len(matches))
	}
}

func TestRotateWriter_MaxAge(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.log")

	// Create some old backup files
	oldFile1 := filepath.Join(tmpDir, "test-20200101-120000.log")
	oldFile2 := filepath.Join(tmpDir, "test-20200102-120000.log")
	recentFile := filepath.Join(tmpDir, fmt.Sprintf("test-%s.log", time.Now().Add(-1*time.Hour).Format("20060102-150405")))

	for _, f := range []string{oldFile1, oldFile2, recentFile} {
		if err := os.WriteFile(f, []byte("old log"), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", f, err)
		}
	}

	// Set modification times
	oldTime := time.Now().AddDate(0, 0, -40) // 40 days ago
	os.Chtimes(oldFile1, oldTime, oldTime)
	os.Chtimes(oldFile2, oldTime, oldTime)

	cfg := &RotateConfig{
		Filename:   filename,
		MaxSize:    50,
		MaxBackups: 10,
		MaxAge:     30, // Keep logs for 30 days
	}

	w, err := NewRotateWriter(cfg)
	if err != nil {
		t.Fatalf("failed to create rotate writer: %v", err)
	}

	// Write to trigger cleanup
	data := []byte(strings.Repeat("x", 60))
	w.Write(data)
	w.Close()

	// Check that old files were removed
	if _, err := os.Stat(oldFile1); !os.IsNotExist(err) {
		t.Errorf("old file %s should have been removed", oldFile1)
	}

	if _, err := os.Stat(oldFile2); !os.IsNotExist(err) {
		t.Errorf("old file %s should have been removed", oldFile2)
	}

	// Recent file should still exist
	if _, err := os.Stat(recentFile); os.IsNotExist(err) {
		t.Errorf("recent file %s should still exist", recentFile)
	}
}

func TestRotateWriter_Close(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.log")

	cfg := DefaultRotateConfig(filename)
	w, err := NewRotateWriter(cfg)
	if err != nil {
		t.Fatalf("failed to create rotate writer: %v", err)
	}

	data := []byte("test data\n")
	w.Write(data)

	err = w.Close()
	if err != nil {
		t.Errorf("close failed: %v", err)
	}

	// Close again should not error
	err = w.Close()
	if err != nil {
		t.Errorf("second close should not error: %v", err)
	}
}

func TestMultiWriter(t *testing.T) {
	var buf1, buf2, buf3 bytes.Buffer

	mw := NewMultiWriter(&buf1, &buf2, &buf3)

	data := []byte("test data")
	n, err := mw.Write(data)
	if err != nil {
		t.Errorf("write failed: %v", err)
	}

	if n != len(data) {
		t.Errorf("expected to write %d bytes, wrote %d", len(data), n)
	}

	// Verify all writers received the data
	if buf1.String() != string(data) {
		t.Errorf("buf1: expected %q, got %q", string(data), buf1.String())
	}

	if buf2.String() != string(data) {
		t.Errorf("buf2: expected %q, got %q", string(data), buf2.String())
	}

	if buf3.String() != string(data) {
		t.Errorf("buf3: expected %q, got %q", string(data), buf3.String())
	}
}

func TestMultiWriter_Error(t *testing.T) {
	var buf bytes.Buffer
	errorWriter := &errorWriter{err: io.ErrShortWrite}

	mw := NewMultiWriter(&buf, errorWriter)

	data := []byte("test data")
	_, err := mw.Write(data)
	if err == nil {
		t.Error("expected error from multi writer")
	}
}

func TestBufferedWriter_Write(t *testing.T) {
	var buf bytes.Buffer

	cfg := &BufferedConfig{
		Writer:      &buf,
		BufferSize:  100,
		FlushSize:   50,
		FlushPeriod: time.Second,
	}

	bw := NewBufferedWriter(cfg)
	defer bw.Close()

	// Write small data (should be buffered)
	data := []byte("test")
	n, err := bw.Write(data)
	if err != nil {
		t.Errorf("write failed: %v", err)
	}

	if n != len(data) {
		t.Errorf("expected to write %d bytes, wrote %d", len(data), n)
	}

	// Data should not be in underlying writer yet
	if buf.Len() > 0 {
		t.Error("data should still be buffered")
	}

	// Flush
	bw.Flush()

	// Now data should be in underlying writer
	if buf.String() != string(data) {
		t.Errorf("expected %q, got %q", string(data), buf.String())
	}
}

func TestBufferedWriter_AutoFlush(t *testing.T) {
	var buf bytes.Buffer

	cfg := &BufferedConfig{
		Writer:      &buf,
		BufferSize:  100,
		FlushSize:   20, // Small flush size
		FlushPeriod: time.Second,
	}

	bw := NewBufferedWriter(cfg)
	defer bw.Close()

	// Write data larger than flush size
	data := []byte(strings.Repeat("x", 30))
	bw.Write(data)

	// Should auto-flush
	if buf.Len() == 0 {
		t.Error("data should have been auto-flushed")
	}

	if buf.Len() != len(data) {
		t.Errorf("expected %d bytes flushed, got %d", len(data), buf.Len())
	}
}

func TestBufferedWriter_LargeWrite(t *testing.T) {
	var buf bytes.Buffer

	cfg := &BufferedConfig{
		Writer:      &buf,
		BufferSize:  100,
		FlushSize:   100,
		FlushPeriod: time.Second,
	}

	bw := NewBufferedWriter(cfg)
	defer bw.Close()

	// Write data larger than buffer
	data := []byte(strings.Repeat("x", 200))
	n, err := bw.Write(data)
	if err != nil {
		t.Errorf("write failed: %v", err)
	}

	if n != len(data) {
		t.Errorf("expected to write %d bytes, wrote %d", len(data), n)
	}

	// Should write directly to underlying writer
	if buf.Len() != len(data) {
		t.Errorf("expected %d bytes in buffer, got %d", len(data), buf.Len())
	}
}

func TestBufferedWriter_PeriodicFlush(t *testing.T) {
	var buf bytes.Buffer

	cfg := &BufferedConfig{
		Writer:      &buf,
		BufferSize:  100,
		FlushSize:   100,
		FlushPeriod: 100 * time.Millisecond, // Short period for testing
	}

	bw := NewBufferedWriter(cfg)
	defer bw.Close()

	// Write small data
	data := []byte("test")
	bw.Write(data)

	// Initially buffered
	if buf.Len() > 0 {
		t.Error("data should still be buffered")
	}

	// Wait for periodic flush
	time.Sleep(200 * time.Millisecond)

	// Should be flushed by timer
	if buf.Len() == 0 {
		t.Error("data should have been flushed by timer")
	}

	if buf.String() != string(data) {
		t.Errorf("expected %q, got %q", string(data), buf.String())
	}
}

func TestBufferedWriter_Close(t *testing.T) {
	var buf bytes.Buffer

	cfg := &BufferedConfig{
		Writer:      &buf,
		BufferSize:  100,
		FlushSize:   100,
		FlushPeriod: time.Second,
	}

	bw := NewBufferedWriter(cfg)

	// Write data
	data := []byte("test")
	bw.Write(data)

	// Data should be buffered
	if buf.Len() > 0 {
		t.Error("data should still be buffered")
	}

	// Close should flush
	err := bw.Close()
	if err != nil {
		t.Errorf("close failed: %v", err)
	}

	// Data should be flushed
	if buf.String() != string(data) {
		t.Errorf("expected %q, got %q", string(data), buf.String())
	}
}

func TestBufferedWriter_DefaultValues(t *testing.T) {
	var buf bytes.Buffer

	cfg := &BufferedConfig{
		Writer:      &buf,
		BufferSize:  0, // Should use default
		FlushSize:   0, // Should use default
		FlushPeriod: 0, // Should use default
	}

	bw := NewBufferedWriter(cfg)
	defer bw.Close()

	if len(bw.buffer) != 4096 {
		t.Errorf("expected default buffer size 4096, got %d", len(bw.buffer))
	}

	if bw.flushSize != 4096 {
		t.Errorf("expected default flush size 4096, got %d", bw.flushSize)
	}
}

// errorWriter is a test writer that always returns an error
type errorWriter struct {
	err error
}

func (w *errorWriter) Write(p []byte) (n int, err error) {
	return 0, w.err
}

// Benchmarks

func BenchmarkRotateWriter_Write(b *testing.B) {
	tmpDir := b.TempDir()
	filename := filepath.Join(tmpDir, "bench.log")

	cfg := &RotateConfig{
		Filename:   filename,
		MaxSize:    100 * 1024 * 1024, // 100MB
		MaxBackups: 5,
		MaxAge:     7,
	}

	w, err := NewRotateWriter(cfg)
	if err != nil {
		b.Fatalf("failed to create rotate writer: %v", err)
	}
	defer w.Close()

	data := []byte("benchmark log entry\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Write(data)
	}
}

func BenchmarkBufferedWriter_Write(b *testing.B) {
	tmpDir := b.TempDir()
	filename := filepath.Join(tmpDir, "bench.log")

	file, err := os.Create(filename)
	if err != nil {
		b.Fatalf("failed to create file: %v", err)
	}
	defer file.Close()

	cfg := &BufferedConfig{
		Writer:      file,
		BufferSize:  8192,
		FlushSize:   8192,
		FlushPeriod: time.Second,
	}

	bw := NewBufferedWriter(cfg)
	defer bw.Close()

	data := []byte("benchmark log entry\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bw.Write(data)
	}
}

func BenchmarkMultiWriter_Write(b *testing.B) {
	var buf1, buf2, buf3 bytes.Buffer
	mw := NewMultiWriter(&buf1, &buf2, &buf3)

	data := []byte("benchmark log entry\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mw.Write(data)
	}
}
