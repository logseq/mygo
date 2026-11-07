package ui

import (
	"fmt"
	"slices"
	"testing"
)

func requestedListWindow(t *testing.T, s *ListState) (int, int, []int) {
	t.Helper()
	api, ok := any(s).(interface{ Requested() (int, int, []int) })
	if !ok {
		t.Fatal("ListState does not expose its committed contiguous window and active row pins")
	}
	return api.Requested()
}

func TestListRequestedWindowUsesLogicalExtentAndOverscan(t *testing.T) {
	var s ListState
	s.SetItemExtent(20)
	s.SetOverscan(2)
	tt := coreNewTester(func(c *context) {
		coreList(c, &s, 100000, func(i int) { coreBox(c).Height(20).Children(func() { coreTextf(c, "Row %d", i) }) }).Height(100)
	}, 300, 100)
	first, last, pins := requestedListWindow(t, &s)
	if first != 0 || last != 7 || len(pins) != 0 {
		t.Fatalf("initial half-open overscan window: %d..%d pins%v", first, last, pins)
	}
	s.ScrollTo(99999, End)
	tt.Frame()
	first, last, pins = requestedListWindow(t, &s)
	if first != 99993 || last != 100000 || len(pins) != 0 {
		t.Fatalf("bottom jump changed logical extent: %d..%d pins%v", first, last, pins)
	}
	if len(s.rows) > 16 {
		t.Fatalf("built %d rows for distant jump", len(s.rows))
	}
}

func TestListRequestedWindowPinsFocusedCompositionAndReleasesRemovedRows(t *testing.T) {
	var s ListState
	texts := make([]string, 1000)
	count := 1000
	s.SetItemExtent(30)
	s.SetOverscan(1)
	tt := coreNewTester(func(c *context) {
		coreList(c, &s, count, func(i int) {
			coreRow(c).Height(30).Children(func() { coreTextInput(c, &texts[i]).Label(fmt.Sprintf("Field %d", i)) })
		}).Height(150)
	}, 300, 150)
	if err := tt.Click("Field 2"); err != nil {
		t.Fatal(err)
	}
	tt.Compose("你", 1)
	focus := tt.rt.focused
	s.ScrollToEnd()
	tt.Frame()
	first, last, pins := requestedListWindow(t, &s)
	if first < 990 || last != 1000 || !slices.Equal(pins, []int{2}) {
		t.Fatalf("composition owner incorrectly expands window: %d..%d pins%v", first, last, pins)
	}
	if tt.rt.focused != focus {
		t.Fatal("composition lost focus")
	}
	tt.Type("你")
	if texts[2] != "你" {
		t.Fatalf("composition committed to wrong row: %q", texts[2])
	}
	count = 0
	tt.Frame()
	first, last, pins = requestedListWindow(t, &s)
	if first != 0 || last != 0 || len(pins) != 0 {
		t.Fatalf("removed source retains requested rows: %d..%d %v", first, last, pins)
	}
}

func TestListRequestedWindowPinsPointerHoldAndDrag(t *testing.T) {
	for _, drag := range []bool{false, true} {
		t.Run(fmt.Sprint(drag), func(t *testing.T) {
			var s ListState
			s.SetItemExtent(30)
			s.SetOverscan(1)
			tt := coreNewTester(func(c *context) {
				coreList(c, &s, 1000, func(i int) {
					e := coreBox(c).Height(30).Label(fmt.Sprintf("Hold %d", i))
					e.flags |= flagClickable
					if drag {
						e.Drag(i)
					}
				}).Height(150)
			}, 300, 150)
			r, ok := tt.Find("Hold 2")
			if !ok {
				t.Fatal("missing press target")
			}
			tt.Press(r.X+10, r.Y+10)
			if drag {
				tt.Move(r.X+30, r.Y+10)
				if tt.rt.drag == nil {
					t.Fatal("drag did not start")
				}
			}
			pressed := tt.rt.pressed
			s.ScrollToEnd()
			tt.Frame()
			if !drag && (pressed == nil || tt.rt.pressed != pressed) {
				t.Fatal("offscreen pointer owner was disposed before release")
			}
			first, last, pins := requestedListWindow(t, &s)
			if first < 990 || last != 1000 || !slices.Equal(pins, []int{2}) {
				t.Fatalf("hold owner incorrectly expands window: %d..%d pins%v", first, last, pins)
			}
			tt.Release(20, 20)
			tt.Frame()
			_, _, pins = requestedListWindow(t, &s)
			if len(pins) != 0 {
				t.Fatalf("released interaction retains rows: %v", pins)
			}
		})
	}
}

func TestListRequestedWindowPinsPopoverOwner(t *testing.T) {
	var s ListState
	s.SetItemExtent(30)
	s.SetOverscan(1)
	open := false
	tt := coreNewTester(func(c *context) {
		coreList(c, &s, 1000, func(i int) {
			b := coreBox(c).Height(30).Label(fmt.Sprintf("Owner %d", i))
			b.flags |= flagClickable
			if b.Clicked() && i == 2 {
				open = true
			}
			if i == 2 {
				corePopoverBase(c, b, &open, func(panel *node) { panel.Size(100, 40); coreText(c, "Owned popup") })
			}
		}).Height(150)
	}, 300, 150)
	if err := tt.Click("Owner 2"); err != nil {
		t.Fatal(err)
	}
	if !tt.HasText("Owned popup") {
		t.Fatal("popup did not open")
	}
	s.ScrollToEnd()
	tt.Frame()
	if !tt.HasText("Owned popup") {
		t.Fatal("scroll disposed live popup owner")
	}
	first, last, pins := requestedListWindow(t, &s)
	if first < 990 || last != 1000 || !slices.Equal(pins, []int{2}) {
		t.Fatalf("popup owner incorrectly expands window: %d..%d pins%v", first, last, pins)
	}
	open = false
	tt.Frame()
	_, _, pins = requestedListWindow(t, &s)
	if len(pins) != 0 {
		t.Fatalf("closed popup retains owner: %v", pins)
	}
}

