package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"
)

type check struct {
	unit  string
	limit float64
}

type budget struct {
	checks []check
	text   string
}

func main() {
	load := flag.String("load", "none named", "other work on the machine during the run")
	flag.Parse()
	p95 := func(ms float64) check { return check{"p95-ns", ms * 1e6} }
	budgets := map[string]budget{
		"OneCell/sync=true":      {[]check{{"bytes/frame", 32}}, "bytes < 32"},
		"ColdStart343":           {[]check{p95(30)}, "p95 < 30 ms"},
		"Render/nodes=10000":     {[]check{{"heap-MB", 20}}, "heap < 20 MB"},
		"KeyToFrame/drive":       {[]check{p95(8)}, "p95 < 8 ms"},
		"KeyToFrame/sync=true":   {[]check{p95(8)}, "p95 < 8 ms"},
		"Idle":                   {[]check{{"cpu-%", 1}}, "cpu < 1% of a core"},
		"AppIdle":                {[]check{{"cpu-%", 1}, {"wakeups", 1}, {"idle-bytes", 1}}, "cpu < 1% of a core, 0 wakeups, 0 bytes"},
		"Surface/sixel/text":     {[]check{{"image-bytes/frame", 1e-9}, p95(8)}, "0 image bytes, p95 < 8 ms"},
		"Surface/sixel/hover":    {[]check{p95(8), {"bytes/frame", 16e3}}, "p95 < 8 ms, bytes < 16 KB"},
		"Surface/sixel/first":    {[]check{p95(50), {"bytes/frame", 300e3}}, "p95 < 50 ms, bytes < 300 KB"},
		"Surface/kitty/first":    {[]check{p95(30)}, "p95 < 30 ms"},
		"Playground/sixel/first": {[]check{p95(50), {"bytes/frame", 300e3}}, "p95 < 50 ms, bytes < 300 KB"},
		"Playground/sixel/type":  {[]check{{"image-bytes/frame", 1e-9}, p95(8)}, "0 image bytes, p95 < 8 ms"},
		"raster card 440x168px":  {nil, "< 1 ms, 0 allocs; measured by twi/raster BenchmarkCard, not in bench/"},
	}
	header := map[string]string{}
	samples := map[string]map[string][]float64{}
	var order []string
	procs := ""
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if key, value, ok := strings.Cut(line, ": "); ok && !strings.Contains(key, " ") {
			header[key] = value
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 || !strings.HasPrefix(fields[0], "Benchmark") {
			continue
		}
		name := strings.TrimPrefix(fields[0], "Benchmark")
		if i := strings.LastIndex(name, "-"); i > 0 {
			name, procs = name[:i], name[i+1:]
		}
		if samples[name] == nil {
			samples[name] = map[string][]float64{}
			order = append(order, name)
		}
		for i := 2; i+1 < len(fields); i += 2 {
			v, err := strconv.ParseFloat(fields[i], 64)
			if err != nil {
				fmt.Fprintln(os.Stderr, "report: bad value in", line)
				os.Exit(1)
			}
			samples[name][fields[i+1]] = append(samples[name][fields[i+1]], v)
		}
	}
	if len(order) == 0 {
		fmt.Fprintln(os.Stderr, "report: no benchmark lines on stdin")
		os.Exit(1)
	}

	commit := command("git", "rev-parse", "--short", "HEAD")
	if command("git", "status", "--porcelain") != "" {
		commit += "+dirty"
	}
	system := command("uname", "-sr")
	if runtime.GOOS == "windows" {
		system = command("cmd", "/c", "ver")
	}
	fmt.Printf("machine: cpu=%q os=%q goos=%s goarch=%s go=%s procs=%s commit=%s date=%s load=%q terminal=none size=80x24, Surface, Playground and AppIdle 120x40 with a 10x20 px cell\n",
		header["cpu"], system, header["goos"], header["goarch"], runtime.Version(), procs, commit, time.Now().Format(time.RFC3339), *load)
	fmt.Println("path: Render, ColdStart and OneCell call twi.Render and terminal.Writer.Diff into a byte counter")
	fmt.Println("path: KeyToFrame/drive times drive.Press wall clock, fake clock stepped one frame per key; the driver exposes no byte count")
	fmt.Println("path: KeyToFrame/sync=* time a key event sent to twi.Backend until the frame's one Write returns, clock stepped past the 60 fps throttle")
	fmt.Println("path: first frame of every KeyToFrame and Surface step arm is drawn before timing and not sampled; its size is first bytes")
	fmt.Println("path: Surface/* drive bench/scenarios/surfaces, the playground's surfaces page and chrome plus a five-row list and a counter card, through twi.Backend reporting Capabilities sixel, kitty or none with a 10x20 px cell, clock stepped past the throttle")
	fmt.Println("path: Surface/*/first times twi.New, Run and the first Write; text sends '+' (counter card text), hover toggles one list row's bg, theme swaps twind dark and light through rt.SetTheme, resize times 120x40 to 160x40 and resizes back untimed")
	fmt.Println("path: Playground/* drive examples/playground/app.App, the example itself, autofocused input, twind-dark set by rt.SetTheme before Run, Sixel, 120x40; type times 'a' into the input and erases it untimed; theme types t and Enter, moves the picker cursor down or up in turn, all untimed, and times the Enter that applies dream-dark or twind-dark and closes the picker")
	fmt.Println("path: Surface and Playground p50 and p95 stop at the first Write; settled p50 and p95 stop when the runtime is quiet (a Dispatch round with no Write), so they include any frame the change needs after the first; the bench counts image tiles after settled stops; extra bytes are what it wrote after the first Write; first bytes on step arms is all of startup; resize back, erase and picker setup are untimed")
	fmt.Println("path: Surface time is the whole frame (tree, layout, paint, raster, encode, diff), so an encode budget that passes here passes alone")
	fmt.Println("path: image bytes and tiles count the Sixel DCS (ESC P to ESC \\) and Kitty APC (ESC _G to ESC \\) sequences in each frame's one Write; a chunked Kitty tile counts once per chunk, and only first runs with kitty")
	fmt.Println("path: Idle runs bench/scenarios/idle, the counter on twi.Backend with the real clock, and reads process CPU time over 10 s after the first frame")
	fmt.Println("path: AppIdle runs bench/scenarios/idle -app surfaces over 60 s with Sixel; rss and private are the process working set and private bytes (K32GetProcessMemoryInfo, Windows; peak rss elsewhere) and heap is Go HeapInuse, read at the end of the window")
	if runtime.GOOS == "windows" {
		fmt.Println("path: Windows counts process CPU time in 15.625 ms ticks, so Idle and AppIdle read 0 below 0.157% of a core over 10 s and 0.026% over 60 s")
	}
	fmt.Println("comparable: only against runs on the machine above")
	fmt.Println()
	fmt.Println("| scenario | runs | p50 ms | p95 ms | p95 worst ms | settled p50 ms | settled p95 ms | bytes/frame | image bytes/frame | tiles/frame | extra bytes | first bytes | heap MB max | rss MB max | private MB max | B/op | allocs/op | cpu % core max | wakeups | idle bytes | budget | verdict |")
	fmt.Println("|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|")
	var misses []string
	for _, name := range order {
		s := samples[name]
		ms := func(unit string, pick func([]float64) float64) string { return cell(s[unit], pick, 1e-6, 3) }
		top := slices.Max[[]float64]
		verdict, target := "-", "-"
		if b, ok := budgets[name]; ok {
			target, verdict = b.text, "pass"
			for _, c := range b.checks {
				if len(s[c.unit]) == 0 || top(s[c.unit]) >= c.limit {
					verdict = "MISS " + c.unit
					misses = append(misses, name+" ("+c.unit+")")
					break
				}
			}
		}
		fmt.Printf("| %s | %d | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
			name, len(s["ns/op"]), ms("p50-ns", median), ms("p95-ns", median), ms("p95-ns", top), ms("settled-p50-ns", median), ms("settled-p95-ns", median),
			cell(s["bytes/frame"], median, 1, 1), cell(s["image-bytes/frame"], median, 1, 1), cell(s["tiles/frame"], median, 1, 2),
			cell(s["extra-bytes/op"], median, 1, 0), cell(s["first-bytes/frame"], median, 1, 0), cell(s["heap-MB"], top, 1, 2), cell(s["rss-MB"], top, 1, 1), cell(s["private-MB"], top, 1, 1),
			cell(s["B/op"], median, 1, 0), cell(s["allocs/op"], median, 1, 0),
			cell(s["cpu-%"], top, 1, 3), cell(s["wakeups"], top, 1, 0), cell(s["idle-bytes"], top, 1, 0),
			target, verdict)
	}
	fmt.Println()
	var absent []string
	for name, b := range budgets {
		if samples[name] == nil {
			absent = append(absent, name+" ("+b.text+")")
		}
	}
	slices.Sort(absent)
	fmt.Println("missed budgets:", list(misses))
	fmt.Println("budgets not run:", list(absent))
	fmt.Println()
	tracked := []struct{ unit, label string }{
		{"p50-ns", "p50-ms"}, {"p95-ns", "p95-ms"}, {"settled-p95-ns", "settled-p95-ms"}, {"bytes/frame", "bytes"}, {"extra-bytes/op", "extra-bytes"}, {"image-bytes/frame", "image-bytes"}, {"tiles/frame", "tiles"},
		{"allocs/op", "allocs"}, {"cpu-%", "cpu-%"}, {"wakeups", "wakeups"}, {"rss-MB", "rss-MB"}, {"heap-MB", "heap-MB"},
	}
	for _, name := range order {
		line := "ledger: " + commit + " " + name
		for _, t := range tracked {
			if v := samples[name][t.unit]; len(v) > 0 {
				scale := 1.0
				if strings.HasSuffix(t.unit, "-ns") {
					scale = 1e-6
				}
				line += " " + t.label + "=" + cell(v, median, scale, 3)
			}
		}
		fmt.Println(line)
	}
}

func list(names []string) string {
	if len(names) == 0 {
		return "none"
	}
	return strings.Join(names, ", ")
}

func median(v []float64) float64 {
	sorted := slices.Sorted(slices.Values(v))
	return (sorted[(len(sorted)-1)/2] + sorted[len(sorted)/2]) / 2
}

func cell(v []float64, pick func([]float64) float64, scale float64, digits int) string {
	if len(v) == 0 {
		return "-"
	}
	return strconv.FormatFloat(pick(v)*scale, 'f', digits, 64)
}

func command(name string, args ...string) string {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}
