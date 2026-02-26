package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// RotateWriter is a writer that supports file rotation.
type RotateWriter struct {
	filename    string
	file        *os.File
	mu          sync.Mutex
	maxSize     int64 // Maximum size in bytes
	maxBackups  int   // Maximum number of old log files to keep
	maxAge      int   // Maximum number of days to keep old log files
	currentSize int64
	openTime    time.Time
}

// RotateConfig holds configuration for log rotation.
type RotateConfig struct {
	Filename   string // Log file path
	MaxSize    int64  // Max size in MB (default: 100MB)
	MaxBackups int    // Max number of old files (default: 10)
	MaxAge     int    // Max days to keep (default: 30)
}

// DefaultRotateConfig returns default rotation configuration.
func DefaultRotateConfig(filename string) *RotateConfig {
	return &RotateConfig{
		Filename:   filename,
		MaxSize:    100 * 1024 * 1024, // 100MB
		MaxBackups: 10,
		MaxAge:     30,
	}
}

// NewRotateWriter creates a new rotating file writer.
func NewRotateWriter(cfg *RotateConfig) (*RotateWriter, error) {
	if cfg == nil {
		return nil, fmt.Errorf("rotate config cannot be nil")
	}

	if cfg.Filename == "" {
		return nil, fmt.Errorf("filename cannot be empty")
	}

	if cfg.MaxSize <= 0 {
		cfg.MaxSize = 100 * 1024 * 1024 // Default 100MB
	}

	if cfg.MaxBackups < 0 {
		cfg.MaxBackups = 0
	}

	if cfg.MaxAge < 0 {
		cfg.MaxAge = 0
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(cfg.Filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	w := &RotateWriter{
		filename:   cfg.Filename,
		maxSize:    cfg.MaxSize,
		maxBackups: cfg.MaxBackups,
		maxAge:     cfg.MaxAge,
		openTime:   time.Now(),
	}

	if err := w.openFile(); err != nil {
		return nil, err
	}

	return w, nil
}

// Write implements io.Writer interface.
func (w *RotateWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Check if rotation is needed
	if w.shouldRotate(len(p)) {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}

	// Write to file
	n, err = w.file.Write(p)
	w.currentSize += int64(n)

	return n, err
}

// Close closes the log file.
func (w *RotateWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		err := w.file.Close()
		w.file = nil // Set to nil to make Close idempotent
		return err
	}

	return nil
}

// openFile opens the log file for writing.
func (w *RotateWriter) openFile() error {
	// Get current file size if file exists
	info, err := os.Stat(w.filename)
	if err == nil {
		w.currentSize = info.Size()
	}

	// Open file for appending
	file, err := os.OpenFile(w.filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	w.file = file
	w.openTime = time.Now()

	return nil
}

// shouldRotate checks if rotation is needed.
func (w *RotateWriter) shouldRotate(writeSize int) bool {
	return w.currentSize+int64(writeSize) > w.maxSize
}

// rotate performs log file rotation.
func (w *RotateWriter) rotate() error {
	// Close current file
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return err
		}
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	ext := filepath.Ext(w.filename)
	nameWithoutExt := strings.TrimSuffix(w.filename, ext)
	backupName := fmt.Sprintf("%s-%s%s", nameWithoutExt, timestamp, ext)

	// Rename current file to backup
	if err := os.Rename(w.filename, backupName); err != nil {
		return fmt.Errorf("failed to rename log file: %w", err)
	}

	// Clean up old backup files
	w.cleanup()

	// Reset current size
	w.currentSize = 0

	// Open new file
	return w.openFile()
}

// cleanup removes old backup files based on maxBackups and maxAge.
func (w *RotateWriter) cleanup() {
	dir := filepath.Dir(w.filename)
	base := filepath.Base(w.filename)
	ext := filepath.Ext(base)
	prefix := strings.TrimSuffix(base, ext)

	// Find all backup files
	pattern := fmt.Sprintf("%s-*%s", prefix, ext)
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return
	}

	// Sort by modification time (oldest first)
	sort.Slice(matches, func(i, j int) bool {
		infoI, errI := os.Stat(matches[i])
		infoJ, errJ := os.Stat(matches[j])
		if errI != nil || errJ != nil {
			return false
		}
		return infoI.ModTime().Before(infoJ.ModTime())
	})

	// Remove files older than maxAge
	if w.maxAge > 0 {
		cutoff := time.Now().AddDate(0, 0, -w.maxAge)
		for _, file := range matches {
			info, err := os.Stat(file)
			if err != nil {
				continue
			}
			if info.ModTime().Before(cutoff) {
				os.Remove(file)
			}
		}
	}

	// Refresh matches after age-based cleanup
	matches, err = filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return
	}

	// Sort again
	sort.Slice(matches, func(i, j int) bool {
		infoI, errI := os.Stat(matches[i])
		infoJ, errJ := os.Stat(matches[j])
		if errI != nil || errJ != nil {
			return false
		}
		return infoI.ModTime().Before(infoJ.ModTime())
	})

	// Remove excess backup files beyond maxBackups
	if w.maxBackups > 0 && len(matches) > w.maxBackups {
		for i := 0; i < len(matches)-w.maxBackups; i++ {
			os.Remove(matches[i])
		}
	}
}

