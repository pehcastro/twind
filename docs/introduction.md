# Introduction

Twind is a UI runtime for terminal programs, written in Go. You build a tree of elements, style it with Tailwind classes, and Twind lays it out, paints it and keeps it on screen. To start one, read [Installation](installation.md).

## Why Twind

Most terminal libraries ask you to think in strings and cursor positions. Twind asks you to think the way a web page works:

- **A retained tree.** Elements with children, keys and event handlers, not a string rebuilt on every update.
- **Real Tailwind.** Classes are compiled by the official Tailwind build into typed Go when you build, so the runtime never parses CSS.
- **Flexbox and grid** in whole cells, with padding, gaps, borders, positioning and scrolling.
- **Events and focus** that bubble, with Tab order, pointer hover and clicks.
- **Surfaces as pixels.** Rounded corners, shadows and translucent panels are drawn as images where the terminal supports them, with real text on top.

## A first element

```go
twi.Element(
	twi.Class("flex flex-col gap-1 rounded-lg border px-2 py-1"),
	twi.Text("Hello from Twind"),
)
```

## What is in this site

This documentation is itself a Twind app. Every preview on a component page runs the Go file shown in its Code tab, so a demo and its code cannot drift apart.

- Press **Ctrl+K** to search every page.
- Press **Tab** to move through the sidebar and **Enter** to open a page.
- Press **t** to change the theme; the arrows preview it live.
