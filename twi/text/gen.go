//go:build ignore

package main

import (
	"flag"
	"fmt"
	"go/format"
	"io"
	"log"
	"maps"
	"net/http"
	"os"
	"path"
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
	wordNames := []string{"Other", "CR", "LF", "Newline", "Extend", "Format", "ZWJ", "Regional_Indicator", "Katakana", "Hebrew_Letter", "ALetter", "Single_Quote", "Double_Quote", "MidNumLet", "MidLetter", "MidNum", "Numeric", "ExtendNumLet", "WSegSpace"}
	lineNames := []string{"AL", "NU", "ID", "NS", "SP", "BA", "HY", "BB", "B2", "GL", "WJ", "ZW", "OP", "CL", "CP", "EX", "IS", "SY", "QU", "PR", "PO", "IN"}
	lineAliases := map[string]string{
		"XX": "AL", "AI": "AL", "SG": "AL", "SA": "AL", "HL": "AL", "CM": "AL", "ZWJ": "AL",
		"BK": "AL", "CR": "AL", "LF": "AL", "NL": "AL", "AK": "AL", "AP": "AL", "AS": "AL", "VI": "AL", "VF": "AL",
		"H2": "ID", "H3": "ID", "JL": "ID", "JV": "ID", "JT": "ID", "EB": "ID", "EM": "ID", "RI": "ID", "CB": "ID",
		"CJ": "NS", "HH": "HY",
	}
	bidiNames := []string{"L", "R", "AL", "EN", "ES", "ET", "AN", "CS", "NSM", "BN", "B", "S", "WS", "ON", "LRE", "LRO", "RLE", "RLO", "PDF", "LRI", "RLI", "FSI", "PDI"}
	bidiAliases := map[string]string{"Left_To_Right": "L", "Right_To_Left": "R", "Arabic_Letter": "AL", "European_Terminator": "ET"}
	count := int(unicode.MaxRune) + 1
	class := make([]byte, count)
	line := make([]byte, count)
	conjunct := make([]byte, count)
	word := make([]byte, count)
	bidi := make([]byte, count)
	wide := make([]bool, count)
	presentation := make([]bool, count)
	modifier := make([]bool, count)
	pictographic := make([]bool, count)
	emoji := make([]bool, count)

	each(fetch(base+"EastAsianWidth.txt"), func(r rune, fields []string) {
		wide[r] = fields[0] == "W" || fields[0] == "F"
	})
	each(fetch(base+"auxiliary/GraphemeBreakProperty.txt"), func(r rune, fields []string) {
		class[r] = index(classNames, fields[0])
	})
	each(fetch(base+"auxiliary/WordBreakProperty.txt"), func(r rune, fields []string) {
		word[r] = index(wordNames, fields[0])
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
		case "Emoji":
			emoji[r] = true
		case "Emoji_Presentation":
			presentation[r] = true
		case "Emoji_Modifier":
			modifier[r] = true
		case "Extended_Pictographic":
			pictographic[r] = true
		}
	})
	each(fetch(base+"extracted/DerivedBidiClass.txt"), func(r rune, fields []string) {
		name := fields[0]
		if alias, ok := bidiAliases[name]; ok {
			name = alias
		}
		bidi[r] = index(bidiNames, name)
	})
	canonical := map[rune]rune{}
	each(fetch(base+"UnicodeData.txt"), func(r rune, fields []string) {
		if d := fields[4]; d != "" && !strings.ContainsAny(d, "< ") {
			canonical[r] = hex(d)
		}
	})
	mirrors := map[rune]rune{}
	each(fetch(base+"BidiMirroring.txt"), func(r rune, fields []string) {
		if fields[0] != "<none>" {
			mirrors[r] = hex(fields[0])
			bidi[r] |= konst.BidiMirrorBit
		}
	})
	brackets := map[rune]rune{}
	each(fetch(base+"BidiBrackets.txt"), func(r rune, fields []string) {
		opener, bit := r, byte(konst.BidiOpenBit)
		if fields[1] == "c" {
			opener, bit = hex(fields[0]), konst.BidiCloseBit
		}
		if c, ok := canonical[opener]; ok {
			opener = c
		}
		brackets[r] = opener
		bidi[r] |= bit
	})

	records := make([][konst.RecordSize]byte, count)
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
		if emoji[r] {
			flags |= konst.EmojiBit
		}
		switch bidiNames[bidi[r]&konst.BidiClassMask] {
		case "R", "AL", "AN", "RLE", "RLO", "RLI":
			flags |= konst.RTLBit
		}
		records[r] = [konst.RecordSize]byte{class[r], flags, line[r], word[r]}
	}

	ids := map[[konst.RecordSize]byte]int{}
	recordIDs := make([]byte, count)
	var recordTable []byte
	for r, rec := range records {
		id, ok := ids[rec]
		if !ok {
			id = len(ids)
			ids[rec] = id
			recordTable = append(recordTable, rec[:]...)
		}
		recordIDs[r] = byte(id)
	}
	if len(ids) > 256 {
		log.Fatalf("%d records do not fit a byte", len(ids))
	}
	var out strings.Builder
	fmt.Fprintf(&out, "package text\n\nconst UnicodeVersion = %q\n\nconst records = %q\n", *version, recordTable)
	stage(&out, "blockIndex", "blocks", recordIDs)
	printable := make([]byte, 256)
	for b := range printable {
		printable[b] = konst.Unprintable
		if b >= ' ' && b <= '~' {
			printable[b] = line[b]
		}
	}
	fmt.Fprintf(&out, "\nconst printableLines = %q\n", printable)
	stage(&out, "bidiIndex", "bidiBlocks", bidi)
	fmt.Fprintf(&out, "\nconst mirrors = %q\n\nconst brackets = %q\n", pairs(mirrors), pairs(brackets))
	src, err := format.Source([]byte(out.String()))
	must(err)
	must(os.WriteFile("tables.go", src, 0o644))
	for _, name := range []string{"auxiliary/GraphemeBreakTest.txt", "auxiliary/WordBreakTest.txt", "BidiCharacterTest.txt"} {
		must(os.WriteFile("testdata/"+path.Base(name), []byte(fetch(base+name)), 0o644))
	}
}

func stage(out *strings.Builder, indexName, blocksName string, values []byte) {
	blockIDs := map[string]int{}
	var index, blocks []byte
	for block := range slices.Chunk(values, konst.BlockSize) {
		id, ok := blockIDs[string(block)]
		if !ok {
			id = len(blockIDs)
			blockIDs[string(block)] = id
			blocks = append(blocks, block...)
		}
		index = append(index, byte(id))
	}
	if len(blockIDs) > 256 {
		log.Fatalf("%s: %d blocks do not fit a byte", blocksName, len(blockIDs))
	}
	chunked(out, indexName, index)
	chunked(out, blocksName, blocks)
}

func chunked(out *strings.Builder, name string, data []byte) {
	fmt.Fprintf(out, "\nconst %s = \"\" +\n", name)
	for chunk := range slices.Chunk(data, konst.BlockSize) {
		fmt.Fprintf(out, "\t%q +\n", chunk)
	}
	out.WriteString("\t\"\"\n")
}

func pairs(m map[rune]rune) []byte {
	var table []byte
	for _, r := range slices.Sorted(maps.Keys(m)) {
		for _, v := range []rune{r, m[r]} {
			table = append(table, byte(v>>16), byte(v>>8), byte(v))
		}
	}
	return table
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
