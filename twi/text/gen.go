//go:build ignore

package main

import (
	"flag"
	"fmt"
	"go/format"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"unicode"

	konst "github.com/twind-dev/twind/internal/konst/text"
)

func main() {
	version := flag.String("version", "", "Unicode version of the UCD files")
	flag.Parse()
	base := "https://www.unicode.org/Public/" + *version + "/ucd/"
	classNames := []string{"Other", "CR", "LF", "Control", "Extend", "ZWJ", "Regional_Indicator", "Prepend", "SpacingMark", "L", "V", "T", "LV", "LVT"}
	conjunctNames := []string{"None", "Consonant", "Extend", "Linker"}
	lineNames := []string{"AL", "NU", "ID", "NS", "SP", "BA", "HY", "BB", "B2", "GL", "WJ", "ZW", "OP", "CL", "CP", "EX", "IS", "SY", "QU", "PR", "PO", "IN"}
	lineAliases := map[string]string{
		"XX": "AL", "AI": "AL", "SG": "AL", "SA": "AL", "HL": "AL", "CM": "AL", "ZWJ": "AL",
		"BK": "AL", "CR": "AL", "LF": "AL", "NL": "AL", "AK": "AL", "AP": "AL", "AS": "AL", "VI": "AL", "VF": "AL",
		"H2": "ID", "H3": "ID", "JL": "ID", "JV": "ID", "JT": "ID", "EB": "ID", "EM": "ID", "RI": "ID", "CB": "ID",
		"CJ": "NS", "HH": "HY",
	}
	count := int(unicode.MaxRune) + 1
	class := make([]byte, count)
	line := make([]byte, count)
	conjunct := make([]byte, count)
	wide := make([]bool, count)
	presentation := make([]bool, count)
	modifier := make([]bool, count)
	pictographic := make([]bool, count)

	each(fetch(base+"EastAsianWidth.txt"), func(r rune, fields []string) {
		wide[r] = fields[0] == "W" || fields[0] == "F"
	})
	each(fetch(base+"auxiliary/GraphemeBreakProperty.txt"), func(r rune, fields []string) {
		class[r] = index(classNames, fields[0])
	})
	each(fetch(base+"LineBreak.txt"), func(r rune, fields []string) {
		name := fields[0]
		if alias, ok := lineAliases[name]; ok {
			name = alias
		}
		line[r] = index(lineNames, name)
	})
	each(fetch(base+"DerivedCoreProperties.txt"), func(r rune, fields []string) {
		if fields[0] == "InCB" {
			conjunct[r] = index(conjunctNames, fields[1])
		}
	})
	each(fetch(base+"emoji/emoji-data.txt"), func(r rune, fields []string) {
		switch fields[0] {
		case "Emoji_Presentation":
			presentation[r] = true
		case "Emoji_Modifier":
			modifier[r] = true
		case "Extended_Pictographic":
			pictographic[r] = true
		}
	})

	records := make([][3]byte, count)
	for r := range records {
		width := byte(1)
		switch classNames[class[r]] {
		case "CR", "LF", "Control", "ZWJ", "V", "T":
			width = 0
		case "Extend":
			if !modifier[r] {
				width = 0
			}
		}
		if width == 1 && (wide[r] || presentation[r]) {
			width = 2
		}
		flags := width | conjunct[r]<<konst.ConjunctShift
		if pictographic[r] {
			flags |= konst.PictographicBit
		}
		records[r] = [3]byte{class[r], flags, line[r]}
	}

	ids := map[[3]byte]int{}
	blockIDs := map[string]int{}
	var recordTable, blockIndex, blocks []byte
	for lo := 0; lo < count; lo += konst.BlockSize {
		block := make([]byte, konst.BlockSize)
		for i := range block {
			id, ok := ids[records[lo+i]]
			if !ok {
				id = len(ids)
				ids[records[lo+i]] = id
				recordTable = append(recordTable, records[lo+i][:]...)
			}
			block[i] = byte(id)
		}
		id, ok := blockIDs[string(block)]
		if !ok {
			id = len(blockIDs)
			blockIDs[string(block)] = id
			blocks = append(blocks, block...)
		}
		blockIndex = append(blockIndex, byte(id))
	}
	if len(ids) > 256 || len(blockIDs) > 256 {
		log.Fatalf("%d records and %d blocks do not fit a byte", len(ids), len(blockIDs))
	}
	var out strings.Builder
	fmt.Fprintf(&out, "package text\n\nconst UnicodeVersion = %q\n\nconst records = %q\n\nconst blockIndex = \"\" +\n", *version, recordTable)
	for chunk := range slices.Chunk(blockIndex, konst.BlockSize) {
		fmt.Fprintf(&out, "\t%q +\n", chunk)
	}
	out.WriteString("\t\"\"\n\nconst blocks = \"\" +\n")
	for chunk := range slices.Chunk(blocks, konst.BlockSize) {
		fmt.Fprintf(&out, "\t%q +\n", chunk)
	}
	out.WriteString("\t\"\"\n")
	src, err := format.Source([]byte(out.String()))
	must(err)
	must(os.WriteFile("tables.go", src, 0o644))
	must(os.WriteFile("testdata/GraphemeBreakTest.txt", []byte(fetch(base+"auxiliary/GraphemeBreakTest.txt")), 0o644))
}

func fetch(url string) string {
	resp, err := http.Get(url)
	must(err)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatal(url + ": " + resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	must(err)
	return string(body)
}

func each(text string, fn func(r rune, fields []string)) {
	for line := range strings.Lines(text) {
		data, _, _ := strings.Cut(strings.TrimPrefix(line, "# @missing:"), "#")
		codes, rest, ok := strings.Cut(data, ";")
		if !ok {
			continue
		}
		lo, hi, isRange := strings.Cut(strings.TrimSpace(codes), "..")
		if !isRange {
			hi = lo
		}
		fields := strings.Split(rest, ";")
		for i := range fields {
			fields[i] = strings.TrimSpace(fields[i])
		}
		for r, last := hex(lo), hex(hi); r <= last; r++ {
			fn(r, fields)
		}
	}
}

func hex(s string) rune {
	v, err := strconv.ParseUint(s, 16, 32)
	must(err)
	return rune(v)
}

func index(names []string, name string) byte {
	i := slices.Index(names, name)
	if i < 0 {
		log.Fatal("unknown property value " + name)
	}
	return byte(i)
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
