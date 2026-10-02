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

type TreeExportNode struct {
	Resource string            `json:"resource"`
	Title    string            `json:"title"`
	Links    []*TreeExportNode `json:"links"`
}

func (e *Exporter) ExportJSON(filePath string, nodes []*models.Node) error {
	if filePath == "" {
		return fmt.Errorf("output file path is empty")
	}

	if nodes == nil {
		nodes = []*models.Node{}
	}

	createdNodes := make(map[string]*TreeExportNode)

	var buildTree func(n *models.Node, visitedPath map[string]bool) *TreeExportNode
	buildTree = func(n *models.Node, visitedPath map[string]bool) *TreeExportNode {
		if n == nil {
			return nil
		}

		if visitedPath[n.Resource] {
			if existing, ok := createdNodes[n.Resource]; ok {
				return &TreeExportNode{
					Resource: existing.Resource,
					Title:    existing.Title,
					Links:    make([]*TreeExportNode, 0), // Пустой массив для листовых/зацикленных узлов
				}
			}
			return nil
		}

		if existing, ok := createdNodes[n.Resource]; ok {
			return existing
		}

		expNode := &TreeExportNode{
			Resource: n.Resource,
			Title:    n.Title,
			Links:    make([]*TreeExportNode, 0),
		}
		createdNodes[n.Resource] = expNode

		nextVisited := make(map[string]bool, len(visitedPath)+1)
		for k, v := range visitedPath {
			nextVisited[k] = v
		}
		nextVisited[n.Resource] = true

		for _, link := range n.Links {
			if childExp := buildTree(link, nextVisited); childExp != nil {
				expNode.Links = append(expNode.Links, childExp)
			}
		}

		return expNode
	}

	exportRoots := make([]*TreeExportNode, 0)
	for _, root := range nodes {
		if rootExp := buildTree(root, make(map[string]bool)); rootExp != nil {
			exportRoots = append(exportRoots, rootExp)
		}
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

	if err := encoder.Encode(exportRoots); err != nil {
		return fmt.Errorf("failed to encode nodes to JSON: %w", err)
	}

	return nil
}
