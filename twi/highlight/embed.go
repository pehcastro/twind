package highlight

import (
	"strings"
	"unsafe"

	konst "github.com/twind-dev/twind/internal/konst/highlight"
)

type region struct {
	start, end int
	hole       bool
	from, to   int
	synthetic  Kind
}

func (w *work) splice(sub *Grammar) {
	src, sources, content := w.src, 0, -1
	for k, r := range w.regions {
		if r.synthetic == Text {
			sources++
			if !r.hole {
				content = k
			}
		}
	}
	var virtual string
	if sources == 1 && content >= 0 {
		virtual = src[w.regions[content].start:w.regions[content].end]
	} else {
		w.virtual = w.virtual[:0]
		for _, r := range w.regions {
			switch {
			case r.synthetic != Text:
			case r.hole:
				for range r.end - r.start {
					w.virtual = append(w.virtual, ' ')
				}
			default:
				w.virtual = append(w.virtual, src[r.start:r.end]...)
			}
		}
		virtual = unsafe.String(unsafe.SliceData(w.virtual), len(w.virtual))
	}
	if w.nested == nil {
		w.nested = &work{}
	}
	inner := w.nested
	sub.fill(inner, virtual)
	at, offset := 0, 0
	for _, r := range w.regions {
		end := offset + r.end - r.start
		switch {
		case r.synthetic != Text:
			w.out = append(w.out, Span{Kind: r.synthetic, Start: r.start, End: r.end})
			continue
		case r.hole:
			w.out = append(w.out, w.spans[r.from:r.to]...)
		default:
			for ; at < len(inner.spans) && inner.spans[at].Start < end; at++ {
				s := inner.spans[at]
				s.Start, s.End = max(s.Start, offset)-offset+r.start, min(s.End, end)-offset+r.start
				if s.End > s.Start {
					w.out = append(w.out, s)
				}
				if inner.spans[at].End > end {
					break
				}
			}
		}
		offset = end
	}
	inner.src = ""
}

func (w *work) embedGroups(sub embeds) {
	for range konst.EmbedPasses {
		w.out = w.out[:0]
		found := false
		for i := 0; i < len(w.spans); {
			if next, g := w.taggedTemplate(i, sub); g != nil {
				w.out = append(w.out, w.spans[i])
				w.splice(g)
				i, found = next, true
				continue
			}
			if g := w.docComment(i, sub); g != nil {
				w.splice(g)
				i, found = i+1, true
				continue
			}
			w.out = append(w.out, w.spans[i])
			i++
		}
		if !found {
			return
		}
		w.spans, w.out = w.out, w.spans
	}
}

func (w *work) taggedTemplate(i int, sub embeds) (int, *Grammar) {
	if w.spans[i].Kind != Identifier {
		return 0, nil
	}
	var g *Grammar
	switch w.text(i) {
	case "html":
		g = sub.html
	case "css":
		g = sub.css
	default:
		return 0, nil
	}
	first := i + 1
	if first >= len(w.spans) || w.spans[first].Kind != Template || w.src[w.spans[first].Start] != '`' {
		return 0, nil
	}
	start := w.spans[first].Start
	w.regions = append(w.regions[:0], region{start: start, end: start + 1, synthetic: Template})
	hole, depth := -1, 0
	for k := first; k < len(w.spans); k++ {
		s := w.spans[k]
		if hole >= 0 {
			if s.Kind != Punctuation {
				continue
			}
			t := w.text(k)
			if depth += strings.Count(t, "{") - strings.Count(t, "}"); depth == 0 {
				w.regions = append(w.regions, region{start: w.spans[hole].Start, end: s.End, hole: true, from: hole, to: k + 1})
				hole = -1
			}
			continue
		}
		switch {
		case s.Kind == Template:
			from := s.Start
			if k == first {
				from++
			}
			closing := s.End > from && w.src[s.End-1] == '`'
			end := s.End
			if closing {
				end--
			}
			if end > from {
				w.regions = append(w.regions, region{start: from, end: end})
			}
			if closing {
				w.regions = append(w.regions, region{start: s.End - 1, end: s.End, synthetic: Template})
				return k + 1, g
			}
		case s.Kind == Punctuation && w.text(k) == "${":
			hole, depth = k, 1
		default:
			return 0, nil
		}
	}
	return 0, nil
}

func (w *work) docComment(i int, sub embeds) *Grammar {
	s := w.spans[i]
	if s.Kind != Comment || s.End-s.Start <= 3 || !strings.HasPrefix(w.text(i), "/**") {
		return nil
	}
	w.regions = append(w.regions[:0], region{start: s.Start, end: s.Start + 3, synthetic: Comment}, region{start: s.Start + 3, end: s.End})
	return sub.jsdoc
}

func (w *work) embedRaw(script, style *Grammar) {
	w.out = w.out[:0]
	for _, s := range w.spans {
		g := style
		switch s.Kind {
		case rawScript:
			g = script
		case rawStyle:
		default:
			w.out = append(w.out, s)
			continue
		}
		w.regions = append(w.regions[:0], region{start: s.Start, end: s.End})
		w.splice(g)
	}
	w.spans, w.out = w.out, w.spans
}
