//go:build linux && (amd64 || arm64)

package linux

// TestAccessibilityFocused reads the real ATK focused state of a snapshot node.
func TestAccessibilityFocused(n TestAccessNode) bool {
	accessTestsOnce.Do(loadAccessTests)
	states := atkObjectRefStateSet(n.obj)
	if states == 0 {
		return false
	}
	defer gObjectUnref(states)
	return atkStateSetContainsState(states, atkStates.focused)
}

// TestPopupImageLabels reads actual GTK image children and their ATK names.
func TestPopupImageLabels() []string {
	loadMenuAccess()
	var names []string
	for _, menu := range theBackend.popups {
		items := gtkContainerGetChildren(menu)
		for at := items; at != 0; at = field[ptr](at, 8) {
			item := field[ptr](at, 0)
			row := menuBinGetChild(item)
			if row == 0 {
				continue
			}
			// Rows added for image menu items are containers; GtkLabel is not.
			if !gTypeCheckInstanceIsA(row, menuContainerGetType()) {
				continue
			}
			children := gtkContainerGetChildren(row)
			for child := children; child != 0; child = field[ptr](child, 8) {
				if gTypeCheckInstanceIsA(field[ptr](child, 0), menuImageGetType()) {
					if label := goStr(menuAccessibleName(menuWidgetAccessible(item))); label != "" {
						names = append(names, label)
					}
					break
				}
			}
			gListFree(children)
		}
		gListFree(items)
	}
	return names
}
