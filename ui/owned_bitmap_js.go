//go:build js

package ui

import "github.com/egoist/mygo/internal/gpu/gl"

func ownedBitmapRetirement(h *windowHost, renderer any, id uint64) func() error {
	if r, ok := renderer.(*gl.Renderer); ok {
		restoreContext := gl.OwnedImageContext()
		r.OwnedCleanup = func(restore bool) {
			if restore {
				restoreContext()
			}
			r.RetireImage(0)
		}
		return func() error {
			restoreContext()
			return r.RetireImage(id)
		}
	}
	return ownedBitmapFallbackRetirement(renderer, id)
}
