//go:build windows && (amd64 || arm64)

package windows

import "unsafe"

// TestAccessibilityFocused reads the actual UIA provider focus property.
func TestAccessibilityFocused(n TestAccessNode) bool {
	return uiaProperty(n.simple, uiaHasKeyboardFocusProperty).Val != 0
}

// TestPopupImageLabels checks real HMENU owner-draw images and bitmaps,
// retaining the OS item's own visible label.
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
			info := menuItemInfo{Mask: miimData | miimBitmap}
			info.Size = uint32(unsafe.Sizeof(info))
			ok, _, _ := getInfo.Call(menu, uintptr(index), 1, uintptr(unsafe.Pointer(&info)))
			if ok == 0 {
				continue
			}
			if look := theBackend.menus.looks[info.ItemData]; look != nil && look.icons && look.icon != nil && drawMenuIcon != nil {
				if iconAt(look.icon, toPx(16, dpiOf(popup)), 0) != nil {
					labels = append(labels, look.label)
				}
			} else if info.BmpItem != 0 {
				var ds dibSection
				if n, _, _ := procGetObjectW.Call(info.BmpItem, unsafe.Sizeof(ds), uintptr(unsafe.Pointer(&ds))); n != 0 && ds.bmWidth > 0 && ds.bmHeight != 0 {
					var label [512]uint16
					procGetMenuStringW.Call(menu, uintptr(index), uintptr(unsafe.Pointer(&label[0])), uintptr(len(label)), mfByPosition)
					labels = append(labels, wstr(uintptr(unsafe.Pointer(&label[0]))))
				}
			}
		}
	}
	return labels
}
