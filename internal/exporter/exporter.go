package exporter

import (
	"log/slog"
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
