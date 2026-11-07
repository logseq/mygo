package ui

// LinkBase creates a link with arbitrary child content. It uses Link's native
// routing, URL activation, focus and keyboard behavior; the caller supplies
// its label, icon and other presentation.
func LinkBase(c *Context, url string) Element {
	ctx := c.build()
	if ctx == nil {
		return Element{}
	}
	ctx.rt.beginBuild()
	e := coreButtonBase(ctx).TextColor(ctx.theme.Accent).Cursor(CursorPointer)
	result := linkBehavior(ctx, e, url)
	ctx.rt.endBuild()
	return wrapElement(result)
}
