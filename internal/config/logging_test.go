package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewRotatingFileWriter_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	rw, err := NewRotatingFileWriter(path, 1024, 3)
	if err != nil {
		t.Fatalf("NewRotatingFileWriter() error: %v", err)
	}
	defer rw.Close()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("log file was not created")
	}
}

func TestNewRotatingFileWriter_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "deep", "test.log")

	rw, err := NewRotatingFileWriter(path, 1024, 3)
	if err != nil {
		t.Fatalf("NewRotatingFileWriter() error: %v", err)
	}
	defer rw.Close()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("log file was not created in nested directory")
	}
}

func TestRotatingFileWriter_Write(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	rw, err := NewRotatingFileWriter(path, 0, 0) // rotation disabled
	if err != nil {
		t.Fatalf("NewRotatingFileWriter() error: %v", err)
	}

	msg := "hello world\n"
	n, err := rw.Write([]byte(msg))
	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if n != len(msg) {
		t.Errorf("Write() returned %d, want %d", n, len(msg))
	}

	rw.Close()

	data, _ := os.ReadFile(path)
	if string(data) != msg {
		t.Errorf("file content = %q, want %q", string(data), msg)
	}
}

func TestRotatingFileWriter_RotatesAtMaxBytes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	maxBytes := 50
	rw, err := NewRotatingFileWriter(path, maxBytes, 2)
	if err != nil {
		t.Fatalf("NewRotatingFileWriter() error: %v", err)
	}

	// Write enough data to trigger rotation
	line := strings.Repeat("x", 30) + "\n" // 31 bytes
	rw.Write([]byte(line))                 // 31 bytes, under threshold
	rw.Write([]byte(line))                 // 62 bytes total -> triggers rotation on this write

	rw.Close()

	// After rotation, original file should be rotated to .1
	backup1 := path + ".1"
	if _, err := os.Stat(backup1); os.IsNotExist(err) {
		t.Error("backup .1 was not created after rotation")
	}

	// Current file should exist (and be small - the new write that triggered rotation)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("current log file missing after rotation")
	}
}

func TestRotatingFileWriter_BackupShifting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	maxBytes := 20
	rw, err := NewRotatingFileWriter(path, maxBytes, 2)
	if err != nil {
		t.Fatalf("NewRotatingFileWriter() error: %v", err)
	}

	line := strings.Repeat("a", 25) + "\n" // 26 bytes > maxBytes

	// Write 3 times to trigger multiple rotations
	rw.Write([]byte(line))
	rw.Write([]byte(line))
	rw.Write([]byte(line))
	rw.Close()

	// Should have .1 and .2 backups
	if _, err := os.Stat(path + ".1"); os.IsNotExist(err) {
		t.Error("backup .1 missing")
	}
	if _, err := os.Stat(path + ".2"); os.IsNotExist(err) {
		t.Error("backup .2 missing")
	}

	// backupCount=2, so .3 should NOT exist
	if _, err := os.Stat(path + ".3"); !os.IsNotExist(err) {
		t.Error("backup .3 should not exist (backupCount=2)")
	}
}

func TestRotatingFileWriter_Close(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")

	rw, err := NewRotatingFileWriter(path, 1024, 1)
	if err != nil {
		t.Fatalf("NewRotatingFileWriter() error: %v", err)
	}

	if err := rw.Close(); err != nil {
		t.Errorf("Close() error: %v", err)
	}

	// Double close should not panic or error
	if err := rw.Close(); err != nil {
		// file already closed - this is acceptable
	}
}

func TestConfigureLogging_WithFileAndStdout(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "test.log")

	cfg := DefaultConfig()
	cfg.LogFile = logPath
	cfg.LogToStdout = true

	cleanup, err := ConfigureLogging(cfg)
	if err != nil {
		t.Fatalf("ConfigureLogging() error: %v", err)
	}
	defer cleanup()

	// Verify log file was created
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		t.Error("log file was not created by ConfigureLogging")
	}
}

func TestConfigureLogging_StdoutOnly(t *testing.T) {
	cfg := DefaultConfig()
	cfg.LogFile = ""
	cfg.LogToStdout = true

	cleanup, err := ConfigureLogging(cfg)
	if err != nil {
		t.Fatalf("ConfigureLogging() error: %v", err)
	}
	defer cleanup()
}

func TestConfigureLogging_NoWriters_FallsBackToStdout(t *testing.T) {
	cfg := DefaultConfig()
	cfg.LogFile = ""
	cfg.LogToStdout = false

	cleanup, err := ConfigureLogging(cfg)
	if err != nil {
		t.Fatalf("ConfigureLogging() error: %v", err)
	}
	defer cleanup()
	// Should not panic - falls back to stdout
}
