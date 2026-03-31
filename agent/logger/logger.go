package logger

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

// handler is a custom slog.Handler that produces clean human-readable output:
//
//	2006/01/02 15:04:05 INFO   message  key=value
type handler struct {
	mu    *sync.Mutex // pointer so derived handlers (WithAttrs/WithGroup) share the same lock
	w     io.Writer
	level slog.Level
	attrs []slog.Attr
}

func newHandler(w io.Writer, level slog.Level) *handler {
	return &handler{mu: &sync.Mutex{}, w: w, level: level}
}

func (h *handler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *handler) Handle(_ context.Context, r slog.Record) error {
	var buf bytes.Buffer

	buf.WriteString(r.Time.Format("2006/01/02 15:04:05"))
	buf.WriteByte(' ')

	switch r.Level {
	case slog.LevelDebug:
		buf.WriteString("DEBUG  ")
	case slog.LevelInfo:
		buf.WriteString("INFO   ")
	case slog.LevelWarn:
		buf.WriteString("WARN   ")
	case slog.LevelError:
		buf.WriteString("ERROR  ")
	default:
		buf.WriteString(r.Level.String())
		buf.WriteString("  ")
	}

	buf.WriteString(r.Message)

	for _, a := range h.attrs {
		writeAttr(&buf, a)
	}
	r.Attrs(func(a slog.Attr) bool {
		writeAttr(&buf, a)
		return true
	})

	buf.WriteByte('\n')

	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.w.Write(buf.Bytes())
	return err
}

func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &handler{
		mu:    h.mu,
		w:     h.w,
		level: h.level,
		attrs: append(append([]slog.Attr{}, h.attrs...), attrs...),
	}
}

// WithGroup is intentionally a no-op: the agent logger does not support
// group prefixing. Groups are silently ignored to keep output flat.
func (h *handler) WithGroup(_ string) slog.Handler {
	return &handler{
		mu:    h.mu,
		w:     h.w,
		level: h.level,
		attrs: h.attrs,
	}
}

func writeAttr(buf *bytes.Buffer, a slog.Attr) {
	if a.Equal(slog.Attr{}) {
		return
	}
	buf.WriteString("  ")
	buf.WriteString(a.Key)
	buf.WriteByte('=')
	v := a.Value.String()
	needsQuote := false
	for _, c := range v {
		if c == ' ' || c == '\t' || c == '\n' || c == '"' {
			needsQuote = true
			break
		}
	}
	if needsQuote {
		buf.WriteByte('"')
		buf.WriteString(strings.ReplaceAll(v, `"`, `\"`))
		buf.WriteByte('"')
	} else {
		buf.WriteString(v)
	}
}

// logLevel returns INFO by default, DEBUG when WATCHFLARE_DEBUG is set.
func logLevel() slog.Level {
	if os.Getenv("WATCHFLARE_DEBUG") != "" {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}

// Init initialises the global slog logger with the clean text handler.
// Must be called once at startup before any log call.
// If the WATCHFLARE_DEBUG environment variable is set, DEBUG level is enabled.
func Init() {
	slog.SetDefault(slog.New(newHandler(os.Stdout, logLevel())))
}

// InitWithFile redirects the global slog logger to write to path instead of stdout.
// The file is opened in append mode and created if it does not exist.
// This is called after loading config when log_file is set.
// The file handle is intentionally kept open for the lifetime of the process.
func InitWithFile(path string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %w", path, err)
	}
	slog.SetDefault(slog.New(newHandler(f, logLevel())))
	return nil
}

// Fatal logs at ERROR level then exits with code 1.
func Fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}
