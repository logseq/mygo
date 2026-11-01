# LUI compatibility branch

This branch started at upstream MyGo v0.6.4, commit
`6255ea40139d06e0762a7c4a32626148ba410478`, and merges the official v0.7.3
release, `6f28d7bb07b31177cf218e51586f6a5e99b38016`. It retains the complete upstream
repository, tests and licenses. Its Go module path and self-imports remain
`github.com/egoist/mygo`, preserving the upstream identity. Downstream applications
can pin an immutable fork version with their Go module replacement mechanism.

No third-party source, module cache or generated build output needs to be
committed to downstream applications.

## v0.7.3 compatibility migration

- `CommittedText.Lines()` now returns `[]CommittedTextLine`, not `[]TextLine`.
  Rename explicit references to the old committed geometry type, for example
  `var lines []ui.CommittedTextLine`. Its `Start`/`End` remain rune offsets
  relative to the bound node, and `Bounds` remains a transformed window-space
  DIP rectangle. Upstream `TextLayout.Lines()` returns `[]TextLine`: its offsets
  are UTF-16 and its `Rect` is layout-local DIPs. Do not interchange these types.
- New menu code should use upstream `MenuItem.Icon`/`NewMenuIcon`/`SetIcon`.
  Internal content-menu snapshots with only `ImagePNG` retain their legacy
  sizing and native owner-draw behavior. If both are present, `Icon` wins;
  clearing it reveals the legacy image again. GTK custom rows retain explicit
  accessibility names. Windows menus with any legacy fallback retain their
  owner-draw mode and 28-DIP legacy icon column so clearing `Icon` needs no
  rebuild. This also applies when both inputs are present: only the typed icon
  is drawn. The upstream typed icon column is also 28 DIPs (16 + 12), but a
  compatibility menu can use owner-draw where a typed-only menu uses a native
  bitmap on Windows 10.
  Menus with only typed icons keep the upstream owner-draw/native-bitmap
  selection and layout. Tray updates restore
  the owned legacy PNG through the existing native export. No second menu
  lifecycle is introduced.
- Explicit compact line boxes still allow ordinary glyph ink to overflow.
  Inline objects and shifted spans include their ascent/descent extents in
  layout, including `ObjectDescent` and `BaselineShift`.

Local extensions are deliberately limited to reusable native runtime primitives required by portable LUI behavior:

- `Context.Measure` isolates intrinsic measurement; `Context.Fitting` chooses using final local layout constraints and constructs only the selected retained candidate. Fitting subtrees bypass cached-layout reuse.
- `Context.AfterFrame` runs once after the final committed layout, including late virtual rows and fitting children, before the next input/render pass.
- `ListState.SetOverscan` and `SetItemExtent` configure bounded virtualization using the actual local viewport. A scroll request fencepost is corrected and `ScrollState.Stop` cancels superseded animation.
- `Element.ZIndex` orders painting and pointer hits without reordering layout. `DisabledOpacity`, percentage minimums with pixel offsets, `ScrollAxes`, `WhiteSpace`, `Inert`, explicit italic and decoration overrides expose portable style behavior. `Tester.Accessibility` returns a copied committed accessibility tree for integration assertions.
- `Dialog.OpenContext` cancels the owned native picker on macOS, GTK and Windows and filters a result if cancellation wins the completion race. `Dialog.Open` preserves its existing pre-start diagnostic.
- `RoleTabPanel`, `OnMenuDismiss`, native cancellation notification, native menu `ImagePNG`/SVG rasterization, `LinkBase`, `CollisionPadding` and late-bound `AttachToHandle`/`MatchAnchorWidth` support accessible tabs and native menus/popovers, including targets constructed later in the same frame.

The opt-in `testing/native` facade (`lui_native_smoke` build tag) exposes existing native host hooks for real Cocoa, GTK and Win32 qualification. Small host helpers read actual ATK/UIA focus and native popup image/name bindings. GTK image rows keep an explicit ATK name and update their retained label without removing the icon. These checks cover native interfaces and input-protocol boundaries; they do not claim audible screen-reader output or keyboard IME candidate UI.

Focused `ui/lui_*_test.go` tests verify these extensions through the real layout, interaction, accessibility and rendering pipeline. Each extension source carries its API documentation; existing runtime files contain only the necessary integration hooks.

Refresh this branch by merging an upstream release, reviewing these extensions,
and rerunning the layout, input, accessibility and native host checks. Preserve
upstream copyright notices and asset licenses. Never patch the Go module cache.
