package server

import (
	"sync"
	"testing"
	"time"
)

func TestNewCounter(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
	}{
		{"zero duration", 0},
		{"positive duration", time.Second},
		{"short duration", 100 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCounter(tt.duration)
			if c == nil {
				t.Fatal("newCounter returned nil")
			}
			if c.duration != tt.duration {
				t.Errorf("duration = %v, want %v", c.duration, tt.duration)
			}
			if c.connections != 0 {
				t.Errorf("connections = %d, want 0", c.connections)
			}
			if c.zeroTimer == nil {
				t.Error("zeroTimer is nil")
			}
		})
	}
}

func TestCounterAdd(t *testing.T) {
	c := newCounter(time.Second)

	// Add first connection
	n := c.add(1)
	if n != 1 {
		t.Errorf("add(1) returned %d, want 1", n)
	}
	if c.count() != 1 {
		t.Errorf("count() = %d, want 1", c.count())
	}

	// Add second connection
	n = c.add(1)
	if n != 2 {
		t.Errorf("add(1) returned %d, want 2", n)
	}
	if c.count() != 2 {
		t.Errorf("count() = %d, want 2", c.count())
	}

	// Add multiple at once
	n = c.add(3)
	if n != 5 {
		t.Errorf("add(3) returned %d, want 5", n)
	}
}

func TestCounterDone(t *testing.T) {
	c := newCounter(time.Second)

	// Add connections
	c.add(3)

	// Done one
	n := c.done()
	if n != 2 {
		t.Errorf("done() returned %d, want 2", n)
	}

	// Done another
	n = c.done()
	if n != 1 {
		t.Errorf("done() returned %d, want 1", n)
	}

	// Done last one
	n = c.done()
	if n != 0 {
		t.Errorf("done() returned %d, want 0", n)
	}
}

func TestCounterCount(t *testing.T) {
	c := newCounter(0)

	if c.count() != 0 {
		t.Errorf("initial count() = %d, want 0", c.count())
	}

	c.add(5)
	if c.count() != 5 {
		t.Errorf("count() after add(5) = %d, want 5", c.count())
	}

	c.done()
	c.done()
	if c.count() != 3 {
		t.Errorf("count() after 2 done() = %d, want 3", c.count())
	}
}

func TestCounterTimer(t *testing.T) {
	c := newCounter(time.Second)

	timer := c.timer()
	if timer == nil {
		t.Error("timer() returned nil")
	}
	if timer != c.zeroTimer {
		t.Error("timer() returned different timer than zeroTimer")
	}
}

func TestCounterWait(t *testing.T) {
	c := newCounter(0)

	c.add(2)

	done := make(chan bool)
	go func() {
		c.wait()
		done <- true
	}()

	// Wait should not complete yet
	select {
	case <-done:
		t.Error("wait() completed before all done()")
	case <-time.After(50 * time.Millisecond):
		// Expected
	}

	// Complete first
	c.done()

	// Still shouldn't complete
	select {
	case <-done:
		t.Error("wait() completed before all done()")
	case <-time.After(50 * time.Millisecond):
		// Expected
	}

	// Complete second
	c.done()

	// Now should complete
	select {
	case <-done:
		// Expected
	case <-time.After(time.Second):
		t.Error("wait() did not complete after all done()")
	}
}

func TestCounterConcurrency(t *testing.T) {
	c := newCounter(0)
	var wg sync.WaitGroup

	// Add many connections concurrently
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.add(1)
		}()
	}
	wg.Wait()

	if c.count() != 100 {
		t.Errorf("count() = %d, want 100 after concurrent adds", c.count())
	}

	// Done many connections concurrently
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.done()
		}()
	}
	wg.Wait()

	if c.count() != 0 {
		t.Errorf("count() = %d, want 0 after concurrent dones", c.count())
	}
}

func TestCounterTimerResetOnZero(t *testing.T) {
	duration := 100 * time.Millisecond
	c := newCounter(duration)

	// Add and immediately done - should reset timer
	c.add(1)
	c.done()

	// Timer should fire after duration
	select {
	case <-c.timer().C:
		// Expected - timer fired
	case <-time.After(2 * duration):
		t.Error("timer did not fire after duration")
	}
}

// Benchmark counter operations
func BenchmarkCounterAdd(b *testing.B) {
	c := newCounter(0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.add(1)
	}
}

func BenchmarkCounterAddDone(b *testing.B) {
	c := newCounter(0)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c.add(1)
		c.done()
	}
}
