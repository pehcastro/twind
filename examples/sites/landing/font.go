package main

import (
	"strings"

	"github.com/twind-dev/twind/twi"
)

func pixels(picture []string, scale, gap int) []twi.NodeOption {
	w, h := len(picture[0])*scale, len(picture)*scale
	lit := func(x, y int) int {
		if y < h && picture[y/scale][x/scale] == '#' {
			return 1
		}
		return 0
	}
	rows := []twi.NodeOption{}
	for y := 0; y < h; y += 2 {
		var b strings.Builder
		for x := range w {
			b.WriteString([...]string{" ", "▀", "▄", "█"}[lit(x, y)+2*lit(x, y+1)])
		}
		rows = append(rows, twi.Text(b.String()+strings.Repeat(" ", gap)))
	}
	return rows
}

func bitmap(class string, scale int, picture ...string) twi.Node {
	return el("flex flex-col shrink-0 whitespace-pre "+class, pixels(picture, scale, 0)...)
}

func display(s string, scale int, colours ...string) twi.Node {
	letters := []twi.NodeOption{}
	runes := []rune(s)
	for i, r := range runes {
		gap := scale
		if i == len(runes)-1 {
			gap = 0
		}
		g := glyph(r)
		letters = append(letters, el("flex flex-col whitespace-pre "+colours[i*len(colours)/len(runes)], pixels(g[:], scale, gap)...))
	}
	return el("flex flex-row shrink-0 font-bold", letters...)
}

func glyph(r rune) [5]string {
	switch r {
	case ' ':
		return [5]string{"...", "...", "...", "...", "..."}
	case '.':
		return [5]string{".", ".", ".", ".", "#"}
	case ':':
		return [5]string{".", "#", ".", "#", "."}
	case '\'':
		return [5]string{"#", "#", ".", ".", "."}
	case '-':
		return [5]string{"...", "...", "###", "...", "..."}
	case 'A':
		return [5]string{".###.", "#...#", "#####", "#...#", "#...#"}
	case 'B':
		return [5]string{"####.", "#...#", "####.", "#...#", "####."}
	case 'C':
		return [5]string{".####", "#....", "#....", "#....", ".####"}
	case 'D':
		return [5]string{"####.", "#...#", "#...#", "#...#", "####."}
	case 'E':
		return [5]string{"#####", "#....", "####.", "#....", "#####"}
	case 'F':
		return [5]string{"#####", "#....", "####.", "#....", "#...."}
	case 'G':
		return [5]string{".####", "#....", "#..##", "#...#", ".###."}
	case 'H':
		return [5]string{"#...#", "#...#", "#####", "#...#", "#...#"}
	case 'I':
		return [5]string{"###", ".#.", ".#.", ".#.", "###"}
	case 'J':
		return [5]string{"....#", "....#", "....#", "#...#", ".###."}
	case 'K':
		return [5]string{"#...#", "#..#.", "###..", "#..#.", "#...#"}
	case 'L':
		return [5]string{"#....", "#....", "#....", "#....", "#####"}
	case 'M':
		return [5]string{"#...#", "##.##", "#.#.#", "#...#", "#...#"}
	case 'N':
		return [5]string{"#...#", "##..#", "#.#.#", "#..##", "#...#"}
	case 'O':
		return [5]string{".###.", "#...#", "#...#", "#...#", ".###."}
	case 'P':
		return [5]string{"####.", "#...#", "####.", "#....", "#...."}
	case 'Q':
		return [5]string{".###.", "#...#", "#...#", "#..#.", ".##.#"}
	case 'R':
		return [5]string{"####.", "#...#", "####.", "#..#.", "#...#"}
	case 'S':
		return [5]string{".####", "#....", ".###.", "....#", "####."}
	case 'T':
		return [5]string{"#####", "..#..", "..#..", "..#..", "..#.."}
	case 'U':
		return [5]string{"#...#", "#...#", "#...#", "#...#", ".###."}
	case 'V':
		return [5]string{"#...#", "#...#", "#...#", ".#.#.", "..#.."}
	case 'W':
		return [5]string{"#...#", "#...#", "#.#.#", "##.##", "#...#"}
	case 'X':
		return [5]string{"#...#", ".#.#.", "..#..", ".#.#.", "#...#"}
	case 'Y':
		return [5]string{"#...#", ".#.#.", "..#..", "..#..", "..#.."}
	case 'Z':
		return [5]string{"#####", "...#.", "..#..", ".#...", "#####"}
	case '0':
		return [5]string{".###.", "#...#", "#...#", "#...#", ".###."}
	case '1':
		return [5]string{"..#..", ".##..", "..#..", "..#..", ".###."}
	case '2':
		return [5]string{"####.", "....#", ".###.", "#....", "#####"}
	case '3':
		return [5]string{"####.", "....#", ".###.", "....#", "####."}
	case '4':
		return [5]string{"#...#", "#...#", "#####", "....#", "....#"}
	case '5':
		return [5]string{"#####", "#....", "####.", "....#", "####."}
	case '6':
		return [5]string{".###.", "#....", "####.", "#...#", ".###."}
	case '7':
		return [5]string{"#####", "....#", "...#.", "..#..", "..#.."}
	case '8':
		return [5]string{".###.", "#...#", ".###.", "#...#", ".###."}
	case '9':
		return [5]string{".###.", "#...#", ".####", "....#", ".###."}
	}
	panic("landing: no glyph for " + string(r))
}
