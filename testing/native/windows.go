//go:build windows && lui_native_smoke

package native

import "github.com/egoist/mygo/internal/windows"

type AccessNode struct {
	Role, Label, Value string
	Focused            bool
}

func Accessibility(handle uintptr) []AccessNode {
	nodes, _ := windows.TestAccessibility(handle)
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
		out = append(out, AccessNode{Role: role, Label: n.Label, Value: n.Value, Focused: windows.TestAccessibilityFocused(n)})
	}
	return out
}
func Compose(handle uintptr, text string, caret int, commit bool) bool {
	return windows.TestComposeOver(handle, text, caret, commit, -1, 0)
}
func WindowNumber(handle uintptr) int { return int(handle) }

var Perform = windows.TestAccessibilityPerform
var RightClick = windows.TestRightClickSurface
var Popups = windows.TestPopups
var ChoosePopup = windows.TestChoosePopupItem
var PopupImages = windows.TestPopupImageLabels
var OnScreen = windows.TestForeground

func InputClient(handle uintptr) ([2]int, string) {
	text, start, length, _ := windows.TestDocumentFeed(handle)
	return [2]int{start, length}, text
}
