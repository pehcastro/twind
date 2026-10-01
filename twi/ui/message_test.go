package ui

import (
	"testing"

	"github.com/twind-dev/twind/twi"
	"github.com/twind-dev/twind/twi/color"
	"github.com/twind-dev/twind/twi/theme"
)

func TestMessageBubbleAttachmentMarkerLook(t *testing.T) {
	look := twi.Element(twi.Class("flex flex-col gap-1 p-1 h-full bg-background text-foreground"),
		Marker(Ruled, MarkerContent(twi.Text("Today"))),
		MessageGroup(twi.Class("w-50"),
			Message(Start,
				MessageAvatar(Avatar(SizeSM, AvatarFallback(twi.Text("AI")))),
				MessageContent(
					MessageHeader(twi.Text("Assistant")),
					Bubble(Muted, Start, BubbleContent(twi.Text("A long answer that has to wrap because a bubble is never wider than four fifths of its message."))),
					MessageFooter(twi.Text("2:14 PM")),
				),
			),
			Message(End,
				MessageAvatar(Avatar(SizeSM, AvatarFallback(twi.Text("ME")))),
				MessageContent(
					Bubble(Default, End, BubbleContent(twi.Text("Thanks"))),
					MessageFooter(twi.Text("read")),
				),
			),
			Message(Start, MessageContent(BubbleGroup(
				Bubble(Secondary, Start, BubbleContent(twi.Text("secondary"))),
				Bubble(Tinted, Start, BubbleContent(twi.Text("tinted"))),
				Bubble(Destructive, Start, BubbleContent(twi.Text("destructive"))),
			))),
		),
		Marker(Bordered, MarkerContent(twi.Text("Earlier"))),
		twi.Element(twi.Class("flex flex-row gap-1"),
			Attachment(Done, Horizontal, AttachmentMedia(Icon, twi.Text("▣")), AttachmentContent(AttachmentTitle(twi.Text("report.pdf")), AttachmentDescription(twi.Text("done")))),
			Attachment(Failed, Horizontal, AttachmentMedia(Icon, twi.Text("▣")), AttachmentContent(AttachmentTitle(twi.Text("photo.png")), AttachmentDescription(twi.Text("failed")))),
			Attachment(Idle, Horizontal, AttachmentContent(AttachmentTitle(twi.Text("drop.txt")), AttachmentDescription(twi.Text("idle")))),
		),
	)
	d := overlayDriver(t, 60, 30, func(rt *twi.Runtime) func() twi.Node {
		rt.SetTheme(zinc(t, theme.Dark))
		return func() twi.Node { return look }
	})
	expect := expecter(t, d)
	cells := d.Frame().Cells()
	type place struct{ x, y int }
	where := map[string]place{}
	for _, s := range []string{"AI", "ME", "A long", "message.", "Thanks", "read", "2:14 PM", "secondary", "tinted", "destructive", "Today", "done", "failed", "drop.txt"} {
		x, y, ok := at(d.Frame(), s)
		if !ok {
			t.Fatalf("no %q on screen:\n%s", s, d.Frame().Text())
		}
		where[s] = place{x, y}
	}
	bg := func(s string) color.RGBA { return cells.At(where[s].x, where[s].y).Bg.RGBA }
	fg := func(s string) color.RGBA { return cells.At(where[s].x, where[s].y).Fg.RGBA }
	expect("a start message puts its avatar left of its bubble, at the bottom", where["AI"].x < where["A long"].x && where["AI"].y > where["A long"].y)
	expect("an end message puts its avatar on the right with its bubble beside it", where["ME"].x > where["Thanks"].x && where["Thanks"].x > 25)
	expect("the end message's footer sits under its bubble, right-aligned", where["read"].y == where["Thanks"].y+1 && where["read"].x+len("read") == where["Thanks"].x+len("Thanks"))
	expect("the long bubble wraps inside four fifths of a 50-cell message", where["message."].y-where["A long"].y >= 2 && where["message."].x+len("message.") <= where["AI"].x+50)
	page := cells.At(0, 0).Bg.RGBA
	if len(map[color.RGBA]bool{bg("Thanks"): true, bg("secondary"): true, bg("tinted"): true, bg("destructive"): true, page: true}) != 5 || bg("A long") == page {
		t.Errorf("default, secondary, tinted and destructive bubbles each paint their own background, muted one too (zinc dark's muted is its secondary): %v %v %v %v %v on %v", bg("Thanks"), bg("secondary"), bg("tinted"), bg("destructive"), bg("A long"), page)
	}
	expect("the destructive bubble and the failed attachment use the destructive colour", fg("destructive") != fg("secondary") && fg("failed") != fg("done"))
	expect("the ruled marker draws a line either side of its content on its own row", cells.At(where["Today"].x-2, where["Today"].y).Grapheme == "─" && cells.At(where["Today"].x+len("Today")+1, where["Today"].y).Grapheme == "─")
	expect("an idle attachment has a dashed border", cells.At(where["drop.txt"].x-2, where["drop.txt"].y).Grapheme == "┆")
	t.Logf("messages, bubbles, attachments and markers, 60x30 dark:\n%s", d.Frame().Text())
}
