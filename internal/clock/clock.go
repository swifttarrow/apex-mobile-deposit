package clock

import (
	"sync"
	"time"
)

// TravelClock supports setting, freezing, and resuming an app-level clock.
// When running, time advances with a real-time offset from system time.
// When frozen, Now() always returns the frozen timestamp.
type TravelClock struct {
	mu       sync.RWMutex
	offset   time.Duration
	frozen   bool
	frozenAt time.Time
}

// NewTravelClock creates a running clock aligned with system time.
func NewTravelClock() *TravelClock {
	return &TravelClock{}
}

// Now returns the current app time.
func (c *TravelClock) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.frozen {
		return c.frozenAt
	}
	return time.Now().Add(c.offset)
}

// Freeze stops time progression at the current app time.
func (c *TravelClock) Freeze() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.frozen {
		return
	}
	c.frozenAt = time.Now().Add(c.offset)
	c.frozen = true
}

// Resume restarts time progression from the frozen timestamp.
func (c *TravelClock) Resume() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.frozen {
		return
	}
	c.offset = c.frozenAt.Sub(time.Now())
	c.frozen = false
	c.frozenAt = time.Time{}
}

// Set sets the app time to an arbitrary value.
// If frozen, it updates the frozen timestamp.
// If running, it updates the running offset to align Now() with t.
func (c *TravelClock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.frozen {
		c.frozenAt = t
		return
	}
	c.offset = t.Sub(time.Now())
}

// IsFrozen reports whether the clock is currently frozen.
func (c *TravelClock) IsFrozen() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.frozen
}
