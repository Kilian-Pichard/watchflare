package logger

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// GORMLogger routes GORM logs to slog:
//   - ErrRecordNotFound  → silent (normal application flow)
//   - other DB errors    → ERROR
//   - slow queries       → WARN  (threshold: 500ms)
//   - everything else    → silent
type GORMLogger struct {
	SlowThreshold time.Duration
}

// NewGORMLogger returns a GORM logger adapter.
func NewGORMLogger() *GORMLogger {
	return &GORMLogger{SlowThreshold: 500 * time.Millisecond}
}

func (l *GORMLogger) LogMode(_ gormlogger.LogLevel) gormlogger.Interface { return l }
func (l *GORMLogger) Info(_ context.Context, _ string, _ ...any)         {}
func (l *GORMLogger) Warn(_ context.Context, _ string, _ ...any)         {}
func (l *GORMLogger) Error(_ context.Context, _ string, _ ...any)        {}

func (l *GORMLogger) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return
		}
		sql, _ := fc()
		slog.Error("database error", "error", err, "elapsed", elapsed, "sql", sql)
		return
	}

	if elapsed > l.SlowThreshold {
		sql, rows := fc()
		slog.Warn("slow query", "elapsed", elapsed, "rows", rows, "sql", sql)
	}
}
