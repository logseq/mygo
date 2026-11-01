package ui

import (
	"errors"
	"sync"
	"sync/atomic"

	"github.com/egoist/mygo/internal/scene"
)

// OwnedBitmapOptions controls the native textures of an OwnedBitmap.
type OwnedBitmapOptions struct {
	// AcquireTexture reserves width*height*4 payload bytes before each renderer
	// allocates a texture. A rejection skips this paint and can be retried on a
	// later paint. It must not block or re-enter rendering. A successful call
	// must return a non-nil release, called once after native references and GPU
	// uses retire. Uncertain retirement quarantines the reservation instead.
	// Nil means no external admission, not that textures need no retirement.
	AcquireTexture func(payloadBytes int64) (release func(), ok bool)
}

// OwnedBitmap is an immutable, explicitly owned ImageSource. Create it with
// NewOwnedBitmapRGBA and dispose it when all its views should stop showing it.
// Unlike Bitmap, it never decodes, restores, updates or makes mipmaps.
// Its public methods are safe from any goroutine.
type OwnedBitmap struct {
	bitmap  *Bitmap
	dead    atomic.Bool
	mu      sync.Mutex
	options OwnedBitmapOptions
	ticket  *BitmapDisposal
	owners  map[*windowHost]bool
	tokens  map[any]*ownedBitmapToken
	drawing *windowHost
	err     error
}

// BitmapDisposal completes after the CPU pixels are no longer reachable inside
// the library and all native texture retirements have completed or failed.
// A failure keeps the corresponding GPU reservation quarantined. Never wait
// for Done on the UI thread; native retirement itself runs on that thread.
type BitmapDisposal struct {
	done chan struct{}
	err  error
}

// Done closes when disposal is complete, successfully or with an error.
func (d *BitmapDisposal) Done() <-chan struct{} { return d.done }

// Err reports a retirement failure after Done closes. Before that it returns nil.
func (d *BitmapDisposal) Err() error {
	select {
	case <-d.done:
		return d.err
	default:
		return nil
	}
}

type ownedBitmapToken struct {
	retire   func() error
	release  func()
	finished bool
	owner    *windowHost
}

type ownedBitmapHooks struct {
	bind   func(*windowHost, *scene.Scene)
	render func(*windowHost, *scene.Scene)
	detach func(*windowHost)
}

var ownedBitmapFeature atomic.Pointer[ownedBitmapHooks]

// NewOwnedBitmapRGBA takes exclusive ownership of exactly width*height*4 packed,
// premultiplied RGBA8 bytes on success. The caller must not keep, mutate or reuse
// aliases of pixels. On error ownership stays with the caller. Dimensions must
// be positive. Pixels are not copied and no encoded representation is retained.
func NewOwnedBitmapRGBA(width, height int, pixels []byte, options OwnedBitmapOptions) (*OwnedBitmap, error) {
	maxInt := int(^uint(0) >> 1)
	if width <= 0 || height <= 0 || width > maxInt/4 || height > maxInt/(4*width) || len(pixels) != 4*width*height {
		return nil, errors.New("mygo: owned bitmap requires exact packed RGBA8 pixels and positive dimensions")
	}
	b := &OwnedBitmap{
		bitmap:  &Bitmap{img: scene.NewImageRGBA(width, height, pixels), w: width, h: height, noMip: true},
		options: options, owners: make(map[*windowHost]bool), tokens: make(map[any]*ownedBitmapToken),
	}
	b.bitmap.img.Lifetime = &scene.ImageLifetime{Owner: b, Alive: func() bool { return !b.dead.Load() }, Acquire: b.acquire}
	ownedBitmapFeature.CompareAndSwap(nil, &ownedBitmapHooks{bind: bindOwnedBitmaps, render: renderOwnedBitmaps, detach: detachOwnedBitmaps})
	return b, nil
}

// Size returns the immutable dimensions in pixels, even after disposal.
func (b *OwnedBitmap) Size() (width, height int) { return b.bitmap.w, b.bitmap.h }

func (b *OwnedBitmap) imageSize() (w, h float32) { return float32(b.bitmap.w), float32(b.bitmap.h) }

// Dispose terminally retires the bitmap in every window. Views borrow it: they
// do not dispose it on unmount. Repeated calls return the same ticket. Stale
// scenes retain only an inert tombstone and can never upload the image again.
func (b *OwnedBitmap) Dispose() *BitmapDisposal {
	b.mu.Lock()
	if b.ticket != nil {
		d := b.ticket
		b.mu.Unlock()
		return d
	}
	b.dead.Store(true)
	b.ticket = &BitmapDisposal{done: make(chan struct{}), err: b.err}
	d := b.ticket
	owners := make([]*windowHost, 0, len(b.owners))
	for h := range b.owners {
		owners = append(owners, h)
	}
	b.completeLocked()
	b.mu.Unlock()
	for _, h := range owners {
		// detach drains these owners even if a closed window drops this post.
		h.conn.Post(func() { b.retireOwner(h) })
	}
	return d
}

