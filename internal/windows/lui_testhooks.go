//go:build windows && (amd64 || arm64)

package windows

import "unsafe"

// TestAccessibilityFocused reads the actual UIA provider focus property.
func TestAccessibilityFocused(n TestAccessNode) bool {
	return uiaProperty(n.simple, uiaHasKeyboardFocusProperty).Val != 0
}

// TestPopupImageLabels checks that real HMENU item data refers to a decoded
// native owner-draw image, retaining the OS item's own visible label.
func TestPopupImageLabels() []string {
	getInfo := user32.NewProc("GetMenuItemInfoW")
	var labels []string
	for _, popup := range threadPopups() {
		menu, _, _ := procSendMessageW.Call(popup, mnGetHMenu, 0, 0)
		if menu == 0 {
			continue
		}
		count, _, _ := procGetMenuItemCount.Call(menu)
		for index := range int(int32(count)) {
			info := menuItemInfo{Mask: miimData}
			info.Size = uint32(unsafe.Sizeof(info))
			ok, _, _ := getInfo.Call(menu, uintptr(index), 1, uintptr(unsafe.Pointer(&info)))
			if ok == 0 {
				continue
			}
			if look := theBackend.menus.looks[info.ItemData]; look != nil && look.icon != nil {
				labels = append(labels, look.label)
			}
		}
	}
	return labels
}
