package ui

// PointerEvents controls pointer hit testing for this element and, by
// inheritance, its descendants. The default is inherited (enabled at the
// root). An explicit true opts a descendant back in below a false ancestor,
// including native controls' internal children. It does not change keyboard
// focus, paint, or layout, and does not override Inert or Invisible.
// Unlike PassThrough, false also suppresses descendants without an override.
// Call only during the element's current build pass, on the UI thread.
func (e Element) PointerEvents(enabled bool) Element {
	if n := e.node(); n != nil {
		n.pointerEvents = -1
		if enabled {
			n.pointerEvents = 1
		}
	}
	return e
}
