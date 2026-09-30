# highlight

Syntax highlighting for Twind, heavily inspired by [twinkleplop](https://github.com/pngwn/twinkleplop) by pngwn (MIT licence, reproduced below).

A grammar is a list of states, each a list of rules. The first rule that matches the next byte wins, multi-byte literals are tried longest first, and a rule may enter, go to or leave a state. `Compile` turns a `Definition` into lookup tables once; the caller holds the `*Grammar` and reuses it. `Tokens` walks a string and yields `Span{Kind, Start, End}` values that cover every byte exactly once, with no allocation per span.

```go
g := highlight.Go()
for s := range highlight.Tokens(src, g) {
	colour := palette[s.Kind]
	draw(src[s.Start:s.End], colour)
}
```

`Go`, `Bash`, `JSON` and `TOML` are ports of twinkleplop's grammars for the same languages, with their reclassifier passes, and their spans match twinkleplop's `tokenize()` at full fidelity for the same input (`testdata/`).

The passes run over the whole file after the grammar, so `Tokens` fills a buffer the grammar reuses before it yields the first span; JSON and TOML have no passes and stream. Go claims, highest first: the name after `package` and an import alias are `Namespace`; parameter names after `func`, including receivers and names that share a type, are `Parameter`; an identifier before `(`, or before `[...](`, is `Function`; an `UPPER_SNAKE` identifier is `Constant`. Bash merges `$` plus the identifier after it into one `Variable`, joins `0xff` and `16#ff` into one `Number`, promotes reserved words, builtins and booleans, and marks `name()` as `Function`. Twinkleplop's Go passes emit no type kind, so neither does this.

Rules are built with the same helpers twinkleplop uses:

| twinkleplop | here |
|---|---|
| `match(patterns, token)` | `Match(kind, literals...)`, with `.Range(lo, hi)`, `.Letters()`, `.Digits()`, `.HexDigits()` |
| `on(patterns)` | `Match(Text, literals...)` |
| `keyword(words, {}, token)` | `Word(kind, words...)` |
| `within(start, end, token, opts)` | `Within(kind, start, end)`, with `.Escaped(escape)` and `.OneLine()` |
| `fallback({ token })` | `Fallback(kind)` |
| `enter(s)`, `goto(s)`, `leave()` | `.Enter(s)`, `.Goto(s)`, `.Leave()` |
| `mode: "probe"`, `fallback: s` | `State{Probe: true, AtEnd: s}` |

`Text` means no token: bytes a rule consumes without a kind, and bytes no rule matches, come out as `Text` spans. `DefaultPalette()` maps each kind to a theme token: `foreground` for text and identifiers, a `syntax-*` token for the rest (`syntax-keyword`, `syntax-string`, `syntax-number`, `syntax-comment`, `syntax-function`, `syntax-constant`, `syntax-namespace`, `syntax-parameter`, `syntax-punctuation`). Every shipped theme defines them after github-light and github-dark, at 4.5:1 or more against `muted` and `background`, so a theme switch needs no new palette.

## twinkleplop licence

```
MIT License

Copyright (c) 2025 pngwn

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```
