# LUI compatibility branch

This branch starts at upstream MyGo v0.6.4, commit
`6255ea40139d06e0762a7c4a32626148ba410478`, and retains the complete upstream
repository, tests and licenses. Its Go module path and self-imports remain
`github.com/egoist/mygo`, preserving the upstream identity. Downstream applications
can pin an immutable fork version with their Go module replacement mechanism.

No third-party source, module cache or generated build output needs to be
committed to downstream applications.

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
