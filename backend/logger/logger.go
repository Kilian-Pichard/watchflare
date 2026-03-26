package logger

import (
	"log/slog"
	"os"
)

// Init initialises the global slog logger with JSON output (SIEM-compatible).
// Must be called once at startup before any log call.
func Init() {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(h))
}

// Fatal logs at ERROR level then exits with code 1.
func Fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}
