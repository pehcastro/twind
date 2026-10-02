package dev

import (
	"bytes"
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/twind-dev/twind/internal/dev/konst"
)

type StaleError string

func (e StaleError) Error() string { return "dev plan: " + string(e) }

type command struct {
	env  []string
	dir  string
	args []string
}

type planned struct {
	path, block string
	compile     command
	importcfg   string
	embedcfg    string
	deps        []string
	files       []string
	plain       bool
}

type Plan struct {
	work     string
	main     string
	order    []*planned
	owner    map[string]*planned
	heads    map[string]string
	have     map[string]string
	link     command
	linkCfg  string
	builds   int
	commits  int
	compiles atomic.Int64
	cache    sync.Mutex
	sums     map[string]string
	exports  map[string]string
	archives map[string]string
	exes     map[string]string
}

func Capture(ctx context.Context, root, pkg, work string) (*Plan, error) {
	dry := exec.CommandContext(ctx, "go", "build", "-n", "-a", "-o", filepath.Join(work, "dry"), pkg)
	dry.Dir = root
	script, err := dry.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("go build -n: %w\n%s", err, script)
	}
	list := exec.CommandContext(ctx, "go", "list", "-export", "-deps", "-f", `{{.ImportPath}}{{"\t"}}{{.Export}}{{"\t"}}{{.DepOnly}}{{"\t"}}{{with .Module}}{{.Main}}{{end}}`, pkg)
	list.Dir = root
	listed, err := list.Output()
	if err != nil {
		return nil, fmt.Errorf("go list -export: %w", err)
	}
	p := &Plan{work: work, owner: map[string]*planned{}, heads: map[string]string{}, have: map[string]string{}, sums: map[string]string{}, exports: map[string]string{}, archives: map[string]string{}, exes: map[string]string{}}
	local := map[string]bool{}
	for line := range strings.Lines(string(listed)) {
		f := strings.Split(strings.TrimRight(line, "\r\n"), "\t")
		p.have[f[0]] = f[1]
		local[f[0]] = f[3] == "true"
		if f[2] == "false" {
			p.main = f[0]
		}
	}
	heredocs := map[string]string{}
	var cwd, open string
	var body strings.Builder
	for line := range strings.Lines(strings.ReplaceAll(string(script), "\r\n", "\n")) {
		line = strings.TrimSuffix(line, "\n")
		if open != "" {
			if before, ok := strings.CutSuffix(line, "EOF"); ok {
				body.WriteString(before)
				heredocs[open], open = body.String(), ""
				continue
			}
			body.WriteString(line + "\n")
			continue
		}
		if rest, ok := strings.CutPrefix(line, "echo '"); ok {
			text, target, _ := strings.Cut(rest, "' > ")
			target, _, _ = strings.Cut(target, " ")
			heredocs[target] = text + "\n"
			continue
		}
		if rest, ok := strings.CutPrefix(line, "cat >"); ok {
			open, _, _ = strings.Cut(rest, " ")
			body.Reset()
			continue
		}
		if rest, ok := strings.CutPrefix(line, "cd "); ok {
			cwd = rest
			continue
		}
		c, tool := parseCommand(line, cwd)
		switch tool {
		case "compile":
			out, path := flagValue(c.args, "-o"), flagValue(c.args, "-p")
			block := filepath.Dir(out)
			if path == "main" {
				path = p.main
			}
			if !local[path] {
				continue
			}
			pk := &planned{path: path, block: block, compile: c, importcfg: heredocs[filepath.Join(block, "importcfg")], embedcfg: heredocs[filepath.Join(block, "embedcfg")]}
			pk.plain = flagValue(c.args, "-symabis") == "" && flagValue(c.args, "-asmhdr") == ""
			for cfg := range strings.Lines(pk.importcfg) {
				if dep, _, ok := strings.Cut(strings.TrimPrefix(cfg, "packagefile "), "="); ok && strings.HasPrefix(cfg, "packagefile ") {
					pk.deps = append(pk.deps, dep)
				}
			}
			files := c.args[indexOf(c.args, "-pack")+1:]
			if pk.embedcfg != "" {
				var embed struct{ Files map[string]string }
				if err := json.Unmarshal([]byte(pk.embedcfg), &embed); err != nil {
					return nil, fmt.Errorf("embedcfg of %s: %w", path, err)
				}
				for _, file := range embed.Files {
					files = append(files, file)
				}
			}
			for _, file := range files {
				if !filepath.IsAbs(file) {
					file = filepath.Join(cwd, file)
				}
				p.owner[file] = pk
				pk.files = append(pk.files, file)
				if filepath.Ext(file) == ".go" {
					if p.heads[file], err = head(file); err != nil {
						return nil, err
					}
				}
			}
			p.order = append(p.order, pk)
		case "link":
			p.link, p.linkCfg = c, heredocs[flagValue(c.args, "-importcfg")]
		}
	}
	if p.main == "" || p.link.args == nil {
		return nil, fmt.Errorf("go build -n gave no main package and link for %s", pkg)
	}
	return p, nil
}

