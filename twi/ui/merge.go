package ui

import (
	"slices"
	"strings"
)

type classKey struct {
	variant          string
	important, color bool
	group            string
}

func Merge(classes ...string) string {
	tokens := make([]string, 0, 32)
	for _, c := range classes {
		for token := range strings.FieldsSeq(c) {
			tokens = append(tokens, token)
		}
	}
	taken := make([]classKey, 0, len(tokens))
	size := 0
	for i, token := range slices.Backward(tokens) {
		k := keyOf(token)
		if slices.ContainsFunc(taken, func(later classKey) bool { return later.replaces(k) }) {
			tokens[i] = ""
			continue
		}
		taken = append(taken, k)
		size += len(token) + 1
	}
	var out strings.Builder
	out.Grow(size)
	for _, token := range tokens {
		if token != "" {
			if out.Len() > 0 {
				out.WriteByte(' ')
			}
			out.WriteString(token)
		}
	}
	return out.String()
}

func (later classKey) replaces(earlier classKey) bool {
	return later.variant == earlier.variant && later.important == earlier.important && later.color == earlier.color &&
		(later.group == earlier.group || covers(later.group, earlier.group))
}

func keyOf(token string) classKey {
	var k classKey
	base := token
	if at := lastOutside(token, ':'); at >= 0 {
		k.variant, base = variant(token[:at]), token[at+1:]
	}
	if trimmed, ok := strings.CutSuffix(base, "!"); ok {
		base, k.important = trimmed, true
	}
	if trimmed, ok := strings.CutPrefix(base, "!"); ok {
		base, k.important = trimmed, true
	}
	k.group, k.color = groupOf(strings.TrimPrefix(base, "-"))
	return k
}

func lastOutside(s string, sep byte) int {
	at, depth := -1, 0
	for i := range len(s) {
		switch s[i] {
		case '[', '(':
			depth++
		case ']', ')':
			depth--
		case sep:
			if depth == 0 {
				at = i
			}
		}
	}
	return at
}

func variant(modifiers string) string {
	if !strings.Contains(modifiers, ":") || strings.ContainsAny(modifiers, "[*") {
		return modifiers
	}
	parts := strings.Split(modifiers, ":")
	slices.Sort(parts)
	return strings.Join(parts, ":")
}

func groupOf(base string) (string, bool) {
	if strings.HasPrefix(base, "[") {
		if at := strings.IndexByte(base, ':'); at > 0 {
			return base[:at], false
		}
		return base, false
	}
	if g := static(base); g != "" {
		return g, false
	}
	end := len(base)
	if at := strings.IndexAny(base, "[("); at >= 0 {
		end = at
	}
	if g, color := functional(base, ""); end == len(base) && g != "" {
		return g, color
	}
	for at := strings.LastIndexByte(base[:end], '-'); at > 0; at = strings.LastIndexByte(base[:at], '-') {
		value := base[at+1:]
		if slash := lastOutside(value, '/'); slash >= 0 {
			value = value[:slash]
		}
		if g, color := functional(base[:at], value); g != "" {
			return g, color
		}
	}
	return base, false
}

