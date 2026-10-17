package parse

import (
	"testing"

	"github.com/s0cks/texttool/internal/doc"
)

func TestPlainTextStructure(t *testing.T) {
	src := "INTRODUCTION\n\nA paragraph wrapped\nover two lines.\n\nSetext Title\n============\n\n1.2 Details\n\n- first bullet\n- second bullet\n"
	blocks, err := PlainText([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	type want struct {
		kind  string
		level int
		plain string
	}
	wants := []want{
		{doc.KindHeading, 1, "INTRODUCTION"},
		{doc.KindParagraph, 0, "A paragraph wrapped over two lines."},
		{doc.KindHeading, 1, "Setext Title"},
		{doc.KindHeading, 2, "1.2 Details"},
		{doc.KindListItem, 0, "first bullet"},
		{doc.KindListItem, 0, "second bullet"},
	}
	if len(blocks) != len(wants) {
		t.Fatalf("got %d blocks: %+v", len(blocks), blocks)
	}
	for i, w := range wants {
		b := blocks[i]
		if b.Kind != w.kind || b.Level != w.level || b.Plain != w.plain {
			t.Errorf("block %d = {%s %d %q}, want %+v", i, b.Kind, b.Level, b.Plain, w)
		}
		if got := src[b.Range.Start:b.Range.End]; len(got) == 0 {
			t.Errorf("block %d has empty range", i)
		}
	}
	if blocks[4].Section[len(blocks[4].Section)-1] != "1.2 Details" {
		t.Errorf("section = %v", blocks[4].Section)
	}
}

func TestSanitizePreservesOffsets(t *testing.T) {
	src := []byte("---\ntitle: X\n---\n\n::: note\ntext\n:::\n\n```\n::: keep\n```\n")
	out := Sanitize(src)
	if len(out) != len(src) {
		t.Fatalf("length changed: %d -> %d", len(src), len(out))
	}
	if string(out[len(out)-12:]) != string(src[len(src)-12:]) {
		t.Errorf("code fence content was altered: %q", out)
	}
	for i := range src {
		if src[i] == '\n' && out[i] != '\n' {
			t.Fatalf("newline lost at %d", i)
		}
	}
}
