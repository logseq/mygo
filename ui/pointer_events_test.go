package ui

import "testing"

func TestPointerEventsInheritThroughNativeChildren(t *testing.T) {
	under, blocked, opted := 0, 0, 0
	prevent, coverEnabled := false, false
	tt := NewTester(func(c *Context) {
		Button(c, "Under").Size(240, 40).OnClick(func() { under++ })
		Box(c).Absolute().Left(0).Top(0).Size(300, 200).PointerEvents(coverEnabled).Children(func() {
			Button(c, "Covered caption").Size(240, 40).OnClick(func() { blocked++ })
			Button(c, "Opted in").Size(240, 40).PointerEvents(true).OnClick(func() { opted++ })
		})
		c.CaptureInput(func(ev InputEvent, _ InputTarget) bool { return prevent && ev.Kind == InputClick })
	}, 400, 240)
	if err := tt.Click("Covered caption"); err != nil {
		t.Fatal(err)
	}
	if under != 1 || blocked != 0 {
		t.Fatalf("native caption leaked through inherited pointer suppression: under=%d blocked=%d", under, blocked)
	}
	if err := tt.Click("Opted in"); err != nil {
		t.Fatal(err)
	}
	if opted != 1 {
		t.Fatal("explicit descendant opt-in did not restore native activation")
	}
	prevent = true
	if err := tt.Click("Covered caption"); err != nil {
		t.Fatal(err)
	}
	if under != 1 || blocked != 0 {
		t.Fatal("pointer policy bypassed capture default prevention")
	}
	prevent, coverEnabled = false, true
	tt.Frame()
	if err := tt.Click("Covered caption"); err != nil {
		t.Fatal(err)
	}
	if blocked != 1 || under != 1 {
		t.Fatalf("updated inherited policy did not commit: under=%d blocked=%d", under, blocked)
	}
}
