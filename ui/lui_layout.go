package ui

import "math"

// AfterFrame runs fn once after this frame's final layout and presentation.
// Callbacks from discarded build passes are discarded too. The callback may
// update retained application state; Element values have expired at this point.
func (c *Context) AfterFrame(fn func()) {
	if raw := c.build(); raw != nil && fn != nil {
		raw.rt.afterFrame = append(raw.rt.afterFrame, fn)
	}
}

// Measure builds an isolated subtree and returns its ideal border-box size.
// A negative maxWidth asks for its unconstrained ideal width; otherwise text
// and wrapping use maxWidth. The real window receives no input, focus,
// accessibility, lifecycle or painting effects from the isolated context.
// Application builders must remain declarative: native services and external
// model mutations belong outside measurement callbacks.
func (c *Context) Measure(view func(*Context), maxWidth float32) Rect {
	raw := c.build()
	if raw == nil || view == nil {
		return Rect{}
	}
	width := maxWidth
	if width < 0 || math.IsInf(float64(width), 0) {
		width = raw.w
	}
	host := &headless{w: width, h: raw.h, scale: 1, lang: raw.rt.lang, rtl: raw.rt.direction(), prefs: raw.rt.host.preferences()}
	rt := newRuntime(nil, host)
	rt.inFrame = true
	rt.frame = 1
	rt.epoch = 1
	rt.dark, rt.darkKnown = raw.theme.Dark, true
	rt.prefsKnown, rt.themeOK = true, true
	rt.theme = *raw.theme
	rt.c.reset(raw.now, width, raw.h)
	rt.c.theme = raw.theme
	rt.c.root.ts = raw.parent.resolvedText()
	view(rt.public)
	root := rt.c.root
	root.width, root.height = length{}, length{}
	measuredWidth := fitWidth(root, inf, inf)
	if maxWidth >= 0 {
		measuredWidth = fitWidth(root, maxWidth, maxWidth)
	}
	measuredHeight := heightAt(root, measuredWidth, inf)
	rt.inFrame = false
	rt.closed = true
	return Rect{W: measuredWidth, H: measuredHeight}
}

// Fitting builds only the first candidate whose ideal size fits the local
// proposed axis; if none fits it builds the last. Probes run through Measure
// and build receives the chosen index during layout, with the current Context.
func (c *Context) Fitting(vertical bool, candidates []func(*Context), build func(*Context, int)) Element {
	raw := c.build()
	if raw == nil {
		return Element{}
	}
	e := coreBox(raw).MinWidth(0)
	e.fitting = &fittingBuilder{c: c, e: e, vertical: vertical, candidates: candidates, chosen: build, index: -1, theme: *raw.theme}
	return wrapElement(e)
}

type fittingBuilder struct {
	c          *Context
	e          *node
	vertical   bool
	candidates []func(*Context)
	chosen     func(*Context, int)
	index      int
	built      bool
	theme      Theme
}

func (f *fittingBuilder) sizes(width float32) []Rect {
	raw := f.c.build()
	parent := raw.parent
	raw.parent = f.e
	defer func() { raw.parent = parent }()
	sizes := make([]Rect, len(f.candidates))
	for i, candidate := range f.candidates {
		sizes[i] = f.c.Measure(candidate, width)
	}
	return sizes
}
func (f *fittingBuilder) ideal(width float32) Rect {
	if len(f.candidates) == 0 {
		return Rect{}
	}
	if !f.vertical {
		width = -1
	}
	sizes := f.sizes(width)
	if width >= 0 {
		for _, size := range sizes {
			if size.W <= width {
				return size
			}
		}
	}
	return sizes[0]
}
func (f *fittingBuilder) build(width, height float32) {
	if f.built || len(f.candidates) == 0 || f.chosen == nil {
		return
	}
	f.built = true
	measureWidth := float32(-1)
	if f.vertical {
		measureWidth = width
	}
	sizes := f.sizes(measureWidth)
	selected := len(sizes) - 1
	for i, size := range sizes {
		if !f.vertical && size.W <= width+0.01 || f.vertical && size.H <= height+0.01 {
			selected = i
			break
		}
	}
	f.index = selected
	raw := f.e.c
	parent, theme := raw.parent, raw.theme
	raw.parent, raw.theme = f.e, &f.theme
	f.chosen(f.c, selected)
	raw.parent, raw.theme = parent, theme
	raw.rt.late = true
}

// SetOverscan sets rows constructed beyond each viewport edge. Zero disables
// overscan; unconfigured lists retain the upstream default of two rows.
func (s *ListState) SetOverscan(rows int) { value := max(0, rows); s.overscan = &value }
func (s *ListState) overscanRows() int {
	if s.overscan == nil {
		return listOverscan
	}
	return *s.overscan
}

// SetItemExtent supplies the initial row-height estimate before real rows are
// measured. Actual measured heights always replace the estimate.
func (s *ListState) SetItemExtent(extent float32) {
	if extent > 0 && !math.IsInf(float64(extent), 0) && !math.IsNaN(float64(extent)) {
		s.itemExtent = extent
	} else {
		s.itemExtent = 0
	}
}

// Stop cancels an explicit ScrollTo animation at its current position.
func (s *ScrollState) Stop() { s.glide = nil }
