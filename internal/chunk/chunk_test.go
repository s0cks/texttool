package chunk

import (
	"testing"

	"github.com/s0cks/texttool/internal/doc"
)

func seg(id string, block int, section []string, plain string, start, end int) doc.Segment {
	return doc.Segment{ID: id, Block: block, Kind: doc.KindParagraph, Section: section, Plain: plain, Range: doc.Range{Start: start, End: end}}
}

func TestBuildRespectsBudgetAndSections(t *testing.T) {
	a := []string{"A"}
	b := []string{"A", "B"}
	segs := []doc.Segment{
		seg("s0", 0, a, "one two three.", 0, 14),     // 4 tokens
		seg("s1", 0, a, "four five six.", 15, 29),    // 4 tokens
		seg("s2", 1, a, "seven eight nine.", 31, 48), // 4 tokens
		seg("s3", 2, b, "other section.", 50, 64),    // new section
	}
	got := Build(segs, 8, false)
	if len(got) != 3 {
		t.Fatalf("want 3 chunks, got %d: %+v", len(got), got)
	}
	if got[0].TokenCount != 8 || len(got[0].SegmentIDs) != 2 {
		t.Errorf("chunk 0 = %+v", got[0])
	}
	if got[0].Range.Start != 0 || got[0].Range.End != 29 {
		t.Errorf("chunk 0 range = %+v", got[0].Range)
	}
	if got[0].Text != "one two three. four five six." {
		t.Errorf("same-block segments should join with a space, got %q", got[0].Text)
	}
	if got[2].Section[1] != "B" {
		t.Errorf("section change must start a new chunk: %+v", got[2])
	}
}

func TestOversizedSegmentIsItsOwnChunk(t *testing.T) {
	segs := []doc.Segment{seg("s0", 0, nil, "a b c d e f g h i j.", 0, 20)}
	got := Build(segs, 3, false)
	if len(got) != 1 || got[0].SegmentIDs[0] != "s0" {
		t.Fatalf("got %+v", got)
	}
}

func TestCodeBlocksAreSkippedAndTTSTextPreferred(t *testing.T) {
	tts := "spoken form."
	s0 := seg("s0", 0, nil, "Written form.", 0, 13)
	s0.TTS = &tts
	code := seg("s1", 1, nil, "make all", 14, 22)
	code.Kind = doc.KindCodeBlock
	got := Build([]doc.Segment{s0, code}, 50, true)
	if len(got) != 1 || got[0].Text != "spoken form." || len(got[0].SegmentIDs) != 1 {
		t.Fatalf("got %+v", got)
	}
}
