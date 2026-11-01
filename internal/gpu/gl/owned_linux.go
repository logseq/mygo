//go:build linux && (amd64 || arm64)

package gl

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

var ownedFinish struct {
	sync.Once
	call func()
	err  error
}

func finishOwnedImages() error {
	ownedFinish.Do(func() {
		lib, err := purego.Dlopen("libepoxy.so.0", purego.RTLD_NOW|purego.RTLD_GLOBAL)
		if err != nil {
			ownedFinish.err = err
			return
		}
		sym, err := purego.Dlsym(lib, "epoxy_glFinish")
		if err != nil || sym == 0 {
			ownedFinish.err = fmt.Errorf("gl: cannot load owned image completion: %v", err)
			return
		}
		// Like the other epoxy entry points, the symbol is a function pointer.
		purego.RegisterFunc(&ownedFinish.call, **(**uintptr)(unsafe.Pointer(&sym)))
	})
	if ownedFinish.err != nil {
		return ownedFinish.err
	}
	ownedFinish.call()
	return nil
}
