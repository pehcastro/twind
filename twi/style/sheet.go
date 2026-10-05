package style

import (
	"cmp"
	"fmt"
	"math"
	"math/bits"
	"slices"
	"sync/atomic"
	"time"

	konst "github.com/pehcastro/twind/internal/konst/style"
	"github.com/pehcastro/twind/twi/color"
	"github.com/pehcastro/twind/twi/theme"
)

type Property uint8

const (
	PropDisplay Property = iota + 1
	PropDirection
	PropGrow
	PropShrink
	PropBasis
	PropAlignItems
	PropAlignSelf
	PropJustify
	PropRowGap
	PropColumnGap
	PropWidth
	PropHeight
	PropMinWidth
	PropMaxWidth
	PropMinHeight
	PropMaxHeight
	PropPaddingTop
	PropPaddingRight
	PropPaddingBottom
	PropPaddingLeft
	PropMarginTop
	PropMarginRight
	PropMarginBottom
	PropMarginLeft
	PropPosition
	PropTop
	PropRight
	PropBottom
	PropLeft
	PropOverflowX
	PropOverflowY
	PropZIndex
	PropTranslateX
	PropTranslateY
	PropBorderTopWidth
	PropBorderRightWidth
	PropBorderBottomWidth
	PropBorderLeftWidth
	PropBorderStyle
	PropBorderColor
	PropRadius
	PropBackground
	PropOpacity
	PropColor
	PropBold
	PropItalic
	PropUnderline
	PropStrikethrough
	PropTextAlign
	PropVisibility
	PropCursor
	PropUserSelect
	PropShadow
	PropInsetShadow
	PropGradient
	PropGradientFrom
	PropGradientVia
	PropGradientTo
	PropGradientFromPosition
	PropGradientViaPosition
	PropGradientToPosition
	PropShadowColor
	PropInsetShadowColor
	PropRingWidth
	PropRingColor
	PropRingInset
	PropRingOffsetWidth
	PropRingOffsetColor
	PropWhiteSpace
	PropTextOverflow
	PropAspectRatio
	PropTransitionProperty
	PropTransitionDuration
	PropTransitionDelay
	PropTransitionEasing
	PropAnimationKeyframes
	PropAnimationDuration
	PropAnimationEasing
	PropAnimationIterations
	PropWrap
	PropGridColumns
	PropGridRows
	PropGridAutoColumns
	PropGridAutoRows
	PropGridColumnStart
	PropGridColumnEnd
	PropGridRowStart
	PropGridRowEnd
	PropGridFlow
	PropJustifyItems
	PropJustifySelf
	PropAlignContent
	PropPointerEvents
	PropOverflowWrap
	PropWordBreak
	PropAnimationDelay
	PropAnimationFill
	PropScaleX
	PropScaleY
	PropEnterOpacity
	PropEnterScale
	PropEnterTranslateX
	PropEnterTranslateY
	PropEnterDegrees
	PropExitOpacity
	PropExitScale
	PropExitTranslateX
	PropExitTranslateY
	PropExitDegrees
	PropTailwindDuration
	PropTailwindEasing
	PropTailwindAnimationDuration
	PropTailwindAnimationDelay
	PropTailwindAnimationIterations
	PropTailwindAnimationFill
	PropRadiusTopLeft
	PropRadiusTopRight
	PropRadiusBottomRight
	PropRadiusBottomLeft
	propEnd
	propPadding
	propMargin
	propInset
	propBorderWidth
	propGap
	propFused
)

type Declaration struct {
	Property     Property
	Length       Length
	Number       float64
	Flag         bool
	Color        color.Color
	Token        theme.Token
	Mix          float64
	Display      Display
	Direction    Direction
	Wrap         Wrapping
	Align        Align
	Justify      Justify
	Position     Position
	Overflow     Overflow
	BorderStyle  BorderStyle
	Radius       Radius
	TextAlign    TextAlign
	Visibility   Visibility
	Cursor       Cursor
	UserSelect   UserSelect
	Shadows      []Shadow
	Line         GradientLine
	WhiteSpace   WhiteSpace
	TextOverflow TextOverflow
	OverflowWrap OverflowWrap
	WordBreak    WordBreak
	Pointer      PointerEvents
	Transition   TransitionProperty
	Keyframes    Keyframes
	Duration     time.Duration
	Easing       Easing
	Fill         Fill
	Fallback     bool
	Tracks       []Track
	GridLine     GridLine
	Flow         GridFlow
}

type State uint8

const (
	StateHover State = 1 << iota
	StateFocus
	StateFocusVisible
	StateActive
	StateDisabled
	StateFocusWithin
	StateChecked
)

type Place uint8

const (
	PlaceFirst Place = 1 << iota
	PlaceLast
	PlaceOdd
	PlaceEven
)

type Negation struct {
	States State
	Attrs  []Attr
	Places Place
}

type Part uint8

