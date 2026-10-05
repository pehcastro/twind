package ui

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/pehcastro/twind/twi/edit"
)

type fieldMemo struct {
	Value          string
	Cursor, Anchor int
	Chips          []string
}

func (e *editor) remember() {
	if e.Key != "" {
		e.rt.Remember("field"+e.Key, e.memo, e.recall)
	}
}

func (e *editor) memo() string {
	m := fieldMemo{Value: e.Value(), Cursor: e.At(e.Cursor())}
	lo, hi := e.Selection()
	m.Anchor = lo
	if m.Cursor == lo {
		m.Anchor = hi
	}
	for start, end := range chipSpans(m.Value) {
		m.Chips = append(m.Chips, strings.TrimPrefix(e.Expand(m.Value[:end]), e.Expand(m.Value[:start])))
	}
	raw, _ := json.Marshal(m)
	return string(raw)
}

func (e *editor) recall(raw string) {
	var m fieldMemo
	if json.Unmarshal([]byte(raw), &m) != nil {
		return
	}
	e.Set("")
	at, next := 0, 0
	for start, end := range chipSpans(m.Value) {
		e.Insert(m.Value[at:start])
		if next < len(m.Chips) && m.Chips[next] != m.Value[start:end] {
			e.Paste(m.Chips[next])
		} else {
			e.Insert(m.Value[start:end])
		}
		at, next = end, next+1
	}
	e.Insert(m.Value[at:])
	e.Press(m.Anchor, edit.Grapheme, false)
	e.Press(m.Cursor, edit.Grapheme, true)
}

func chipSpans(s string) func(yield func(start, end int) bool) {
	return func(yield func(start, end int) bool) {
		for at := 0; ; {
			open := strings.Index(s[at:], edit.ChipHead)
			if open < 0 {
				return
			}
			open += at
			shut := strings.Index(s[open:], edit.ChipTail)
			if shut < 0 {
				return
			}
			end := open + shut + len(edit.ChipTail)
			if _, err := strconv.Atoi(s[open+len(edit.ChipHead) : open+shut]); err == nil && !yield(open, end) {
				return
			}
			at = end
		}
	}
}

func (o *overlay) remember() {
	if o.Key != "" {
		o.rt.Remember("open"+o.Key, o.memo, o.recall)
	}
}

func (o *overlay) memo() string { return strconv.FormatBool(o.Open) }

func (o *overlay) recall(raw string) { o.Open, _ = strconv.ParseBool(raw) }
