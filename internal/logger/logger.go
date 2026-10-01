package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

func InitLogger(logPath string) (*slog.Logger, func(), error) {
	var writer io.Writer = os.Stdout
	cleanup := func() {}

	if logPath != "" {
		if dir := filepath.Dir(logPath); dir != "." && dir != "" {
			_ = os.MkdirAll(dir, 0o755)
		}

		file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o666)
		if err != nil {
			handler := slog.NewTextHandler(writer, &slog.HandlerOptions{Level: slog.LevelInfo})
			return slog.New(handler), cleanup, fmt.Errorf("failed to open log file, falling back to stdout: %w", err)
		}

		writer = file
		cleanup = func() {
			_ = file.Close()
		}
	}

	handler := slog.NewTextHandler(writer, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	return slog.New(handler), cleanup, nil
}
