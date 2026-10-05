package ui

import (
	"strings"
	"testing"

	"github.com/pehcastro/twind/twi"
	"github.com/pehcastro/twind/twi/drive"
	"github.com/pehcastro/twind/twi/style"
	"github.com/pehcastro/twind/twi/theme"
)

func BenchmarkMerge(b *testing.B) {
	base := button(Default, SizeDefault, idleRing(Default)+" "+focusRing)
	for name, build := range map[string]func(){
		"button":         func() { Button(Ghost, SizeDefault, twi.Text("1")) },
		"button-w-26":    func() { Button(Ghost, SizeDefault, twi.Class("w-26"), twi.Text("1")) },
		"skeleton":       func() { Skeleton() },
		"skeleton-round": func() { Skeleton(twi.Class("rounded-full")) },
		"field-group":    func() { FieldGroup() },
		"field-group-w":  func() { FieldGroup(twi.Class("w-26")) },
		"previous-link":  func() { PaginationPrevious() },
		"merge-button-w": func() { Merge(base, "w-26") },
		"merge-field-w":  func() { Merge("flex flex-col w-full gap-1", "w-26") },
	} {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				build()
			}
		})
	}
}

func TestOverride(t *testing.T) {
	light := zinc(t, theme.Light)
	width := func(n float64) func(style.ComputedStyle) bool {
		return func(s style.ComputedStyle) bool { return s.Width == cells(n) }
	}
	radius := func(r style.Radius) func(style.ComputedStyle) bool {
		return func(s style.ComputedStyle) bool { return s.Radius == r }
	}
	checkParts(t, []partCase{
		{"skeleton rounded-full", light, Skeleton(twi.Class("rounded-full")), nil, radius(style.RadiusFull)},
		{"field group w-26", light, FieldGroup(twi.Class("w-26")), nil, width(26)},
		{"the later of two caller classes", light, FieldGroup(twi.Class("w-26"), twi.Class("w-20")), nil, width(20)},
		{"a hover class keeps the base background", light, Skeleton(twi.Class("hover:bg-muted")), nil, func(s style.ComputedStyle) bool {
			return s.Background == light.Tokens[theme.Accent] && s.Radius == style.RadiusMd
		}},
		{"a child's class stays its own", light, Skeleton(Skeleton(twi.Class("rounded-full"))), nil, radius(style.RadiusMd)},
		{"the child gets it", light, Skeleton(Skeleton(twi.Class("rounded-full"))), []int{0}, radius(style.RadiusFull)},
		{"through Button to the pagination link", light, PaginationLink(false, twi.Class("w-5")), nil, width(5)},
		{"a control's root", light, NewSwitch(twi.New()).Node(twi.Class("w-6")), nil, width(6)},
		{"spaces only", light, Skeleton(twi.Class("   ")), nil, radius(style.RadiusMd)},
	})
	sheet, err := styles()
	if err != nil {
		t.Fatal(err)
	}
	frame := func(n twi.Node) drive.Frame {
		d := drive.New(func(rt *twi.Runtime) func() twi.Node {
			rt.SetTheme(light)
			return func() twi.Node {
				return twi.Element(twi.Class("flex flex-col w-60 gap-1 p-1 h-full bg-background text-foreground"), n)
			}
		}, drive.Size(64, 6), drive.With(twi.Styles(sheet)))
		defer func() { _ = d.Close() }()
		return d.Frame()
	}
	group := frame(FieldGroup(twi.Class("w-26"), twi.Element(twi.Class("h-1 bg-primary"))))
	filled := 0
	for x := range group.Width() {
		if group.At(x, 1).Bg.RGBA == light.Tokens[theme.Primary].RGBA {
			filled++
		}
	}
	if filled != 26 {
		t.Errorf("FieldGroup(twi.Class(\"w-26\")) fills %d cells, want 26:\n%s", filled, group.Text())
	}
	square, round := strings.Split(frame(Skeleton(twi.Class("h-3 w-6 border rounded-none"))).Text(), "\n")[1], strings.Split(frame(Skeleton(twi.Class("h-3 w-6 border"))).Text(), "\n")[1]
	t.Logf("rounded-none %q, own rounded-md %q", square, round)
	if !strings.HasPrefix(strings.TrimSpace(square), "┌") || !strings.HasPrefix(strings.TrimSpace(round), "╭") {
		t.Error("Skeleton(twi.Class(\"h-3 w-6 border rounded-none\")) does not draw square corners over its own rounded-md")
	}
}

