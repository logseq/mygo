package ui

import (
	"reflect"
	"testing"
)

func TestLUIZIndexPreservesFlowAndOrdersPaintAndHitTesting(t *testing.T) {
	red, blue := 0, 0
	var failure string
	tester := NewTester(func(c *Context) {
		Grid(c).ColumnTracks(Fixed(80)).RowTracks(Fixed(40)).Children(func() {
			e := Box(c).ColumnStart(1).RowStart(1).Background(RGB(255, 0, 0)).OnClick(func() { red++ })
			index, ok := any(e).(interface{ ZIndex(int) Element })
			if !ok {
				failure = "missing paint-only z-index"
				return
			}
			index.ZIndex(10)
			Box(c).ColumnStart(1).RowStart(1).Background(RGB(0, 0, 255)).OnClick(func() { blue++ })
		})
	}, 200, 100)
	if failure != "" {
		t.Fatal(failure)
	}
	pixel := tester.Image().RGBAAt(40, 20)
	if pixel.R < 250 || pixel.B > 5 {
		t.Fatalf("z-order did not affect painting: %#v", pixel)
	}
	tester.ClickAt(40, 20)
	if red != 1 || blue != 0 {
		t.Fatalf("paint and hit order diverged: red=%d blue=%d", red, blue)
	}
}

func TestLUITesterExposesCommittedAccessibilitySnapshot(t *testing.T) {
	tester := NewTester(func(c *Context) { Text(c, "Heading").Role(RoleHeading).Level(3) }, 200, 100)
	method := reflect.ValueOf(tester).MethodByName("Accessibility")
	if !method.IsValid() {
		t.Fatal("missing accessibility snapshot")
	}
	if method.Call(nil)[0].Len() == 0 {
		t.Fatal("committed accessibility snapshot is empty")
	}
}

func TestLUINegativePercentageOffsetsRemainSigned(t *testing.T) {
	var relative, absolute Handle
	var relativeX, absoluteX float32
	NewTester(func(c *Context) {
		Box(c).Width(200).Height(100).Children(func() {
			Box(c).Width(20).Height(20).StartPercent(-25).Bind(&relative)
			Box(c).Absolute().Width(20).Height(20).LeftPercent(-10).Bind(&absolute)
		})
		c.AfterFrame(func() {
			relativeX, absoluteX = relative.Bounds(c).X, absolute.Bounds(c).X
		})
	}, 300, 200)
	if relativeX != -50 || absoluteX != -20 {
		t.Fatalf("negative offsets clamped: relative=%v, absolute=%v", relativeX, absoluteX)
	}
}

func TestLUIExplicitLineHeightMayBeSmallerThanFontMetrics(t *testing.T) {
	for _, fixed := range []bool{true, false} {
		t.Run(map[bool]string{true: "logical-pixels", false: "font-multiplier"}[fixed], func(t *testing.T) {
			tester := NewTester(func(c *Context) {
				e := Text(c, "first\nsecond").FontSize(40).Label("two lines")
				if fixed {
					e.FixedLineHeight(10)
				} else {
					e.LineHeight(.25)
				}
			}, 300, 200)
			r, ok := tester.Find("two lines")
			if !ok || r.H != 20 {
				t.Fatalf("explicit line boxes must preserve requested height despite larger font ink: found=%v rect=%v", ok, r)
			}
		})
	}
}