type Step struct {
	Package  string
	Took     time.Duration
	Exported bool
	Reused   bool
}

func (p *Plan) Build(ctx context.Context, changed []string, exe string) (string, []Step, error) {
	dirty := map[string]bool{}
	for _, file := range changed {
		pk, ok := p.owner[file]
		if !ok {
			return "", nil, StaleError(file + " is in no planned package")
		}
		if !pk.plain {
			return "", nil, StaleError(pk.path + " has assembly")
		}
		if want, ok := p.heads[file]; ok {
			if now, err := head(file); err != nil || now != want {
				return "", nil, StaleError("the imports or build constraints of " + file + " changed")
			}
		} else if _, err := os.Stat(file); err != nil {
			return "", nil, StaleError(file + " is gone")
		}
		dirty[pk.path] = true
	}
	p.builds++
	work := filepath.Join(p.work, "b"+strconv.Itoa(p.builds))
	next, steps, err := p.compileDirty(ctx, dirty, work)
	if err != nil {
		return "", nil, err
	}
	began, ids := time.Now(), p.identities(next)
	key := ids[p.main] + "\n" + rewrite(p.linkCfg, ids)
	p.cache.Lock()
	kept, ok := p.exes[key]
	p.cache.Unlock()
	if ok {
		if _, err := os.Stat(kept); err == nil {
			p.cache.Lock()
			p.have, p.commits = next, p.commits+1
			p.cache.Unlock()
			return kept, append(steps, Step{Package: "link", Took: time.Since(began), Reused: true}), nil
		}
	}
	cfg := flagValue(p.link.args, "-importcfg")
	link := command{env: p.link.env, dir: p.link.dir, args: slices.Insert(slices.Clone(p.link.args), 1, konst.NoDWARF)}
	link.args[len(link.args)-1] = next[p.main]
	if err := os.MkdirAll(filepath.Dir(strings.Replace(cfg, "$WORK", work, 1)), 0o755); err != nil {
		return "", nil, err
	}
	if err := os.WriteFile(strings.Replace(cfg, "$WORK", work, 1), []byte(rewrite(p.linkCfg, next)), 0o600); err != nil {
		return "", nil, err
	}
	if err := run(ctx, link, work, exe); err != nil {
		return "", nil, err
	}
	p.cache.Lock()
	defer p.cache.Unlock()
	p.have, p.commits, p.exes[key] = next, p.commits+1, exe
	return exe, append(steps, Step{Package: "link", Took: time.Since(began)}), nil
}

func (p *Plan) Warm(ctx context.Context) error {
	dirty := map[string]bool{}
	for _, pk := range p.order {
		dirty[pk.path] = pk.plain
	}
	p.cache.Lock()
	base := p.commits
	p.cache.Unlock()
	next, _, err := p.compileDirty(ctx, dirty, filepath.Join(p.work, "warm"))
	p.cache.Lock()
	defer p.cache.Unlock()
	if err == nil && p.commits == base {
		p.have = next
	}
	return err
}

