//go:build linux && (amd64 || arm64)

package ui

import "github.com/egoist/mygo/internal/linux"

func linkMenuIcons() { linux.LinkMenuIcons() }
