package mygo

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func TestContentNativeMenuPreservesOwnedImagesThroughHostSnapshots(t *testing.T) {
	pixels := image.NewRGBA(image.Rect(0, 0, 2, 2))
	pixels.SetRGBA(1, 1, color.RGBA{R: 255, A: 255})
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, pixels); err != nil {
		t.Fatal(err)
	}
	icon := encoded.Bytes()
	makeItem := func(id int, label string) *platform.MenuItem {
		return &platform.MenuItem{ID: id, Label: label, Enabled: true, Visible: true, ImagePNG: append([]byte(nil), icon...)}
	}
	plain, checked, nested := makeItem(1, "Plain icon"), makeItem(2, "Checked icon"), makeItem(4, "Nested icon")
	checked.Type, checked.Checked = platform.MenuItemCheckbox, true
	submenu := makeItem(3, "Submenu icon")
	submenu.Type, submenu.Submenu = platform.MenuItemSubmenu, &platform.Menu{Items: []*platform.MenuItem{nested}}
	chosen := 0
	items := contentMenu(&platform.Menu{Items: []*platform.MenuItem{plain, checked, submenu}}, func(id int) { chosen = id })
	// The host bridge owns an opening snapshot after Content returns; callers
	// may publish a replacement or release their original PNG buffers.
	for _, item := range []*platform.MenuItem{plain, checked, submenu, nested} {
		clear(item.ImagePNG)
	}
	menu := NewMenu(items)
	snapshot := menu.snapshot()
	all := []*platform.MenuItem{snapshot.Items[0], snapshot.Items[1], snapshot.Items[2], snapshot.Items[2].Submenu.Items[0]}
	for _, item := range all {
		decoded, err := png.Decode(bytes.NewReader(item.ImagePNG))
		if err != nil {
			t.Errorf("native host lost %q decoded image: %v", item.Label, err)
			continue
		}
		if got := color.RGBAModel.Convert(decoded.At(1, 1)).(color.RGBA); got != (color.RGBA{R: 255, A: 255}) {
			t.Errorf("native host changed %q image pixels: %v", item.Label, got)
		}
	}
	if !snapshot.Items[1].Checked || snapshot.Items[1].Type != platform.MenuItemCheckbox {
		t.Fatal("image forwarding changed checked menu behavior")
	}
	items[2].Submenu[0].Click(nil, nil)
	if chosen != 4 {
		t.Fatalf("nested image item selected %d, want 4", chosen)
	}
	clear(snapshot.Items[0].ImagePNG)
	if again := menu.snapshot().Items[0].ImagePNG; !bytes.Equal(again, icon) {
		t.Fatal("native snapshot shares mutable image bytes with retained menu")
	}
}
