//go:build js

package gl

import (
	"errors"
	"syscall/js"
)

// OwnedImageContext captures the actual WebGL context while an owned upload
// is admitted. A later cleanup need not depend on another canvas's frame.
func OwnedImageContext() func() {
	c := cur
	return func() { c.MakeCurrent() }
}

func finishOwnedImages() error {
	if cur != nil {
		// Clear upload staging even on context loss, independently of GPU credit.
		defer func() { cur.scratch, cur.size = js.Undefined(), 0 }()
	}
	if cur == nil || cur.Lost() {
		return errors.New("gl: owned image retirement lost its WebGL context")
	}
	cur.v.Call("finish")
	if cur.Lost() {
		return errors.New("gl: WebGL context lost during owned image completion")
	}
	return nil
}
