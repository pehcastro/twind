package layout

import (
	"fmt"
	"math"
	"slices"
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
	basis, grow, shrink int
}

func newItem(s Style, size, lo, hi Length, space int, definite bool) (flexItem, bool) {
	it := flexItem{bounds: limit(lo, hi, space, definite), grow: s.Grow, shrink: s.Shrink}
	var ok bool
	if it.basis, ok = resolve(s.Basis, space, definite); !ok {
		it.basis, ok = resolve(size, space, definite)
	}
	return it, !ok
}

func flexSizes(items []flexItem, space int) []int {
	sizes := make([]int, len(items))
	hypothetical := 0
	for i, it := range items {
		sizes[i] = it.clamp(it.basis)
		hypothetical += sizes[i]
	}
	growing := hypothetical < space
	weights := make([]int, len(items))
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
	for {
		free, total := space, 0
		for i := range items {
			free -= sizes[i]
			total += weights[i]
		}
		if total == 0 {
			return sizes
		}
		shares := distribute(max(free, -free), weights)
		raw := make([]int, len(items))
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

func distribute(amount int, weights []int) []int {
	total := 0
	for _, w := range weights {
		total += w
	}
	shares := make([]int, len(weights))
	rest := make([]int, len(weights))
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

func justify(j Justify, free, n int) (int, []int) {
	extra := make([]int, n)
	switch j {
	case JustifyStart:
		return 0, extra
	case JustifyEnd:
		return free, extra
	case JustifyCenter:
		return free / 2, extra
	case JustifyBetween:
		if free <= 0 || n < 2 {
			return 0, extra
		}
		copy(extra, distribute(free, slices.Repeat([]int{1}, n-1)))
		return 0, extra
	case JustifyAround:
		if free <= 0 || n == 0 {
			return 0, extra
		}
		slots := distribute(free, slices.Repeat([]int{1}, 2*n))
		for i := range n - 1 {
			extra[i] = slots[2*i+1] + slots[2*i+2]
		}
		return slots[0], extra
	case JustifyEvenly:
		if free <= 0 || n == 0 {
			return 0, extra
		}
		slots := distribute(free, slices.Repeat([]int{1}, n+1))
		copy(extra, slots[1:n])
		return slots[0], extra
	}
	panic(fmt.Sprintf("layout: unknown justify %d", j))
}
