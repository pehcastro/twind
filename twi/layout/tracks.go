package layout

import konst "github.com/twind-dev/twind/internal/konst/layout"

func (it *gridItem) minimum(definite bool, under sizing) int {
	switch {
	case definite:
		return it.least
	case under == minContent:
		return it.low
	}
	return it.high
}

func grown(limit, contribution int) int {
	if limit == infinite {
		return contribution
	}
	return max(limit, contribution)
}

func flexible(ts []track) bool {
	for _, t := range ts {
		if t.max.Kind == SizeFr {
			return true
		}
	}
	return false
}

func (a *arena) sizeTracks(ts []track, items []gridItem, ax, gap, size int, definite bool, under sizing) {
	widest := 1
	for i := range items {
		it := &items[i]
		sp := it.area[ax]
		widest = max(widest, sp.end-sp.start)
		t := &ts[sp.start]
		if sp.end-sp.start != 1 || t.max.Kind == SizeFr {
			continue
		}
		switch t.min.Kind {
		case SizeMinContent:
			t.base = max(t.base, it.low)
		case SizeMaxContent:
			t.base = max(t.base, it.high)
		case SizeAuto:
			t.base = max(t.base, it.minimum(definite, under))
		}
		switch t.max.Kind {
		case SizeMinContent:
			t.limit = grown(t.limit, it.low)
		case SizeMaxContent, SizeAuto:
			t.limit = grown(t.limit, it.high)
		}
		t.limit = max(t.limit, t.base)
	}
	for n := 2; n <= widest; n++ {
		for i := range items {
			it := &items[i]
			sp := it.area[ax]
			spanned := ts[sp.start:sp.end]
			if sp.end-sp.start != n || flexible(spanned) {
				continue
			}
			a.spread(spanned, it.minimum(definite, under)-gap*(n-1), false)
			a.spread(spanned, it.high-gap*(n-1), true)
		}
	}
	for i := range items {
		it := &items[i]
		sp := it.area[ax]
		spanned := ts[sp.start:sp.end]
		if !flexible(spanned) {
			continue
		}
		extra, mark := it.minimum(definite, under)-gap*(len(spanned)-1), len(a.ints)
		weights := grabZero(&a.ints, len(spanned))
		sum := 0
		for k, t := range spanned {
			extra -= t.base
			if t.max.Kind == SizeFr && t.min.contentBased() {
				weights[k] = t.max.Value
				sum += weights[k]
			}
		}
		if extra > 0 && sum > 0 {
			for k, share := range a.distribute(extra, weights) {
				spanned[k].base += share
			}
		}
		a.ints = a.ints[:mark]
	}
	for i := range ts {
		if ts[i].limit == infinite {
			ts[i].limit = ts[i].base
		}
	}
	switch {
	case definite:
		for i := range ts {
			ts[i].frozen = false
		}
		a.grow(ts, size-total(ts, gap))
	case under == maxContent:
		for i := range ts {
			ts[i].base = ts[i].limit
		}
	}
	a.expand(ts, items, ax, gap, size, definite, under)
}

func (t track) finiteLimit() int {
	if t.limit == infinite {
		return t.base
	}
	return t.limit
}

func (a *arena) grow(ts []track, extra int) int {
	for extra > 0 {
		open := 0
		for _, t := range ts {
			if !t.frozen && t.base < t.limit {
				open++
			}
		}
		if open == 0 {
			return extra
		}
		mark := len(a.ints)
		shares := a.even(extra, open)
		k := 0
		for i := range ts {
			t := &ts[i]
			if t.frozen || t.base >= t.limit {
				continue
			}
			give := min(shares[k], t.limit-t.base)
			t.base, extra, k = t.base+give, extra-give, k+1
		}
		a.ints = a.ints[:mark]
	}
	return extra
}

