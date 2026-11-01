//go:build linux && (amd64 || arm64)

package ui

import "github.com/egoist/mygo/internal/gpu/gl"

func ownedBitmapRetirement(h *windowHost, renderer any, id uint64) func() error {
	if r, ok := renderer.(*gl.Renderer); ok {
		area := h.conn.Surface.Native().GLArea
		r.OwnedCleanup = func(restore bool) {
			if restore && !h.detached {
				gl.MakeAreaCurrent(area)
			}
			r.RetireImage(0)
		}
		return func() error {
			if !h.detached {
				gl.MakeAreaCurrent(area)
			}
			return r.RetireImage(id)
		}
	}
	return ownedBitmapFallbackRetirement(renderer, id)
}
