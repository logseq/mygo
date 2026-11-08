package ui

type navigationKeys struct {
	keys uint8
	fn   func(InputEvent) bool
}

func navigationKeyMask(key Key) uint8 {
	switch key {
	case KeyLeft:
		return 1
	case KeyRight:
		return 2
	case KeyUp:
		return 4
	case KeyDown:
		return 8
	}
	return 0
}

// HandleNavigationKeys handles arrows left by the focused control in the
// element's subtree, before they scroll. keys may contain only KeyLeft,
// KeyRight, KeyUp and KeyDown; another key panics. With no keys or a nil fn,
// the element stops handling navigation. Omission in a build does the same.
//
// The nearest registered element owns navigation: an arrow it did not list,
// or that fn leaves, never goes to an outer navigation handler. Existing
// shortcuts and native control handling take priority. Text fields, editors
// (including read-only editors), compound fields and overlay boundaries
// between the focus and the owner prevent routing. Only no modifiers or Shift
// are admitted; typing, composition and other keys are unchanged.
//
// fn runs on the UI thread as the key arrives, between builds; true takes it,
// false leaves its default handling. A frame follows either result. Do not
// build UI, retain Elements or run a nested event loop in fn.
func (e *node) HandleNavigationKeys(keys []Key, fn func(InputEvent) bool) *node {
	var mask uint8
	for _, key := range keys {
		bit := navigationKeyMask(key)
		if bit == 0 {
			panic("ui: HandleNavigationKeys requires arrow keys")
		}
		mask |= bit
	}
	e.navigation = nil
	if mask != 0 && fn != nil {
		e.navigation = &navigationKeys{keys: mask, fn: fn}
		e.c.rt.routeNavigation = e.c.rt.navigationKey
	}
	return e
}

func (rt *engine) navigationKey(ev InputEvent, focused uint64) bool {
	bit := navigationKeyMask(ev.Key)
	if bit == 0 || ev.Mods&^Shift != 0 || focused == 0 ||
		rt.focused != focused || !rt.windowFocus() || rt.claimed(keyEvent{ev.Mods, ev.Key, ev.Repeat}) {
		return false
	}
	for s := rt.states[focused]; s != nil; s = rt.states[s.parent] {
		if s.navigationBarrier || s.flags&(flagDisabled|flagInert|flagEditable) != 0 ||
			s.ignoreInput || s.editor != nil || s.textClient != nil || s.takesText ||
			s.role == RoleTextField || s.role == RoleComboBox {
			return false
		}
		if scope := s.navigation; scope != nil {
			if scope.keys&bit == 0 {
				return false
			}
			ev.Time = rt.now()
			handled := scope.fn(ev)
			rt.requestFrame()
			return handled
		}
	}
	return false
}