func (b *OwnedBitmap) completeLocked() {
	if b.ticket == nil || len(b.owners) != 0 || len(b.tokens) != 0 {
		return
	}
	b.bitmap.img.Pix = nil
	b.bitmap.img.Restore, b.bitmap.img.Dropped = nil, nil
	b.drawing = nil
	b.options = OwnedBitmapOptions{}
	close(b.ticket.done)
}

func (b *OwnedBitmap) acquire(renderer any, bytes int64) (func(error), bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.dead.Load() || b.drawing == nil || b.drawing.detached || b.drawing.ownedClosing || b.drawing.conn.Post == nil {
		return nil, false
	}
	t := &ownedBitmapToken{owner: b.drawing}
	if acquire := b.options.AcquireTexture; acquire != nil {
		release, ok := acquire(bytes)
		if !ok {
			return nil, false
		}
		if release == nil {
			return nil, false
		}
		t.release = release
	}
	finish := func(err error) {
		b.mu.Lock()
		if t.finished {
			b.mu.Unlock()
			return
		}
		t.finished = true
		delete(b.tokens, renderer)
		if err != nil {
			b.err = errors.Join(b.err, err)
			if b.ticket != nil {
				b.ticket.err = b.err
			}
		}
		b.mu.Unlock()
		if err == nil && t.release != nil {
			t.release()
		}
	}
	retire := ownedBitmapRetirement(b.drawing, renderer, b.bitmap.img.ID())
	t.retire = func() error {
		err := retire()
		finish(err)
		return err
	}
	b.tokens[renderer] = t
	return finish, true
}

func ownedBitmapFallbackRetirement(renderer any, id uint64) func() error {
	if r, ok := renderer.(interface{ RetireImage(uint64) error }); ok {
		return func() error { return r.RetireImage(id) }
	}
	return func() error { return errors.New("mygo: renderer cannot retire owned images") }
}

func (b *OwnedBitmap) retireOwner(h *windowHost) {
	b.mu.Lock()
	if !b.owners[h] {
		b.mu.Unlock()
		return
	}
	var retire []func() error
	for _, t := range b.tokens {
		if t.owner == h {
			retire = append(retire, t.retire)
		}
	}
	b.mu.Unlock()
	for _, fn := range retire {
		fn()
	}
	delete(h.ownedBitmaps, b)
	// A fresh frame must not reuse a Differ's old bitmap operation.
	h.soft.Release()
	if !h.detached && h.conn.Invalidate != nil {
		h.invalidate()
	}
	b.mu.Lock()
	delete(b.owners, h)
	if b.drawing == h {
		b.drawing = nil
	}
	b.completeLocked()
	b.mu.Unlock()
}

func bindOwnedBitmaps(h *windowHost, s *scene.Scene) {
	if h.detached || h.ownedClosing {
		return
	}
	visitOwnedBitmaps(s, func(b *OwnedBitmap) {
		b.mu.Lock()
		defer b.mu.Unlock()
		if b.dead.Load() || h.conn.Post == nil || h.detached {
			return
		}
		b.owners[h] = true
		if h.ownedBitmaps == nil {
			h.ownedBitmaps = make(map[*OwnedBitmap]bool)
			h.conn.RetireImages = func() {
				h.ownedClosing = true
				detachOwnedBitmaps(h)
			}
		}
		h.ownedBitmaps[b] = true
	})
}

func renderOwnedBitmaps(h *windowHost, s *scene.Scene) {
	visitOwnedBitmaps(s, func(b *OwnedBitmap) {
		b.mu.Lock()
		b.drawing = h
		b.mu.Unlock()
	})
}

func detachOwnedBitmaps(h *windowHost) {
	for b := range h.ownedBitmaps {
		b.retireOwner(h)
	}
}

func visitOwnedBitmaps(s *scene.Scene, fn func(*OwnedBitmap)) {
	seen := make(map[*scene.Scene]bool)
	var visit func(*scene.Scene)
	var image func(*scene.Image)
	image = func(img *scene.Image) {
		if img == nil {
			return
		}
		if img.Lifetime != nil {
			if b, ok := img.Lifetime.Owner.(*OwnedBitmap); ok {
				fn(b)
			}
		}
		if img.Scene != nil {
			visit(img.Scene)
		}
	}
	visit = func(s *scene.Scene) {
		if s == nil || seen[s] {
			return
		}
		seen[s] = true
		for _, op := range s.Ops {
			image(op.Image)
		}
		for _, effect := range s.Effects {
			for _, img := range effect.Images {
				image(img)
			}
		}
		for _, img := range s.Canvases {
			image(img)
		}
	}
	visit(s)
}
