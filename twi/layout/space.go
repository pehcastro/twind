package layout

import (
	"fmt"
	"math"
)

type bounds struct{ min, max int }

func limit(lo, hi Length, base int, baseDefinite bool) bounds {
	r := bounds{max: math.MaxInt}
	if lo.Unit == Auto && hi.Unit == Auto {
		return r
	}
	if v, ok := resolve(lo, base, baseDefinite); ok {
		r.min = max(v, 0)
	}
	if v, ok := resolve(hi, base, baseDefinite); ok {
		r.max = v
	}
	return r
}

func (r bounds) clamp(v int) int {
	return max(min(v, r.max), r.min)
}

type flexItem struct {
	box *Box
	bounds
	basis, grow, shrink, margins, size int
	content                            bool
}

func (it *flexItem) set(c *Box, size, lo, hi Length, space int, definite bool) {
	s := &c.Style
	it.grow, it.shrink = s.Grow, s.Shrink
	if c.plain {
		it.bounds, it.basis, it.content = bounds{max: math.MaxInt}, 0, true
		return
	}
	it.bounds = limit(lo, hi, space, definite)
	var ok bool
	if it.basis, ok = resolve(s.Basis, space, definite); !ok {
		it.basis, ok = resolve(size, space, definite)
	}
	it.content = !ok
}

func (it *flexItem) weight(growing bool) int {
	switch {
	case growing && it.basis > it.size, !growing && it.basis < it.size:
		return 0
	case growing:
		return it.grow
	}
	return it.shrink * it.basis
}

func rigid(items []flexItem, free int) bool {
	for i := range items {
		if items[i].weight(free > 0) != 0 && (free != 0 || items[i].basis != items[i].size) {
			return false
		}
	}
	return true
}

func (a *arena) flexSizes(items []flexItem, space int) int {
	hypothetical := 0
	for i := range items {
		it := &items[i]
		it.size = it.clamp(it.basis)
		hypothetical += it.size
	}
	growing := hypothetical < space
	if rigid(items, space-hypothetical) {
		return hypothetical
	}
	total, weights := 0, grab(&a.ints, len(items))
	for i := range items {
		it := &items[i]
		w := it.weight(growing)
		if w > 0 {
			it.size = it.basis
		}
		weights[i], total = w, total+w
	}
	for total != 0 {
		free := space
		for i := range items {
			free -= items[i].size
		}
		mark := len(a.ints)
		raw := a.distribute(max(free, -free), weights)
		violation := 0
		for i := range items {
			if it := &items[i]; weights[i] != 0 {
				if free < 0 {
					raw[i] = -raw[i]
				}
				raw[i] += it.basis
				it.size = it.clamp(raw[i])
				violation += it.size - raw[i]
			}
		}
		total = 0
		for i := range items {
			it := &items[i]
			if weights[i] == 0 {
				continue
			}
			if violation == 0 || violation > 0 && it.size > raw[i] || violation < 0 && it.size < raw[i] {
				weights[i] = 0
			} else {
				it.size = it.basis
			}
			total += weights[i]
		}
		a.ints = a.ints[:mark]
	}
	sum := 0
	for i := range items {
		sum += items[i].size
	}
	return sum
}

func (a *arena) distribute(amount int, weights []int) []int {
	total := 0
	for _, w := range weights {
		total += w
	}
	shares := grab(&a.ints, len(weights))
	if len(weights) == 1 {
		shares[0] = amount
		return shares
	}
	rest := grab(&a.ints, len(weights))
	left := amount
	for i, w := range weights {
		shares[i], rest[i] = amount*w/total, amount*w%total
		left -= shares[i]
	}
	for ; left > 0; left-- {
		best := 0
		for i := range rest {
			if rest[i] > rest[best] {
				best = i
			}
		}
		shares[best]++
		rest[best] = -1
	}
	return shares
}

func (a *arena) even(amount, n int) []int {
	weights := grab(&a.ints, n)
	for i := range weights {
		weights[i] = 1
	}
	return a.distribute(amount, weights)
}

func (a *arena) justify(j Justify, free, n int) (int, []int) {
	switch j {
	case JustifyStart, JustifyStretch:
		return 0, nil
	case JustifyEnd:
		return free, nil
	case JustifyCenter:
		return free / 2, nil
	case JustifyBetween:
		if free <= 0 || n < 2 {
			return 0, nil
		}
		extra := grabZero(&a.ints, n)
		copy(extra, a.even(free, n-1))
		return 0, extra
	case JustifyAround:
		if free <= 0 || n == 0 {
			return 0, nil
		}
		extra := grabZero(&a.ints, n)
		slots := a.even(free, 2*n)
		for i := range n - 1 {
			extra[i] = slots[2*i+1] + slots[2*i+2]
		}
		return slots[0], extra
	case JustifyEvenly:
		if free <= 0 || n == 0 {
			return 0, nil
		}
		extra := grabZero(&a.ints, n)
		slots := a.even(free, n+1)
		copy(extra, slots[1:n])
		return slots[0], extra
	}
	panic(fmt.Sprintf("layout: unknown justify %d", j))
}
