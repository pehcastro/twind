package text

import (
	"strings"

	konst "github.com/twind-dev/twind/internal/konst/text"
)

func Wrap(s string, width int) []string {
	return Widths{}.Wrap(s, width)
}

func (w Widths) Wrap(s string, width int) []string {
	var lines []string
	for paragraph := range strings.SplitSeq(s, "\n") {
		line, lineWidth := "", 0
		for word := range strings.SplitSeq(paragraph, " ") {
			wordWidth := w.Width(word)
			switch {
			case word == "":
			case line != "" && lineWidth+1+wordWidth <= width:
				line, lineWidth = line+" "+word, lineWidth+1+wordWidth
			default:
				if line != "" {
					lines = append(lines, line)
					line, lineWidth = "", 0
				}
				for cluster := range Graphemes(word) {
					clusterWidth := w.Width(cluster)
					if line != "" && lineWidth+clusterWidth > width {
						lines = append(lines, line)
						line, lineWidth = "", 0
					}
					line, lineWidth = line+cluster, lineWidth+clusterWidth
				}
			}
		}
		lines = append(lines, line)
	}
	return lines
}

func Truncate(s string, width int) string {
	return Widths{}.Truncate(s, width)
}

func (w Widths) Truncate(s string, width int) string {
	if w.Width(s) <= width {
		return s
	}
	room := width - w.Width(konst.Ellipsis)
	if room < 0 {
		return ""
	}
	n, used := 0, 0
	for cluster := range Graphemes(s) {
		used += w.Width(cluster)
		if used > room {
			break
		}
		n += len(cluster)
	}
	return s[:n] + konst.Ellipsis
}
