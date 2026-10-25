# Window input capture

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

Capture reports raw pointer press/release/movement and `InputClick` when MyGo's
native hit/press/drag rules qualify a primary release as a click. The latter is
capture-only and includes the native click count. Click
recognition, focus, IME, editor commands and menus remain MyGo's responsibility
unless explicitly prevented. A pointer release prevented by capture cancels the
pending native press/drag so it cannot produce a click later.

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
