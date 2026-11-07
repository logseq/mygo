//go:build windows && (amd64 || arm64)

package ui

import "github.com/egoist/mygo/internal/windows"

func linkMenuIcons() { windows.LinkMenuIcons() }