func static(base string) string {
	switch base {
	case "block", "inline-block", "inline", "flex", "inline-flex", "grid", "inline-grid", "hidden", "contents", "flow-root", "list-item",
		"table", "inline-table", "table-caption", "table-cell", "table-column", "table-column-group", "table-footer-group", "table-header-group", "table-row-group", "table-row":
		return "display"
	case "static", "fixed", "absolute", "relative", "sticky":
		return "position"
	case "visible", "invisible", "collapse":
		return "visibility"
	case "isolate", "isolation-auto":
		return "isolation"
	case "box-border", "box-content":
		return "box-sizing"
	case "sr-only", "not-sr-only":
		return "sr-only"
	case "flex-row", "flex-row-reverse", "flex-col", "flex-col-reverse":
		return "flex-direction"
	case "flex-wrap", "flex-nowrap", "flex-wrap-reverse":
		return "flex-wrap"
	case "flex-auto", "flex-initial", "flex-none":
		return "flex"
	case "text-left", "text-center", "text-right", "text-justify", "text-start", "text-end":
		return "text-align"
	case "text-wrap", "text-nowrap", "text-balance", "text-pretty":
		return "text-wrap"
	case "truncate", "text-ellipsis", "text-clip":
		return "text-overflow"
	case "italic", "not-italic":
		return "font-style"
	case "uppercase", "lowercase", "capitalize", "normal-case":
		return "text-transform"
	case "underline", "overline", "line-through", "no-underline":
		return "text-decoration-line"
	case "antialiased", "subpixel-antialiased":
		return "font-smoothing"
	case "break-normal", "break-all", "break-keep":
		return "word-break"
	case "wrap-anywhere", "wrap-break-word", "wrap-normal":
		return "overflow-wrap"
	case "border-solid", "border-dashed", "border-dotted", "border-double", "border-hidden", "border-none":
		return "border-style"
	case "outline-hidden", "outline-none", "outline-solid", "outline-dashed", "outline-dotted", "outline-double":
		return "outline-style"
	case "decoration-solid", "decoration-double", "decoration-dotted", "decoration-dashed", "decoration-wavy":
		return "decoration-style"
	case "bg-auto", "bg-cover", "bg-contain":
		return "bg-size"
	case "bg-fixed", "bg-local", "bg-scroll":
		return "bg-attachment"
	case "bg-top", "bg-top-left", "bg-top-right", "bg-bottom", "bg-bottom-left", "bg-bottom-right", "bg-left", "bg-right", "bg-center":
		return "bg-position"
	case "bg-repeat", "bg-no-repeat", "bg-repeat-x", "bg-repeat-y", "bg-repeat-round", "bg-repeat-space":
		return "bg-repeat"
	case "bg-none":
		return "bg-image"
	case "object-contain", "object-cover", "object-fill", "object-none", "object-scale-down":
		return "object-fit"
	case "resize", "resize-none", "resize-x", "resize-y":
		return "resize"
	case "ring-inset":
		return "ring-inset"
	case "content-none":
		return "content"
	}
	return ""
}

func functional(root, value string) (string, bool) {
	switch root {
	case "p", "px", "py", "pt", "pr", "pb", "pl", "ps", "pe", "m", "mx", "my", "mt", "mr", "mb", "ml", "ms", "me",
		"w", "h", "min-w", "min-h", "max-w", "max-h", "size", "inset", "inset-x", "inset-y", "top", "right", "bottom", "left", "start", "end",
		"gap", "gap-x", "gap-y", "space-x", "space-y", "z", "order", "basis", "grow", "shrink", "opacity", "leading", "tracking", "indent", "line-clamp",
		"col", "col-span", "col-start", "col-end", "row", "row-span", "row-start", "row-end", "grid-cols", "grid-rows", "auto-cols", "auto-rows", "grid-flow",
		"items", "justify", "justify-items", "justify-self", "self", "place-content", "place-items", "place-self",
		"overflow", "overflow-x", "overflow-y", "overscroll", "overscroll-x", "overscroll-y", "whitespace", "select", "pointer-events", "cursor", "touch",
		"rounded", "rounded-t", "rounded-r", "rounded-b", "rounded-l", "rounded-s", "rounded-e", "rounded-tl", "rounded-tr", "rounded-br", "rounded-bl", "rounded-ss", "rounded-se", "rounded-ee", "rounded-es",
		"divide-x", "divide-y", "outline-offset", "ring-offset", "underline-offset", "aspect", "columns", "origin", "animate", "transition", "duration", "delay", "ease",
		"translate", "translate-x", "translate-y", "scale", "scale-x", "scale-y", "rotate", "skew-x", "skew-y", "blur", "brightness", "contrast", "grayscale", "invert", "saturate", "sepia",
		"backdrop-blur", "appearance", "scheme", "will-change", "field-sizing", "list-image", "bg-linear", "bg-radial", "bg-conic", "bg-size", "bg-position", "bg-clip", "bg-origin", "bg-blend", "mix-blend":
		return root, false
	case "flex":
		if value == "" {
			return "", false
		}
		return "flex", false
	case "content":
		if slices.Contains([]string{"normal", "center", "start", "end", "center-safe", "end-safe", "between", "around", "evenly", "baseline", "stretch"}, value) {
			return "align-content", false
		}
		return "content", false
	case "text":
		if slices.Contains([]string{"xs", "sm", "base", "lg", "xl", "2xl", "3xl", "4xl", "5xl", "6xl", "7xl", "8xl", "9xl"}, value) || sized(value) {
			return "font-size", false
		}
		return root, true
	case "font":
		if slices.Contains([]string{"thin", "extralight", "light", "normal", "medium", "semibold", "bold", "extrabold", "black"}, value) || digits(value) {
			return "font-weight", false
		}
		return "font-family", false
	case "border", "border-x", "border-y", "border-t", "border-r", "border-b", "border-l", "border-s", "border-e", "outline", "ring", "inset-ring", "stroke", "decoration":
		if value == "" || digits(value) || sized(value) || value == "px" || value == "auto" || value == "from-font" {
			return root, false
		}
		return root, true
	case "shadow", "inset-shadow", "text-shadow", "drop-shadow":
		if slices.Contains([]string{"", "2xs", "xs", "sm", "md", "lg", "xl", "2xl", "none"}, value) || strings.HasPrefix(value, "[") && !strings.HasPrefix(value, "[color:") {
			return root, false
		}
		return root, true
	case "bg", "fill", "caret", "accent", "placeholder", "from", "via", "to", "divide", "scrollbar-thumb", "scrollbar-track":
		return root, true
	case "object":
		return "object-position", false
	case "list":
		return "list-style", false
	}
	return "", false
}

