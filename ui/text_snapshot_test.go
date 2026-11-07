package ui

import "testing"

func TestHandleLiveTextSnapshotLifetime(t *testing.T) {
	var field Handle
	var ctx *Context
	value, shown, identity := "a😀b", true, 0
	var changed string
	tt := NewTester(func(c *Context) {
		ctx = c
		if shown {
			TextArea(c.Key(identity), &value).Bind(&field).AutoFocus().OnChange(func() {
				text, _, _, ok := field.TextSnapshot(c)
				if !ok {
					t.Fatal("live input unavailable in OnChange")
				}
				changed = text
			})
		}
	}, 300, 120)
	defer tt.rt.close()
	tt.Frame()
	tt.Key(Shift, KeyLeft)
	if _, a, z, ok := field.TextSnapshot(ctx); !ok || a != 3 || z != 2 {
		t.Fatalf("reversed rune selection: %d %d %v", a, z, ok)
	}
	b := field.binding(tt.rt, false)
	client := tt.rt.states[b.id].editor.client
	client.ReplaceText(nil, "x")
	text, anchor, caret, ok := field.TextSnapshot(ctx)
	if !ok || text != "a😀x" || anchor != 3 || caret != 3 || value != "a😀b" {
		t.Fatalf("native snapshot before bound-value publication: %q %d %d %v (value %q)", text, anchor, caret, ok, value)
	}
	tt.Frame()
	if changed != "a😀x" {
		t.Fatalf("post-edit OnChange snapshot: %q", changed)
	}
	old := field
	shown = false
	tt.Frame()
	if _, _, _, ok := old.TextSnapshot(ctx); ok {
		t.Fatal("removed control remained readable")
	}
	field, value, shown, identity = Handle{}, "fresh", true, identity+1
	tt.Frame()
	if _, _, _, ok := old.TextSnapshot(ctx); ok {
		t.Fatal("retired handle aliased remounted input")
	}
	if text, _, _, ok := field.TextSnapshot(ctx); !ok || text != "fresh" {
		t.Fatalf("remount: %q %v", text, ok)
	}
	tt.rt.close()
	if _, _, _, ok := field.TextSnapshot(ctx); ok {
		t.Fatal("closed input remained readable")
	}
}
