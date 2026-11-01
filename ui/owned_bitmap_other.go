//go:build !windows && !darwin && !(linux && (amd64 || arm64)) && !js

package ui

func ownedBitmapRetirement(h *windowHost, renderer any, id uint64) func() error {
	return ownedBitmapFallbackRetirement(renderer, id)
}
