package ui

import "testing"

func TestCaptureClickOnPlainHit(t *testing.T) {
	for _, tc := range []struct {
		name    string
		native  bool
		prevent InputKind
		action  string
		clicks  int
		presses int
	}{
		{name: "plain", clicks: 1},
		{name: "native", native: true, clicks: 1, presses: 1},
		{name: "prevent native click", native: true, prevent: InputClick, clicks: 1},
		{name: "prevent down", prevent: InputPointerDown, clicks: 1},
		{name: "prevent native down", native: true, prevent: InputPointerDown, clicks: 1},
		{name: "prevent up", prevent: InputPointerUp},
		{name: "prevent plain click", prevent: InputClick, clicks: 1},
		{name: "drag and return", action: "drag"},
		{name: "disabled", action: "disabled"},
		{name: "retired", action: "retired"},
		{name: "replaced", action: "replaced"},
		{name: "pointer disabled", action: "pointer"},
		{name: "capture removed", action: "capture"},
		{name: "window blur", action: "blur"},
		{name: "prevent down then drag", prevent: InputPointerDown, action: "drag"},
		{name: "prevent down then replace", prevent: InputPointerDown, action: "replaced"},
		{name: "prevent down then blur", prevent: InputPointerDown, action: "blur"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var hit Handle
			clicks, presses := 0, 0
			present, capture, enabled, disabled, key := true, true, true, false, "original"
			var value string
			tt := NewTester(func(c *Context) {
				if capture {
					c.CaptureInput(func(ev InputEvent, target InputTarget) bool {
						if ev.Kind == InputClick {
							clicks++
							if target.Distance(hit) < 0 || !tc.native && target.Distance(hit) != 0 || ev.X != 50 || ev.Y != 14 || ev.Clicks != 1 {
								t.Fatalf("click has stale target or wrong window coordinates/count: %+v", ev)
							}
						}
						return ev.Kind == tc.prevent
					})
				}
				TextInput(c, &value).Label("Editor").AutoFocus().Absolute().Left(110).Top(30).Size(80, 28)
				if !present {
					return
				}
				var e Element
				if tc.native {
					e = Button(c, "Action").OnClick(func() { presses++ })
				} else {
					e = Box(c)
				}
				e.Key(key).Size(100, 28).Bind(&hit).Disabled(disabled).PointerEvents(enabled)
			}, 200, 60)
			tt.Press(50, 14)
			switch tc.action {
			case "drag":
				tt.Move(70, 14)
				tt.Move(50, 14)
			case "disabled":
				disabled = true
			case "retired":
				present = false
				tt.Frame()
				present = true
			case "replaced":
				key = "replacement"
			case "pointer":
				enabled = false
			case "capture":
				capture = false
				tt.Frame()
				capture = true
			case "blur":
				tt.SetFocused(false)
			}
			tt.Frame()
			tt.Release(50, 14)
			if tc.prevent == InputPointerDown && tc.action == "" && !tt.Focused("Editor") {
				t.Fatal("prevented down stole native editor focus")
			}
			if clicks != tc.clicks || presses != tc.presses {
				t.Fatalf("capture clicks/native presses = %d/%d, want %d/%d", clicks, presses, tc.clicks, tc.presses)
			}
		})
	}
}

func TestCaptureInputBeforeNativeConsumption(t *testing.T) {
	var field Handle
	var value string
	var capture bool = true
	var saved InputTarget
	var keys int
	tt := NewTester(func(c *Context) {
		if capture {
			c.CaptureInput(func(ev InputEvent, target InputTarget) bool {
				if ev.Kind != InputKeyDown && ev.Kind != InputCommand {
					return false
				}
				if target.Distance(field) != 0 {
					t.Fatal("capture did not identify the focused native field")
				}
				saved = target
				if ev.Kind == InputKeyDown {
					keys++
				}
				return ev.Kind == InputKeyDown && ev.Key == KeyBackspace
			})
		}
		TextInput(c, &value).Bind(&field).Label("Query").AutoFocus()
	}, 300, 100)
	tt.Frame()
	tt.Type("search")
	tt.Key(0, KeyBackspace)
	if value != "search" || keys != 1 {
		t.Fatalf("capture failed to prevent native deletion: %q, %d key events", value, keys)
	}
	if saved.Distance(field) != -1 {
		t.Fatal("input target remained live outside its callback")
	}
	tt.Command("selectAll")
	tt.Type("next")
	if value != "next" {
		t.Fatalf("capture swallowed native selectAll: %q", value)
	}
	capture = false
	tt.Frame()
	tt.Key(0, KeyBackspace)
	if value != "nex" || keys != 1 {
		t.Fatalf("removed capture still handled input: %q, %d key events", value, keys)
	}
}
