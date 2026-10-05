package render_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/pehcastro/twind/internal/buffer"
	"github.com/pehcastro/twind/internal/paint"
	"github.com/pehcastro/twind/internal/render"
	"github.com/pehcastro/twind/internal/render/testdata/sheet"
)

func TestWordBreakReachesThePage(t *testing.T) {
	const hash, cjk = "commit 0123456789abcdef0123456789abcdef01234567", "中文排版 需要断行。日本語"
	var tree render.Tree
	f := cssFrame(t, 30)
	for _, step := range []struct {
		name, hash, cjk string
		want            []string
	}{
		{"break-all and break-keep", sheet.Hash, sheet.KeepAll, []string{
			"commit 01234", "56789abcdef0", "123456789abc", "def01234567",
			"中文排版", "需要断行。", "日本語",
		}},
		{"both normal", "w-12", "w-12", []string{
			"commit", "0123456789ab", "cdef01234567", "89abcdef0123", "4567",
			"中文排版 需", "要断行。日本", "語",
		}},
		{"break-all and break-keep again", sheet.Hash, sheet.KeepAll, []string{
			"commit 01234", "56789abcdef0", "123456789abc", "def01234567",
			"中文排版", "需要断行。", "日本語",
		}},
	} {
		root, err := tree.Scene(node(sheet.Page, node(step.hash, text(hash)), node(step.cjk, text(cjk))), f)
		if err != nil {
			t.Fatal(err)
		}
		buf := buffer.New(f.Width, root.Bounds.H)
		paint.Paint(buf, root, paint.Composited)
		got := rowsOf(buf)
		t.Logf("%s, 30 wide:\n%s", step.name, strings.Join(got, "\n"))
		if !slices.Equal(got, step.want) {
			t.Errorf("%s: rows %q, want %q", step.name, got, step.want)
		}
	}
}
