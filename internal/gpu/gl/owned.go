//go:build (linux && (amd64 || arm64)) || js

package gl

import "fmt"

// RetireImage retires one explicitly owned image, or all of them when id is
// zero. It is opt-in: normal Release paths reach it only through OwnedCleanup.
// The renderer's exact context must be current; a lost/mismatched context is
// not proof of GPU completion and quarantines reservations.
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
	r.b.Release()
	r.diff.Release()
	r.held = false
	for _, t := range r.canvases {
		t.b.Release()
		t.ver = 0
	}
	var err error
	if r.context == 0 || currentContext() != r.context || !hasContext() {
		err = fmt.Errorf("gl: owned image retirement has no matching context")
	} else {
		// Images can also have been read by an effect's image units.
		for unit := range uint32(9) {
			glActiveTexture(glTexture0 + unit)
			glBindTexture(glTexture2D, 0)
		}
		glActiveTexture(glTexture0)
		err = finishOwnedImages()
		for _, t := range targets {
			glDeleteTextures(1, &t.tex)
		}
		if code := glGetError(); code != glNoError && err == nil {
			err = fmt.Errorf("gl: owned image retirement failed: %#x", code)
		}
	}
	for _, t := range targets {
		t.owned(err)
		t.owned = nil
	}
	return err
}
