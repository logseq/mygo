package ui

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/internal/platform"
)

func TestCapturedPointerSurvivesOtherButtons(t *testing.T) {
	for _, button := range []int{1, 2} {
		for _, intercept := range []bool{false, true} {
			for _, cancel := range []bool{false, true} {
				t.Run(fmt.Sprintf("button=%d/intercept=%v/cancel=%v", button, intercept, cancel), func(t *testing.T) {
					moves, releases, cancels, downs, ups := 0, 0, 0, 0, 0
					tt := NewTester(func(c *Context) {
						c.CaptureInput(func(ev InputEvent, _ InputTarget) bool {
							return intercept && ev.Button == button && (ev.Kind == InputPointerDown || ev.Kind == InputPointerUp)
						})
						Box(c).Size(20, 30).HandleCapturedPointer(func(ev InputEvent) bool {
							switch ev.Kind {
							case InputPointerDown:
								if ev.Button == button {
									downs++
								}
							case InputPointerUp:
								if ev.Button == button {
									ups++
								} else if ev.Button == 0 {
									releases++
								}
							case InputPointerMove:
								moves++
							case InputPointerCancel:
								cancels++
							}
							return true
						})
					}, 200, 60)
					tt.Press(10, 15)
					tt.send(platform.SurfaceEvent{Kind: platform.PointerDown, X: 10, Y: 15, Button: button})
					tt.send(platform.SurfaceEvent{Kind: platform.PointerUp, X: 10, Y: 15, Button: button})
					moves = 0
					tt.Move(180, 15)
					if moves != 1 {
						t.Fatalf("primary movement after other-button release = %d; want 1", moves)
					}
					if cancel {
						tt.Key(0, KeyEscape)
					}
					tt.Release(180, 15)
					wantOther, wantRelease, wantCancel := 1, 1, 0
					if intercept {
						wantOther = 0
					}
					if cancel {
						wantRelease, wantCancel = 0, 1
					}
					if downs != wantOther || ups != wantOther || releases != wantRelease || cancels != wantCancel {
						t.Fatalf("other down/up, primary release/cancel = %d/%d, %d/%d; want %d/%d, %d/%d", downs, ups, releases, cancels, wantOther, wantOther, wantRelease, wantCancel)
					}
				})
			}
		}
	}
}

func TestCapturedPointerLifetime(t *testing.T) {
	for _, end := range []string{"release", "escape", "blur", "cancel", "remove", "disable", "close"} {
		t.Run(end, func(t *testing.T) {
			present, disabled := true, false
			moves, releases, cancels, clicks := 0, 0, 0, 0
			tt := NewTester(func(c *Context) {
				if !present {
					return
				}
				Box(c).Key("handle").Size(20, 30).Disabled(disabled).
					OnClick(func() { clicks++ }).HandleCapturedPointer(func(ev InputEvent) bool {
					switch ev.Kind {
					case InputPointerDown:
						return ev.Button == 0
					case InputPointerMove:
						moves++
					case InputPointerUp:
						releases++
					case InputPointerCancel:
						cancels++
					}
					return true
				})
			}, 200, 60)
			tt.Press(10, 15)
			moves = 0
			tt.Move(180, 15)
			if moves != 1 {
				t.Fatalf("owned movement outside handle = %d, want 1", moves)
			}
			switch end {
			case "escape":
				tt.Key(0, KeyEscape)
			case "blur":
				tt.SetFocused(false)
			case "cancel":
				tt.send(platform.SurfaceEvent{Kind: platform.PointerCancel})
			case "remove":
				present = false
				tt.Frame()
			case "disable":
				disabled = true
				tt.Frame()
			case "close":
				tt.rt.close()
			}
			if end != "close" {
				tt.Release(180, 15)
			}
			wantRelease, wantCancel := 0, 1
			if end == "release" {
				wantRelease, wantCancel = 1, 0
			}
			if releases != wantRelease || cancels != wantCancel || clicks != 0 {
				t.Fatalf("release/cancel/click = %d/%d/%d, want %d/%d/0",
					releases, cancels, clicks, wantRelease, wantCancel)
			}
		})
	}
}
