//go:build darwin

package metal

// RetireImage retires one explicitly owned image, or all of them for id zero.
// Dropping a pending frame can commit its preparation commands; waitLast must
// therefore follow dropPending, before releasing either textures or credit.
func (r *Renderer) RetireImage(id uint64) error {
	var targets []*texture
	for key, t := range r.images {
		if t.owned != nil && (id == 0 || key == id) {
			targets = append(targets, t)
			delete(r.images, key)
		}
	}
	if len(targets) == 0 {
		return nil
	}
	pool(func() {
		r.dropPending()
		r.waitLast()
		r.b.Release()
		r.b.Classes, r.b.Opaque = true, true
		r.diff.Release()
		r.forgetFrames()
		for _, t := range r.canvases {
			t.b.Release()
			t.ver = 0
		}
		for _, t := range targets {
			release(&t.tex)
		}
	})
	for _, t := range targets {
		t.owned(r.err)
		t.owned = nil
	}
	return r.err
}
