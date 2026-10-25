//go:build darwin

package darwin

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"runtime"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func TestLUINativeMenuImagePixelsAndRemoval(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	load()
	source := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := range 16 {
		for x := range 16 {
			source.SetRGBA(x, y, color.RGBA{R: 255, A: 255})
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, source); err != nil {
		t.Fatal(err)
	}
	withPool(func() {
		item := autorelease(send(send(class("NSMenuItem"), "alloc"), "initWithTitle:action:keyEquivalent:", uintptr(nsString("Inspect")), 0, uintptr(nsString(""))))
		applyItemState(item, &platform.MenuItem{Label: "Inspect", Visible: true, Enabled: true, ImagePNG: encoded.Bytes()})
		native := send(item, "image")
		if native == 0 {
			t.Fatal("NSMenuItem did not retain its native image")
		}
		result, err := png.Decode(bytes.NewReader(pngOfImage(native, 16)))
		if err != nil {
			t.Fatal(err)
		}
		r, g, b, a := result.At(8, 8).RGBA()
		if r < 60000 || g > 500 || b > 500 || a < 60000 {
			t.Fatalf("native menu image pixels: %d %d %d %d", r, g, b, a)
		}
		applyItemState(item, &platform.MenuItem{Label: "Inspect", Visible: true, Enabled: true})
		if send(item, "image") != 0 {
			t.Fatal("native image survived removal")
		}
	})
}