const (
	PartNode Part = iota
	PartPlaceholder
	PartSelection
)

type NodeState struct {
	States State
	Attrs  []Attr
	Places Place
}

type Attr struct {
	Name     string
	Value    string
	AnyValue bool
}

type Scheme uint8

const (
	SchemeAny Scheme = iota
	SchemeLight
	SchemeDark
)

type Condition struct {
	States    State
	Attrs     []Attr
	MinCols   int
	BelowCols int
	Scheme    Scheme
	Places    Place
	Not       Negation
}

type Rule struct {
	Class  string
	When   Condition
	Decls  []Declaration
	Target Match
	Near   Match
	Part   Part
}

type Sheet struct {
	rules     []Rule
	steps     []step
	start     *ComputedStyle
	universal []step
	theme     *theme.Theme
	bounds    []int
	columns   int
	shaded    *shadeCache
	index
}

type gate uint8

const (
	gateOpen gate = iota
	gateBand
	gateNode
)

type step struct {
	decls []Declaration
	when  *Condition
	rank  int16
	part  Part
	gate  gate
	need  State
	twin  theme.Token
	bare  bool
}

type key struct{ head, next, prev, tail uint64 }

type slot struct {
	key  key
	size int32
	id   int32
	own  []step
}

type class struct {
	name        string
	parts       []step
	near, hands []int
	mark        Markers
}

type index struct {
	classes []class
	slots   []slot
}

func (x *index) find(name string) (*slot, key) {
	var k key
	switch n := len(name); {
	case len(x.slots) == 0:
		return nil, k
	case n > konst.KeyBytes/2:
		k.next, k.prev = le64(name[8:]), le64(name[n-konst.KeyBytes/2:])
		fallthrough
	case n >= 8:
		k.head, k.tail = le64(name), le64(name[n-8:])
	case n >= 4:
		k.head, k.tail = le32(name), le32(name[n-4:])
	case n > 0:
		k.head = uint64(name[0])<<16 | uint64(name[n/2])<<8 | uint64(name[n-1])
	}
	mask := uint64(len(x.slots) - 1)
	for i := mix(k.head^konst.HashA^uint64(len(name)), k.tail^konst.HashB) & mask; ; i = (i + 1) & mask {
		sl := &x.slots[i]
		if sl.id == 0 || sl.key == k && int(sl.size) == len(name) && (len(name) <= konst.KeyBytes || middle(x.classes[sl.id-1].name, name)) {
			return sl, k
		}
	}
}

