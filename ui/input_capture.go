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
// a nested event loop. Inspector-owned input is private to developer tools.
// Events and pointer coordinates use window DIPs.
func (c *Context) CaptureInput(fn func(InputEvent, InputTarget) bool) {
	if raw := c.build(); raw != nil {
		rt := raw.rt
		if fn == nil {
			rt.inputCapture, rt.captureClicks, rt.releaseCapture = nil, nil, nil
			return
		}
		if rt.captureClicks == nil {
			rt.captureClicks = &inputCaptureState{rt: rt}
			rt.releaseCapture = rt.captureClicks.release
		}
		rt.captureClicks.fn = fn
		rt.inputCapture = rt.captureClicks.input
	}
}

// inputCaptureState belongs only to windows that register capture. Plain hits
// have no native press: track their gesture without making them controls.
type inputCaptureState struct {
	rt              *engine
	fn              func(InputEvent, InputTarget) bool
	id, born        uint64
	x, y            float32
	clicks          int
	plain, released bool
}

func (s *inputCaptureState) input(ev InputEvent, target InputTarget) bool {
	if ev.Kind == InputPointerMove && s.id != 0 {
		dx, dy := ev.X-s.x, ev.Y-s.y
		if dx*dx+dy*dy >= dragStart*dragStart {
			s.id = 0
		}
	}
	prevented := s.fn(ev, target)
	switch ev.Kind {
	case InputPointerDown:
		if ev.Button == 0 {
			s.id, s.plain, s.released = 0, false, false
			if hit := s.rt.states[target.id]; !prevented && hit != nil && hit.flags&flagDisabled == 0 {
				s.id, s.born, s.x, s.y = hit.id, hit.born, ev.X, ev.Y
			}
		}
	case InputPointerUp:
		if ev.Button == 0 {
			s.released = !prevented
			if prevented {
				s.id = 0
			}
		}
	case InputClick:
		s.id = 0
	}
	return prevented
}

func (s *inputCaptureState) release(button int) {
	if button != 0 {
		return
	}
	id := s.id
	s.id = 0
	rt := s.rt
	hit := rt.states[id]
	if !s.plain || !s.released || hit == nil || hit.born != s.born || hit.flags&flagDisabled != 0 {
		return
	}
	chain := rt.hitChain(rt.pointerX, rt.pointerY)
	if len(chain) != 0 && chain[0] == id {
		rt.captureInput(InputEvent{Kind: InputClick, Button: button, Clicks: max(1, s.clicks), Mods: rt.mods, X: rt.pointerX, Y: rt.pointerY}, id)
	}
}

func (rt *engine) captureInput(ev InputEvent, id uint64) bool {
	if rt.inputCapture == nil {
		return false
	}
	if rt.insp.ownsInput(rt, ev, id) {
		if rt.captureClicks != nil {
			rt.captureClicks.id = 0
		}
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
