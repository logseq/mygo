//go:build windows

package d3d11

import (
	"fmt"
	"runtime"
	"time"
	"unsafe"
)

// ID3D11Device and ID3D11DeviceContext indices from the Windows SDK.
const (
	devCreateQuery = 24
	ctxEnd         = 28
	ctxGetData     = 29
)

func (r *Renderer) finishOwnedImages() error {
	if r.device == 0 || r.ctx == 0 {
		return fmt.Errorf("d3d11: owned image retirement has no device")
	}
	// D3D11_QUERY_EVENT, no miscellaneous flags.
	desc := [2]uint32{0, 0}
	var query uintptr
	hr := call(r.device, devCreateQuery, uintptr(unsafe.Pointer(&desc[0])), uintptr(unsafe.Pointer(&query)))
	if failed(hr) || query == 0 {
		return fmt.Errorf("d3d11: cannot create retirement event query: %#x", uint32(hr))
	}
	defer free(&query)
	call(r.ctx, ctxEnd, query)
	call(r.ctx, ctxFlush)
	guard := time.Now().Add(30 * time.Second)
	for {
		var done int32                                                            // BOOL, not Go bool.
		hr = call(r.ctx, ctxGetData, query, uintptr(unsafe.Pointer(&done)), 4, 1) // DONOTFLUSH
		if hr == 0 && done != 0 {
			return nil
		}
		if failed(hr) || time.Now().After(guard) {
			return fmt.Errorf("d3d11: retirement query failed or did not complete: %#x", uint32(hr))
		}
		runtime.Gosched()
	}
}

// RetireImage retires one explicitly owned image, or all of them for id zero.
// COM release and Flush alone do not prove completion of queued draws.
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
	if r.ctx != 0 {
		var none [11]uintptr
		call(r.ctx, ctxPSSetShaderResources, 0, uintptr(len(none)), uintptr(unsafe.Pointer(&none[0])))
	}
	err := r.finishOwnedImages()
	for _, t := range targets {
		free(&t.srv)
		free(&t.tex)
		t.owned(err)
		t.owned = nil
	}
	return err
}