func digits(v string) bool {
	return v != "" && strings.Trim(v, "0123456789.") == ""
}

func sized(v string) bool {
	inner, ok := strings.CutPrefix(v, "[")
	if !ok {
		return false
	}
	return strings.HasPrefix(inner, "length:") || inner != "" && (inner[0] >= '0' && inner[0] <= '9' || inner[0] == '.' || strings.HasPrefix(inner, "calc("))
}

func covers(shorthand, longhand string) bool {
	switch shorthand {
	case "p":
		return slices.Contains([]string{"px", "py", "pt", "pr", "pb", "pl", "ps", "pe"}, longhand)
	case "px":
		return slices.Contains([]string{"pr", "pl", "ps", "pe"}, longhand)
	case "py":
		return slices.Contains([]string{"pt", "pb"}, longhand)
	case "m":
		return slices.Contains([]string{"mx", "my", "mt", "mr", "mb", "ml", "ms", "me"}, longhand)
	case "mx":
		return slices.Contains([]string{"mr", "ml", "ms", "me"}, longhand)
	case "my":
		return slices.Contains([]string{"mt", "mb"}, longhand)
	case "size":
		return slices.Contains([]string{"w", "h"}, longhand)
	case "inset":
		return slices.Contains([]string{"inset-x", "inset-y", "top", "right", "bottom", "left", "start", "end"}, longhand)
	case "inset-x":
		return slices.Contains([]string{"right", "left", "start", "end"}, longhand)
	case "inset-y":
		return slices.Contains([]string{"top", "bottom"}, longhand)
	case "gap":
		return slices.Contains([]string{"gap-x", "gap-y"}, longhand)
	case "flex":
		return slices.Contains([]string{"basis", "grow", "shrink"}, longhand)
	case "overflow":
		return slices.Contains([]string{"overflow-x", "overflow-y"}, longhand)
	case "overscroll":
		return slices.Contains([]string{"overscroll-x", "overscroll-y"}, longhand)
	case "place-content":
		return slices.Contains([]string{"align-content", "justify"}, longhand)
	case "place-items":
		return slices.Contains([]string{"items", "justify-items"}, longhand)
	case "place-self":
		return slices.Contains([]string{"self", "justify-self"}, longhand)
	case "font-size":
		return longhand == "leading"
	case "line-clamp":
		return slices.Contains([]string{"display", "overflow"}, longhand)
	case "translate":
		return slices.Contains([]string{"translate-x", "translate-y"}, longhand)
	case "scale":
		return slices.Contains([]string{"scale-x", "scale-y"}, longhand)
	case "rounded":
		return strings.HasPrefix(longhand, "rounded-")
	case "rounded-t":
		return slices.Contains([]string{"rounded-tl", "rounded-tr"}, longhand)
	case "rounded-r":
		return slices.Contains([]string{"rounded-tr", "rounded-br"}, longhand)
	case "rounded-b":
		return slices.Contains([]string{"rounded-br", "rounded-bl"}, longhand)
	case "rounded-l":
		return slices.Contains([]string{"rounded-tl", "rounded-bl"}, longhand)
	case "rounded-s":
		return slices.Contains([]string{"rounded-ss", "rounded-es"}, longhand)
	case "rounded-e":
		return slices.Contains([]string{"rounded-se", "rounded-ee"}, longhand)
	case "border":
		return slices.Contains([]string{"border-x", "border-y", "border-t", "border-r", "border-b", "border-l", "border-s", "border-e"}, longhand)
	case "border-x":
		return slices.Contains([]string{"border-r", "border-l", "border-s", "border-e"}, longhand)
	case "border-y":
		return slices.Contains([]string{"border-t", "border-b"}, longhand)
	}
	return false
}
