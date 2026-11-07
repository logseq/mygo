//go:build darwin && lui_native_smoke

// Package native exposes the native backend's existing test hooks only for
// explicitly opted-in desktop smoke tools. Every call must run on the UI thread.
package native

import "github.com/egoist/mygo/internal/darwin"

type AccessNode = darwin.TestAccessNode

var (
	OnScreen      = darwin.TestOnScreen
	WindowNumber  = darwin.TestWindowNumber
	Click         = darwin.TestClick
	AppClick      = darwin.TestAppClick
	AppPress      = darwin.TestAppPress
	DragMove      = darwin.TestDragMove
	Key           = darwin.TestKey
	KeyShift      = darwin.TestKeyShift
	Scroll        = darwin.TestScroll
	Drag          = darwin.TestDrag
	Compose       = darwin.TestCompose
	InputClient   = darwin.TestInputClient
	Accessibility = darwin.TestAccessibility
	Perform       = darwin.TestAccessibilityPerform
)
