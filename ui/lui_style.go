package ui

import (
	"github.com/egoist/mygo/internal/platform"
	"slices"
	"strings"
)

// ZIndex orders drawing and pointer hits among siblings without changing flow.
func (e Element) ZIndex(index int) Element {
	if n := e.node(); n != nil {
		n.zIndex, n.zIndexSet = index, true
	}
	return e
}
func (e *node) layeredChildren() []*node {
	active := false
	for child := e.first; child != nil; child = child.next {
		active = active || child.zIndexSet
	}
	if !active {
		return nil
	}
	children := make([]*node, 0, e.nchild)
	for child := e.first; child != nil; child = child.next {
		children = append(children, child)
	}
	slices.SortStableFunc(children, func(a, b *node) int {
		if a.zIndex < b.zIndex {
			return -1
		}
		if a.zIndex > b.zIndex {
			return 1
		}
		return int(b2i(a.flags&flagAbsolute != 0)) - int(b2i(b.flags&flagAbsolute != 0))
	})
	return children
}

// DisabledOpacity replaces the toolkit's default disabled alpha. Element
// Opacity still applies independently to its complete composited subtree.
func (e Element) DisabledOpacity(alpha float32) Element {
	if n := e.node(); n != nil {
		value := max(0, min(1, alpha))
		n.disabledOpacity = &value
	}
	return e
}

// MinWidthPercentOffset is a percentage minimum plus a logical-pixel offset.
func (e Element) MinWidthPercentOffset(percent, offset float32) Element {
	if n := e.node(); n != nil {
		n.minW = length{v: percent, u: unitPercent, offset: offset}
	}
	return e
}
func (e Element) MinHeightPercentOffset(percent, offset float32) Element {
	if n := e.node(); n != nil {
		n.minH = length{v: percent, u: unitPercent, offset: offset}
	}
	return e
}

// ScrollAxes makes an existing container scroll on the chosen axes.
func (e Element) ScrollAxes(horizontal, vertical bool) Element {
	if n := e.node(); n != nil {
		n.flags &^= flagScrollX | flagScrollY
		if horizontal || vertical {
			n.flags |= flagHover
		}
		if horizontal {
			n.flags |= flagScrollX
		}
		if vertical {
			n.flags |= flagScrollY
		}
	}
	return e
}

// WhiteSpace controls collapsing and wrapping for the inline text in e.
func (e Element) WhiteSpace(mode string) Element {
	var apply func(*node)
	apply = func(n *node) {
		if n.kind == kindText {
			switch mode {
			case "normal", "nowrap":
				n.text = strings.Join(strings.Fields(n.text), " ")
			case "pre-line":
				lines := strings.Split(n.text, "\n")
				for i, line := range lines {
					lines[i] = strings.Join(strings.Fields(line), " ")
				}
				n.text = strings.Join(lines, "\n")
			}
			if mode == "nowrap" {
				n.NoWrap()
			}
		}
		for child := n.first; child != nil; child = child.next {
			apply(child)
		}
	}
	if n := e.node(); n != nil {
		apply(n)
	}
	return e
}

// AccessibleNode is a copied view of a committed assistive-technology node.
type AccessibleNode struct {
	ID       uint64 `json:"id"`
	Role     Role   `json:"role"`
	Label    string `json:"label"`
	Value    string `json:"value"`
	Level    int    `json:"level"`
	Bounds   Rect   `json:"bounds"`
	Disabled bool   `json:"disabled"`
	Selected bool   `json:"selected"`
	Expanded bool   `json:"expanded"`
}

// Accessibility returns the actual committed accessibility tree for testing.
// Returned values own no frame handles and may be kept across later frames.
func (t *Tester) Accessibility() []AccessibleNode {
	tree := t.rt.accessTree()
	out := make([]AccessibleNode, 0, len(tree.Nodes))
	for _, n := range tree.Nodes {
		out = append(out, AccessibleNode{ID: n.ID, Role: Role(n.Role) + RoleGroup, Label: n.Label, Value: n.Value, Level: n.Level, Bounds: Rect{X: float32(n.Bounds.X), Y: float32(n.Bounds.Y), W: float32(n.Bounds.W), H: float32(n.Bounds.H)}, Disabled: n.States&platform.AccessDisabled != 0, Selected: n.States&platform.AccessSelected != 0, Expanded: n.States&platform.AccessExpanded != 0})
	}
	return out
}

// Inert keeps the subtree visible while removing it from interaction and access.
func (e Element) Inert(enabled bool) Element {
	if n := e.node(); n != nil {
		n.flags &^= flagInert
		if enabled {
			n.flags |= flagInert
		}
	}
	return e
}

// FontItalic sets an explicit inherited italic value, including normal text.
func (e Element) FontItalic(enabled bool) Element {
	if n := e.node(); n != nil {
		n.ts.italic = enabled
		n.ts.set |= setItalic
	}
	return e
}

// TextDecorations explicitly overrides inherited underlines and strike-through.
func (e Element) TextDecorations(underline, strike bool) Element {
	if n := e.node(); n != nil {
		n.ts.underline, n.ts.wavy, n.ts.strike = underline, false, strike
		n.ts.set |= setUnderline | setStrike
	}
	return e
}
