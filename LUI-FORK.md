# LUI compatibility fork

This fork integrates upstream MyGo through commit
`1db4e93bd48efa7e3f5a93985c8e89a4c6fa9e3d`, retaining the complete upstream
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

`ZIndex` uses upstream's opt-in paint/commit ordering, not a separate sorter.
Upstream `TextLayout`, accessibility, physical keys and `KeyCode` remain intact.
Committed handle geometry, live text snapshots, focus epochs and build-time
text-flow queries retain their separate lifetime and coordinate contracts.

## Source and behavior migration

- The fork's committed `TextLine` result is now `CommittedTextLine`:
  `CommittedText.Lines() ([]CommittedTextLine, bool)`. Its ranges are
  node-relative rune offsets and its `Bounds` are window DIPs. Upstream's
  `TextLine` remains UTF-16, with layout-local `Rect` and baseline/RTL metrics.
  Change explicit committed-result annotations from `[]ui.TextLine` to
  `[]ui.CommittedTextLine`; inferred callers need no rename. Do not convert
  between the types by aliasing offsets or coordinates.
- `Handle.ScrollOffset(c)` retains `(x, y float32, ok bool)`.
- Custom `ui.TextInputClient` implementations must implement both
  `EndComposition()` (transient end, pending transaction retained) and
  `CancelComposition()` (rollback), alongside `UnmarkText()` (accept).
  Release on blur, replacement or disposal cancels unfinished preedit rather
  than accepting it. The internal native client contract also requires typed
  cancellation; retired adapters cannot mutate replacement clients. See
  [the migration guide](docs/ui/migration.md#text-input-composition-phases).
- `InputEvent.Text` carries real native glyph metadata on normal/raw key down
  and up alongside logical, physical and code identities. It is not a second
  insertion route. Consumed input denotes actual capture/handler/shortcut
  ownership, not ordinary builtin printable-key bookkeeping.
- Native cancellation remains canceled `InputPointerUp` for `HandleInput`;
  `HandleCapturedPointer` translates its owned primary cancellation into one
  `InputPointerCancel`. Ordinary completed release never becomes cancellation.
- Menus use upstream `SetMenuPresentation`. Choice is applied before one
  dismissal callback; retired hosts/owners cannot complete a replacement menu.
  `ImagePNG` copies legacy full-color bytes and lazily uses upstream icons only
  when no explicit `Icon` was supplied. Unused apps do not opt into decoders.

The opt-in `testing/native` facade (`lui_native_smoke` build tag) exposes existing native host hooks for real Cocoa, GTK and Win32 qualification. Small host helpers read actual ATK/UIA focus and native popup image/name bindings. GTK image rows keep an explicit ATK name and update their retained label without removing the icon. These checks cover native interfaces and input-protocol boundaries; they do not claim audible screen-reader output or keyboard IME candidate UI.

Focused `ui/lui_*_test.go` tests verify these extensions through the real layout, interaction, accessibility and rendering pipeline. Each extension source carries its API documentation; existing runtime files contain only the necessary integration hooks.

Refresh this fork by merging an upstream release, reviewing these extensions,
and rerunning the layout, input, accessibility and native host checks. Preserve
upstream copyright notices and asset licenses. Never patch the Go module cache.
