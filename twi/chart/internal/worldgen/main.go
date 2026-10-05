package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"

	konst "github.com/pehcastro/twind/internal/konst/chart"
)

type feature struct {
	Properties struct {
		ISO  string `json:"ISO_A3"`
		ADM0 string `json:"ADM0_A3"`
		Name string `json:"NAME"`
	} `json:"properties"`
	Geometry struct {
		Type        string          `json:"type"`
		Coordinates json.RawMessage `json:"coordinates"`
	} `json:"geometry"`
}

type pt struct{ x, y float64 }

type country struct {
	code, name string
	rings      [][]pt
}

func main() {
	if len(os.Args) != 3 {
		fail(errors.New("usage: worldgen ne_110m_admin_0_countries.geojson world.bin"))
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		fail(err)
	}
	var doc struct{ Features []feature }
	if err := json.Unmarshal(raw, &doc); err != nil {
		fail(err)
	}
	var countries []country
	lo, hi := pt{math.Inf(1), math.Inf(1)}, pt{math.Inf(-1), math.Inf(-1)}
	for _, f := range doc.Features {
		code := f.Properties.ISO
		if code == konst.WorldNoCode {
			code = f.Properties.ADM0
		}
		if code == konst.WorldDropped {
			continue
		}
		var polys [][][][2]float64
		switch f.Geometry.Type {
		case "Polygon":
			var p [][][2]float64
			err = json.Unmarshal(f.Geometry.Coordinates, &p)
			polys = append(polys, p)
		case "MultiPolygon":
			err = json.Unmarshal(f.Geometry.Coordinates, &polys)
		default:
			err = fmt.Errorf("%s: geometry %s", code, f.Geometry.Type)
		}
		if err != nil {
			fail(err)
		}
		c := country{code: code, name: f.Properties.Name}
		for _, poly := range polys {
			for _, ring := range poly {
				var out []pt
				for _, lonlat := range ring[:len(ring)-1] {
					p := project(lonlat[0]*math.Pi/180, lonlat[1]*math.Pi/180)
					lo, hi = pt{min(lo.x, p.x), min(lo.y, p.y)}, pt{max(hi.x, p.x), max(hi.y, p.y)}
					out = append(out, p)
				}
				c.rings = append(c.rings, out)
			}
		}
		countries = append(countries, c)
	}
	scale := konst.WorldGrid / (hi.x - lo.x)
	height := int(math.Ceil((hi.y - lo.y) * scale))
	out := binary.AppendUvarint(nil, konst.WorldGrid)
	out = binary.AppendUvarint(out, uint64(height))
	out = binary.AppendUvarint(out, uint64(len(countries)))
	var prev [2]int64
	points := 0
	for _, c := range countries {
		out = append(out, c.code...)
		out = binary.AppendUvarint(out, uint64(len(c.name)))
		out = append(out, c.name...)
		var rings [][][2]int64
		for _, ring := range c.rings {
			var q [][2]int64
			for _, p := range simplify(ring, konst.WorldSimplify/scale) {
				g := [2]int64{int64(math.Round((p.x - lo.x) * scale)), int64(math.Round((hi.y - p.y) * scale))}
				if len(q) == 0 || q[len(q)-1] != g {
					q = append(q, g)
				}
			}
			if len(q) >= konst.WorldMinRing {
				rings = append(rings, q)
			}
		}
		out = binary.AppendUvarint(out, uint64(len(rings)))
		for _, q := range rings {
			out = binary.AppendUvarint(out, uint64(len(q)))
			for _, g := range q {
				out = binary.AppendVarint(out, g[0]-prev[0])
				out = binary.AppendVarint(out, g[1]-prev[1])
				prev = g
			}
			points += len(q)
		}
	}
	if err := os.WriteFile(os.Args[2], out, konst.WorldFileMode); err != nil {
		fail(err)
	}
	fmt.Printf("%d countries, %d points, %dx%d grid, %d bytes\n", len(countries), points, konst.WorldGrid, height, len(out))
}

func project(lambda, phi float64) pt {
	p2 := phi * phi
	p4 := p2 * p2
	return pt{
		lambda * (konst.ProjX0 + p2*(konst.ProjX2+p2*(konst.ProjX4+p2*p4*(konst.ProjX10+p2*konst.ProjX12)))),
		phi * (konst.ProjY0 + p2*(konst.ProjY2+p4*(konst.ProjY6+p2*(konst.ProjY8+p2*konst.ProjY10)))),
	}
}

func simplify(ring []pt, tolerance float64) []pt {
	keep := make([]bool, len(ring))
	keep[0], keep[len(ring)-1] = true, true
	var walk func(a, b int)
	walk = func(a, b int) {
		far, at := 0.0, -1
		for i := a + 1; i < b; i++ {
			if d := distance(ring[i], ring[a], ring[b]); d > far {
				far, at = d, i
			}
		}
		if at >= 0 && far > tolerance {
			keep[at] = true
			walk(a, at)
			walk(at, b)
		}
	}
	walk(0, len(ring)-1)
	var out []pt
	for i, p := range ring {
		if keep[i] {
			out = append(out, p)
		}
	}
	return out
}

func distance(p, a, b pt) float64 {
	dx, dy := b.x-a.x, b.y-a.y
	if dx == 0 && dy == 0 {
		return math.Hypot(p.x-a.x, p.y-a.y)
	}
	return math.Abs(dy*(p.x-a.x)-dx*(p.y-a.y)) / math.Hypot(dx, dy)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "worldgen:", err)
	os.Exit(1)
}
