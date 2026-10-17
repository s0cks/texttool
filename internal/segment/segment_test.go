package segment

import (
	"os"
	"testing"

	"github.com/s0cks/texttool/internal/parse"
)

func TestRangeInvariant(t *testing.T) {
	src, err := os.ReadFile("../../testdata/sample.md")
	if err != nil {
		t.Fatal(err)
	}
	blocks, err := parse.Markdown(src)
	if err != nil {
		t.Fatal(err)
	}
	sp, err := New([]string{"e.g", "dr"})
	if err != nil {
		t.Fatal(err)
	}
	segs := sp.Split(blocks, src, "sample.md")
	if len(segs) == 0 {
		t.Fatal("no segments")
	}
	prevEnd := -1
	for _, s := range segs {
		t.Logf("%-10s sec=%v [%d,%d) plain=%q text=%q", s.Kind, s.Section, s.Range.Start, s.Range.End, s.Plain, s.Text)
		if s.Text != string(src[s.Range.Start:s.Range.End]) {
			t.Errorf("%s: text does not match source range", s.ID)
		}
		if s.Range.Start < prevEnd && s.Kind != "table_cell" {
			t.Errorf("%s: range overlaps previous segment", s.ID)
		}
		prevEnd = s.Range.End
	}
}
