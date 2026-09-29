package log

import "sync"

const (
	// maxPathCap limits retained attribute-path bytes, not path length.
	maxPathCap = 4 * 1024

	// maxGroupCap limits retained group slots, not nesting depth.
	maxGroupCap = 64

	// maxLineCap limits retained line-buffer bytes, not log length.
	maxLineCap = 64 * 1024
)

// pool reuses buffers for both record output and preformatted handler attributes.
var pool = &sync.Pool{
	New: func() any {
		return &state{
			attr: attrState{
				path:   make([]byte, 0, 256),
				groups: make([]string, 0, 8),
			},
			line: lineState{
				buf: make([]byte, 0, 1024),
			},
		}
	},
}

// state holds attribute and line buffers with a shared lifetime.
type state struct {
	attr attrState
	line lineState
}

// acquireState returns cleared state for exclusive use until releaseState.
func acquireState() *state {
	return pool.Get().(*state)
}

// releaseState resets state, drops oversized buffers, and returns it to the pool.
// The caller must not use the state or its buffers after release.
func releaseState(o *state) {
	if cap(o.attr.path) > maxPathCap {
		o.attr.path = nil
	}
	if cap(o.attr.groups) > maxGroupCap {
		o.attr.groups = nil
	}
	if cap(o.line.buf) > maxLineCap {
		o.line.buf = nil
	}
	o.attr.reset()
	o.line.reset()
	pool.Put(o)
}