func middle(a, b string) bool {
	for i := konst.KeyBytes / 2; i < len(b)-konst.KeyBytes/2; i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (x *index) class(name string) *class {
	if sl, _ := x.find(name); sl != nil && sl.id != 0 {
		return &x.classes[sl.id-1]
	}
	return nil
}

func mix(a, b uint64) uint64 {
	hi, lo := bits.Mul64(a, b)
	return hi ^ lo
}

func le64(s string) uint64 {
	_ = s[7]
	return uint64(s[0]) | uint64(s[1])<<8 | uint64(s[2])<<16 | uint64(s[3])<<24 | uint64(s[4])<<32 | uint64(s[5])<<40 | uint64(s[6])<<48 | uint64(s[7])<<56
}

func le32(s string) uint64 {
	_ = s[3]
	return uint64(s[0]) | uint64(s[1])<<8 | uint64(s[2])<<16 | uint64(s[3])<<24
}

type shading struct {
	shadows, inset *Shadow
	counts         [2]int
	tints          [2]color.Color
	ring           Ring
	theme          *theme.Theme
}

type shade struct {
	key    shading
	tokens theme.Tokens
	out    [2][]Shadow
}

type shadeCache [konst.ShadeEntries]atomic.Pointer[shade]

func (s Sheet) WithColumns(columns int) Sheet {
	s.columns = columns
	return s
}

func (s Sheet) Band(columns int) int {
	band, _ := slices.BinarySearch(s.bounds, columns+1)
	return band
}

func (s Sheet) Responsive(classes []string) bool {
	bounded := func(st step) bool { return st.when.MinCols != 0 || st.when.BelowCols != 0 }
	rule := func(i int) bool { return bounded(s.steps[i]) }
	return slices.ContainsFunc(s.universal, bounded) || slices.ContainsFunc(classes, func(name string) bool {
		sl, _ := s.find(name)
		if sl == nil || sl.id == 0 {
			return false
		}
		c := &s.classes[sl.id-1]
		return slices.ContainsFunc(sl.own, bounded) || slices.ContainsFunc(c.near, rule) || slices.ContainsFunc(c.hands, rule) || slices.ContainsFunc(c.parts, bounded)
	})
}

func (s Sheet) Marks(classes []string) Markers {
	var marks Markers
	for _, name := range classes {
		if c := s.class(name); c != nil {
			marks |= c.mark
		}
	}
	return marks
}

func (s Sheet) Near(classes []string, into []int) []int {
	for _, name := range classes {
		if c := s.class(name); c != nil {
			into = append(into, c.near...)
		}
	}
	return into
}

func (s Sheet) Hands(classes []string, node NodeState, into []int) []int {
	scheme := s.scheme()
	for _, name := range classes {
		c := s.class(name)
		if c == nil {
			continue
		}
		for _, i := range c.hands {
			if when := &s.rules[i].When; s.fits(when, scheme) && node.holds(when.States, when.Attrs, when.Places, &when.Not) {
				into = append(into, i)
			}
		}
	}
	return into
}

func (s Sheet) Rule(i int) *Rule { return &s.rules[i] }

func (m *Match) Accepts(element Element, marks Markers, node NodeState) bool {
	return (m.Element == ElementAny || m.Element == element) && marks&m.mark == m.mark && node.holds(m.States, m.Attrs, m.Places, &m.Not)
}

func (s Sheet) WithTheme(t *theme.Theme) Sheet {
	s.theme = t
	return s
}

type VersionError struct{ Got int }

func (e VersionError) Error() string {
	return fmt.Sprintf("style: IR version %d, runtime reads %d", e.Got, konst.IRVersion)
}

func NewSheet(version int, rules []Rule) (Sheet, error) {
	if version != konst.IRVersion {
		return Sheet{}, VersionError{Got: version}
	}
	if len(rules) > konst.MaxRules {
		return Sheet{}, fmt.Errorf("style: more than %d rules", konst.MaxRules)
	}
	s := Sheet{
		rules:  slices.Clone(rules),
		shaded: &shadeCache{},
		steps:  make([]step, len(rules)),
	}
	ids := map[string]int{}
	var owns [][]step
	entry := func(name string) int {
		if _, ok := ids[name]; !ok {
			ids[name] = len(s.classes)
			s.classes = append(s.classes, class{name: name})
			owns = append(owns, nil)
		}
		return ids[name]
	}
	start, baked, marked := initial(), 0, 0
	var early winners
	for i := range s.rules {
		r := &s.rules[i]
		if slices.ContainsFunc(r.Decls, func(d Declaration) bool { return d.Property == 0 || d.Property >= propEnd }) {
			return Sheet{}, fmt.Errorf("style: rule %d declares an unknown property", i)
		}
		w := &r.When
		st := step{decls: fused(r.Decls), when: w, rank: int16(i + 1), part: r.Part}
		switch {
		case r.Target.Relation != RelationSelf:
		case len(w.Attrs) > 0 || w.Places != 0 || w.Not.States != 0 || len(w.Not.Attrs) > 0 || w.Not.Places != 0:
			st.gate, st.need = gateNode, w.States
		default:
			st.need = w.States
		}
		if st.gate == gateOpen && (w.MinCols != 0 || w.BelowCols != 0 || w.Scheme != SchemeAny) {
			st.gate = gateBand
		}
		st.bare = st.gate == gateOpen && st.need == 0 && st.part == PartNode
		s.steps[i] = st
		for _, m := range [...]*Match{&r.Near, &r.Target} {
			if m.Class == "" {
				continue
			}
			c := &s.classes[entry(m.Class)]
			if c.mark == 0 {
				if marked == konst.MaxMarkers {
					return Sheet{}, fmt.Errorf("style: more than %d group, peer and target classes", konst.MaxMarkers)
				}
				c.mark = 1 << marked
				marked++
			}
			m.mark = c.mark
		}
		switch {
		case r.Target.Relation != RelationSelf:
			c := &s.classes[entry(r.Class)]
			c.hands = append(c.hands, i)
		case r.Near.Relation != RelationSelf:
			c := &s.classes[entry(r.Class)]
			c.near = append(c.near, i)
		case r.Part != PartNode:
			c := &s.classes[entry(r.Class)]
			c.parts = append(c.parts, st)
		case r.Class != "":
			k := entry(r.Class)
			st.twin = st.pairs(owns[k])
			owns[k] = append(owns[k], st)
		case baked == i && st.bare && !slices.ContainsFunc(r.Decls, func(d Declaration) bool { return d.Token != 0 || d.Property.inherited() }):
			start.apply(r.Decls, st.rank, &early, nil, &color.Color{})
			baked++
		default:
			st.twin = st.pairs(s.universal)
			s.universal = append(s.universal, st)
		}
		for _, bound := range [...]int{r.When.MinCols, r.When.BelowCols} {
			if bound != 0 {
				s.bounds = append(s.bounds, bound)
			}
		}
	}
	slices.Sort(s.bounds)
	s.bounds = slices.Compact(s.bounds)
	s.start = &start
	if len(s.classes) > 0 {
		s.slots = make([]slot, 1<<bits.Len(uint(len(s.classes)*konst.ClassSlots)))
	}
	for k, c := range s.classes {
		sl, key := s.find(c.name)
		*sl = slot{key: key, size: int32(len(c.name)), id: int32(k + 1), own: owns[k]}
	}
	return s, nil
}

func fused(decls []Declaration) []Declaration {
	var out []Declaration
	for i := 0; i < len(decls); i++ {
		d := decls[i]
		for _, g := range [...]struct {
			first, fused Property
			count        int
		}{
			{PropPaddingTop, propPadding, 4},
			{PropMarginTop, propMargin, 4},
			{PropTop, propInset, 4},
			{PropBorderTopWidth, propBorderWidth, 4},
			{PropRowGap, propGap, 2},
		} {
			run := decls[i:min(i+g.count, len(decls))]
			same := len(run) == g.count
			for j := range run {
				same = same && run[j].Property == g.first+Property(j) && run[j].Length == d.Length
			}
			if same {
				d, i = Declaration{Property: g.fused, Length: d.Length}, i+g.count-1
				break
			}
		}
		out = append(out, d)
	}
	return out
}

func (p Property) inherited() bool {
	switch p {
	case PropColor, PropBold, PropItalic, PropUnderline, PropStrikethrough, PropTextAlign, PropVisibility, PropCursor, PropUserSelect, PropWhiteSpace, PropOverflowWrap, PropWordBreak, PropPointerEvents:
		return true
	}
	return false
}

func initial() ComputedStyle {
	ease := Easing{X1: konst.EaseX1, Y1: konst.EaseY1, X2: konst.EaseX2, Y2: konst.EaseY2}
	still := Pose{Opacity: 1, Scale: 1}
	return ComputedStyle{
		Shrink:       1,
		ScaleX:       1,
		ScaleY:       1,
		AlignItems:   AlignStretch,
		JustifyItems: AlignStretch,
		Justify:      JustifyStretch,
		AlignContent: JustifyStretch,
		Basis:        Length{Unit: Auto},
		Width:        Length{Unit: Auto},
		Height:       Length{Unit: Auto},
		MinWidth:     Length{Unit: Auto},
		MinHeight:    Length{Unit: Auto},
		MaxWidth:     Length{Unit: None},
		MaxHeight:    Length{Unit: None},
		Inset:        Edges{Length{Unit: Auto}, Length{Unit: Auto}, Length{Unit: Auto}, Length{Unit: Auto}},
		BorderColor:  color.Color{Kind: color.Current},
		Opacity:      1,
		Gradient: Gradient{
			From: GradientStop{Color: color.Color{Kind: color.Literal}, Position: konst.FromPosition},
			Via:  GradientStop{Color: color.Color{Kind: color.Literal}, Position: konst.ViaPosition},
			To:   GradientStop{Color: color.Color{Kind: color.Literal}, Position: konst.ToPosition},
		},
		Ring: Ring{
			Color:       color.Color{Kind: color.Current},
			OffsetColor: color.Color{Kind: color.Literal, RGBA: color.RGBA{R: konst.RingOffsetWhite, G: konst.RingOffsetWhite, B: konst.RingOffsetWhite, A: konst.RingOffsetWhite}},
		},
		Transition: Transition{Properties: TransitionAll, Easing: ease},
		Animation:  Animation{Iterations: 1, Easing: ease, Enter: still, Exit: still},
	}
}

func (n NodeState) holds(states State, attrs []Attr, places Place, not *Negation) bool {
	if states&^n.States != 0 || places&^n.Places != 0 || not.States&n.States != 0 || not.Places != 0 && (n.Places == 0 || not.Places&n.Places != 0) || len(attrs) > 0 && len(n.Attrs) == 0 {
		return false
	}
	for _, want := range attrs {
		if !n.carries(want) {
			return false
		}
	}
	for _, unwanted := range not.Attrs {
		if n.carries(unwanted) {
			return false
		}
	}
	return true
}

func (n NodeState) carries(want Attr) bool {
	return slices.ContainsFunc(n.Attrs, func(a Attr) bool { return a.Name == want.Name && (want.AnyValue || a.Value == want.Value) })
}

func themed(t *theme.Theme, c color.Color, token theme.Token, mix float64) color.Color {
	if token == 0 || t == nil || t.Tokens[token].Kind == color.Unset {
		return c
	}
	c = t.Tokens[token]
	if mix != konst.OpaquePercent {
		c.RGBA.A = uint8(math.Round(float64(c.RGBA.A) * mix / konst.OpaquePercent))
	}
	return c
}

func (s Sheet) Compute(parent ComputedStyle, classes []string) (out ComputedStyle) {
	s.compute(&out, PartNode, &parent, classes, &NodeState{}, nil)
	return out
}

func (s Sheet) ComputeState(parent ComputedStyle, classes []string, node NodeState) (out ComputedStyle) {
	s.compute(&out, PartNode, &parent, classes, &node, nil)
	return out
}

func (s *Sheet) scheme() Scheme {
	switch {
	case s.theme == nil:
		return SchemeAny
	case s.theme.Scheme == theme.Dark:
		return SchemeDark
	}
	return SchemeLight
}

func (s *Sheet) fits(when *Condition, scheme Scheme) bool {
	narrow, wide := when.MinCols > s.columns, when.BelowCols != 0 && s.columns >= when.BelowCols
	return !narrow && !wide && (when.Scheme == SchemeAny || when.Scheme == scheme)
}

func PlaceOf(index, count int) Place {
	place := PlaceOdd
	if index%2 == 1 {
		place = PlaceEven
	}
	if index == 0 {
		place |= PlaceFirst
	}
	if index == count-1 {
		place |= PlaceLast
	}
	return place
}

func (s Sheet) ComputeRelated(parent ComputedStyle, classes []string, node NodeState, related []int) (out ComputedStyle) {
	s.compute(&out, PartNode, &parent, classes, &node, related)
	return out
}

func (s Sheet) ComputePart(part Part, origin ComputedStyle, classes []string, node NodeState, related []int) (out ComputedStyle) {
	s.compute(&out, part, &origin, classes, &node, related)
	return out
}

type winners [propFused]int16

func (s *Sheet) compute(out *ComputedStyle, part Part, parent *ComputedStyle, classes []string, node *NodeState, related []int) {
	if s.start == nil {
		*out = initial()
	} else {
		*out = *s.start
	}
	out.Color, out.Bold, out.Italic, out.Underline, out.Strikethrough = parent.Color, parent.Bold, parent.Italic, parent.Underline, parent.Strikethrough
	out.TextAlign, out.Visibility, out.Cursor, out.UserSelect = parent.TextAlign, parent.Visibility, parent.Cursor, parent.UserSelect
	out.WhiteSpace, out.OverflowWrap, out.WordBreak, out.PointerEvents = parent.WhiteSpace, parent.OverflowWrap, parent.WordBreak, parent.PointerEvents
	var won winners
	scheme, steps := s.scheme(), s.universal
	if part != PartNode {
		steps = nil
	}
	for n := 0; ; n++ {
		for k := range steps {
			st := &steps[k]
			switch {
			case st.bare && part == PartNode:
			case st.need&^node.States != 0, st.twin != 0 && s.theme != nil && s.theme.Tokens[st.twin].Kind != color.Unset, !s.passes(st, part, scheme, node):
				continue
			}
			out.apply(st.decls, st.rank, &won, s.theme, &parent.Color)
		}
		switch {
		case n < len(classes):
			steps = nil
			switch sl, _ := s.find(classes[n]); {
			case sl == nil:
			case part == PartNode:
				steps = sl.own
			case sl.id != 0:
				steps = s.classes[sl.id-1].parts
			}
		case n < len(classes)+len(related):
			i := related[n-len(classes)]
			steps = s.steps[i : i+1]
		default:
			if out.Shadows != nil || out.InsetShadows != nil || out.Ring.Width > 0 {
				s.finish(out)
			}
			if out.Animation.Keyframes != KeyframesNone {
				s.settle(out, &won)
			}
			return
		}
	}
}

func (s *Sheet) passes(st *step, part Part, scheme Scheme, node *NodeState) bool {
	w := st.when
	return st.part == part && (st.gate == gateOpen || s.fits(w, scheme) && (st.gate == gateBand || node.holds(w.States, w.Attrs, w.Places, &w.Not)))
}

func (st *step) pairs(list []step) theme.Token {
	if len(list) == 0 || len(st.decls) != 1 || st.gate != gateBand || st.when.MinCols != 0 || st.when.BelowCols != 0 {
		return 0
	}
	base, d := &list[len(list)-1], &st.decls[0]
	if base.gate != gateOpen || base.need != st.need || base.rank != st.rank-1 || len(base.decls) != 1 || base.decls[0].Property != d.Property || base.decls[0].Token != d.Token || base.decls[0].Mix != d.Mix {
		return 0
	}
	return d.Token
}

func (s *Sheet) settle(out *ComputedStyle, ranks *winners) {
	won := func(p Property) *Declaration {
		if ranks[p] == 0 {
			return nil
		}
		decls := s.rules[ranks[p]-1].Decls
		for i := len(decls) - 1; ; i-- {
			if decls[i].Property == p {
				return &decls[i]
			}
		}
	}
	for _, read := range [...]struct {
		longhand  Property
		variables [2]Property
	}{
		{PropAnimationDuration, [2]Property{PropTailwindAnimationDuration, PropTailwindDuration}},
		{PropAnimationEasing, [2]Property{PropTailwindEasing}},
		{PropAnimationDelay, [2]Property{PropTailwindAnimationDelay}},
		{PropAnimationIterations, [2]Property{PropTailwindAnimationIterations}},
		{PropAnimationFill, [2]Property{PropTailwindAnimationFill}},
	} {
		if d := won(read.longhand); d == nil || !d.Fallback {
			continue
		}
		if v := cmp.Or(won(read.variables[0]), won(read.variables[1])); v != nil {
			set := [1]Declaration{*v}
			set[0].Property = read.longhand
			out.apply(set[:], 0, &winners{}, nil, &color.Color{})
		}
	}
}

func (s *Sheet) finish(out *ComputedStyle) {
	tokened := func(sh Shadow) bool { return sh.Token != 0 }
	themed := s.theme != nil && (slices.ContainsFunc(out.Shadows, tokened) || slices.ContainsFunc(out.InsetShadows, tokened))
	if out.Ring.Width <= 0 && !themed && out.ShadowColor.Kind == color.Unset && out.InsetShadowColor.Kind == color.Unset {
		out.Shadows, out.InsetShadows = slices.Clip(out.Shadows), slices.Clip(out.InsetShadows)
		return
	}
	key := shading{counts: [2]int{len(out.Shadows), len(out.InsetShadows)}, tints: [2]color.Color{out.ShadowColor, out.InsetShadowColor}, ring: out.Ring}
	if len(out.Shadows) > 0 {
		key.shadows = &out.Shadows[0]
	}
	if len(out.InsetShadows) > 0 {
		key.inset = &out.InsetShadows[0]
	}
	if themed {
		key.theme = s.theme
	}
	way := key.hash()
	for i := range uint64(konst.ShadeWays) {
		if done := s.shaded[(way+i)%konst.ShadeEntries].Load(); done != nil && done.key == key && (!themed || done.tokens == s.theme.Tokens) {
			out.Shadows, out.InsetShadows = done.out[0], done.out[1]
			return
		}
	}
	out.Shadows = s.shade(out.Shadows, out.ShadowColor)
	out.InsetShadows = s.shade(out.InsetShadows, out.InsetShadowColor)
	if r := out.Ring; r.Width > 0 {
		var ring []Shadow
		if r.OffsetWidth > 0 {
			ring = append(ring, Shadow{Spread: r.OffsetWidth, Color: r.OffsetColor, Inset: r.Inset})
		}
		ring = append(ring, Shadow{Spread: r.Width + r.OffsetWidth, Color: r.Color, Inset: r.Inset})
		if r.Inset {
			out.InsetShadows = slices.Concat(out.InsetShadows, ring)
		} else {
			out.Shadows = slices.Concat(ring, out.Shadows)
		}
	}
	out.Shadows, out.InsetShadows = slices.Clip(out.Shadows), slices.Clip(out.InsetShadows)
	done := &shade{key: key, out: [2][]Shadow{out.Shadows, out.InsetShadows}}
	if themed {
		done.tokens = s.theme.Tokens
	}
	free := way
	for i := range uint64(konst.ShadeWays) {
		if s.shaded[(way+i)%konst.ShadeEntries].Load() == nil {
			free = way + i
			break
		}
	}
	s.shaded[free%konst.ShadeEntries].Store(done)
}

func (k *shading) hash() uint64 {
	h := mix(packed(k.ring.Color)^uint64(k.ring.Width+k.ring.OffsetWidth)^konst.HashA, packed(k.tints[0])^packed(k.tints[1])^uint64(k.counts[0]+k.counts[1])^konst.HashB)
	for _, sh := range [...]*Shadow{k.shadows, k.inset} {
		if sh != nil {
			h = mix(h^uint64(sh.X)^packed(sh.Color), uint64(sh.Y+sh.Blur+sh.Spread)^konst.HashB)
		}
	}
	return h
}

func packed(c color.Color) uint64 {
	return uint64(c.Kind)<<32 | uint64(c.RGBA.R)<<24 | uint64(c.RGBA.G)<<16 | uint64(c.RGBA.B)<<8 | uint64(c.RGBA.A)
}

func (s *Sheet) shade(shadows []Shadow, tint color.Color) []Shadow {
	if tint.Kind == color.Unset && (s.theme == nil || !slices.ContainsFunc(shadows, func(sh Shadow) bool { return sh.Token != 0 })) {
		return shadows
	}
	var out []Shadow
	for _, sh := range shadows {
		sh.Color = themed(s.theme, sh.Color, sh.Token, sh.Mix)
		if sh.Tintable && tint.Kind != color.Unset {
			sh.Color = tint
		}
		if sh.Color.Kind != color.Literal || sh.Color.RGBA.A > 0 {
			out = append(out, sh)
		}
	}
	return out
}

func (w *winners) sides(top Property, rank int16, e *Edges, l Length) {
	if w[top] <= rank {
		w[top], e.Top = rank, l
	}
	if w[top+1] <= rank {
		w[top+1], e.Right = rank, l
	}
	if w[top+2] <= rank {
		w[top+2], e.Bottom = rank, l
	}
	if w[top+3] <= rank {
		w[top+3], e.Left = rank, l
	}
}

func (s *ComputedStyle) apply(decls []Declaration, rank int16, won *winners, t *theme.Theme, inherited *color.Color) {
	for i := range decls {
		d := &decls[i]
		if won[d.Property] > rank {
			continue
		}
		won[d.Property] = rank
		switch d.Property {
		case PropRadius:
			for p := PropRadiusTopLeft; p <= PropRadiusBottomLeft; p++ {
				if won[p] <= rank {
					won[p] = rank
					s.Radius = s.Radius.With(Corner(p-PropRadiusTopLeft), d.Radius)
				}
			}
		case propPadding:
			won.sides(PropPaddingTop, rank, &s.Padding, d.Length)
		case propMargin:
			won.sides(PropMarginTop, rank, &s.Margin, d.Length)
		case propInset:
			won.sides(PropTop, rank, &s.Inset, d.Length)
		case propBorderWidth:
			won.sides(PropBorderTopWidth, rank, &s.BorderWidth, d.Length)
		case propGap:
			if won[PropRowGap] <= rank {
				won[PropRowGap], s.RowGap = rank, d.Length
			}
			if won[PropColumnGap] <= rank {
				won[PropColumnGap], s.ColumnGap = rank, d.Length
			}
		case PropDisplay:
			s.Display = d.Display
		case PropDirection:
			s.Direction = d.Direction
		case PropWrap:
			s.Wrap = d.Wrap
		case PropGrow:
			s.Grow = d.Number
		case PropShrink:
			s.Shrink = d.Number
		case PropBasis:
			s.Basis = d.Length
		case PropAlignItems:
			s.AlignItems = d.Align
		case PropAlignSelf:
			s.AlignSelf = d.Align
		case PropJustify:
			s.Justify = d.Justify
		case PropAlignContent:
			s.AlignContent = d.Justify
		case PropJustifyItems:
			s.JustifyItems = d.Align
		case PropJustifySelf:
			s.JustifySelf = d.Align
		case PropGridColumns:
			s.GridColumns = d.Tracks
		case PropGridRows:
			s.GridRows = d.Tracks
		case PropGridAutoColumns:
			s.GridAutoColumns = d.Tracks
		case PropGridAutoRows:
			s.GridAutoRows = d.Tracks
		case PropGridColumnStart:
			s.GridColumn.Start = d.GridLine
		case PropGridColumnEnd:
			s.GridColumn.End = d.GridLine
		case PropGridRowStart:
			s.GridRow.Start = d.GridLine
		case PropGridRowEnd:
			s.GridRow.End = d.GridLine
		case PropGridFlow:
			s.GridFlow = d.Flow
		case PropRowGap:
			s.RowGap = d.Length
		case PropColumnGap:
			s.ColumnGap = d.Length
		case PropWidth:
			s.Width = d.Length
		case PropHeight:
			s.Height = d.Length
		case PropMinWidth:
			s.MinWidth = d.Length
		case PropMaxWidth:
			s.MaxWidth = d.Length
		case PropMinHeight:
			s.MinHeight = d.Length
		case PropMaxHeight:
			s.MaxHeight = d.Length
		case PropPaddingTop:
			s.Padding.Top = d.Length
		case PropPaddingRight:
			s.Padding.Right = d.Length
		case PropPaddingBottom:
			s.Padding.Bottom = d.Length
		case PropPaddingLeft:
			s.Padding.Left = d.Length
		case PropMarginTop:
			s.Margin.Top = d.Length
		case PropMarginRight:
			s.Margin.Right = d.Length
		case PropMarginBottom:
			s.Margin.Bottom = d.Length
		case PropMarginLeft:
			s.Margin.Left = d.Length
		case PropPosition:
			s.Position = d.Position
		case PropTop:
			s.Inset.Top = d.Length
		case PropRight:
			s.Inset.Right = d.Length
		case PropBottom:
			s.Inset.Bottom = d.Length
		case PropLeft:
			s.Inset.Left = d.Length
		case PropOverflowX:
			s.OverflowX = d.Overflow
		case PropOverflowY:
			s.OverflowY = d.Overflow
		case PropZIndex:
			s.ZIndex = int(d.Number)
		case PropTranslateX:
			s.TranslateX = d.Length
		case PropTranslateY:
			s.TranslateY = d.Length
		case PropBorderTopWidth:
			s.BorderWidth.Top = d.Length
		case PropBorderRightWidth:
			s.BorderWidth.Right = d.Length
		case PropBorderBottomWidth:
			s.BorderWidth.Bottom = d.Length
		case PropBorderLeftWidth:
			s.BorderWidth.Left = d.Length
		case PropBorderStyle:
			s.BorderStyle = d.BorderStyle
		case PropBorderColor:
			s.BorderColor = themed(t, d.Color, d.Token, d.Mix)
		case PropRadiusTopLeft, PropRadiusTopRight, PropRadiusBottomRight, PropRadiusBottomLeft:
			s.Radius = s.Radius.With(Corner(d.Property-PropRadiusTopLeft), d.Radius)
		case PropBackground:
			s.Background = themed(t, d.Color, d.Token, d.Mix)
		case PropOpacity:
			s.Opacity = d.Number
		case PropColor:
			s.Color = themed(t, d.Color, d.Token, d.Mix)
			if s.Color.Kind == color.Current {
				s.Color = *inherited
			}
		case PropBold:
			s.Bold = d.Flag
		case PropItalic:
			s.Italic = d.Flag
		case PropUnderline:
			s.Underline = d.Flag
		case PropStrikethrough:
			s.Strikethrough = d.Flag
		case PropTextAlign:
			s.TextAlign = d.TextAlign
		case PropVisibility:
			s.Visibility = d.Visibility
		case PropCursor:
			s.Cursor = d.Cursor
		case PropUserSelect:
			s.UserSelect = d.UserSelect
		case PropShadow:
			s.Shadows = d.Shadows
		case PropInsetShadow:
			s.InsetShadows = d.Shadows
		case PropShadowColor:
			s.ShadowColor = themed(t, d.Color, d.Token, d.Mix)
		case PropInsetShadowColor:
			s.InsetShadowColor = themed(t, d.Color, d.Token, d.Mix)
		case PropGradient:
			s.Gradient.GradientLine = d.Line
		case PropGradientFrom:
			s.Gradient.From.Color = themed(t, d.Color, d.Token, d.Mix)
		case PropGradientVia:
			s.Gradient.Via.Color = themed(t, d.Color, d.Token, d.Mix)
			s.Gradient.HasVia = true
		case PropGradientTo:
			s.Gradient.To.Color = themed(t, d.Color, d.Token, d.Mix)
		case PropGradientFromPosition:
			s.Gradient.From.Position = d.Number
		case PropGradientViaPosition:
			s.Gradient.Via.Position = d.Number
		case PropGradientToPosition:
			s.Gradient.To.Position = d.Number
		case PropRingWidth:
			s.Ring.Width = Pixels(d.Number)
		case PropRingColor:
			s.Ring.Color = themed(t, d.Color, d.Token, d.Mix)
		case PropRingInset:
			s.Ring.Inset = d.Flag
		case PropRingOffsetWidth:
			s.Ring.OffsetWidth = Pixels(d.Number)
		case PropRingOffsetColor:
			s.Ring.OffsetColor = themed(t, d.Color, d.Token, d.Mix)
		case PropWhiteSpace:
			s.WhiteSpace = d.WhiteSpace
		case PropTextOverflow:
			s.TextOverflow = d.TextOverflow
		case PropOverflowWrap:
			s.OverflowWrap = d.OverflowWrap
		case PropWordBreak:
			s.WordBreak = d.WordBreak
		case PropPointerEvents:
			s.PointerEvents = d.Pointer
		case PropAspectRatio:
			s.AspectRatio = d.Number
		case PropTransitionProperty:
			s.Transition.Properties = d.Transition
		case PropTransitionDuration:
			s.Transition.Duration = d.Duration
		case PropTransitionDelay:
			s.Transition.Delay = d.Duration
		case PropTransitionEasing:
			s.Transition.Easing = d.Easing
		case PropAnimationKeyframes:
			s.Animation.Keyframes = d.Keyframes
		case PropAnimationDuration:
			s.Animation.Duration = d.Duration
		case PropAnimationEasing:
			s.Animation.Easing = d.Easing
		case PropAnimationIterations:
			s.Animation.Iterations, s.Animation.Infinite = d.Number, d.Flag
		case PropAnimationDelay:
			s.Animation.Delay = d.Duration
		case PropAnimationFill:
			s.Animation.Fill = d.Fill
		case PropScaleX:
			s.ScaleX = d.Number
		case PropScaleY:
			s.ScaleY = d.Number
		case PropEnterOpacity:
			s.Animation.Enter.Opacity = d.Number
		case PropEnterScale:
			s.Animation.Enter.Scale = d.Number
		case PropEnterTranslateX:
			s.Animation.Enter.TranslateX = d.Length
		case PropEnterTranslateY:
			s.Animation.Enter.TranslateY = d.Length
		case PropEnterDegrees:
			s.Animation.Enter.Degrees = d.Number
		case PropExitOpacity:
			s.Animation.Exit.Opacity = d.Number
		case PropExitScale:
			s.Animation.Exit.Scale = d.Number
		case PropExitTranslateX:
			s.Animation.Exit.TranslateX = d.Length
		case PropExitTranslateY:
			s.Animation.Exit.TranslateY = d.Length
		case PropExitDegrees:
			s.Animation.Exit.Degrees = d.Number
		case PropTailwindDuration, PropTailwindEasing, PropTailwindAnimationDuration, PropTailwindAnimationDelay, PropTailwindAnimationIterations, PropTailwindAnimationFill:
		default:
			panic(fmt.Sprintf("style: unknown property %d", d.Property))
		}
	}
}
