package ui

// TrackLayoutWidth records a non-scrolling box's horizontal layout extent for
// CommittedSpace.LayoutWidth. It does not change layout constraints or clipping.
// Call it on every build whose committed width will be queried.
func (e *node) TrackLayoutWidth() *node {
	e.trackLayoutWidth = true
	return e
}

// LayoutWidth returns a tracked box's client and content widths in local DIPs.
// Both include padding and exclude borders. Content is at least client and
// includes the box's own laid-out flow span and its final direct children's
// border boxes and margins toward inline-end, followed by that end's padding.
// Hidden and leaving children are excluded. Right-to-left layout measures
// toward the left. Transforms, painted ink and overflowing grandchildren are
// not included: this is not a recursive DOM scrollWidth query.
//
// Like Bounds, this reads only the final committed frame on the UI thread.
// Untracked, scrolling, non-box, hidden, expired and uncommitted elements report
// false, including measurement probes. No scrolling is enabled or changed.
func (g CommittedSpace) LayoutWidth() (clientWidth, contentWidth float32, ok bool) {
	n, ok := g.node()
	if !ok || !n.trackLayoutWidth || n.kind != kindBox || n.scrolls() {
		return 0, 0, false
	}
	borders := n.border[1] + n.border[3]
	clientWidth = max(n.w-borders, 0)
	return clientWidth, max(clientWidth, float32(n.contentW)-borders), true
}

func (e *node) keepLayoutWidth(flow, client float32) {
	width := max(flow, client) + e.padX()
	for c := e.first; c != nil; c = c.next {
		if c.hidden || c.leaving != 0 {
			continue
		}
		end := c.x + c.w + max(c.m(1), 0) + e.pad[1] + e.border[1]
		if e.rtl() {
			end = e.w - c.x + max(c.m(3), 0) + e.pad[3] + e.border[3]
		}
		width = max(width, end)
	}
	e.contentW = float64(width)
}
