package ui

import "testing"

func TestNavigationKeysRespectFocusedChildAndNearestOwner(t *testing.T) {
	for _, tc := range []struct {
		name, child string
		key         Key
		mods        Modifiers
		declared    []Key
		decline     bool
		wantScope   int
		wantChild   int
	}{
		{name: "left", key: KeyLeft, wantScope: 1},
		{name: "right", key: KeyRight, wantScope: 1},
		{name: "up", key: KeyUp, wantScope: 1},
		{name: "down", key: KeyDown, wantScope: 1},
		{name: "shift-left", key: KeyLeft, mods: Shift, wantScope: 1},
		{name: "shift-right", key: KeyRight, mods: Shift, wantScope: 1},
		{name: "shift-up", key: KeyUp, mods: Shift, wantScope: 1},
		{name: "shift-down", key: KeyDown, mods: Shift, wantScope: 1},
		{name: "ctrl", key: KeyRight, mods: Ctrl},
		{name: "alt", key: KeyRight, mods: Alt},
		{name: "super", key: KeyRight, mods: Super},
		{name: "unlisted", key: KeyLeft, declared: []Key{KeyRight}},
		{name: "declined", key: KeyRight, decline: true, wantScope: 1},
		{name: "child-handled", child: "handler", key: KeyRight, wantChild: 1},
		{name: "child-shortcut", child: "shortcut", key: KeyRight, wantChild: 1},
		{name: "field", child: "field", key: KeyRight},
		{name: "readonly", child: "readonly", key: KeyRight},
		{name: "password", child: "password", key: KeyRight},
		{name: "combobox", child: "combobox", key: KeyRight},
		{name: "enter", key: KeyEnter, wantChild: 1},
		{name: "space", key: KeySpace, wantChild: 1},
		{name: "inside-overlay", child: "inside-overlay", key: KeyRight, wantScope: 1},
		{name: "outside-overlay", child: "outside-overlay", key: KeyRight},
	} {
		t.Run(tc.name, func(t *testing.T) {
			scopeCalls, outerCalls, childCalls := 0, 0, 0
			text, selected := "text", "Alpha"
			keys := tc.declared
			if keys == nil {
				keys = []Key{KeyLeft, KeyRight, KeyUp, KeyDown}
			}
			var received InputEvent
			child := func(c *Context) {
				switch tc.child {
				case "field", "readonly", "password":
					input := TextInput(c, &text).Focus()
					if tc.child == "readonly" {
						input.ReadOnly()
					}
					if tc.child == "password" {
						input.Password()
					}
				case "combobox":
					Combobox(c, &selected, []string{"Alpha", "Beta"}).Focus()
				default:
					button := Button(c, "Pill").Focus().OnClick(func() { childCalls++ })
					if tc.child == "handler" {
						button.HandleInput(func(ev InputEvent) bool {
							if ev.Kind == InputKeyDown && ev.Key == tc.key {
								childCalls++
								return true
							}
							return false
						})
					}
					if tc.child == "shortcut" {
						button.OnShortcut(tc.mods, tc.key, func() { childCalls++ })
					}
				}
			}
			tt := NewTester(func(c *Context) {
				build := func() {
					Column(c).HandleNavigationKeys([]Key{KeyLeft, KeyRight, KeyUp, KeyDown}, func(InputEvent) bool {
						outerCalls++
						return true
					}).Children(func() {
						Column(c).HandleNavigationKeys(keys, func(ev InputEvent) bool {
							scopeCalls++
							received = ev
							return !tc.decline
						}).Children(func() {
							if tc.child == "outside-overlay" {
								Overlay(c, func() { child(c) })
							} else {
								child(c)
							}
						})
					})
				}
				if tc.child == "inside-overlay" {
					Overlay(c, func() { Box(c).Modal().Children(build) })
				} else {
					build()
				}
			}, 300, 200)
			tt.Key(tc.mods, tc.key)
			if scopeCalls != tc.wantScope || outerCalls != 0 || childCalls != tc.wantChild {
				t.Fatalf("scope %d, outer %d, child %d; want %d, 0, %d", scopeCalls, outerCalls, childCalls, tc.wantScope, tc.wantChild)
			}
			if scopeCalls > 0 && (received.Kind != InputKeyDown || received.Key != tc.key || received.Mods != tc.mods) {
				t.Fatalf("routed event lost key/modifiers: %+v", received)
			}
			tt.Frame()
			if scopeCalls != tc.wantScope {
				t.Fatal("a later frame redelivered the arrow")
			}
		})
	}
}
