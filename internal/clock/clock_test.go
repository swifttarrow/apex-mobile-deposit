package clock

import (
	"testing"
	"time"
)

func TestTravelClock_SetRunning(t *testing.T) {
	c := NewTravelClock()
	target := time.Now().Add(2 * time.Hour).UTC()
	c.Set(target)

	got := c.Now().UTC()
	if d := got.Sub(target); d < -250*time.Millisecond || d > 250*time.Millisecond {
		t.Fatalf("expected now near %s, got %s (delta=%s)", target.Format(time.RFC3339Nano), got.Format(time.RFC3339Nano), d)
	}
}

func TestTravelClock_FreezeSetResume(t *testing.T) {
	c := NewTravelClock()
	c.Freeze()
	if !c.IsFrozen() {
		t.Fatal("expected frozen=true after Freeze")
	}

	initial := c.Now()
	time.Sleep(10 * time.Millisecond)
	if got := c.Now(); !got.Equal(initial) {
		t.Fatalf("expected frozen time to remain constant: %s vs %s", initial, got)
	}

	target := initial.Add(24 * time.Hour)
	c.Set(target)
	if got := c.Now(); !got.Equal(target) {
		t.Fatalf("expected Set while frozen to update frozen time: want %s got %s", target, got)
	}

	c.Resume()
	if c.IsFrozen() {
		t.Fatal("expected frozen=false after Resume")
	}

	afterResume := c.Now()
	if d := afterResume.Sub(target); d < -250*time.Millisecond || d > 250*time.Millisecond {
		t.Fatalf("expected resumed clock near %s, got %s (delta=%s)", target, afterResume, d)
	}
}
