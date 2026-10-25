package mygo

import (
	"context"
	"errors"
	"github.com/egoist/mygo/internal/platform"
	"testing"
	"time"
)

func TestLUIOpenContextCancelledBeforeApplication(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Dialog.OpenContext(ctx, OpenDialogOptions{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled request must not start application or native dialog: %v", err)
	}
}
func TestLUIOpenContextClosesTheNativeDialogOnItsUIQueue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	queue := make(chan func(), 8)
	closed := 0
	result := startContextOpenDialog(ctx, &platform.OpenDialogOptions{}, func(opts *platform.OpenDialogOptions, done func([]string, error)) {
		if opts.RegisterCancel != nil {
			opts.RegisterCancel(func() { closed++; done(nil, nil) })
		}
	}, func(fn func()) bool { queue <- fn; return true }, func() {})
	(<-queue)()
	cancel()
	select {
	case closeNative := <-queue:
		closeNative()
	case <-time.After(time.Second):
		t.Fatal("cancellation did not queue native presentation dismissal")
	}
	if closed != 1 {
		t.Fatalf("native close count: %d", closed)
	}
	select {
	case reply := <-result:
		if !errors.Is(reply.err, context.Canceled) {
			t.Fatalf("cancel result: %v", reply.err)
		}
	case <-time.After(time.Second):
		t.Fatal("native dismissal did not release waiter")
	}
}
func TestLUIOpenContextDoesNotCloseAnAlreadyDisposedNativeHandle(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	queue := make(chan func(), 8)
	closed := 0
	result := startContextOpenDialog(ctx, &platform.OpenDialogOptions{}, func(opts *platform.OpenDialogOptions, done func([]string, error)) {
		if opts.RegisterCancel != nil {
			opts.RegisterCancel(func() { closed++ })
		}
		done([]string{"/picked"}, nil)
	}, func(fn func()) bool { queue <- fn; return true }, func() {})
	(<-queue)()
	reply := <-result
	if len(reply.paths) != 1 {
		t.Fatal("native selection was lost")
	}
	cancel()
	select {
	case fn := <-queue:
		fn()
	case <-time.After(10 * time.Millisecond):
	}
	if closed != 0 {
		t.Fatal("late cancellation touched released native dialog")
	}
}
