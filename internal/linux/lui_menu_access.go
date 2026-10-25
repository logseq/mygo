//go:build linux && (amd64 || arm64)

package linux

import "sync"

var menuAccessOnce sync.Once
var menuWidgetAccessible func(ptr) ptr
var menuAccessibleName func(ptr) ptr
var menuSetAccessibleName func(ptr, *byte)
var menuImageGetType func() uintptr
var menuBinGetChild func(ptr) ptr
var menuContainerGetType func() uintptr
var menuLabelSetText func(ptr, *byte)

func loadMenuAccess() {
	menuAccessOnce.Do(func() {
		atk, err := open("libatk-1.0.so.0")
		if err != nil {
			panic(err)
		}
		mustBind(libGTK, &menuWidgetAccessible, "gtk_widget_get_accessible")
		mustBind(atk, &menuAccessibleName, "atk_object_get_name")
		mustBind(atk, &menuSetAccessibleName, "atk_object_set_name")
		mustBind(libGTK, &menuImageGetType, "gtk_image_get_type")
		mustBind(libGTK, &menuBinGetChild, "gtk_bin_get_child")
		mustBind(libGTK, &menuContainerGetType, "gtk_container_get_type")
		mustBind(libGTK, &menuLabelSetText, "gtk_label_set_text")
	})
}

// Explicit ATK names remain stable when a GtkMenuItem contains an image row
// rather than GtkLabel, which GTK's default accessible-name lookup requires.
func nameMenuItem(item ptr, label string) {
	loadMenuAccess()
	if obj := menuWidgetAccessible(item); obj != 0 {
		menuSetAccessibleName(obj, cs(label))
	}
}
func menuItemText(item ptr) string {
	if text := goStr(gtkMenuItemGetLabel(item)); text != "" {
		return text
	}
	loadMenuAccess()
	return goStr(menuAccessibleName(menuWidgetAccessible(item)))
}
