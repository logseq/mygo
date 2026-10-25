package ui

import "testing"

type measurementContext interface {
	Measure(func(*Context), float32) Rect
	Fitting(bool, []func(*Context), func(*Context, int)) Element
	AfterFrame(func())
}

func TestLUIFittingMeasuresFinalLocalRoomWithoutActivatingProbes(t *testing.T) {
	measured, selected, committed := 0, -1, 0
	var failure string
	tester := NewTester(func(c *Context) {
		api, ok := any(c).(measurementContext)
		if !ok {
			failure = "missing isolated measure/fitting/after-frame API"
			return
		}
		Column(c).Width(100).Children(func() {
			fit := api.Fitting(false, []func(*Context){
				func(p *Context) { measured++; TextInput(p, new(string)).Width(300).Focus() },
				func(p *Context) { measured++; Text(p, "compact").Width(80) },
			}, func(p *Context, index int) { selected = index; Text(p, "chosen").Width(80) })
			fit.Width(100).Height(40)
		})
		api.AfterFrame(func() { committed++ })
	}, 600, 200)
	if failure != "" {
		t.Fatal(failure)
	}
	if measured == 0 || selected != 1 || !tester.HasText("chosen") || tester.HasText("compact") {
		t.Fatalf("fitting: measured=%d selected=%d visible=%v", measured, selected, tester.Texts())
	}
	if committed != 1 {
		t.Fatalf("after-frame ran %d times", committed)
	}
}

func TestLUIMeasureUsesActualTypographyAndDoesNotCommitFocus(t *testing.T) {
	var wide, narrow Rect
	var failure string
	tester := NewTester(func(c *Context) {
		api, ok := any(c).(measurementContext)
		if !ok {
			failure = "missing measure API"
			return
		}
		Column(c).FontSize(28).Children(func() {
			wide = api.Measure(func(p *Context) { Text(p, "MMMM") }, -1)
			narrow = api.Measure(func(p *Context) { Text(p, "MMMM").FontSize(14) }, -1)
			api.Measure(func(p *Context) { TextInput(p, new(string)).Label("probe").Focus() }, 100)
			TextInput(c, new(string)).Label("actual").Focus()
		})
	}, 300, 200)
	if failure != "" {
		t.Fatal(failure)
	}
	if wide.W < narrow.W*1.8 || wide.H < narrow.H*1.8 {
		t.Fatalf("typography not inherited: wide=%+v narrow=%+v", wide, narrow)
	}
	if !tester.Focused("actual") || tester.HasText("probe") {
		t.Fatal("measurement committed probe focus/content")
	}
}

func TestLUIListTuningControlsActualRowsBuilt(t *testing.T) {
	for _, overscan := range []int{0, 8} {
		var state ListState
		tuning, ok := any(&state).(interface {
			SetOverscan(int)
			SetItemExtent(float32)
		})
		if !ok {
			t.Fatal("ListState lacks viewport tuning")
		}
		tuning.SetOverscan(overscan)
		tuning.SetItemExtent(20)
		built := map[int]bool{}
		NewTester(func(c *Context) {
			List(c, &state, 10000, func(row int) { built[row] = true; Text(c, "row").Height(20) }).Width(200).Height(100)
		}, 300, 200)
		if len(built) != 5+overscan {
			t.Fatalf("overscan %d built %d rows", overscan, len(built))
		}
	}
}
