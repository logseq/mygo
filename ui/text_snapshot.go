package ui

// TextSnapshot reads the live, already-consumed edit state of a committed
// TextInput or TextArea. Offsets count runes and preserve selection direction.
// The returned string is immutable; passwords are not redacted for this trusted
// host query. Marked/preedit text is excluded, as from the bound string value.
//
// Call on the UI thread, including between builds in CaptureInput, or after
// input in OnChange. The window-stable Context may be retained for this query;
// no Element may be retained. This reads the native editor, not the bound value
// or previous layout. It does not dispatch queued input or change selection.
// Missing, unbound, remounted-before-commit, non-input and closed controls return
// false. Bind on every build before querying from callbacks.
func (r Handle) TextSnapshot(c *Context) (text string, anchor, caret int, ok bool) {
	rt := c.runtime()
	b := (&r).binding(rt, false)
	if b == nil || b.closed || b.id == 0 || b.element.owner != rt.arena || b.element.gen != rt.arena.generation {
		return
	}
	n := rt.nodeAt(b.element.slot)
	s := rt.states[b.id]
	if n == nil || n.id != b.id || n.kind != kindInput || s == nil || n.st != s || s.seen != rt.frame || s.pass != rt.pass || s.flags&flagEditable == 0 || s.editor == nil || s.editor.document != nil {
		return
	}
	return s.editor.String(), s.editor.anchor, s.editor.caret, true
}
