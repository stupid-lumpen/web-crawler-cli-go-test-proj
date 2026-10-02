package exporter_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"web-crawler-go-test-proj/internal/exporter"
	"web-crawler-go-test-proj/internal/models"
)

func TestNewExporter(t *testing.T) {
	t.Run("with custom logger", func(t *testing.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		exp := exporter.NewExporter(logger)

		if exp == nil {
			t.Fatal("expected non-nil Exporter")
		}
	})

	t.Run("with nil logger fallback to default", func(t *testing.T) {
		exp := exporter.NewExporter(nil)

		if exp == nil {
			t.Fatal("expected non-nil Exporter")
		}
	})
}

func TestExportJSON(t *testing.T) {
	var logBuf bytes.Buffer
	testLogger := slog.New(slog.NewTextHandler(&logBuf, nil))
	exp := exporter.NewExporter(testLogger)

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

		var result []*exporter.TreeExportNode
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("failed to unmarshal JSON output: %v", err)
		}

		if len(result) != 1 {
			t.Fatalf("expected 1 root node, got %d", len(result))
		}
		if result[0].Resource != "https://google.com" || result[0].Title != "Google" {
			t.Errorf("unexpected root node values: %+v", result[0])
		}
		if len(result[0].Links) != 1 {
			t.Fatalf("expected 1 child link, got %d", len(result[0].Links))
		}
		if result[0].Links[0].Resource != "https://google.com/about" {
			t.Errorf("unexpected child resource: %s", result[0].Links[0].Resource)
		}
	})

	t.Run("handles nil node in links list", func(t *testing.T) {
		tempDir := t.TempDir()
		outputFile := filepath.Join(tempDir, "nil_node.json")

		nodes := []*models.Node{
			{
				Resource: "https://example.com",
				Title:    "Root",
				Links:    []*models.Node{nil}, // Содержит nil узел
			},
			nil, // Корневой nil узел
		}

		err := exp.ExportJSON(outputFile, nodes)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		data, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatalf("failed to read output file: %v", err)
		}

		var result []*exporter.TreeExportNode
		_ = json.Unmarshal(data, &result)

		if len(result) != 1 {
			t.Fatalf("expected 1 root node (nil roots skipped), got %d", len(result))
		}
		if len(result[0].Links) != 0 {
			t.Errorf("expected nil child node to be ignored, got %d links", len(result[0].Links))
		}
	})

	t.Run("handles duplicate nodes across different branches (diamond graph)", func(t *testing.T) {
		tempDir := t.TempDir()
		outputFile := filepath.Join(tempDir, "diamond.json")

		sharedNode := &models.Node{
			Resource: "https://example.com/shared",
			Title:    "Shared Page",
			Links:    []*models.Node{},
		}

		nodeA := &models.Node{
			Resource: "https://example.com/a",
			Title:    "Page A",
			Links:    []*models.Node{sharedNode},
		}

		nodeB := &models.Node{
			Resource: "https://example.com/b",
			Title:    "Page B",
			Links:    []*models.Node{sharedNode},
		}

		root := &models.Node{
			Resource: "https://example.com/root",
			Title:    "Root Page",
			Links:    []*models.Node{nodeA, nodeB},
		}

		err := exp.ExportJSON(outputFile, []*models.Node{root})
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}

		data, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		var result []*exporter.TreeExportNode
		_ = json.Unmarshal(data, &result)

		// Проверяем, что sharedNode повторно вернулся из createdNodes без пересоздания
		if len(result[0].Links) != 2 {
			t.Fatalf("expected 2 child nodes under root, got %d", len(result[0].Links))
		}
	})

	t.Run("handles cyclical links without infinite recursion", func(t *testing.T) {
		tempDir := t.TempDir()
		outputFile := filepath.Join(tempDir, "cycle.json")

		nodeA := &models.Node{
			Resource: "https://example.com/a",
			Title:    "Page A",
		}
		nodeB := &models.Node{
			Resource: "https://example.com/b",
			Title:    "Page B",
		}

		nodeA.Links = []*models.Node{nodeB}
		nodeB.Links = []*models.Node{nodeA}

		err := exp.ExportJSON(outputFile, []*models.Node{nodeA})
		if err != nil {
			t.Fatalf("expected no error on cycle export, got: %v", err)
		}

		data, err := os.ReadFile(outputFile)
		if err != nil {
			t.Fatalf("failed to read output file: %v", err)
		}

		var result []*exporter.TreeExportNode
		_ = json.Unmarshal(data, &result)

		if len(result) != 1 {
			t.Fatalf("expected 1 root node, got %d", len(result))
		}

		childB := result[0].Links[0]
		childAInCycle := childB.Links[0]
		if len(childAInCycle.Links) != 0 {
			t.Errorf("expected empty links for cycle leaf, got %d", len(childAInCycle.Links))
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
