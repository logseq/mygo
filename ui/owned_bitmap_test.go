package ui

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/egoist/mygo/internal/scene"
	"github.com/egoist/mygo/internal/surface"
	"github.com/egoist/mygo/internal/wait"
)

func TestOwnedBitmapPixelOwnership(t *testing.T) {
	maxInt := int(^uint(0) >> 1)
	for _, c := range []struct{ w, h, n int }{
		{0, 1, 0}, {-1, 1, 0}, {1, 0, 0}, {1, -1, 0},
		{maxInt, 2, 4}, {2, maxInt, 4}, {maxInt / 4, 2, 4},
		{1, 1, 3}, {1, 1, 5},
	} {
		pixels := make([]byte, c.n)
		if b, err := NewOwnedBitmapRGBA(c.w, c.h, pixels, OwnedBitmapOptions{}); b != nil || err == nil {
			t.Fatalf("accepted %dx%d with %d bytes", c.w, c.h, c.n)
		}
	}
	pixels := stripedImage(32, 32, 1, 0).Pix
	b, err := NewOwnedBitmapRGBA(32, 32, pixels, OwnedBitmapOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if &b.bitmap.img.Pix[0] != &pixels[0] {
		t.Fatal("owned input was copied")
	}
	s := &scene.Scene{Width: 8, Height: 8, Scale: 1}
	p := &Painter{s: s, scale: 1, opacity: 1}
	p.Image(b, Rect{W: 4, H: 4}, Contain)
	if len(s.Ops) != 1 || s.Ops[0].Image != b.bitmap.img || len(b.bitmap.levels) != 0 || b.bitmap.img.Restore != nil {
		t.Fatal("drawing introduced a mipmap or restore path")
	}
	// A stale worker result has no UI owner or texture to wait for.
	d := b.Dispose()
	wait.For(t, "unpublished pixels to retire", d.Done())
	if d.Err() != nil || b.bitmap.img.Pix != nil || b.bitmap.img.Pixels() != nil {
		t.Fatal("disposed pixel storage remains reachable")
	}
	p.Image(b, Rect{W: 4, H: 4}, Contain)
	if len(s.Ops) != 1 || b.Dispose() != d {
		t.Fatal("terminal owner was painted again or got a different ticket")
	}
	if w, h := b.Size(); w != 32 || h != 32 {
		t.Fatalf("disposed dimensions changed to %dx%d", w, h)
	}
}

// ownedTestRenderer implements the same admission/completion protocol as native
// renderers. Its controllable barrier protects ownership ordering, not a driver.
type ownedTestRenderer struct {
	finish map[uint64]func(error)
	err    error
}

func (r *ownedTestRenderer) Render(s *scene.Scene) error {
	for _, op := range s.Ops {
		img := op.Image
		if img == nil || img.Lifetime == nil || !img.Lifetime.Alive() || r.finish[img.ID()] != nil {
			continue
		}
		if finish, ok := img.Lifetime.Acquire(r, int64(img.W)*int64(img.H)*4); ok {
			if r.finish == nil {
				r.finish = make(map[uint64]func(error))
			}
			r.finish[img.ID()] = finish
		}
	}
	return nil
}

func (r *ownedTestRenderer) RetireImage(id uint64) error {
	if finish := r.finish[id]; finish != nil {
		delete(r.finish, id)
		finish(r.err)
	}
	return r.err
}

func (r *ownedTestRenderer) Release() {
	for id := range r.finish {
		r.RetireImage(id)
	}
}

func TestOwnedBitmapRetirementConservesCredit(t *testing.T) {
	for _, c := range []struct {
		name       string
		close      bool
		closeFirst bool
		err        error
	}{
		{name: "two windows"},
		{name: "close drops posted work", close: true},
		{name: "Run exits before Dispose", close: true, closeFirst: true},
		{name: "quarantine", close: true, err: errors.New("device or context lost")},
	} {
		t.Run(c.name, func(t *testing.T) {
			var admissions, credits atomic.Int32
			b, err := NewOwnedBitmapRGBA(1, 1, []byte{255, 0, 0, 255}, OwnedBitmapOptions{
				AcquireTexture: func(bytes int64) (func(), bool) {
					if bytes != 4 {
						t.Errorf("admission: %d bytes, want 4", bytes)
					}
					if admissions.Add(1) == 1 {
						return nil, false
					}
					return func() { credits.Add(1) }, true
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			s := &scene.Scene{Width: 8, Height: 8, Scale: 1, Ops: []scene.Op{
				{Kind: scene.OpImage, Image: b.bitmap.img, Rect: scene.Rect{W: 8, H: 8}, Opacity: 1},
			}}
			posts := make(chan func(), 2)
			var hosts []*windowHost
			for range 2 {
				r := &ownedTestRenderer{}
				h := &windowHost{gpu: r, conn: &surface.Conn{
					Surface: &testSurface{}, Post: func(fn func()) { posts <- fn }, Invalidate: func() {},
				}}
				bindOwnedBitmaps(h, s)
				renderOwnedBitmaps(h, s)
				r.Render(s)
				if len(hosts) == 0 {
					if len(r.finish) != 0 {
						t.Fatal("denied texture was uploaded")
					}
					r.Render(s)
				}
				if len(r.finish) != 1 {
					t.Fatal("admission was not independently retried/reserved")
				}
				hosts = append(hosts, h)
			}
			closeOwners := func() {
				hosts[0].conn.RetireImages()
				hosts[1].gpu.(*ownedTestRenderer).err = c.err
				hosts[1].detached = true
				detachOwnedBitmaps(hosts[1])
			}
			if c.closeFirst {
				closeOwners()
			}
			// No sleeps: all disposers join before the UI owner drains work.
			var group sync.WaitGroup
			tickets := make(chan *BitmapDisposal, 16)
			for range 16 {
				group.Go(func() { tickets <- b.Dispose() })
			}
			group.Wait()
			d := <-tickets
			for range 15 {
				if <-tickets != d {
					t.Fatal("concurrent Dispose returned different tickets")
				}
			}
			select {
			case <-d.Done():
				if !c.closeFirst {
					t.Fatal("CPU credit returned before UI/GPU retirement")
				}
			default:
			}
			if c.close {
				// No visible frame or queued Post is needed for close/quit.
				if !c.closeFirst {
					closeOwners()
				}
			} else {
				wait.For(t, "first owner post", posts)()
				select {
				case <-d.Done():
					t.Fatal("first renderer ack retired the second renderer's CPU borrow")
				default:
				}
				wait.For(t, "second owner post", posts)()
			}
			wait.For(t, "all owned resources to retire", d.Done())
			wantCredits := int32(2)
			if c.err != nil {
				wantCredits--
			}
			if !errors.Is(d.Err(), c.err) || credits.Load() != wantCredits || b.bitmap.img.Pix != nil {
				t.Fatalf("retirement: err=%v credits=%d pixels=%d", d.Err(), credits.Load(), len(b.bitmap.img.Pix))
			}
			// Late posts/old scenes cannot return credit twice or re-upload.
			for len(posts) != 0 {
				(<-posts)()
			}
			for _, h := range hosts {
				h.gpu.Render(s)
				h.gpu.Release()
			}
			if b.Dispose() != d || credits.Load() != wantCredits || admissions.Load() != 3 {
				t.Fatal("stale scene or repeated completion changed admission/credit")
			}
		})
	}
}
