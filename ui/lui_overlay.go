package ui

import "math"

// CollisionPadding keeps an attached overlay this far inside its window.
// The toolkit's default is four DIPs; a zero padding permits touching the edge.
func (e Element) CollisionPadding(padding float32) Element {
	if n := e.node(); n != nil && padding >= 0 && !math.IsNaN(float64(padding)) && !math.IsInf(float64(padding), 0) {
		value := padding
		n.collisionPadding = &value
	}
	return e
}

// AttachToHandle attaches an overlay to a stable handle, including a target
// built later in this pass. Geometry, focus order and outside presses use the
// actual target. An absent target hides the overlay for that pass.
func (e Element) AttachToHandle(target *Handle, at, self Anchor) Element {
	n := e.node()
	if n == nil || target == nil {
		return e
	}
	target.init()
	n.flags |= flagAbsolute
	n.attach = attachment(1 + min(at, AnchorBottomEnd)*15 + min(self, AnchorBottomEnd))
	n.popoverHandle = target
	n.popover = n.c.rt.public.Resolve(*target).node()
	return e
}

// MatchAnchorWidth uses the committed attachment target's width when this
// overlay has no explicit width. It also resolves targets built later.
func (e Element) MatchAnchorWidth() Element {
	if n := e.node(); n != nil {
		n.matchAnchorWidth = true
	}
	return e
}
