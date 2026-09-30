package layout

import (
	"fmt"
	"math"
)

type bounds struct{ min, max int }

func limit(lo, hi Length, base int, baseDefinite bool) bounds {
	r := bounds{max: math.MaxInt}
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
	bounds
	basis, grow, shrink, margins int
	content                      bool
}

func newItem(s *Style, size, lo, hi Length, space int, definite bool) flexItem {
	it := flexItem{bounds: limit(lo, hi, space, definite), grow: s.Grow, shrink: s.Shrink}
	var ok bool
	if it.basis, ok = resolve(s.Basis, space, definite); !ok {
		it.basis, ok = resolve(size, space, definite)
	}
	it.content = !ok
	return it
}

func (a *arena) flexSizes(items []flexItem, space int) []int {
	sizes := grab(&a.ints, len(items))
	hypothetical := 0
	for i, it := range items {
		sizes[i] = it.clamp(it.basis)
		hypothetical += sizes[i]
	}
	growing := hypothetical < space
	weights := grab(&a.ints, len(items))
	for i, it := range items {
		weights[i] = it.shrink * it.basis
		if growing {
			weights[i] = it.grow
		}
		if growing && it.basis > sizes[i] || !growing && it.basis < sizes[i] {
			weights[i] = 0
		}
		if weights[i] > 0 {
			sizes[i] = it.basis
		}
	}
	raw := grab(&a.ints, len(items))
	for {
		free, total := space, 0
		for i := range items {
			free -= sizes[i]
			total += weights[i]
		}
		if total == 0 {
			return sizes
		}
		mark := len(a.ints)
		shares := a.distribute(max(free, -free), weights)
		violation := 0
		for i, it := range items {
			if weights[i] == 0 {
				continue
			}
			raw[i] = it.basis + shares[i]
			if free < 0 {
				raw[i] = it.basis - shares[i]
			}
			sizes[i] = it.clamp(raw[i])
			violation += sizes[i] - raw[i]
		}
		a.ints = a.ints[:mark]
		for i, it := range items {
			if weights[i] == 0 {
				continue
			}
			if violation == 0 || violation > 0 && sizes[i] > raw[i] || violation < 0 && sizes[i] < raw[i] {
				weights[i] = 0
			} else {
				sizes[i] = it.basis
			}
		}
	}
}

func (a *arena) distribute(amount int, weights []int) []int {
	total := 0
	for _, w := range weights {
		total += w
	}
	shares := grab(&a.ints, len(weights))
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
	extra := grab(&a.ints, n)
	switch j {
	case JustifyStart, JustifyStretch:
		return 0, extra
	case JustifyEnd:
		return free, extra
	case JustifyCenter:
		return free / 2, extra
	case JustifyBetween:
		if free <= 0 || n < 2 {
			return 0, extra
		}
		copy(extra, a.even(free, n-1))
		return 0, extra
	case JustifyAround:
		if free <= 0 || n == 0 {
			return 0, extra
		}
		slots := a.even(free, 2*n)
		for i := range n - 1 {
			extra[i] = slots[2*i+1] + slots[2*i+2]
		}
		return slots[0], extra
	case JustifyEvenly:
		if free <= 0 || n == 0 {
			return 0, extra
		}
		slots := a.even(free, n+1)
		copy(extra, slots[1:n])
		return slots[0], extra
	}
	panic(fmt.Sprintf("layout: unknown justify %d", j))
}
