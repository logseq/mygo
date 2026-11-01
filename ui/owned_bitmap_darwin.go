//go:build darwin

package ui

import "github.com/egoist/mygo/internal/gpu/metal"

func ownedBitmapRetirement(h *windowHost, renderer any, id uint64) func() error {
	if r, ok := renderer.(*metal.Renderer); ok {
		r.OwnedCleanup = func(bool) { r.RetireImage(0) }
		return func() error { return r.RetireImage(id) }
	}
	return ownedBitmapFallbackRetirement(renderer, id)
}