// MultiWriter creates a writer that duplicates writes to all provided writers.
type MultiWriter struct {
	writers []io.Writer
}

// NewMultiWriter creates a writer that writes to multiple destinations.
func NewMultiWriter(writers ...io.Writer) io.Writer {
	w := make([]io.Writer, len(writers))
	copy(w, writers)
	return &MultiWriter{writers: w}
}

// Write implements io.Writer interface.
func (m *MultiWriter) Write(p []byte) (n int, err error) {
	for _, w := range m.writers {
		n, err = w.Write(p)
		if err != nil {
			return
		}
		if n != len(p) {
			err = io.ErrShortWrite
			return
		}
	}
	return len(p), nil
}

// BufferedWriter wraps a writer with buffering for better performance.
type BufferedWriter struct {
	writer     io.Writer
	buffer     []byte
	size       int
	flushSize  int
	flushTimer *time.Timer
	mu         sync.Mutex
}

// BufferedConfig holds configuration for buffered writer.
type BufferedConfig struct {
	Writer      io.Writer
	BufferSize  int           // Buffer size in bytes
	FlushSize   int           // Auto-flush when buffer reaches this size
	FlushPeriod time.Duration // Auto-flush period
}

// NewBufferedWriter creates a new buffered writer.
func NewBufferedWriter(cfg *BufferedConfig) *BufferedWriter {
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 4096 // 4KB default
	}

	if cfg.FlushSize <= 0 {
		cfg.FlushSize = cfg.BufferSize
	}

	if cfg.FlushPeriod <= 0 {
		cfg.FlushPeriod = time.Second
	}

	bw := &BufferedWriter{
		writer:    cfg.Writer,
		buffer:    make([]byte, cfg.BufferSize),
		flushSize: cfg.FlushSize,
	}

	// Start periodic flush
	bw.flushTimer = time.AfterFunc(cfg.FlushPeriod, func() {
		bw.Flush()
		bw.flushTimer.Reset(cfg.FlushPeriod)
	})

	return bw
}

// Write implements io.Writer interface.
func (bw *BufferedWriter) Write(p []byte) (n int, err error) {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	n = len(p)

	// If data is larger than buffer, write directly
	if len(p) > len(bw.buffer) {
		if bw.size > 0 {
			if err := bw.flush(); err != nil {
				return 0, err
			}
		}
		return bw.writer.Write(p)
	}

	// If buffer would overflow, flush first
	if bw.size+len(p) > len(bw.buffer) {
		if err := bw.flush(); err != nil {
			return 0, err
		}
	}

	// Copy to buffer
	copy(bw.buffer[bw.size:], p)
	bw.size += len(p)

	// Auto-flush if reached flush size
	if bw.size >= bw.flushSize {
		if err := bw.flush(); err != nil {
			return 0, err
		}
	}

	return n, nil
}

// Flush writes buffered data to the underlying writer.
func (bw *BufferedWriter) Flush() error {
	bw.mu.Lock()
	defer bw.mu.Unlock()
	return bw.flush()
}

// flush is the internal flush without locking.
func (bw *BufferedWriter) flush() error {
	if bw.size == 0 {
		return nil
	}

	_, err := bw.writer.Write(bw.buffer[:bw.size])
	bw.size = 0

	return err
}

// Close flushes remaining data and stops the flush timer.
func (bw *BufferedWriter) Close() error {
	bw.mu.Lock()
	defer bw.mu.Unlock()

	if bw.flushTimer != nil {
		bw.flushTimer.Stop()
	}

	return bw.flush()
}
