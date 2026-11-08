package ui

import "testing"

func TestCommittedLayoutWidth(t *testing.T) {
	cases := []struct {
		name    string
		build   func(*Context, Element)
		content float32
	}{
		{"empty", func(c *Context, e Element) {}, 96},
		{"nowrap", func(c *Context, e Element) {
			e.Children(func() {
				for range 3 {
					Box(c).Size(60, 20).Shrink(0)
				}
			})
		}, 210},
		{"wrap", func(c *Context, e Element) {
			e.Wrap().Children(func() {
				for range 3 {
					Box(c).Size(60, 20).Shrink(0)
				}
			})
		}, 96},
		{"relative", func(c *Context, e Element) {
			e.Children(func() { Box(c).Size(60, 20).Shrink(0).Left(100) })
		}, 180},
		{"absolute", func(c *Context, e Element) {
			e.Children(func() { Box(c).Size(20, 20).Absolute().Left(160) })
		}, 190},
		{"negative end margin", func(c *Context, e Element) {
			e.Children(func() { Box(c).Size(200, 20).Shrink(0).Margin(0, -150, 0, 0) })
		}, 220},
		{"rtl", func(c *Context, e Element) {
			e.Direction(RightToLeft).Children(func() {
				for range 3 {
					Box(c).Size(60, 20).Shrink(0)
				}
			})
		}, 210},
		{"hidden child", func(c *Context, e Element) {
			e.Children(func() {
				Box(c).Size(60, 20).Shrink(0)
				Box(c).Size(500, 20).Shrink(0).Hidden(true)
			})
		}, 96},
		{"direct layout only", func(c *Context, e Element) {
			e.Children(func() {
				Box(c).Size(40, 20).Shrink(0).Children(func() {
					Box(c).Size(400, 20).Shrink(0).Translate(300, 0)
				})
			})
		}, 96},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var h Handle
			var space CommittedSpace
			tracked := false
			tt := NewTester(func(c *Context) {
				e := Row(c).Size(100, 120).Padding(0, 10).BorderWidth(0, 2).Gap(5).Clip().Bind(&h)
				if tracked {
					e.TrackLayoutWidth()
				}
				test.build(c, e)
				c.AfterFrame(func() {
					var ok bool
					space, ok = h.Coordinates(c)
					if !ok {
						t.Fatal("missing committed box")
					}
				})
			}, 300, 200)
			defer tt.rt.close()
			if _, _, ok := space.LayoutWidth(); ok {
				t.Fatal("untracked box published layout width")
			}
			tracked = true
			for range 2 {
				tt.Frame()
				client, content, ok := space.LayoutWidth()
				if !ok || client != 96 || content != test.content {
					t.Fatalf("layout width = %v, %v, %v; want 96, %v, true", client, content, ok, test.content)
				}
			}
			tracked = false
			tt.Frame()
			if _, _, ok := space.LayoutWidth(); ok {
				t.Fatal("untracked rebuild retained layout width")
			}
		})
	}
}
