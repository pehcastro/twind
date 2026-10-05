package present

import (
	"bytes"
	"image"
	"maps"
	"slices"

	konst "github.com/pehcastro/twind/internal/konst/present"
	"github.com/pehcastro/twind/internal/terminal"
	"github.com/pehcastro/twind/twi/color"
)

type store struct {
	tiles  map[uint64]*stored
	images map[twin]*picture
	on     storedFor
}

type storedFor struct {
	page     uint32
	profile  color.Profile
	graphics terminal.Graphics
}

type stored struct {
	key     []byte
	runs    []run
	rows    []storedRow
	samples []color.Color
	hash    uint64
	used    uint64
}

type storedRow struct {
	span [2]int32
	n    int
	hash uint64
}

type picture struct {
	body []byte
	used uint64
}

func (st *store) open(on storedFor) {
	if st.tiles == nil || st.on != on {
		st.tiles, st.images, st.on = map[uint64]*stored{}, map[twin]*picture{}, on
	}
}

func (st *store) tile(hash uint64, key []byte, frame uint64) *stored {
	e, ok := st.tiles[hash]
	if !ok || !bytes.Equal(e.key, key) {
		return nil
	}
	e.used = frame
	return e
}

func (st *store) keep(hash uint64, key []byte, runs []run, rows []storedRow, tile, frame uint64) *stored {
	if _, ok := st.tiles[hash]; ok {
		return nil
	}
	e := &stored{key: slices.Clone(key), runs: slices.Clone(runs), rows: slices.Clone(rows), hash: tile, used: frame}
	st.tiles[hash] = e
	return e
}

func (e *stored) sampled(s *Screen, cells image.Rectangle) bool {
	s.lock.Lock()
	saved := e.samples
	s.lock.Unlock()
	if saved == nil {
		return false
	}
	for y := cells.Min.Y; y < cells.Max.Y; y++ {
		at := y*s.cols + cells.Min.X
		copy(s.samples[at:at+cells.Dx()], saved[(y-cells.Min.Y)*cells.Dx():])
		for i := range cells.Dx() {
			s.sampled[at+i] = true
		}
	}
	return true
}

func (e *stored) sample(s *Screen, cells image.Rectangle) {
	samples := make([]color.Color, 0, cells.Dx()*cells.Dy())
	for y := cells.Min.Y; y < cells.Max.Y; y++ {
		samples = append(samples, s.samples[y*s.cols+cells.Min.X:y*s.cols+cells.Max.X]...)
	}
	s.lock.Lock()
	e.samples = samples
	s.lock.Unlock()
}

func (st *store) image(key twin, frame uint64) ([]byte, bool) {
	e, ok := st.images[key]
	if !ok {
		return nil, false
	}
	e.used = frame
	return e.body, true
}

func (st *store) remember(key twin, body []byte, frame uint64) {
	if _, ok := st.images[key]; !ok {
		st.images[key] = &picture{body: slices.Clone(body), used: frame}
	}
}

func (st *store) sweep(tiles int, frame uint64) {
	for _, since := range [2]uint64{frame - min(frame, konst.StoredFrames), frame} {
		if len(st.tiles)+len(st.images) <= konst.StoredPerTile*tiles {
			return
		}
		maps.DeleteFunc(st.tiles, func(_ uint64, e *stored) bool { return e.used < since })
		maps.DeleteFunc(st.images, func(_ twin, e *picture) bool { return e.used < since })
	}
}
