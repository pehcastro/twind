# A diff renderer in one evening

<Meta date="2026-09-12" tags="go, terminals" />

Every terminal UI eventually learns the same lesson: redrawing the whole screen is cheap until it is not. Last week I wrote the smallest renderer I could think of that only sends what changed, and it fit in one evening.

## The idea

Keep two grids of cells, the frame on screen and the frame you want. Walk both, and for every run of cells that differs, move the cursor once and write the run.

```go
func diff(prev, next [][]rune) []string {
	var out []string
	for y := range next {
		for x := 0; x < len(next[y]); x++ {
			if prev[y][x] == next[y][x] {
				continue
			}
			start := x
			for x < len(next[y]) && prev[y][x] != next[y][x] {
				x++
			}
			out = append(out, fmt.Sprintf("\x1b[%d;%dH%s", y+1, start+1, string(next[y][start:x])))
		}
	}
	return out
}
```

## What it does not handle

- Wide characters, which take two cells and break the run arithmetic.
- Colour, which needs a style per cell and a reset at the end of each run.
- Scrolling, where a smarter renderer moves whole regions instead of rewriting them.

Still, for a status bar that ticks once a second, this cut the bytes written by **ninety percent**.
