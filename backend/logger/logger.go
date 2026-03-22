package logger

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
)

// handler is a custom slog.Handler that produces clean human-readable output:
//
//	2006/01/02 15:04:05 INFO   message  key=value
type handler struct {
	mu    sync.Mutex
	w     io.Writer
	level slog.Level
	attrs []slog.Attr
}

func newHandler(w io.Writer, level slog.Level) *handler {
	return &handler{w: w, level: level}
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
		w:     h.w,
		level: h.level,
		attrs: append(append([]slog.Attr{}, h.attrs...), attrs...),
	}
}

func (h *handler) WithGroup(_ string) slog.Handler {
	return &handler{
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
		buf.WriteString(v)
		buf.WriteByte('"')
	} else {
		buf.WriteString(v)
	}
}

// Init initialises the global slog logger with the clean text handler.
// Must be called once at startup before any log call.
func Init() {
	h := newHandler(os.Stdout, slog.LevelInfo)
	slog.SetDefault(slog.New(h))
}

// Fatal logs at ERROR level then exits with code 1.
func Fatal(msg string, args ...any) {
	slog.Error(msg, args...)
	os.Exit(1)
}