func (p *Plan) compileDirty(ctx context.Context, dirty map[string]bool, work string) (map[string]string, []Step, error) {
	p.cache.Lock()
	next := maps.Clone(p.have)
	p.cache.Unlock()
	var steps []Step
	var failed error
	var mu sync.Mutex
	var wg sync.WaitGroup
	cond := sync.NewCond(&mu)
	exported, finished, inPlan := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, pk := range p.order {
		inPlan[pk.path] = true
	}
	pending := func(pk *planned) (unfinished []string) {
		for _, dep := range pk.deps {
			if inPlan[dep] && !finished[dep] {
				unfinished = append(unfinished, dep)
			}
			dirty[pk.path] = dirty[pk.path] || exported[dep]
		}
		return unfinished
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	for _, pk := range p.order {
		wg.Go(func() {
			mu.Lock()
			defer func() {
				finished[pk.path] = true
				cond.Broadcast()
				mu.Unlock()
			}()
			for len(pending(pk)) > 0 && !dirty[pk.path] && failed == nil {
				cond.Wait()
			}
			if !dirty[pk.path] || failed != nil {
				return
			}
			began, old := time.Now(), next[pk.path]
			for {
				early, importcfg, deps := pending(pk), rewrite(pk.importcfg, next), map[string]string{}
				for _, dep := range pk.deps {
					deps[dep] = next[dep]
				}
				mu.Unlock()
				key, err := p.sourceKey(pk, deps)
				p.cache.Lock()
				archive, reused := p.archives[key]
				p.cache.Unlock()
				if err == nil && !reused {
					archive, err = p.compile(ctx, pk, work, importcfg)
				}
				if err == nil && !reused {
					if again, _ := p.sourceKey(pk, deps); again == key {
						err = p.remember(key, archive)
					}
				}
				changed := false
				if err == nil {
					changed, err = exportChanged(old, archive)
				}
				mu.Lock()
				for err == nil && failed == nil && len(pending(pk)) > 0 {
					cond.Wait()
				}
				if err != nil || failed != nil {
					failed = cmp.Or(failed, err)
					cancel()
					return
				}
				if !slices.ContainsFunc(early, func(dep string) bool { return exported[dep] }) {
					next[pk.path], exported[pk.path] = archive, changed
					steps = append(steps, Step{pk.path, time.Since(began), changed, reused})
					return
				}
			}
		})
	}
	wg.Wait()
	return next, steps, failed
}

func (p *Plan) identities(archives map[string]string) map[string]string {
	p.cache.Lock()
	defer p.cache.Unlock()
	ids := make(map[string]string, len(archives))
	for pkg, archive := range archives {
		ids[pkg] = cmp.Or(p.sums[archive], archive)
	}
	return ids
}

func (p *Plan) exportID(archive string) (string, error) {
	p.cache.Lock()
	id, ok := p.exports[archive]
	p.cache.Unlock()
	if ok {
		return id, nil
	}
	export, err := exportData(archive)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(export)
	p.cache.Lock()
	defer p.cache.Unlock()
	p.exports[archive] = hex.EncodeToString(sum[:])
	return p.exports[archive], nil
}

func (p *Plan) remember(key, archive string) error {
	data, err := os.ReadFile(archive)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	p.cache.Lock()
	defer p.cache.Unlock()
	p.archives[key], p.sums[archive] = archive, hex.EncodeToString(sum[:])
	return nil
}

func (p *Plan) sourceKey(pk *planned, deps map[string]string) (string, error) {
	ids := map[string]string{}
	for dep, archive := range deps {
		id, err := p.exportID(archive)
		if err != nil {
			return "", err
		}
		ids[dep] = id
	}
	h := sha256.New()
	h.Write([]byte(strings.Join(pk.compile.args, "\x00") + "\x00" + pk.embedcfg + "\x00" + rewrite(pk.importcfg, ids)))
	for _, file := range slices.Sorted(slices.Values(pk.files)) {
		src, err := os.ReadFile(file)
		if err != nil {
			return "", err
		}
		h.Write([]byte(file + "\x00"))
		h.Write(src)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (p *Plan) compile(ctx context.Context, pk *planned, work, importcfg string) (string, error) {
	work = filepath.Join(work, strconv.FormatInt(p.compiles.Add(1), 10))
	block := strings.Replace(pk.block, "$WORK", work, 1)
	if err := os.MkdirAll(block, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(block, "importcfg"), []byte(importcfg), 0o600); err != nil {
		return "", err
	}
	if pk.embedcfg != "" {
		if err := os.WriteFile(filepath.Join(block, "embedcfg"), []byte(pk.embedcfg), 0o600); err != nil {
			return "", err
		}
	}
	lean := pk.compile
	lean.args = slices.Insert(slices.Clone(lean.args), indexOf(lean.args, "-pack"), konst.NoInline)
	return filepath.Join(block, "_pkg_.a"), run(ctx, lean, work, "")
}

func exportChanged(old, archive string) (bool, error) {
	before, err := exportData(old)
	if err != nil {
		return false, err
	}
	after, err := exportData(archive)
	return !bytes.Equal(before, after), err
}

func run(ctx context.Context, c command, work, exe string) error {
	args := make([]string, len(c.args))
	for i, a := range c.args {
		args[i] = strings.ReplaceAll(a, "$WORK", work)
		if exe != "" && i > 0 && c.args[i-1] == "-o" {
			args[i] = exe
		}
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Dir, cmd.Env = c.dir, append(os.Environ(), c.env...)
	if out, err := cmd.CombinedOutput(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("%s%w", out, err)
	}
	return nil
}

func rewrite(cfg string, archives map[string]string) string {
	var out strings.Builder
	for line := range strings.Lines(cfg) {
		if rest, ok := strings.CutPrefix(line, "packagefile "); ok {
			path, _, _ := strings.Cut(rest, "=")
			line = "packagefile " + path + "=" + archives[path] + "\n"
		}
		out.WriteString(line)
	}
	return out.String()
}

func parseCommand(line, cwd string) (command, string) {
	var words []string
	for rest := strings.TrimSpace(line); rest != ""; rest = strings.TrimLeft(rest, " ") {
		var word strings.Builder
		for rest != "" && rest[0] != ' ' {
			switch rest[0] {
			case '"':
				quoted, err := strconv.QuotedPrefix(rest)
				if err != nil {
					return command{}, ""
				}
				text, _ := strconv.Unquote(quoted)
				word.WriteString(text)
				rest = rest[len(quoted):]
			case '\'':
				end := strings.IndexByte(rest[1:], '\'')
				if end < 0 {
					return command{}, ""
				}
				word.WriteString(rest[1 : end+1])
				rest = rest[end+2:]
			default:
				word.WriteByte(rest[0])
				rest = rest[1:]
			}
		}
		words = append(words, word.String())
	}
	for i, w := range words {
		if !strings.Contains(w, "=") {
			return command{env: words[:i], dir: cwd, args: words[i:]}, strings.TrimSuffix(filepath.Base(w), ".exe")
		}
	}
	return command{}, ""
}

func flagValue(args []string, name string) string {
	if i := indexOf(args, name); i >= 0 && i+1 < len(args) {
		return args[i+1]
	}
	return ""
}

func indexOf(args []string, name string) int {
	for i, a := range args {
		if a == name {
			return i
		}
	}
	return -1
}

func exportData(archive string) ([]byte, error) {
	data, err := os.ReadFile(archive)
	if err != nil {
		return nil, err
	}
	rest, ok := bytes.CutPrefix(data, []byte(konst.ArchiveMagic))
	for ok && len(rest) >= konst.ArchiveHeader {
		size, err := strconv.Atoi(strings.TrimSpace(string(rest[konst.ArchiveSizeAt:konst.ArchiveSizeEnd])))
		if err != nil || konst.ArchiveHeader+size > len(rest) {
			break
		}
		body := rest[konst.ArchiveHeader : konst.ArchiveHeader+size]
		if strings.TrimSpace(string(rest[:konst.ArchiveNameEnd])) == "__.PKGDEF" {
			if _, export, found := bytes.Cut(body, []byte("\n$$B\n")); found {
				return export, nil
			}
		}
		rest = rest[konst.ArchiveHeader+size+size%2:]
	}
	return nil, fmt.Errorf("%s: no export data", archive)
}

func head(path string) (string, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	f, err := parser.ParseFile(token.NewFileSet(), path, src, parser.ImportsOnly)
	if err != nil {
		return "", err
	}
	var h string
	for line := range strings.Lines(string(src[:f.Package-1])) {
		if strings.HasPrefix(line, "//go:build") || strings.HasPrefix(line, "// +build") {
			h += line
		}
	}
	for _, imp := range f.Imports {
		h += "\n" + imp.Path.Value
	}
	return h, nil
}
