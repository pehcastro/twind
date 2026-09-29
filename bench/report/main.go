package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"
)

type budget struct {
	unit  string
	limit float64
	text  string
}

func main() {
	budgets := map[string]budget{
		"OneCell/sync=true":  {"bytes/frame", 32, "bytes < 32"},
		"ColdStart343":       {"p95-ns", 30e6, "p95 < 30 ms"},
		"Render/nodes=10000": {"heap-MB", 20, "heap < 20 MB"},
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
	fmt.Printf("machine: cpu=%q os=%q goos=%s goarch=%s go=%s procs=%s commit=%s date=%s terminal=none width=80\n",
		header["cpu"], system, header["goos"], header["goarch"], runtime.Version(), procs, commit, time.Now().Format(time.RFC3339))
	fmt.Println("path: twi.Render and terminal.Writer.Diff into a byte counter; no runtime or driver exists yet")
	fmt.Println("comparable: only against runs on the machine above")
	fmt.Println()
	fmt.Println("| scenario | runs | p50 ms | p95 ms | p95 worst ms | bytes/frame | first bytes | heap MB max | B/op | allocs/op | budget | verdict |")
	fmt.Println("|---|---|---|---|---|---|---|---|---|---|---|---|")
	var misses []string
	for _, name := range order {
		s := samples[name]
		ms := func(unit string, pick func([]float64) float64) string { return cell(s[unit], pick, 1e-6, 3) }
		verdict, target := "-", "-"
		if b, ok := budgets[name]; ok {
			target, verdict = b.text, "pass"
			if len(s[b.unit]) == 0 {
				verdict = "no data"
				misses = append(misses, name)
			} else if slices.Max(s[b.unit]) >= b.limit {
				verdict = "MISS"
				misses = append(misses, name)
			}
		}
		fmt.Printf("| %s | %d | %s | %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
			name, len(s["ns/op"]), ms("p50-ns", median), ms("p95-ns", median), ms("p95-ns", slices.Max[[]float64]),
			cell(s["bytes/frame"], median, 1, 1), cell(s["first-bytes/frame"], median, 1, 0), cell(s["heap-MB"], slices.Max[[]float64], 1, 2),
			cell(s["B/op"], median, 1, 0), cell(s["allocs/op"], median, 1, 0), target, verdict)
	}
	fmt.Println()
	if len(misses) > 0 {
		fmt.Println("missed budgets:", strings.Join(misses, ", "))
	} else {
		fmt.Println("missed budgets: none")
	}
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
