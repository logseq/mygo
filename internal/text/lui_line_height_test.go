package text

import "testing"

func TestLUIExplicitLineBoxesPreserveInkAndInlineObjectMetrics(t *testing.T) {
	s := Shared()
	for _, family := range []string{"system-ui", "serif", "monospace"} {
		t.Run(family, func(t *testing.T) {
			style := Style{Size: 40, Family: family, LineHeight: .25}
			layout := s.Layout(Params{Text: "first\nsecond", Style: style})
			if len(layout.Lines) != 2 || layout.Height != 20 {
				t.Errorf("explicit line boxes: lines=%d height=%v, want 2 lines of 10", len(layout.Lines), layout.Height)
			}
			if layout.Lines[0].Ascent+layout.Lines[0].Descent <= 10 {
				t.Error("font ink metrics were truncated to the explicit line height")
			}
			style.LineHeight = 0
			natural := s.Layout(Params{Text: "first\nsecond", Style: style})
			if natural.Height <= 20 {
				t.Errorf("default line spacing lost natural font metrics: %v", natural.Height)
			}
			style.LineHeight = .25
			object := s.Layout(Params{Text: "a\ufffcb", Style: style, Spans: EncodeSpans([]Span{
				{End: 1}, {End: 2, ObjectWidth: 30, ObjectHeight: 70}, {End: 3},
			})})
			if object.Height < 70 {
				t.Errorf("inline object no longer contributes its layout height: %v", object.Height)
			}
		})
	}
}
