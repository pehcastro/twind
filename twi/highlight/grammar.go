package highlight

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"unicode/utf8"

	konst "github.com/twind-dev/twind/internal/konst/highlight"
)

type step uint8

const (
	stay step = iota
	push
	jump
	pop
)

type runeRange struct{ lo, hi rune }

type Rule struct {
	literals []string
	ranges   []runeRange
	fallback bool
	until    string
	escape   string
	oneLine  bool
	kind     Kind
	boundary bool
	step     step
	target   string
}

func Match(kind Kind, literals ...string) Rule { return Rule{kind: kind, literals: literals} }

func Word(kind Kind, words ...string) Rule { return Rule{kind: kind, literals: words, boundary: true} }

func Within(kind Kind, start, end string) Rule {
	return Rule{kind: kind, literals: []string{start}, until: end}
}

func Fallback(kind Kind) Rule { return Rule{kind: kind, fallback: true} }

func (r Rule) Range(lo, hi rune) Rule {
	r.ranges = append(slices.Clip(r.ranges), runeRange{lo, hi})
	return r
}

func (r Rule) Letters() Rule { return r.Range('a', 'z').Range('A', 'Z') }

func (r Rule) Digits() Rule { return r.Range('0', '9') }

func (r Rule) HexDigits() Rule { return r.Digits().Range('a', 'f').Range('A', 'F') }

func (r Rule) Escaped(escape string) Rule {
	r.escape = escape
	return r
}

func (r Rule) OneLine() Rule {
	r.oneLine = true
	return r
}

func (r Rule) Enter(state string) Rule {
	r.step, r.target = push, state
	return r
}

func (r Rule) Goto(state string) Rule {
	r.step, r.target = jump, state
	return r
}

func (r Rule) Leave() Rule {
	r.step, r.target = pop, ""
	return r
}

type State struct {
	Name  string
	Rules []Rule
	Probe bool
	AtEnd string
}

type Definition struct {
	States []State
	Words  map[string]Kind
}

const (
	noRule  = konst.MaxRules
	noState = konst.MaxStates
)

type rule struct {
	kind     Kind
	step     step
	boundary bool
	repeats  bool
	target   uint16
}

type pattern struct {
	text string
	rule uint8
}

type wideRange struct {
	runeRange
	rule uint8
}

type state struct {
	byByte   [utf8.RuneSelf]uint8
	first    [utf8.RuneSelf + 1]int32
	patterns []pattern
	rules    []rule
	wide     []wideRange
	fallback uint8
	probe    bool
	atEnd    uint16
}

type Grammar struct {
	states []state
	words  map[string]Kind
}

func Compile(d Definition) (*Grammar, error) {
	if len(d.States) == 0 {
		return nil, errors.New("a grammar needs a state")
	}
	if d.States[0].Probe {
		return nil, fmt.Errorf("state %q: the first state cannot probe", d.States[0].Name)
	}
	var defs []State
	index := map[string]uint16{}
	add := func(s State) error {
		if _, dup := index[s.Name]; dup {
			return fmt.Errorf("state %q declared twice", s.Name)
		}
		if len(defs) >= noState {
			return fmt.Errorf("more than %d states", noState)
		}
		index[s.Name] = uint16(len(defs))
		defs = append(defs, s)
		return nil
	}
	for _, s := range d.States {
		s.Rules = slices.Clone(s.Rules)
		if err := add(s); err != nil {
			return nil, err
		}
	}
	for i := 0; i < len(defs); i++ {
		for r, ru := range defs[i].Rules {
			if ru.until == "" {
				continue
			}
			body := State{Name: defs[i].Name + " within " + strconv.Itoa(r)}
			if ru.escape != "" {
				escape := State{Name: body.Name + " escape", Rules: []Rule{Match(ru.kind).Range(0, utf8.MaxRune).Leave()}}
				body.Rules = append(body.Rules, Match(ru.kind, ru.escape).Enter(escape.Name))
				if err := add(escape); err != nil {
					return nil, err
				}
			}
			content := Match(ru.kind).Range(0, utf8.MaxRune)
			if ru.oneLine {
				content = Match(ru.kind).Range(0, '\n'-1).Range('\n'+1, utf8.MaxRune)
			}
			body.Rules = append(body.Rules, Match(ru.kind, ru.until).Leave(), content)
			defs[i].Rules[r] = Match(ru.kind, ru.literals...).Enter(body.Name)
			if err := add(body); err != nil {
				return nil, err
			}
		}
	}
	g := &Grammar{states: make([]state, len(defs)), words: d.Words}
	for i, s := range defs {
		if err := g.states[i].compile(s, index); err != nil {
			return nil, fmt.Errorf("state %q: %w", s.Name, err)
		}
	}
	opening := 0
	for i, s := range g.states {
		for r, ru := range s.rules {
			if s.probe || ru.step != push && ru.step != jump || !g.states[ru.target].probe {
				continue
			}
			if ru.kind != Text {
				return nil, fmt.Errorf("state %q rule %d: a rule that opens a probe cannot emit", defs[i].Name, r)
			}
			opening++
		}
	}
	if opening > konst.FailedProbes {
		return nil, fmt.Errorf("%d rules open a probe, at most %d", opening, konst.FailedProbes)
	}
	return g, nil
}

