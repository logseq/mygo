//go:build linux && lui_native_smoke

package native

import "github.com/egoist/mygo/internal/linux"

type AccessNode struct {
	Role, Label, Value string
	Focused            bool
}

func Accessibility(handle uintptr) []AccessNode {
	nodes, _ := linux.TestAccessibility(handle)
	out := make([]AccessNode, 0, len(nodes))
	for _, n := range nodes {
		role := n.Role
		switch role {
		case "entry", "Edit":
			role = "textbox"
		case "Button":
			role = "button"
		case "check box", "CheckBox":
			role = "checkbox"
		}
		out = append(out, AccessNode{Role: role, Label: n.Label, Value: n.Value, Focused: linux.TestAccessibilityFocused(n)})
	}
	return out
}
func Compose(handle uintptr, text string, caret int, commit bool) bool {
	return linux.TestComposeOver(handle, text, caret, commit, -1, 0)
}
func WindowNumber(handle uintptr) int { return int(handle) }

var Perform = linux.TestAccessibilityPerform
var RightClick = linux.TestRightClickSurface
var Popups = linux.TestPopups
var ChoosePopup = linux.TestChoosePopupItem
var PopupImages = linux.TestPopupImageLabels

func OnScreen(handle uintptr) bool { return len(linux.TestSurfaceOnScreen(handle)) > 0 }
func InputClient(handle uintptr) ([2]int, string) {
	text, caret, _ := linux.TestSurrounding(handle)
	return [2]int{caret, 0}, text
}
