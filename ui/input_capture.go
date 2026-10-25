package ui

import "weak"

// InputTarget identifies the committed focused control (keyboard, text and
// commands) or deepest hit element (pointer and scroll). It is valid only
// during the CaptureInput callback; it does not retain a frame's elements.
type InputTarget struct {
	owner  weak.Pointer[engine]
	id     uint64
	serial uint64
}

// Distance returns the number of committed ancestors from the target to h,
// zero for the target itself, or -1 for an unrelated or expired handle.
func (t InputTarget) Distance(h Handle) int {
	rt := t.owner.Value()
	if rt == nil || t.serial == 0 || rt.capturing != t.serial {
		return -1
	}
	b := h.binding(rt, false)
	if b == nil || b.closed || b.id == 0 {
		return -1
	}
	for id, distance := t.id, 0; id != 0; distance++ {
		s := rt.states[id]
		if s == nil {
			return -1
		}
		if id == b.id {
			return distance
		}
		id = s.parent
	}
	return -1
}

// CaptureInput installs this build's window input capture callback. Call once
// per view build, on the UI thread. It runs before focused controls, shortcuts,
// menus and pointer handlers consume input; true prevents native handling.
// Return false to observe without interfering with native editing. Rebuilds
// that omit the registration remove it. The callback must not build UI or run
// a nested event loop. Events and pointer coordinates use window DIPs.
func (c *Context) CaptureInput(fn func(InputEvent, InputTarget) bool) {
	if raw := c.build(); raw != nil {
		raw.rt.inputCapture = fn
	}
}

func (rt *engine) captureInput(ev InputEvent, id uint64) bool {
	if rt.inputCapture == nil {
		return false
	}
	rt.captureSerial++
	rt.capturing = rt.captureSerial
	defer func() { rt.capturing = 0 }()
	ev.Time = rt.frameTime()
	return rt.inputCapture(ev, InputTarget{owner: rt.owner, id: id, serial: rt.capturing})
}

func (rt *engine) capturePointer(ev InputEvent, x, y float32) bool {
	if rt.inputCapture == nil {
		return false
	}
	var id uint64
	if chain := rt.hitChain(x, y); len(chain) != 0 {
		id = chain[0]
	}
	ev.X, ev.Y = x, y
	return rt.captureInput(ev, id)
}
