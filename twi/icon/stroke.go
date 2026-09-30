package icon

import (
	"image"
	"math"
	"slices"
	"strconv"

	konst "github.com/twind-dev/twind/internal/konst/icon"
)

type point struct{ x, y float32 }

type Stroker struct {
	segments [][2]point
	inside   []uint8
}

func (s *Stroker) Draw(dst *image.Alpha, name Name) {
	b := dst.Rect
	side := min(b.Dx(), b.Dy())
	if side <= 0 {
		return
	}
	_, _, path := name.lucide()
	scale := float32(side) / konst.ViewBox
	at := point{float32((b.Dx() - side) / 2), float32((b.Dy() - side) / 2)}
	s.flatten(path, scale, at)
	const n = konst.Samples
	half := scale * konst.StrokeWidth / 2
	wide, high := b.Dx()*n, b.Dy()*n
	s.inside = slices.Grow(s.inside[:0], wide*high)[:wide*high]
	clear(s.inside)
	sample := func(v float32) float64 { return float64(v*n - 0.5) }
	for _, seg := range s.segments {
		a, e := seg[0], seg[1]
		top, bottom := max(int(math.Ceil(sample(min(a.y, e.y)-half))), 0), min(int(math.Floor(sample(max(a.y, e.y)+half))), high-1)
		for k := top; k <= bottom; k++ {
			lo, hi := capsule(a, e, (float32(k)+0.5)/n, half)
			if lo > hi {
				continue
			}
			row := s.inside[k*wide:][:wide]
			for i := max(int(math.Ceil(sample(lo))), 0); i <= min(int(math.Floor(sample(hi))), wide-1); i++ {
				row[i] = 1
			}
		}
	}
	for y := range b.Dy() {
		row := dst.Pix[dst.PixOffset(b.Min.X, b.Min.Y+y):][:b.Dx()]
		clear(row)
		for k := range n {
			for i, in := range s.inside[(y*n+k)*wide:][:wide] {
				row[i/n] += in
			}
		}
		for x, count := range row {
			row[x] = uint8((int(count)*math.MaxUint8 + n*n/2) / (n * n))
		}
	}
}

func capsule(a, e point, y, r float32) (lo, hi float32) {
	lo, hi = math.MaxFloat32, -math.MaxFloat32
	for _, c := range [2]point{a, e} {
		if d := y - c.y; d*d <= r*r {
			w := float32(math.Sqrt(float64(r*r - d*d)))
			lo, hi = min(lo, c.x-w), max(hi, c.x+w)
		}
	}
	dx, dy, rise := e.x-a.x, e.y-a.y, y-a.y
	length := dx*dx + dy*dy
	if dy == 0 {
		if rise*rise <= r*r {
			lo, hi = min(lo, a.x, e.x), max(hi, a.x, e.x)
		}
		return lo, hi
	}
	reach := r * float32(math.Sqrt(float64(length)))
	x0, x1 := order(a.x+(rise*dx-reach)/dy, a.x+(rise*dx+reach)/dy)
	if dx != 0 {
		t0, t1 := order(a.x-rise*dy/dx, a.x+(length-rise*dy)/dx)
		x0, x1 = max(x0, t0), min(x1, t1)
	} else if t := rise * dy / length; t < 0 || t > 1 {
		return lo, hi
	}
	if x0 <= x1 {
		lo, hi = min(lo, x0), max(hi, x1)
	}
	return lo, hi
}

func order(p, q float32) (float32, float32) { return min(p, q), max(p, q) }

func (s *Stroker) flatten(path string, scale float32, at point) {
	s.segments = s.segments[:0]
	var pen, start point
	var op byte
	var args [6]float32
	count := 0
	for i := 0; i < len(path); {
		j := i
		for j < len(path) && path[j] != ' ' {
			j++
		}
		field := path[i:j]
		i = j + 1
		switch field {
		case "M", "L", "C":
			op, count = field[0], 0
			continue
		case "Z":
			s.segments = append(s.segments, [2]point{pen, start})
			pen = start
			continue
		}
		v, _ := strconv.ParseFloat(field, 32)
		args[count] = float32(v)*scale + [2]float32{at.x, at.y}[count%2]
		count++
		switch {
		case op == 'M' && count == 2:
			pen, start, count = point{args[0], args[1]}, point{args[0], args[1]}, 0
		case op == 'L' && count == 2:
			to := point{args[0], args[1]}
			s.segments = append(s.segments, [2]point{pen, to})
			pen, count = to, 0
		case op == 'C' && count == 6:
			c1, c2, to := point{args[0], args[1]}, point{args[2], args[3]}, point{args[4], args[5]}
			ddx := max(abs(pen.x-2*c1.x+c2.x), abs(c1.x-2*c2.x+to.x))
			ddy := max(abs(pen.y-2*c1.y+c2.y), abs(c1.y-2*c2.y+to.y))
			steps := min(max(int(math.Ceil(math.Sqrt(3*math.Hypot(float64(ddx), float64(ddy))/(4*konst.Flatness)))), 1), konst.MaxSteps)
			from := pen
			for k := 1; k <= steps; k++ {
				t := float32(k) / float32(steps)
				u := 1 - t
				on := point{
					u*u*u*pen.x + 3*u*u*t*c1.x + 3*u*t*t*c2.x + t*t*t*to.x,
					u*u*u*pen.y + 3*u*u*t*c1.y + 3*u*t*t*c2.y + t*t*t*to.y,
				}
				s.segments = append(s.segments, [2]point{from, on})
				from = on
			}
			pen, count = to, 0
		}
	}
}

func abs(v float32) float32 { return max(v, -v) }
