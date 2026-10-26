package ui

import (
	"image"
	"image/color"
)

// NativeMenuOpen reports whether owner has a native menu presented or queued
// by the committed frame. Query after layout or between frames on the UI
// thread. Hidden, omitted and expired bindings return false. This does not
// observe retained in-window menus or popovers.
func (c *Context) NativeMenuOpen(owner Handle) bool {
	space, ok := owner.Coordinates(c)
	if !ok {
		return false
	}
	menu := &c.runtime().menu
	return menu.active != nil && menu.active.id == space.id || menu.pending != nil && menu.pending.id == space.id
}

// OnMenuDismiss observes a native menu closing, after its chosen item has
// been delivered to ContextMenu or Menu. Cancellation also closes a menu.
// The latest observer runs after the final frame, only while its host still
// exists; rebuilding a control never replays an already delivered close.
func (e Element) OnMenuDismiss(fn func()) Element {
	n := e.node()
	if n == nil || fn == nil {
		return e
	}
	rt, id := n.c.rt, n.id
	if rt.menu.dismissed == id {
		rt.public.AfterFrame(func() {
			if rt.menu.dismissed != id {
				return
			}
			rt.menu.dismissed = 0
			fn()
		})
	}
	return e
}

// ImagePNG sets the native menu item's icon. Its bytes are copied into this
// opening's snapshot; native hosts retain their decoded image until dismissal.
func (it *MenuItem) ImagePNG(data []byte) *MenuItem {
	if it.p != nil {
		it.p.ImagePNG = append([]byte(nil), data...)
	}
	return it
}

// RasterizeIcon draws an SVG's alpha mask in the supplied color into an owned
// image. Native menus and other OS services can share the same icon resources
// as in-window Icon elements without constructing a second UI runtime.
func (s *SVG) RasterizeIcon(width, height int, ink Color) *image.RGBA {
	if s == nil || width <= 0 || height <= 0 || width > 4096 || height > 4096 {
		return nil
	}
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	s.doc.Draw(dst, color.NRGBA{R: ink.R, G: ink.G, B: ink.B, A: 255}, false)
	for i := 0; i < len(dst.Pix); i += 4 {
		a := uint32(dst.Pix[i+3]) * uint32(ink.A) / 255
		dst.Pix[i] = uint8(uint32(ink.R) * a / 255)
		dst.Pix[i+1] = uint8(uint32(ink.G) * a / 255)
		dst.Pix[i+2] = uint8(uint32(ink.B) * a / 255)
		dst.Pix[i+3] = uint8(a)
	}
	return dst
}

// ContextMenuBase consumes the toolkit's ordinary secondary-click and keyboard
// context request without asking the OS to present a menu. Its caller supplies
// retained content at the requested window position; MyGo still owns targeting,
// disabled checks and Shift+F10 routing, including text-selection semantics.
func (e Element) ContextMenuBase(present func(x, y float32)) Element {
	n := e.node()
	if n == nil || present == nil {
		return e
	}
	n.flags |= flagContextMenu
	request := &n.c.rt.menu
	if request.asked == n.id && !request.button {
		request.asked = 0
		present(request.x, request.y)
	}
	return e
}
