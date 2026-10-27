# Window input capture

## Inherited pointer hit policy

`Element.PointerEvents(false)` makes an element and its descendants transparent
to pointer hit testing, including native controls' internal labels. Unset
descendants inherit the policy; an explicit `PointerEvents(true)` re-enables a
subtree, for example an interactive popover inside a non-interactive overlay
layer. The root defaults to enabled. The policy is committed each frame and
does not change paint, layout, keyboard focus, accessibility or input capture
default-prevention. Ancestors still participate in event bubbling from an
enabled descendant. `Inert` and `Invisible` cannot be undone with this policy.

This differs from `PassThrough`, which excludes only the element itself:

```go
ui.Box(c).Fill().PointerEvents(false).Children(func() {
    ui.Box(c).PointerEvents(true).Children(func() {
        ui.Button(c, "Action").OnClick(action)
    })
})
```

This native UI API is identical on Linux, macOS, Windows and wasm, adds no
backend calls or cgo, and is valid only during the element's current build
pass on the UI thread. Keep a `Handle`, not an `Element`, between frames.

## Bounded list ownership

`ListState.Requested()` reports a half-open overscan window and separate indices
retained for active input or an open popup. Pins do not enlarge the contiguous
window. A host that mounts only this window must preserve those pinned rows and
their identities until they are no longer requested. `ListState.SetKeyIndex`
accepts an authoritative key lookup for distant reorders; missing keys must
return false. Passing nil restores the nearby-key search for existing callers.

These APIs coexist with inherited pointer policy: retained rows still inherit
`PointerEvents`, and keeping a row for keyboard focus does not opt it into
pointer hits. Keep application-owned handles for identities needed across
frames. Abandoned handle bindings can be collected; an `Element` or callback-
scoped `InputTarget` is not a persistent ownership token.

## Pre-dispatch capture

Install `Context.CaptureInput(func(InputEvent, InputTarget) bool)` once per view
build. It observes input before focused text clients, element handlers, shortcuts,
menus and native editing consume it. Returning true prevents native handling;
return false for observation. A build that omits registration removes it. The
last registration wins. Callbacks run on the UI thread, must not build UI or
enter a nested event loop, and must remain short.

Keyboard, text, composition and native menu commands identify the committed
focused element. Pointer and scroll events identify the deepest committed hit
element, including non-interactive text children. `InputTarget.Distance(handle)`
reports zero for an exact binding, the number of ancestors for a containing
binding, and -1 otherwise. An input target expires on callback return; retain
application identity instead, and validate its incarnation before delayed use.
No geometry guessing or private engine access is needed. Pointer positions and
scroll deltas in capture use window DIPs, unlike local `HandleInput` coordinates.

Capture is for application input, not the built-in inspector. Keyboard release
and press, text, composition and menu commands focused in the inspector never
reach application capture. Pointer/scroll targets in its committed subtree are
also excluded; a native inspector press retains this isolation while moving or
releasing outside the panel. The check uses committed ownership, not coordinates
or application labels. Inspector controls still receive their normal native
input, and application capture/default-prevention continues while it is open.
An inspector event cancels any pending plain-hit capture click candidate.
Hosts must not use inspector input as document shortcuts or editing commands;
remove any such forwarding. No host-side handler or wire change is needed.

Capture reports raw pointer press/release/movement and one `InputClick` when a
primary gesture qualifies as a click. Native controls keep their existing
press/drag rules. A plain hit without a native gesture also reports a
capture-only click: the same committed hit incarnation must survive to release,
remain enabled and hit-testable, and movement must stay below the native drag
threshold (4 DIPs), even if the pointer later returns. A prevented up,
window blur, omitted capture registration or retired hit cancels that candidate.
Plain hits do not acquire native press, focus or activation semantics. The
callback identifies the actual release hit in window DIPs and carries the
engine's click count; native controls still activate at most once. Click
recognition, focus, IME, editor commands and menus remain MyGo's responsibility
unless explicitly prevented. A pointer release prevented by capture cancels the
pending native press/drag so it cannot produce a click later.

Preventing pointer-down blocks native focus, selection and press activation,
but does not cancel the separate capture-only `InputClick`. It follows the same
live-hit, movement and lifecycle rules as a plain hit. This lets a host cancel
DOM `mousedown` to keep editor focus while still selecting an autocomplete
candidate on the matching release. This corrects the earlier behavior where
preventing down also discarded the capture click. Hosts that need to suppress
the click must prevent the release, or ignore/prevent `InputClick` itself;
native control activation remains suppressed by the prevented down. Do not
work around this by forwarding another click from a local pointer handler.

Hosts upgrading from interactive-only click capture should remove plain-node
click synthesis or placeholder `OnClick` handlers used solely to enable global
delegation. For example, leave `ui.Box(c).Bind(&article)` non-interactive and
observe `InputClick` with `target.Distance(article)` in the window callback.
Do not forward the same physical click again from an element-local handler.

Native menu commands arrive as `InputCommand` (for example `copy`, `paste`,
`selectAll`, `undo`). Do not re-inject fabricated physical keys into the runtime.
A host may normalize a menu command to its existing document accelerator
vocabulary, but must return false unless its document explicitly prevents the
command's default action. The focused native text control then handles the
original command exactly once. Embedded editors may handle the command through
their own focused input handler.

Capture does not synthesize key events on platforms where an input method
reports only committed/composing text, nor intercept OS-reserved shortcuts.
Applications forwarding document events must suppress duplicate forwarding from
their element-local handlers, not disable those handlers' native editing logic.
