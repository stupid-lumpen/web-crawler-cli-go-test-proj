package exporter

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"web-crawler-go-test-proj/internal/models"
)

func TestNewExporter(t *testing.T) {
	t.Run("with custom logger", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		exp := NewExporter(logger)

		if exp == nil {
			t.Fatal("expected non-nil Exporter")
		}
		if exp.logger != logger {
			t.Errorf("expected logger to be assigned correctly")
		}
	})

	t.Run("with nil logger fallback to default", func(t *testing.T) {
		exp := NewExporter(nil)

		if exp == nil {
			t.Fatal("expected non-nil Exporter")
		}
		if exp.logger == nil {
			t.Errorf("expected default logger, got nil")
		}
	})
}

func TestExportJSON(t *testing.T) {
	var logBuf bytes.Buffer
	testLogger := slog.New(slog.NewTextHandler(&logBuf, nil))
	exp := NewExporter(testLogger)

	t.Run("successfully exports tree to JSON", func(t *testing.T) {
		tempDir := t.TempDir()
		outputFile := filepath.Join(tempDir, "result.json")

		nodes := []*models.Node{
			{
				Resource: "https://google.com",
				Title:    "Google",
				Links: []*models.Node{
					{
						Resource: "https://google.com/about",
						Title:    "About Google",
						Links:    []*models.Node{},
					},
				},
			},
		}

		err := exp.ExportJSON(outputFile, nodes)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		data, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatalf("failed to read output file: %v", err)
		}

		expectedJSON := `[
  {
    "resource": "https://google.com",
    "title": "Google",
    "links": [
      {
        "resource": "https://google.com/about",
        "title": "About Google",
        "links": []
      }
    ]
  }
]
`
		if string(data) != expectedJSON {
			t.Errorf("unexpected file content:\nGot:\n%s\nExpected:\n%s", string(data), expectedJSON)
		}
	})

	t.Run("handles nil nodes slice as empty JSON array", func(t *testing.T) {
		tempDir := t.TempDir()
		outputFile := filepath.Join(tempDir, "empty.json")

		err := exp.ExportJSON(outputFile, nil)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		data, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatalf("failed to read output file: %v", err)
		}

		if string(data) != "[]\n" {
			t.Errorf("expected '[]\\n', got %q", string(data))
		}
	})

	t.Run("automatically creates nested directories", func(t *testing.T) {
		tempDir := t.TempDir()
		outputFile := filepath.Join(tempDir, "nested", "dir", "output.json")

		err := exp.ExportJSON(outputFile, []*models.Node{})
		if err != nil {
			t.Fatalf("expected no error when creating directories, got: %v", err)
		}

		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			t.Errorf("expected output file to exist at nested path")
		}
	})

	t.Run("returns error on empty file path", func(t *testing.T) {
		err := exp.ExportJSON("", []*models.Node{})
		if err == nil {
			t.Fatal("expected error for empty filepath, got nil")
		}

		expectedErr := "output file path is empty"
		if err.Error() != expectedErr {
			t.Errorf("expected error %q, got %q", expectedErr, err.Error())
		}
	})

	t.Run("returns error on invalid directory path", func(t *testing.T) {
		tempDir := t.TempDir()
		blockingFilePath := filepath.Join(tempDir, "blocked")

		if err := os.WriteFile(blockingFilePath, []byte("content"), 0o644); err != nil {
			t.Fatalf("failed to create dummy file: %v", err)
		}

		invalidPath := filepath.Join(blockingFilePath, "result.json")

		err := exp.ExportJSON(invalidPath, []*models.Node{})
		if err == nil {
			t.Fatal("expected error when directory creation fails, got nil")
		}

		if !strings.Contains(err.Error(), "failed to create directory for output file") {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}
