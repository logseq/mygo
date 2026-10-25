package ui

import "github.com/egoist/mygo/internal/text"

// CommittedSpace reads a control's committed border-box coordinate system.
// Like CommittedText it expires at the next build or window close, and never
// retains a build Context or Element. All methods run on the UI thread.
type CommittedSpace struct {
	owner *elementOwner
	epoch uint64
	slot  uint32
	id    uint64
}

// Coordinates captures a control's committed coordinate system after layout.
func (r Handle) Coordinates(c *Context) (CommittedSpace, bool) {
	rt := c.runtime()
	b := (&r).binding(rt, false)
	if b == nil || b.closed || b.id == 0 || rt.committedEpoch != rt.epoch || b.element.owner != rt.arena || b.element.gen != rt.arena.generation {
		return CommittedSpace{}, false
	}
	g := CommittedSpace{rt.arena, rt.epoch, b.element.slot, b.id}
	_, ok := g.node()
	return g, ok
}

func (g CommittedSpace) node() (*node, bool) {
	if g.owner == nil || g.owner.rt == nil {
		return nil, false
	}
	rt := g.owner.rt
	if rt.closed || rt.epoch != g.epoch || rt.committedEpoch != g.epoch {
		return nil, false
	}
	n := rt.nodeAt(g.slot)
	if n == nil || n.id != g.id {
		return nil, false
	}
	for p := n; p != nil; p = p.parent {
		if p.flags&(flagInvisible|flagInert) != 0 || p.backface && p.transform.facesAway() {
			return nil, false
		}
	}
	return n, true
}

// Bounds returns the transformed border box in window DIPs.
func (g CommittedSpace) Bounds() (Rect, bool) {
	n, ok := g.node()
	if !ok {
		return Rect{}, false
	}
	return transformRect(n.world, Rect{n.x, n.y, n.w, n.h}), true
}

// LocalToWindow maps a border-box point to window DIPs.
func (g CommittedSpace) LocalToWindow(point Point) (Point, bool) {
	n, ok := g.node()
	if !ok {
		return Point{}, false
	}
	x, y := n.world.Point(n.x+point.X, n.y+point.Y)
	return Point{x, y}, true
}

// WindowToLocal inverts LocalToWindow; singular transforms report false.
func (g CommittedSpace) WindowToLocal(point Point) (Point, bool) {
	n, ok := g.node()
	if !ok {
		return Point{}, false
	}
	inv, ok := n.world.Inverse()
	if !ok {
		return Point{}, false
	}
	x, y := inv.Point(point.X, point.Y)
	return Point{x - n.x, y - n.y}, true
}

// CommittedText reads the exact shaped layout painted by a Text or RichText,
// including an inline child of a paragraph. Offsets are runes relative to the
// bound element's text, not the enclosing paragraph. Coordinates are window
// DIPs, including scrolling and transforms. Use only on the UI thread.
//
// A value expires when another build starts or its window closes. Every query
// reports false after expiry. It holds neither document state nor an Element.
type CommittedText struct {
	space CommittedSpace
}

// TextGeometry reads committed text after layout (for example in AfterFrame),
// or between frames. It reports false during an uncommitted build and for
// missing, hidden, non-text or closed controls. It never reshapes text.
func (r Handle) TextGeometry(c *Context) (CommittedText, bool) {
	space, ok := r.Coordinates(c)
	if !ok {
		return CommittedText{}, false
	}
	g := CommittedText{space}
	_, _, _, _, ok = g.layout()
	return g, ok
}

func (g CommittedText) layout() (n, para *node, start, end int, ok bool) {
	n, ok = g.space.node()
	if !ok || n.kind != kindText {
		return n, nil, 0, 0, false
	}
	para = n
	if n.isInline() {
		start, end = n.runes[0], n.runes[1]
		para = n.parent
		for para.isInline() {
			start, end = start+para.runes[0], end+para.runes[0]
			para = para.parent
		}
	} else if n.tl != nil {
		end = len(n.tl.Runes)
	}
	ok = para.tl != nil && start >= 0 && end >= start && end <= len(para.tl.Runes)
	return
}

// Text returns the displayed text, including nested inline children.
func (g CommittedText) Text() (string, bool) {
	_, p, a, b, ok := g.layout()
	if !ok {
		return "", false
	}
	return string(p.tl.Runes[a:b]), true
}

// Caret returns a whole-grapheme caret, with downstream affinity.
func (g CommittedText) Caret(index int) (Rect, bool) {
	n, p, a, b, ok := g.layout()
	if !ok || index < 0 || index > b-a {
		return Rect{}, false
	}
	x, y, h := p.tl.CaretAt(text.CaretPosition{Index: a + index, Affinity: text.Downstream})
	return transformRect(n.world, Rect{p.x + p.contentX() + x, p.y + p.contentY() + y, 1, h}), true
}

// SelectionRects preserves wrapped and bidi-discontiguous selections.
func (g CommittedText) SelectionRects(first, last int) ([]Rect, bool) {
	n, p, a, b, ok := g.layout()
	if !ok || first < 0 || last < first || last > b-a {
		return nil, false
	}
	var out []Rect
	for _, r := range p.tl.Selection(a+first, a+last) {
		out = append(out, transformRect(n.world, Rect{p.x + p.contentX() + r.X, p.y + p.contentY() + r.Y, r.W, r.H}))
	}
	return out, true
}

// IndexAt hit-tests a window point against the retained shaped glyphs.
func (g CommittedText) IndexAt(point Point) (int, bool) {
	n, p, a, b, ok := g.layout()
	if !ok {
		return 0, false
	}
	inv, ok := n.world.Inverse()
	if !ok {
		return 0, false
	}
	x, y := inv.Point(point.X, point.Y)
	i := p.tl.PositionAt(x-p.x-p.contentX(), y-p.y-p.contentY()).Index
	return max(a, min(b, i)) - a, true
}

// TextLine is a visual line's rune range and window-space box. Newline
// separators are excluded, as in the shaped layout's line table.
type TextLine struct {
	Start, End int
	Bounds     Rect
}

// Lines returns the visual lines intersecting this element's text range.
func (g CommittedText) Lines() ([]TextLine, bool) {
	n, p, a, b, ok := g.layout()
	if !ok {
		return nil, false
	}
	var out []TextLine
	for _, line := range p.tl.Lines {
		if line.End < a || line.Start > b {
			continue
		}
		if a != b && (line.End == a && line.Start != a || line.Start == b && line.End != b) {
			continue
		}
		lo, hi := max(a, line.Start), min(b, line.End)
		var box Rect
		found := false
		for _, r := range p.tl.Selection(lo, hi) {
			if r.Y >= line.Y+line.Height || r.Y+r.H <= line.Y {
				continue
			}
			f := Rect{p.x + p.contentX() + r.X, p.y + p.contentY() + r.Y, r.W, r.H}
			if !found {
				box = f
				found = true
			} else {
				box = union(box, f)
			}
		}
		if !found {
			x, y, h := p.tl.CaretAt(text.CaretPosition{Index: lo, Affinity: text.Downstream})
			box = Rect{p.x + p.contentX() + x, p.y + p.contentY() + y, 1, h}
		}
		out = append(out, TextLine{lo - a, hi - a, transformRect(n.world, box)})
	}
	return out, true
}
