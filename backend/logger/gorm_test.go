package logger

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/stretchr/testify/assert"
)

// captureHandler records slog entries for test assertions.
type captureHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *captureHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }
func (h *captureHandler) WithAttrs(_ []slog.Attr) slog.Handler          { return h }
func (h *captureHandler) WithGroup(_ string) slog.Handler               { return h }
func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r)
	return nil
}

func (h *captureHandler) last() (slog.Record, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.records) == 0 {
		return slog.Record{}, false
	}
	return h.records[len(h.records)-1], true
}

func setupCapture(t *testing.T) *captureHandler {
	t.Helper()
	h := &captureHandler{}
	slog.SetDefault(slog.New(h))
	t.Cleanup(func() { Init() }) // restore JSON logger after test
	return h
}

func TestGORMLogger_Trace(t *testing.T) {
	logger := NewGORMLogger()
	ctx := context.Background()
	fc := func() (string, int64) { return "SELECT 1", 1 }

	t.Run("ErrRecordNotFound is silent", func(t *testing.T) {
		h := setupCapture(t)
		logger.Trace(ctx, time.Now(), fc, gorm.ErrRecordNotFound)
		_, got := h.last()
		assert.False(t, got, "expected no log entry for ErrRecordNotFound")
	})

	t.Run("other error logs at ERROR level", func(t *testing.T) {
		h := setupCapture(t)
		logger.Trace(ctx, time.Now(), fc, errors.New("connection refused"))
		r, got := h.last()
		assert.True(t, got)
		assert.Equal(t, slog.LevelError, r.Level)
		assert.Equal(t, "database error", r.Message)
	})

	t.Run("slow query logs at WARN level", func(t *testing.T) {
		h := setupCapture(t)
		begin := time.Now().Add(-600 * time.Millisecond) // 600ms ago > 500ms threshold
		logger.Trace(ctx, begin, fc, nil)
		r, got := h.last()
		assert.True(t, got)
		assert.Equal(t, slog.LevelWarn, r.Level)
		assert.Equal(t, "slow query", r.Message)
	})

	t.Run("fast query with no error is silent", func(t *testing.T) {
		h := setupCapture(t)
		begin := time.Now().Add(-10 * time.Millisecond) // 10ms < 500ms threshold
		logger.Trace(ctx, begin, fc, nil)
		_, got := h.last()
		assert.False(t, got, "expected no log entry for fast query")
	})
}
