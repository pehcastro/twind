package present

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/pehcastro/twind/internal/present/demo"
	"github.com/pehcastro/twind/internal/scene"
	"github.com/pehcastro/twind/internal/terminal"
)

func gdiOutputs(t *testing.T) []string {
	trees := append([]scene.Node{tree(t, demo.Dialog()), tree(t, withText(demo.Dialog(), "Dashboard  Projects  Settings", "Dashboard")), tree(t, demo.List(0)), tree(t, demo.List(1)), tree(t, demo.Page())}, scrolled(t, 0, 1, 2, 10)...)
	var lines []string
	for _, id := range []terminal.Identity{terminal.IdentityConhost, terminal.IdentityZed} {
		s, out, p := gdiScreen()
		s.Identity = id
		for _, root := range trees {
			frame(t, s, root)
		}
		sum := sha256.New()
		for _, f := range out.frames {
			sum.Write(f)
		}
		for _, call := range p.calls {
			sum.Write(binary.LittleEndian.AppendUint32(nil, uint32(len(call.Tiles))))
			if call.Clear {
				sum.Write([]byte("clear"))
			}
			for _, tile := range call.Tiles {
				sum.Write([]byte(tile.Cells.String()))
				sum.Write(tile.Pix)
			}
		}
		lines = append(lines, fmt.Sprintf("identity %d frames %d paints %d %s", id, len(out.frames), len(p.calls), hex.EncodeToString(sum.Sum(nil)[:8])))
	}
	return lines
}

func TestGDIOutputUnchanged(t *testing.T) {
	fixture, err := os.ReadFile("testdata/gdi.txt")
	if err != nil {
		t.Fatal(err)
	}
	want, got := strings.Split(strings.TrimSpace(strings.ReplaceAll(string(fixture), "\r", "")), "\n"), gdiOutputs(t)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("GDI bytes and pixels differ from 2e9fb66:\ngot  %q\nwant %q", got, want)
	}
}
