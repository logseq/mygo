package ui

import (
	"runtime"
	"testing"
	"weak"
)

func TestLUIEphemeralHandleBindingsFollowTheirOwners(t *testing.T) {
	shown := true
	var bindings []weak.Pointer[refState]
	tt := NewTester(func(c *Context) {
		if !shown {
			Text(c, "Cleared")
			return
		}
		for i := range 16 {
			var handle Handle
			Button(c.Key(i), "Row").Bind(&handle)
			bindings = append(bindings, weak.Make(handle.state.bindings[0]))
		}
	}, 160, 100)
	defer tt.rt.close()
	for range 15 {
		tt.Frame()
	}
	shown = false
	tt.Frame()
	runtime.GC()
	tt.Frame()
	retained := 0
	for _, binding := range bindings {
		if binding.Value() != nil {
			retained++
		}
	}
	if retained != 0 {
		t.Errorf("clearing ephemeral controls retained %d of %d abandoned handle bindings", retained, len(bindings))
	}
	if len(tt.rt.refs) != 0 {
		t.Errorf("focus processing still scans %d abandoned handle bindings", len(tt.rt.refs))
	}
	runtime.KeepAlive(tt)
}

func TestLUIHiddenLiveHandleKeepsItsPendingFocus(t *testing.T) {
	var handle Handle
	shown := true
	tt := NewTester(func(c *Context) {
		Column(c).Children(func() {
			if shown {
				Button(c.Key("target"), "Target").Bind(&handle)
			}
			Button(c.Key("other"), "Other")
		})
	}, 160, 100)
	defer tt.rt.close()
	shown = false
	tt.Frame()
	runtime.GC()
	handle.Focus()
	runtime.GC()
	tt.Frame()
	if tt.Focused("Target") || !handle.state.bindings[0].requested {
		t.Fatal("hidden control lost its pending focus request or took focus")
	}
	shown = true
	tt.Frame()
	if !tt.Focused("Target") || handle.state.bindings[0].requested {
		t.Fatal("returning live handle did not receive pending focus")
	}
	shown = false
	tt.Frame()
	if err := tt.Click("Other"); err != nil {
		t.Fatal(err)
	}
	handle.Focus()
	handle.CancelFocus()
	runtime.GC()
	shown = true
	tt.Frame()
	if tt.Focused("Target") {
		t.Fatal("cancelled hidden focus request was delivered")
	}
	runtime.KeepAlive(handle)
}

func TestLUITransientHandleFocusSurvivesCollectionDuringBuild(t *testing.T) {
	tt := NewTester(func(c *Context) {
		var handle Handle
		Button(c.Key("target"), "Target").Bind(&handle)
		handle.Focus()
		runtime.GC()
	}, 160, 100)
	defer tt.rt.close()
	if !tt.Focused("Target") {
		t.Fatal("a focus request made during construction was collected before delivery")
	}
}

func TestLUILiveHandleSeparatesWindowBindingsAcrossCollectionAndClose(t *testing.T) {
	var handle Handle
	var firstContext, secondContext *Context
	first := NewTester(func(c *Context) {
		firstContext = c
		Button(c, "First").Bind(&handle)
	}, 160, 100)
	second := NewTester(func(c *Context) {
		secondContext = c
		Button(c, "Second").Bind(&handle)
	}, 160, 100)
	defer second.rt.close()
	runtime.GC()
	handle.Focus(firstContext)
	first.Frame()
	if !handle.Focused(firstContext) || handle.Focused(secondContext) {
		t.Fatal("focus crossed window bindings after collection")
	}
	firstOwner := weak.Make(first.rt)
	first.rt.close()
	first = nil
	runtime.GC()
	if firstOwner.Value() != nil {
		t.Fatal("a live handle retained its closed window")
	}
	if !handle.state.bindings[0].closed || handle.state.bindings[0].requested {
		t.Fatal("closing a window did not retire its live handle binding")
	}
	handle.Focus()
	second.Frame()
	if !handle.Focused(secondContext) || handle.Focused(firstContext) {
		t.Fatal("closing one window damaged the remaining binding")
	}
	runtime.KeepAlive(handle)
}