func TestListRequestedWindowPinsNativeMenuOwnerUntilChoice(t *testing.T) {
	var s ListState
	s.SetItemExtent(30)
	s.SetOverscan(1)
	chosen := false
	tt := coreNewTester(func(c *context) {
		coreList(c, &s, 1000, func(i int) {
			coreBox(c).Height(30).Label(fmt.Sprintf("Menu owner %d", i)).ContextMenu(func(m *Menu) {
				if m.Item("Choose").Chosen() {
					chosen = true
				}
			})
		}).Height(150)
	}, 300, 150)
	if err := tt.RightClick("Menu owner 2"); err != nil {
		t.Fatal(err)
	}
	if len(tt.Menu()) == 0 {
		t.Fatal("menu did not open")
	}
	s.ScrollToEnd()
	tt.Frame()
	first, last, pins := requestedListWindow(t, &s)
	if first < 990 || last != 1000 || !slices.Equal(pins, []int{2}) {
		t.Fatalf("native menu owner incorrectly expands window: %d..%d pins%v", first, last, pins)
	}
	if err := tt.ChooseMenuItem("Choose"); err != nil {
		t.Fatal(err)
	}
	if !chosen {
		t.Fatal("menu choice lost after owner scrolled offscreen")
	}
	tt.Frame()
	_, _, pins = requestedListWindow(t, &s)
	if len(pins) != 0 {
		t.Fatalf("chosen menu retains owner: %v", pins)
	}
}

func TestListRequestedWindowFollowsPinnedKeyAcrossDistantReorder(t *testing.T) {
	var s ListState
	s.SetItemExtent(30)
	s.SetOverscan(1)
	moved := false
	keyAt := func(i int) int {
		if moved {
			if i == 2 {
				return 99999
			}
			if i == 99999 {
				return 2
			}
		}
		return i
	}
	s.Key = func(i int) any { return keyAt(i) }
	if lookup, ok := any(&s).(interface{ SetKeyIndex(func(any) (int, bool)) }); ok {
		lookup.SetKeyIndex(func(key any) (int, bool) {
			i, ok := key.(int)
			if !ok {
				return 0, false
			}
			return keyAt(i), true
		})
	}
	text := ""
	tt := coreNewTester(func(c *context) {
		coreList(c, &s, 100000, func(i int) {
			coreBox(c).Height(30).Children(func() {
				if keyAt(i) == 2 {
					coreTextInput(c, &text).Label("moving editor")
				} else {
					coreTextf(c, "Row %d", keyAt(i))
				}
			})
		}).Height(150)
	}, 300, 150)
	if err := tt.Click("moving editor"); err != nil {
		t.Fatal(err)
	}
	moved = true
	tt.Frame()
	if !tt.Focused("moving editor") {
		t.Fatal("distant source reorder retired the pinned editor key")
	}
	_, _, pins := requestedListWindow(t, &s)
	if !slices.Equal(pins, []int{99999}) {
		t.Fatalf("pin followed old numeric index instead of key: %v", pins)
	}
	tt.Type("kept")
	if text != "kept" {
		t.Fatal("reordered editor lost native input state")
	}
}

// Native focus transfer ends preedit through the existing text-input client
// boundary. A finished composition must not pin an otherwise unused row.
func TestListRequestedWindowCancelsNativeCompositionBeforeBlurEviction(t *testing.T) {
	var s ListState
	s.SetItemExtent(30)
	s.SetOverscan(1)
	text := ""
	tt := coreNewTester(func(c *context) {
		coreColumn(c).Children(func() {
			coreList(c, &s, 1000, func(i int) {
				coreBox(c).Height(30).Children(func() {
					if i == 2 {
						coreTextInput(c, &text).Label("composing owner")
					} else {
						coreTextf(c, "Row %d", i)
					}
				})
			}).Height(150)
			coreButton(c, "Other focus")
		})
	}, 300, 220)
	if err := tt.Click("composing owner"); err != nil {
		t.Fatal(err)
	}
	client := tt.rt.states[tt.rt.focused].editor.client
	client.SetMarkedText(nil, "你", TextInputRange{Start: 1, End: 1})
	tt.Frame()
	if _, marked := client.MarkedRange(); !marked {
		t.Fatal("native preedit did not begin")
	}
	if err := tt.Click("Other focus"); err != nil {
		t.Fatal(err)
	}
	if _, marked := client.MarkedRange(); marked {
		t.Fatal("focus transfer left native preedit unresolved")
	}
	if text != "" {
		t.Fatalf("blur saved unfinished native composition instead of preserving original text: %q", text)
	}
	s.ScrollToEnd()
	tt.Frame()
	_, _, pins := requestedListWindow(t, &s)
	if len(pins) != 0 {
		t.Fatalf("finished composition retained unused row: %v", pins)
	}
	if _, ok := tt.Find("composing owner"); ok {
		t.Fatal("blurred row stayed materialized")
	}
}
