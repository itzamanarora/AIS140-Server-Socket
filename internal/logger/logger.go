package logger

import (
	"log/slog"
	"os"
)

// New creates a structured slog logger writing to stdout.
func New() *slog.Logger {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	return slog.New(handler)
}