func (a *arena) spread(ts []track, need int, limits bool) {
	open, extra := 0, need
	for i := range ts {
		t := &ts[i]
		t.frozen = !t.min.contentBased()
		extra -= t.base
		if limits {
			t.frozen = !t.max.contentBased()
			extra -= t.finiteLimit() - t.base
		}
		if !t.frozen {
			open++
		}
	}
	if open == 0 || extra <= 0 {
		return
	}
	if !limits {
		extra = a.grow(ts, extra)
	}
	if extra <= 0 {
		return
	}
	mark := len(a.ints)
	shares := a.even(extra, open)
	k := 0
	for i := range ts {
		t := &ts[i]
		if t.frozen {
			continue
		}
		if limits {
			t.limit = t.finiteLimit() + shares[k]
		} else {
			t.base += shares[k]
			t.limit = max(t.limit, t.base)
		}
		k++
	}
	a.ints = a.ints[:mark]
}

func frSize(ts []track, space int) (leftover, factors int) {
	leftover = space
	for i := range ts {
		t := &ts[i]
		t.frozen = t.max.Kind != SizeFr
		if t.frozen {
			leftover -= t.base
		}
	}
	for {
		factors = 0
		for _, t := range ts {
			if !t.frozen {
				factors += t.max.Value
			}
		}
		whole, inflexible := max(factors, konst.FrUnit), false
		for i := range ts {
			t := &ts[i]
			if !t.frozen && leftover*t.max.Value < t.base*whole {
				t.frozen, leftover, inflexible = true, leftover-t.base, true
			}
		}
		if !inflexible {
			return leftover, factors
		}
	}
}

func (a *arena) expand(ts []track, items []gridItem, ax, gap, size int, definite bool, under sizing) {
	if !flexible(ts) {
		return
	}
	if definite {
		leftover, factors := frSize(ts, size-gap*(len(ts)-1))
		if leftover <= 0 || factors == 0 {
			return
		}
		if factors < konst.FrUnit {
			leftover = leftover * factors / konst.FrUnit
		}
		mark := len(a.ints)
		weights := grabZero(&a.ints, len(ts))
		for i, t := range ts {
			if !t.frozen {
				weights[i] = t.max.Value
			}
		}
		for i, share := range a.distribute(leftover, weights) {
			if weights[i] > 0 {
				ts[i].base = max(ts[i].base, share)
			}
		}
		a.ints = a.ints[:mark]
		return
	}
	if under == minContent {
		return
	}
	cells, per := 0, 1
	larger := func(n, d int) {
		if n*per > cells*d {
			cells, per = n, d
		}
	}
	for _, t := range ts {
		if t.max.Kind == SizeFr {
			larger(t.base*konst.FrUnit, max(t.max.Value, konst.FrUnit))
		}
	}
	for _, it := range items {
		sp := it.area[ax]
		if spanned := ts[sp.start:sp.end]; flexible(spanned) {
			leftover, factors := frSize(spanned, it.high-gap*(len(spanned)-1))
			larger(leftover*konst.FrUnit, max(factors, konst.FrUnit))
		}
	}
	for i := range ts {
		if t := &ts[i]; t.max.Kind == SizeFr {
			whole := per * konst.FrUnit
			t.base = max(t.base, (cells*t.max.Value+whole-1)/whole)
		}
	}
}

func (a *arena) position(ts []track, gap, size int, definite bool, j Justify) {
	mark, free := len(a.ints), 0
	if definite {
		free = size - total(ts, gap)
	}
	auto := 0
	for _, t := range ts {
		if t.max.Kind == SizeAuto {
			auto++
		}
	}
	if j == JustifyStretch && free > 0 && auto > 0 {
		shares, k := a.even(free, auto), 0
		for i := range ts {
			if ts[i].max.Kind == SizeAuto {
				ts[i].base, k = ts[i].base+shares[k], k+1
			}
		}
		free = 0
	}
	pos, extra := a.justify(j, free, len(ts))
	for i := range ts {
		ts[i].pos = pos
		pos += ts[i].base + gap + extra[i]
	}
	a.ints = a.ints[:mark]
}
