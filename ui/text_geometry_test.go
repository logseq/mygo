package ui

import "testing"

func TestCommittedInlineTextGeometryLifetime(t *testing.T) {
	var h Handle
	var ctx *Context
	var g CommittedText
	shown := true
	tt := NewTester(func(c *Context) {
		ctx = c
		if _, ok := h.TextGeometry(c); ok {
			t.Fatal("accepted uncommitted geometry")
		}
		if shown {
			Text(c, "prefix ").Width(120).FontSize(20).Padding(7).Children(func() {
				Text(c, "😀 alpha beta gamma").Bind(&h)
			})
		}
		c.AfterFrame(func() {
			if shown {
				var ok bool
				g, ok = h.TextGeometry(c)
				if !ok {
					t.Fatal("missing committed inline text")
				}
			}
		})
	}, 180, 200)
	defer tt.rt.close()
	tt.Frame()
	if text, ok := g.Text(); !ok || text != "😀 alpha beta gamma" {
		t.Fatalf("inline text: %q %v", text, ok)
	}
	r, ok := g.Caret(0)
	if !ok || r.X <= 7 || r.H <= 0 {
		t.Fatalf("lost paragraph prefix/padding: %+v", r)
	}
	if i, ok := g.IndexAt(Point{r.X, r.Y + r.H/2}); !ok || i != 0 {
		t.Fatalf("caret hit: %d %v", i, ok)
	}
	if rects, ok := g.SelectionRects(0, 18); !ok || len(rects) < 2 {
		t.Fatalf("wrapped inline selection: %+v %v", rects, ok)
	}
	if lines, ok := g.Lines(); !ok || len(lines) < 2 || lines[0].Start != 0 || lines[len(lines)-1].End != 18 {
		t.Fatalf("visual lines: %+v %v", lines, ok)
	}
	old := g
	tt.Frame()
	if _, ok := old.Text(); ok {
		t.Fatal("old frame remained valid")
	}
	shown = false
	tt.Frame()
	if _, ok := h.TextGeometry(ctx); ok {
		t.Fatal("removed node remained valid")
	}
	if _, ok := g.Caret(0); ok {
		t.Fatal("removed frame geometry remained valid")
	}
}