func TestMergeInPagination(t *testing.T) {
	light := zinc(t, theme.Light)
	for name, n := range map[string]twi.Node{"previous": PaginationPrevious(), "next": PaginationNext()} {
		if s := computed(t, light, n, nil, nil); s.Padding.Left != cells(1) || s.Padding.Right != cells(1) {
			t.Errorf("%s: padding %+v, want the link's px-1 over the button's px-2", name, s.Padding)
		}
	}
}

func TestMerge(t *testing.T) {
	for _, c := range []struct{ base, extra, want string }{
		{"flex flex-col w-full gap-1", "w-26", "flex flex-col gap-1 w-26"},
		{"px-4 py-2", "px-2", "py-2 px-2"},
		{"px-4 py-1 pt-1 mt-1", "p-2", "mt-1 p-2"},
		{"px-4", "pl-2", "px-4 pl-2"},
		{"w-4 h-2 min-w-0", "size-4", "min-w-0 size-4"},
		{"top-1 left-2 z-50", "inset-0", "z-50 inset-0"},
		{"rounded-t-lg rounded-bl-sm", "rounded-md", "rounded-md"},
		{"gap-x-1 gap-y-2", "gap-2", "gap-2"},
		{"bg-primary hover:bg-primary/90", "hover:bg-accent", "bg-primary hover:bg-accent"},
		{"bg-primary", "hover:bg-accent", "bg-primary hover:bg-accent"},
		{"data-[state=active]:bg-background", "data-[active=true]:bg-muted", "data-[state=active]:bg-background data-[active=true]:bg-muted"},
		{"data-[state=active]:bg-background", "data-[state=active]:bg-muted", "data-[state=active]:bg-muted"},
		{"dark:hover:bg-input/50", "hover:dark:bg-accent", "hover:dark:bg-accent"},
		{"text-sm text-left text-muted-foreground", "text-foreground", "text-sm text-left text-foreground"},
		{"leading-6 text-muted-foreground", "text-lg", "text-muted-foreground text-lg"},
		{"border border-input border-dashed", "border-2", "border-input border-dashed border-2"},
		{"border-t border-border", "border-x-transparent", "border-t border-border border-x-transparent"},
		{"shadow-sm", "shadow-[0_0_0_1px_var(--color-ring)]", "shadow-[0_0_0_1px_var(--color-ring)]"},
		{"shadow-sm shadow-red-500", "shadow-md", "shadow-red-500 shadow-md"},
		{"font-medium font-sans", "font-mono", "font-medium font-mono"},
		{"bg-primary bg-cover bg-linear-to-r", "bg-muted/50", "bg-cover bg-linear-to-r bg-muted/50"},
		{"focus-visible:shadow-[0_0_0_1px_var(--x),0_0_0_3px_color-mix(in_oklab,var(--x)_50%,transparent)] [&:hover]:underline", "focus-visible:shadow-none", "[&:hover]:underline focus-visible:shadow-none"},
		{"w-4 h-[calc(100%-1px)]", "w-[3px] h-1", "w-[3px] h-1"},
		{"[color:red] [background:blue]", "[color:blue]", "[background:blue] [color:blue]"},
		{"mx-2 top-0", "-mx-1 -top-1", "-mx-1 -top-1"},
		{"w-full", "w-4!", "w-full w-4!"},
		{"w-full", "!w-4", "w-full !w-4"},
		{"flex flex-1 flex-col flex-wrap", "flex-row", "flex flex-1 flex-wrap flex-row"},
		{"flex items-center", "hidden", "items-center hidden"},
		{"flex", "grid", "grid"},
		{"group my-card peer", "my-card", "group peer my-card"},
		{"  px-2   ", "", "px-2"},
		{"grow shrink-0 basis-4", "flex-1", "flex-1"},
		{"overflow-x-auto overflow-y-hidden", "overflow-hidden", "overflow-hidden"},
		{"justify-between max-h-(--radix-select-content-available-height)", "justify-items-center max-h-8", "justify-between justify-items-center max-h-8"},
		{"ring-2 ring-ring/50", "ring-[3px]", "ring-ring/50 ring-[3px]"},
	} {
		if got := Merge(c.base, c.extra); got != c.want {
			t.Errorf("Merge(%q, %q) = %q, want %q", c.base, c.extra, got, c.want)
		}
	}
}