func (c *state) compile(s State, index map[string]uint16) error {
	if len(s.Rules) >= noRule {
		return fmt.Errorf("%d rules, at most %d", len(s.Rules), noRule-1)
	}
	*c = state{fallback: noRule, probe: s.Probe, atEnd: noState}
	for b := range c.byByte {
		c.byByte[b] = noRule
	}
	claim := func(b rune, r uint8) {
		if c.byByte[b] == noRule {
			c.byByte[b] = r
		}
	}
	if s.AtEnd != "" {
		t, ok := index[s.AtEnd]
		if !ok {
			return fmt.Errorf("unknown state %q", s.AtEnd)
		}
		c.atEnd = t
	}
	for i, ru := range s.Rules {
		r := uint8(i)
		compiled := rule{kind: ru.kind, step: ru.step, boundary: ru.boundary, repeats: ru.step == stay && !ru.boundary && !s.Probe, target: noState}
		if ru.step == push || ru.step == jump {
			t, ok := index[ru.target]
			if !ok {
				return fmt.Errorf("rule %d: unknown state %q", i, ru.target)
			}
			compiled.target = t
		}
		c.rules = append(c.rules, compiled)
		for _, lit := range ru.literals {
			if lit == "" || !isASCII(lit) {
				return fmt.Errorf("rule %d: literal %q is empty or not ASCII", i, lit)
			}
			if len(lit) == 1 {
				claim(rune(lit[0]), r)
				continue
			}
			c.patterns = append(c.patterns, pattern{lit, r})
		}
		for _, rg := range ru.ranges {
			if rg.lo > rg.hi || rg.lo < 0 {
				return fmt.Errorf("rule %d: range %U to %U", i, rg.lo, rg.hi)
			}
			for b := rg.lo; b <= min(rg.hi, utf8.RuneSelf-1); b++ {
				claim(b, r)
			}
			if rg.hi < utf8.RuneSelf {
				continue
			}
			wide := wideRange{runeRange{max(rg.lo, utf8.RuneSelf), rg.hi}, r}
			for _, w := range c.wide {
				if wide.lo <= w.hi && wide.hi >= w.lo {
					return fmt.Errorf("rules %d and %d both match %U", w.rule, r, max(wide.lo, w.lo))
				}
			}
			c.wide = append(c.wide, wide)
		}
		if ru.fallback {
			if c.fallback != noRule {
				return fmt.Errorf("rules %d and %d are both fallbacks", c.fallback, r)
			}
			c.fallback = r
			for b := range rune(utf8.RuneSelf) {
				claim(b, r)
			}
		}
	}
	slices.SortStableFunc(c.patterns, func(a, b pattern) int {
		if a.text[0] != b.text[0] {
			return int(a.text[0]) - int(b.text[0])
		}
		return len(b.text) - len(a.text)
	})
	at := 0
	for b := range utf8.RuneSelf + 1 {
		for at < len(c.patterns) && int(c.patterns[at].text[0]) < b {
			at++
		}
		c.first[b] = int32(at)
	}
	return nil
}

func isASCII(s string) bool {
	for i := range len(s) {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}
