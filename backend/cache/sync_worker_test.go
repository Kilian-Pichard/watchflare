package cache

import (
	"testing"
	"time"
)

func TestSyncWorker_StopExitsStart(t *testing.T) {
	// Use a very long interval so syncToDatabase never fires during the test
	w := NewSyncWorker(24 * time.Hour)

	done := make(chan struct{})
	go func() {
		w.Start()
		close(done)
	}()

	w.Stop()

	select {
	case <-done:
		// expected: Start returned after Stop
	case <-time.After(time.Second):
		t.Fatal("Start did not exit within 1s after Stop")
	}
}

func TestStaleChecker_StopExitsStart(t *testing.T) {
	c := NewStaleChecker(24*time.Hour, 15*time.Second)

	done := make(chan struct{})
	go func() {
		c.Start()
		close(done)
	}()

	c.Stop()

	select {
	case <-done:
		// expected
	case <-time.After(time.Second):
		t.Fatal("StaleChecker.Start did not exit within 1s after Stop")
	}
}

func TestSyncWorker_StopIsIdempotent(t *testing.T) {
	w := NewSyncWorker(24 * time.Hour)
	done := make(chan struct{})
	go func() {
		w.Start()
		close(done)
	}()
	w.Stop()
	w.Stop() // second Stop must not panic
	<-done
}

func TestStaleChecker_StopIsIdempotent(t *testing.T) {
	c := NewStaleChecker(24*time.Hour, 15*time.Second)
	done := make(chan struct{})
	go func() {
		c.Start()
		close(done)
	}()
	c.Stop()
	c.Stop() // second Stop must not panic
	<-done
}
