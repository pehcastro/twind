package ui

import (
	"testing"

	"github.com/pehcastro/twind/twi"
)

func TestEveryVariantAndSize(t *testing.T) {
	d := overlayDriver(t, 200, 400, func(rt *twi.Runtime) func() twi.Node {
		return func() twi.Node {
			var all []twi.NodeOption
			for v := ButtonDefault; v <= ButtonLink; v++ {
				for s := ButtonSizeDefault; s <= ButtonSizeIcon; s++ {
					all = append(all, Button(v, s, twi.Text("b")))
				}
			}
			for v := ItemDefault; v <= ItemMuted; v++ {
				for s := ItemSizeDefault; s <= ItemSizeSM; s++ {
					all = append(all, Item(v, s, twi.Text("i")))
				}
			}
			for v := ToggleDefault; v <= ToggleOutline; v++ {
				for s := ToggleSizeDefault; s <= ToggleSizeLG; s++ {
					tg := NewToggle(rt)
					tg.Variant, tg.Size = v, s
					all = append(all, tg.Node(twi.Text("t")))
				}
			}
			for v := AlertDefault; v <= AlertDestructive; v++ {
				all = append(all, Alert(v, twi.Text("a")))
			}
			for v := BadgeDefault; v <= BadgeLink; v++ {
				all = append(all, Badge(v, twi.Text("b")))
			}
			for s := AvatarSizeDefault; s <= AvatarSizeLG; s++ {
				all = append(all, Avatar(s, AvatarFallback(twi.Text("CN"))))
			}
			for v := ItemMediaDefault; v <= ItemMediaImage; v++ {
				all = append(all, ItemMedia(v, twi.Text("m")))
			}
			for v := EmptyMediaDefault; v <= EmptyMediaIcon; v++ {
				all = append(all, EmptyMedia(v, twi.Text("m")))
			}
			for v := AttachmentMediaIcon; v <= AttachmentMediaImage; v++ {
				for u := UploadDone; u <= UploadFailed; u++ {
					for o := Horizontal; o <= Vertical; o++ {
						all = append(all, Attachment(u, o, AttachmentMedia(v, twi.Text("m"))))
					}
				}
			}
			for v := MarkerDefault; v <= MarkerBorder; v++ {
				all = append(all, Marker(v, MarkerContent(twi.Text("m"))))
			}
			for s := SidebarMenuButtonSizeDefault; s <= SidebarMenuButtonSizeLG; s++ {
				all = append(all, SidebarMenuButton(s, Active(s == SidebarMenuButtonSizeLG), twi.Text("s")))
			}
			for a := AlignCenter; a <= AlignEnd; a++ {
				for v := BubbleDefault; v <= BubbleDestructive; v++ {
					all = append(all, Message(a, MessageContent(Bubble(v, a, BubbleContent(twi.Text("b"))))))
				}
				for s := SideBottom; s <= SideLeft; s++ {
					all = append(all, Bubble(BubbleDefault, a, BubbleContent(twi.Text("b")), BubbleReactions(s, a, twi.Text("r"))))
				}
			}
			var addons []Addon
			for s := SideBottom; s <= SideLeft; s++ {
				addons = append(addons, InputGroupAddon(s, twi.Text("x")))
			}
			all = append(all, NewInput(rt).Group(addons...))
			return twi.Element(append([]twi.NodeOption{twi.Class("flex flex-col")}, all...)...)
		}
	})
	if f := d.Frame(); f.Width() != 200 || f.Height() != 400 {
		t.Fatalf("frame %dx%d", f.Width(), f.Height())
	}
}
