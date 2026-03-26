package services

import (
	"testing"
	"time"
)

func TestNewAggregatedMetricsScheduler(t *testing.T) {
	s := NewAggregatedMetricsScheduler(30 * time.Second)

	if s.interval != 30*time.Second {
		t.Errorf("interval: got %v, want 30s", s.interval)
	}
	if s.ctx == nil {
		t.Error("ctx must not be nil")
	}
	if s.cancel == nil {
		t.Error("cancel must not be nil")
	}
	s.Stop()
}

func TestAggregatedMetricsScheduler_StopCancelsContext(t *testing.T) {
	s := NewAggregatedMetricsScheduler(30 * time.Second)

	s.Stop()

	select {
	case <-s.ctx.Done():
		// expected
	default:
		t.Error("context must be done after Stop()")
	}
}

func TestAggregatedMetricsScheduler_StartStops(t *testing.T) {
	s := NewAggregatedMetricsScheduler(time.Minute)

	done := make(chan struct{})
	go func() {
		s.Start()
		close(done)
	}()

	s.Stop()

	select {
	case <-done:
		// expected: Start() returned after Stop()
	case <-time.After(time.Second):
		t.Error("Start() did not return within 1s after Stop()")
	}
}
