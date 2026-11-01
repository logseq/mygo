//go:build linux && (amd64 || arm64)

package gl

import (
	"testing"

	"github.com/ebitengine/purego"
	"github.com/egoist/mygo/internal/scene"
)

func TestOwnedRetirementRequiresMatchingContext(t *testing.T) {
	for _, api := range apis {
		for _, state := range []string{"matching", "different", "unknown"} {
			t.Run(api.name+"/"+state, func(t *testing.T) {
				offscreen(t, api.es, 8, 8)
				r, err := New()
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(r.Release)
				var completed, credited bool
				var completionErr error
				img := scene.NewImageRGBA(1, 1, []byte{255, 0, 0, 255})
				img.Lifetime = &scene.ImageLifetime{
					Alive: func() bool { return true },
					Acquire: func(any, int64) (func(error), bool) {
						return func(err error) { completed = true; completionErr = err; credited = err == nil }, true
					},
				}
				s := &scene.Scene{Width: 8, Height: 8, Scale: 1, Ops: []scene.Op{{Kind: scene.OpImage, Image: img, Rect: scene.Rect{W: 8, H: 8}, Src: scene.Rect{W: 1, H: 1}, Opacity: 1}}}
				if err := r.Render(s); err != nil {
					t.Fatal(err)
				}
				if state == "different" {
					lib, err := purego.Dlopen("libEGL.so.1", purego.RTLD_NOW|purego.RTLD_GLOBAL)
					if err != nil {
						t.Fatal(err)
					}
					var display, context func() uintptr
					var makeCurrent func(uintptr, uintptr, uintptr, uintptr) uint32
					purego.RegisterLibFunc(&display, lib, "eglGetCurrentDisplay")
					purego.RegisterLibFunc(&context, lib, "eglGetCurrentContext")
					purego.RegisterLibFunc(&makeCurrent, lib, "eglMakeCurrent")
					d, c := display(), context()
					// The original Offscreen terminates the shared display and both contexts.
					if _, err := NewOffscreen(api.es, 8, 8); err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() {
						if makeCurrent(d, 0, 0, c) == 0 {
							t.Error("restore original EGL context")
						}
					})
					if context() == c {
						t.Fatal("probe did not switch EGL contexts")
					}
				} else if state == "unknown" {
					r.context = 0
				}
				err = r.RetireImage(img.ID())
				wantCredit := state == "matching"
				if !completed || credited != wantCredit || (err == nil) != wantCredit || (completionErr == nil) != wantCredit {
					t.Fatalf("state=%s completed=%v credited=%v retirement=%v completion=%v", state, completed, credited, err, completionErr)
				}
			})
		}
	}
}
