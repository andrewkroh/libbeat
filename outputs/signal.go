package outputs

import "sync/atomic"

// Signaler signals the completion of potentially asynchronous output operation.
// Completed is called by the output plugin when all events have been sent. On
// failure or if only a subset of the data is published then Failed will be
// invoked.
type Signaler interface {
	Completed()

	Failed()
}

// SplitSignal guards one output signaler from multiple calls
// by using a simple reference counting scheme. If one Signaler consumer
// reports a Failed event, the Failed event will be send to the guarded Signaler
// once the reference count becomes zero.
//
// Example use cases:
//   - Push signaler to multiple outputers
//   - split data to be send in smaller batches
type SplitSignal struct {
	count    int32
	failed   bool
	signaler Signaler
}

// CompositeSignal combines multiple signalers into one Signaler forwarding an event to
// to all signalers.
type CompositeSignal struct {
	signalers []Signaler
}

// NewSplitSignaler creates a new SplitSignal if s is not nil.
// If s is nil, nil will be returned. The count is the number of events to be
// received before publishing the final event to the guarded Signaler.
func NewSplitSignaler(
	s Signaler,
	count int,
) *SplitSignal {
	if s == nil {
		return nil
	}

	return &SplitSignal{
		count:    int32(count),
		signaler: s,
	}
}

// Completed signals a Completed event to s.
func (s *SplitSignal) Completed() {
	s.onEvent()
}

// Failed signals a Failed event to s.
func (s *SplitSignal) Failed() {
	s.failed = true
	s.onEvent()
}

func (s *SplitSignal) onEvent() {
	res := atomic.AddInt32(&s.count, -1)
	if res == 0 {
		if s.failed {
			s.signaler.Failed()
		} else {
			s.signaler.Completed()
		}
	}
}

// NewCompositeSignaler creates a new composite signaler.
func NewCompositeSignaler(signalers ...Signaler) *CompositeSignal {
	if len(signalers) == 0 {
		return nil
	}
	return &CompositeSignal{signalers}
}

// Completed sends the Completed signal to all signalers.
func (cs *CompositeSignal) Completed() {
	for _, s := range cs.signalers {
		if s != nil {
			cs.Completed()
		}
	}
}

// Failed sends the Failed signal to all signalers.
func (cs *CompositeSignal) Failed() {
	for _, s := range cs.signalers {
		if s != nil {
			cs.Failed()
		}
	}
}

// SignalCompleted sends the Completed event to s if s is not nil.
func SignalCompleted(s Signaler) {
	if s != nil {
		s.Completed()
	}
}

// SignalFailed sends the Failed event to s if s is not nil
func SignalFailed(s Signaler) {
	if s != nil {
		s.Failed()
	}
}

// Signal will send the Completed or Failed event to s depending
// on err being set if s is not nil.
func Signal(s Signaler, err error) {
	if s != nil {
		if err == nil {
			s.Completed()
		} else {
			s.Failed()
		}
	}
}

// SignalAll send the Completed or Failed event to all given signalers
// depending on err being set.
func SignalAll(signalers []Signaler, err error) {
	if signalers != nil {
		Signal(NewCompositeSignaler(signalers...), err)
	}
}
