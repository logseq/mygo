package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"reflect"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func TestLUINativeMenuReportsSelectionThenDismissal(t *testing.T) {
	var events []string
	tt := NewTester(func(c *Context) {
		e := Button(c, "Document").ContextMenu(func(m *Menu) {
			if m.Item("Inspect").Chosen() {
				events = append(events, "inspect")
			}
		})
		e.OnMenuDismiss(func() { events = append(events, "dismiss") })
	}, 400, 200)
	if err := tt.RightClick("Document"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("Inspect"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(events, []string{"inspect", "dismiss"}) {
		t.Fatalf("native menu close order: %v", events)
	}
	tt.CloseMenu()
	tt.Frame()
	if len(events) != 2 {
		t.Fatalf("menu dismissed twice: %v", events)
	}
}

func TestLUINativeMenuCancellationAndRetiredHost(t *testing.T) {
	visible := true
	dismissals := 0
	tt := NewTester(func(c *Context) {
		if !visible {
			return
		}
		e := Button(c, "Document").ContextMenu(func(m *Menu) { m.Item("Inspect") })
		e.OnMenuDismiss(func() { dismissals++ })
	}, 400, 200)
	if err := tt.RightClick("Document"); err != nil {
		t.Fatal(err)
	}
	tt.CloseMenu()
	if dismissals != 1 {
		t.Fatalf("cancel must notify once: %d", dismissals)
	}
	if err := tt.RightClick("Document"); err != nil {
		t.Fatal(err)
	}
	visible = false
	tt.Frame()
	tt.CloseMenu()
	tt.Frame()
	if dismissals != 1 {
		t.Fatal("retired host delivered stale menu callback")
	}
}

func TestLUITabPanelRetainsAccessibleChildren(t *testing.T) {
	// The appended role preserves the numerical ABI of existing toolkit roles.
	panelRole := RoleTabPanel
	tt := NewTester(func(c *Context) {
		Column(c).Role(panelRole).Label("Home page").Children(func() { Text(c, "Page content") })
	}, 400, 200)
	tt.send(platform.SurfaceEvent{Kind: platform.AccessibilityOn})
	if tt.h.access == nil {
		t.Fatal("missing accessible tree")
	}
	found := false
	for _, n := range tt.h.access.Nodes {
		if n.Label == "Page content" {
			found = true
		}
	}
	if !found {
		t.Fatalf("tab panel hid accessible child: %+v", tt.h.access.Nodes)
	}
}

func TestLUINativeMenuImageIsDeliveredAndRemoved(t *testing.T) {
	picture := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := range 16 {
		for x := range 16 {
			picture.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	var data bytes.Buffer
	png.Encode(&data, picture)
	showImage := true
	tt := NewTester(func(c *Context) {
		Button(c, "Open image menu").ContextMenu(func(menu *Menu) {
			item := menu.Item("Inspect")
			if showImage {
				if api, ok := any(item).(interface{ ImagePNG([]byte) *MenuItem }); ok {
					api.ImagePNG(data.Bytes())
				}
			}
		})
	}, 300, 200)
	tt.RightClick("Open image menu")
	api, ok := any(tt).(interface{ MenuImage(string) image.Image })
	if !ok || api.MenuImage("Inspect") == nil {
		t.Fatal("native popup service did not receive an image")
	}
	r, g, b, a := api.MenuImage("Inspect").At(8, 8).RGBA()
	if r < 60000 || g != 0 || b != 0 || a < 60000 {
		t.Fatal("native menu pixels changed")
	}
	tt.CloseMenu()
	showImage = false
	tt.Frame()
	tt.RightClick("Open image menu")
	if api.MenuImage("Inspect") != nil {
		t.Fatal("image leaked into subsequent native menu opening")
	}
}
