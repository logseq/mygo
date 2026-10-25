package ui

import "testing"

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
