package exporter

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"web-crawler-go-test-proj/internal/models"
)

type Exporter struct {
	logger *slog.Logger
}

func NewExporter(logger *slog.Logger) *Exporter {
	if logger == nil {
		logger = slog.Default()
	}
	return &Exporter{
		logger: logger,
	}
}

func (e *Exporter) ExportJSON(filePath string, nodes []*models.Node) error {
	if filePath == "" {
		err := fmt.Errorf("output file path is empty")
		return err
	}

	if nodes == nil {
		nodes = []*models.Node{}
	}

	if dir := filepath.Dir(filePath); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory for output file: %w", err)
		}
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(nodes); err != nil {
		return fmt.Errorf("failed to encode nodes to JSON: %w", err)
	}

	return nil
}
