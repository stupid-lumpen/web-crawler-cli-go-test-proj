package logger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitLogger(t *testing.T) {
	t.Run("stdout fallback when logPath is empty", func(t *testing.T) {
		log, cleanup, err := InitLogger("")
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if log == nil {
			t.Fatal("expected non-nil logger")
		}

		cleanup()
	})

	t.Run("creates log file and writes to it", func(t *testing.T) {
		tempDir := t.TempDir()
		logPath := filepath.Join(tempDir, "test.log")

		log, cleanup, err := InitLogger(logPath)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if log == nil {
			t.Fatal("expected non-nil logger")
		}

		log.Info("test log message", "key", "value")

		cleanup()

		data, err := os.ReadFile(logPath)
		if err != nil {
			t.Fatalf("failed to read created log file: %v", err)
		}

		if len(data) == 0 {
			t.Error("expected log file to contain written data, but it is empty")
		}
	})

	t.Run("automatically creates nested directories for log file", func(t *testing.T) {
		tempDir := t.TempDir()
		logPath := filepath.Join(tempDir, "nested", "dir", "test.log")

		_, cleanup, err := InitLogger(logPath)
		if err != nil {
			t.Fatalf("expected no error when creating nested dirs, got: %v", err)
		}
		defer cleanup()

		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			t.Errorf("expected log file to exist in nested directory")
		}
	})

	t.Run("fallback to stdout when file opening fails", func(t *testing.T) {
		tempDir := t.TempDir()
		blockingFilePath := filepath.Join(tempDir, "blocked")
		if err := os.WriteFile(blockingFilePath, []byte("content"), 0o644); err != nil {
			t.Fatalf("failed to create dummy file: %v", err)
		}

		invalidLogPath := filepath.Join(blockingFilePath, "app.log")

		log, cleanup, err := InitLogger(invalidLogPath)
		if err == nil {
			t.Fatal("expected error due to invalid file path, got nil")
		}
		if log == nil {
			t.Fatal("expected non-nil fallback logger for stdout, got nil")
		}

		log.Info("fallback test message")
		cleanup()
	})
}
