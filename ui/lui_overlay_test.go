package ui

import "testing"

func TestLUIOverlayCollisionPadding(t *testing.T) {
	open := true
	tt := NewTester(func(c *Context) {
		var anchor Element
		Overlay(c, func() { anchor = Box(c).Absolute().Left(699).Top(499).Size(0, 0) })
		PopoverBase(c, anchor, &open, func(panel Element) {
			panel.Width(100).Height(50).Label("Popup")
			if styled, ok := any(panel).(interface{ CollisionPadding(float32) Element }); ok {
				styled.CollisionPadding(8)
			}
		})
	}, 700, 500)
	r, ok := tt.Find("Popup")
	if !ok {
		t.Fatal("missing popup")
	}
	if r.X < 8 || r.Y < 8 || r.X+r.W > 692 || r.Y+r.H > 492 {
		t.Fatalf("explicit collision padding ignored: %+v", r)
	}
}

func TestLUIForwardHandleAnchorPreservesGeometryAndAnchorPress(t *testing.T) {
	open := true
	var target Handle
	tt := NewTester(func(c *Context) {
		PopoverBase(c, Element{}, &open, func(panel Element) {
			panel.Width(100).Height(50).Label("Popup")
			if attached, ok := any(panel).(interface {
				AttachToHandle(*Handle, Anchor, Anchor) Element
			}); ok {
				attached.AttachToHandle(&target, AnchorBottomLeft, AnchorTopLeft)
			}
			panel.Children(func() { Button(c, "Popup action") })
		})
		Column(c).Padding(100, 80).Children(func() { Button(c, "Anchor").Width(120).Height(40).Bind(&target) })
	}, 700, 500)
	r, ok := tt.Find("Popup")
	if !ok {
		t.Fatal("missing popup")
	}
	if r.X != 80 || r.Y != 140 {
		t.Fatalf("future target did not position popup: %+v", r)
	}
	if err := tt.Click("Anchor"); err != nil {
		t.Fatal(err)
	}
	if !open {
		t.Fatal("pressing attached handle anchor dismissed popup")
	}
	tt.Key(0, KeyTab)
	if !tt.Focused("Popup action") {
		t.Fatal("popup did not follow actual anchor in focus order")
	}
}
