//go:build linux && (amd64 || arm64)

package linux

import (
	"bytes"
	"sync"
)

var menuAccessOnce sync.Once
var menuWidgetAccessible func(ptr) ptr
var menuAccessibleName func(ptr) ptr
var menuSetAccessibleName func(ptr, *byte)
var menuImageGetType func() uintptr
var menuBinGetChild func(ptr) ptr
var menuContainerGetType func() uintptr
var menuLabelSetText func(ptr, *byte)
var menuImageSetFromPixbuf func(ptr, ptr)

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
		mustBind(libGTK, &menuImageSetFromPixbuf, "gtk_image_set_from_pixbuf")
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

func setLegacyMenuItemImage(n *nativeItem, data []byte) {
	n.imagePNG = bytes.Clone(data)
	if len(data) == 0 && n.image == 0 {
		return
	}
	loadMenuAccess()
	var pix ptr
	if len(data) != 0 {
		pix, _ = pixbufFromPNG(data)
	}
	if pix != 0 {
		defer gObjectUnref(pix)
	}
	if n.image == 0 {
		if pix == 0 {
			return
		}
		label := menuBinGetChild(n.item)
		if label == 0 {
			return
		}
		gObjectRef(label)
		gtkContainerRemove(n.item, label)
		row := gtkBoxNew(0, 6)
		n.image = gtkImageNewFromPixbuf(0)
		gtkWidgetSetNoShowAll(n.image, true)
		gtkBoxPackStart(row, n.image, false, false, 0)
		gtkBoxPackStart(row, label, true, true, 0)
		gObjectUnref(label)
		gtkContainerAdd(n.item, row)
		gtkWidgetShow(row)
		gtkWidgetShow(label)
		n.label = label
	}
	menuImageSetFromPixbuf(n.image, pix)
	gtkWidgetSetVisible(n.image, pix != 0)
}
