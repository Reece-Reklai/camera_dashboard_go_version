package config

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// =============================================================================
// Rotating File Writer
// =============================================================================

// RotatingFileWriter implements io.Writer with log rotation by file size.
// Matches Python's logging.handlers.RotatingFileHandler behaviour:
// when the current file exceeds MaxBytes, it is rotated to .1, .2, etc.
type RotatingFileWriter struct {
	mu          sync.Mutex
	path        string
	maxBytes    int
	backupCount int
	file        *os.File
	currentSize int64
}

// Log levels for coarse filtering when using Go's standard log package.
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarning
	LevelError
	LevelCritical
)

func parseLogLevel(level string) LogLevel {
	switch strings.ToUpper(strings.TrimSpace(level)) {
	case "DEBUG":
		return LevelDebug
	case "INFO", "":
		return LevelInfo
	case "WARNING", "WARN":
		return LevelWarning
	case "ERROR":
		return LevelError
	case "CRITICAL":
		return LevelCritical
	default:
		return LevelInfo
	}
}

func detectMessageLevel(msg string) LogLevel {
	upper := strings.ToUpper(msg)
	switch {
	case strings.Contains(upper, "CRITICAL"):
		return LevelCritical
	case strings.Contains(upper, "ERROR"):
		return LevelError
	case strings.Contains(upper, "WARNING"), strings.Contains(upper, "WARN"):
		return LevelWarning
	case strings.Contains(upper, "DEBUG"):
		return LevelDebug
	default:
		// Most current logs are untagged; treat them as INFO.
		return LevelInfo
	}
}

type levelFilterWriter struct {
	minLevel LogLevel
	next     io.Writer
}

func (w *levelFilterWriter) Write(p []byte) (int, error) {
	if detectMessageLevel(string(p)) < w.minLevel {
		return len(p), nil
	}
	return w.next.Write(p)
}

// NewRotatingFileWriter creates a new rotating file writer.
// maxBytes <= 0 disables rotation (single unbounded file).
func NewRotatingFileWriter(path string, maxBytes, backupCount int) (*RotatingFileWriter, error) {
	dir := filepath.Dir(path)
	if dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("config: create log dir: %w", err)
		}
	}

	rw := &RotatingFileWriter{
		path:        path,
		maxBytes:    maxBytes,
		backupCount: backupCount,
	}

	if err := rw.openFile(); err != nil {
		return nil, err
	}
	return rw, nil
}

func (rw *RotatingFileWriter) openFile() error {
	f, err := os.OpenFile(rw.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("config: open log file: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	rw.file = f
	rw.currentSize = info.Size()
	return nil
}

// Write implements io.Writer. It writes p to the current log file,
// rotating first if the write would exceed MaxBytes.
func (rw *RotatingFileWriter) Write(p []byte) (int, error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	if rw.maxBytes > 0 && rw.currentSize+int64(len(p)) > int64(rw.maxBytes) {
		rw.rotate()
	}

	n, err := rw.file.Write(p)
	rw.currentSize += int64(n)
	return n, err
}

// Close closes the underlying file.
func (rw *RotatingFileWriter) Close() error {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	if rw.file != nil {
		return rw.file.Close()
	}
	return nil
}

// rotate performs log rotation: file -> file.1, file.1 -> file.2, etc.
func (rw *RotatingFileWriter) rotate() {
	rw.file.Close()

	// Shift existing backups
	for i := rw.backupCount; i > 0; i-- {
		src := rw.path
		if i > 1 {
			src = fmt.Sprintf("%s.%d", rw.path, i-1)
		}
		dst := fmt.Sprintf("%s.%d", rw.path, i)
		os.Remove(dst)
		os.Rename(src, dst)
	}

	// Open fresh file
	if err := rw.openFile(); err != nil {
		// If we can't reopen the log file, write to stderr as a fallback.
		// This avoids silent data loss.
		fmt.Fprintf(os.Stderr, "config: failed to reopen log file after rotation: %v\n", err)
	}
}

// =============================================================================
// ConfigureLogging — matches Python's configure_logging()
// =============================================================================

// ConfigureLogging sets up Go's standard log package based on Config.
// It configures a rotating file handler and optional stdout handler,
// matching Python's configure_logging().
//
// Returns a cleanup function that should be called on shutdown.
func ConfigureLogging(cfg *Config) (cleanup func(), err error) {
	var writers []io.Writer
	var closers []io.Closer

	// Rotating file handler
	if cfg.LogFile != "" {
		rw, err := NewRotatingFileWriter(cfg.LogFile, cfg.LogMaxBytes, cfg.LogBackupCount)
		if err != nil {
			log.Printf("[Config] WARNING: Failed to configure file logging: %v", err)
		} else {
			writers = append(writers, rw)
			closers = append(closers, rw)
		}
	}

	// Stdout handler
	if cfg.LogToStdout {
		writers = append(writers, os.Stdout)
	}

	// Fallback: if no writers, use stdout
	if len(writers) == 0 {
		writers = append(writers, os.Stdout)
	}

	var w io.Writer
	if len(writers) == 1 {
		w = writers[0]
	} else {
		w = io.MultiWriter(writers...)
	}
	w = &levelFilterWriter{
		minLevel: parseLogLevel(cfg.LogLevel),
		next:     w,
	}

	// Configure standard logger
	log.SetOutput(w)
	log.SetFlags(log.Ldate | log.Ltime) // "2006/01/02 15:04:05" matches Python's "%(asctime)s"

	cleanup = func() {
		for _, c := range closers {
			c.Close()
		}
	}
	return cleanup, nil
}
