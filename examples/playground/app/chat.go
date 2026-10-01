package app

import (
	"strconv"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/ui"
)

func chatLines() []string {
	return []string{
		"Can you look at why the build is slow on Windows?",
		"Sure. The first run spends most of its time fetching the Tailwind binary.",
		"It is cached after that, right?",
		"Yes, in .twind/bin, checked against a pinned checksum.",
		"Then what is slow on the second run?",
		"go generate runs Tailwind on every package, even the ones that did not change.",
		"Can we skip the fresh ones?",
		"twirgen -check already knows: it hashes the inputs. I can skip a package whose hash matches.",
		"Do it, and show me the numbers.",
		"Second run: 9.8 s before, 1.4 s after. The report is attached.",
	}
}

func chatMessage(i int, text string, extra ...twi.NodeOption) twi.Node {
	a, v, who, name := ui.Start, ui.Muted, "AI", "Assistant"
	if i%2 == 0 {
		a, v, who, name = ui.End, ui.Default, "PD", "Pedro"
	}
	return ui.Message(a,
		ui.MessageAvatar(ui.Avatar(ui.SizeSM, ui.AvatarFallback(twi.Text(who)))),
		ui.MessageContent(append([]twi.NodeOption{ui.MessageHeader(twi.Text(name)), ui.Bubble(v, a, ui.BubbleContent(twi.Text(text)))}, extra...)...),
	)
}

func messagePage(controls) twi.Node {
	return show("Message", "an avatar, a header, bubbles and a footer; end-aligned for the sender",
		ui.MessageGroup(twi.Class("w-70"),
			chatMessage(0, "Is the release notes draft ready?", ui.MessageFooter(twi.Text("read 2:14 PM"))),
			chatMessage(1, "Almost. Two sections left: the driver and the themes.", ui.MessageFooter(twi.Text("2:15 PM"))),
		))
}

func bubblePage(controls) twi.Node {
	var bubbles []twi.NodeOption
	for i, v := range []ui.Variant{ui.Default, ui.Secondary, ui.Muted, ui.Tinted, ui.Outline, ui.Ghost, ui.Destructive} {
		label := []string{"default", "secondary", "muted", "tinted", "outline", "ghost", "destructive"}[i]
		bubbles = append(bubbles, ui.Bubble(v, map[bool]ui.Alignment{true: ui.End, false: ui.Start}[i%2 == 1], ui.BubbleContent(twi.Text(label+" bubble"))))
	}
	bubbles = append(bubbles, ui.Bubble(ui.Secondary, ui.Start, ui.BubbleContent(twi.Text("with reactions")), ui.BubbleReactions(ui.Bottom, ui.End, twi.Text("✓ 2"))))
	return show("Bubble", "seven variants, either side, and reactions under a corner",
		ui.BubbleGroup(append([]twi.NodeOption{twi.Class("w-60")}, bubbles...)...))
}

func messageScrollerPage(c controls) twi.Node {
	k := c.kit
	lines := chatLines()
	items := []twi.NodeOption{twi.Class("px-1"), k.scroller.Item(ui.Marker(ui.Ruled, ui.MarkerContent(twi.Text("Today"))))}
	for i := range k.sent {
		text := lines[i%len(lines)]
		if i >= len(lines) {
			text = "Reply " + strconv.Itoa(i+1) + ": " + text
		}
		var extra []twi.NodeOption
		if i == len(lines)-1 {
			extra = append(extra, ui.Attachment(ui.Done, ui.Horizontal, ui.AttachmentMedia(ui.Icon, twi.Text("▤")), ui.AttachmentContent(ui.AttachmentTitle(twi.Text("build-times.md")), ui.AttachmentDescription(twi.Text("2 KB")))))
		}
		items = append(items, k.scroller.Item(chatMessage(i+1, text, extra...)))
	}
	return show("Message scroller", "follows new messages; the wheel or PageUp stops it, the arrow jumps back; n sends one",
		k.scroller.Node(twi.Class("h-16 w-70 rounded-lg border"), k.scroller.Viewport(items...), k.scroller.Button()),
	)
}

func attachmentPage(controls) twi.Node {
	file := func(u ui.Upload, o ui.Orientation, glyph, name, about string) twi.Node {
		return ui.Attachment(u, o, ui.AttachmentMedia(ui.Icon, twi.Text(glyph)), ui.AttachmentContent(ui.AttachmentTitle(twi.Text(name)), ui.AttachmentDescription(twi.Text(about))),
			ui.AttachmentActions(ui.AttachmentAction(twi.Text("✕"))))
	}
	return show("Attachment", "done, uploading, failed and idle; in a row and stacked",
		ui.AttachmentGroup(twi.Class("w-80"),
			file(ui.Done, ui.Horizontal, "▤", "report.pdf", "1.2 MB"),
			file(ui.Uploading, ui.Horizontal, "▣", "photo.png", "uploading 40%"),
			file(ui.Failed, ui.Horizontal, "▣", "video.mp4", "too large"),
		),
		row(
			ui.Attachment(ui.Idle, ui.Horizontal, ui.AttachmentContent(ui.AttachmentTitle(twi.Text("Drop a file")), ui.AttachmentDescription(twi.Text("or click to browse"))), ui.AttachmentTrigger()),
			file(ui.Done, ui.Vertical, "▤", "notes.txt", "3 KB"),
		),
	)
}

func markerPage(controls) twi.Node {
	return show("Marker", "a quiet line between messages: plain, ruled and bordered",
		el("w-60 flex flex-col gap-1",
			ui.Marker(ui.Default, ui.MarkerIcon(twi.Text("•")), ui.MarkerContent(twi.Text("Pedro joined the conversation"))),
			ui.Marker(ui.Ruled, ui.MarkerContent(twi.Text("Yesterday"))),
			ui.Marker(ui.Bordered, ui.MarkerContent(twi.Text("Earlier messages"))),
		))
}

func resizablePage(c controls) twi.Node {
	k := c.kit
	percents := func(sizes []int) string {
		out := ""
		for i, s := range sizes {
			if i > 0 {
				out += " / "
			}
			out += strconv.Itoa(s) + "%"
		}
		return out
	}
	center := func(s string) twi.Node { return txt("flex-1 flex items-center justify-center font-semibold", s) }
	return show("Resizable", "drag a handle, or Tab to it and use the arrows, Home and End",
		el("h-14 w-[70%] flex rounded-lg border",
			k.panes.Node(
				k.panes.Panel(center("One")),
				k.panes.Handle(true),
				k.panes.Panel(k.stack.Node(k.stack.Panel(center("Two")), k.stack.Handle(true), k.stack.Panel(center("Three")))),
			)),
		txt("text-muted-foreground", "across "+percents(k.panes.Sizes)+", down "+percents(k.stack.Sizes)),
	)
}

func carouselPage(c controls) twi.Node {
	k := c.kit
	var slides []twi.NodeOption
	for i := 1; i <= 5; i++ {
		slides = append(slides, k.carousel.Item(ui.Card(twi.Class("h-full items-center justify-center"), txt("font-semibold", strconv.Itoa(i)))))
	}
	return show("Carousel", "arrows on the focused carousel or the buttons; it wraps at both ends",
		k.carousel.Node(twi.Class("w-30 h-9"), k.carousel.Content(slides...), k.carousel.Previous(), k.carousel.Next()),
		txt("text-muted-foreground", "slide "+strconv.Itoa(k.carousel.Index+1)+" of 5"),
	)
}
