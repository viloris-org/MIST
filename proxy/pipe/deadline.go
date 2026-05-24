package pipe

import (
	"sync"
	"time"
)

// PipeDeadline is an abstraction for handling timeouts.
type PipeDeadline struct {
	mu      sync.Mutex // Guards timer and cancel
	timer   *time.Timer
	cancel  chan struct{} // Must be non-nil
	timedOut bool
}

func MakePipeDeadline() PipeDeadline {
	return PipeDeadline{cancel: make(chan struct{})}
}

// Set sets the point in time when the deadline will time out.
// A timeout event is signaled by closing the channel returned by waiter.
// Once a timeout has occurred, the deadline can be refreshed by specifying a
// t value in the future.
//
// A zero value for t prevents timeout.
func (d *PipeDeadline) Set(t time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.cancel == nil {
		d.cancel = make(chan struct{})
	}

	if t.IsZero() {
		if d.timer != nil {
			d.timer.Stop()
			d.timer = nil
		}
		if d.timedOut {
			d.cancel = make(chan struct{})
			d.timedOut = false
		}
		return
	}

	if dur := time.Until(t); dur > 0 {
		if d.timedOut {
			d.cancel = make(chan struct{})
			d.timedOut = false
		}
		if d.timer == nil {
			d.timer = time.AfterFunc(dur, func() {
				d.mu.Lock()
				defer d.mu.Unlock()
				if d.timedOut {
					return
				}
				d.timedOut = true
				close(d.cancel)
			})
			return
		}
		d.timer.Reset(dur)
		return
	}

	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
	if !d.timedOut {
		d.timedOut = true
		close(d.cancel)
	}
}

// Wait returns a channel that is closed when the deadline is exceeded.
func (d *PipeDeadline) Wait() chan struct{} {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.cancel
}

func isClosedChan(c <-chan struct{}) bool {
	select {
	case <-c:
		return true
	default:
		return false
	}
}
