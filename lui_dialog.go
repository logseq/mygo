package mygo

import (
	"context"
	"sync"

	"github.com/egoist/mygo/internal/platform"
)

type contextDialogResult struct {
	paths []string
	err   error
}

// OpenContext shows the native file dialog and closes that presentation when
// ctx is cancelled. Native handles remain owned until their close callback;
// cancellation never calls a released handle. Like Open, it is safe from any
// goroutine, including the main thread, where native events keep being pumped.
func (DialogModule) OpenContext(ctx context.Context, opts OpenDialogOptions) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	needsApp("Dialog.OpenContext")
	native := &platform.OpenDialogOptions{
		Title: opts.Title, DefaultPath: opts.DefaultPath, ButtonLabel: opts.ButtonLabel, Message: opts.Message, Filters: toFilters(opts.Filters),
		OpenFiles: !opts.Directory, OpenDirectories: opts.Directory, Multiple: opts.Multiple, ShowHiddenFiles: opts.ShowHiddenFiles,
		CreateDirectories: opts.CreateDirectories, TreatPackagesAsDirectories: opts.TreatPackagesAsDirectories,
	}
	result := startContextOpenDialog(ctx, native, func(options *platform.OpenDialogOptions, done func([]string, error)) {
		backend().Dialogs().ShowOpenDialog(opts.Parent.nativeOrNil(), options, done)
	}, postMain, func() { backend().Wake() })
	reply := await(result)
	return reply.paths, reply.err
}

// startContextOpenDialog keeps OS-specific handles on their owning UI queue.
// The injected functions are the native boundary, shared by desktop hosts and
// exercised without replacing the public file-picker UI or its state machine.
func startContextOpenDialog(ctx context.Context, opts *platform.OpenDialogOptions, show func(*platform.OpenDialogOptions, func([]string, error)), post func(func()) bool, wake func()) <-chan contextDialogResult {
	ch := make(chan contextDialogResult, 1)
	var mu sync.Mutex
	finished := false
	var nativeCancel func()
	var stop func() bool
	finish := func(paths []string, err error) {
		mu.Lock()
		if finished {
			mu.Unlock()
			return
		}
		finished = true
		nativeCancel = nil
		stopWatcher := stop
		mu.Unlock()
		if stopWatcher != nil {
			stopWatcher()
		}
		if ctx.Err() != nil {
			paths = nil
			err = ctx.Err()
		}
		ch <- contextDialogResult{paths: paths, err: err}
		wake()
	}
	opts.RegisterCancel = func(cancel func()) {
		mu.Lock()
		defer mu.Unlock()
		if !finished {
			nativeCancel = cancel
		}
	}
	stopWatcher := context.AfterFunc(ctx, func() {
		if !post(func() {
			mu.Lock()
			cancel := nativeCancel
			active := !finished
			mu.Unlock()
			if active && cancel != nil {
				cancel()
			}
		}) {
			finish(nil, errLoopStopped)
		}
	})
	mu.Lock()
	stop = stopWatcher
	alreadyFinished := finished
	mu.Unlock()
	if alreadyFinished {
		stopWatcher()
	}
	if !post(func() {
		if err := ctx.Err(); err != nil {
			finish(nil, err)
			return
		}
		show(opts, finish)
	}) {
		finish(nil, errLoopStopped)
	}
	return ch
}
