package pipeline

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/s0cks/texttool/internal/doc"
)

var update = flag.Bool("update", false, "rewrite golden files")

func process(t *testing.T, opt Options, src, name string) *doc.Document {
	t.Helper()
	p, err := New(opt)
	if err != nil {
		t.Fatal(err)
	}
	d, err := p.Process([]byte(src), name)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// Every segment's text must be exactly the source bytes at its range, and
// segments within a block must be ordered and disjoint.
func checkInvariants(t *testing.T, src string, d *doc.Document) {
	t.Helper()
	prevBlock, prevEnd := -1, 0
	for _, s := range d.Segments {
		if s.Range.Start < 0 || s.Range.End > len(src) || s.Range.Start >= s.Range.End {
			t.Errorf("%s: bad range %+v", s.ID, s.Range)
			continue
		}
		if got := src[s.Range.Start:s.Range.End]; got != s.Text {
			t.Errorf("%s: text %q != source %q", s.ID, s.Text, got)
		}
		if s.Block == prevBlock && s.Range.Start < prevEnd {
			t.Errorf("%s: overlaps previous segment in block %d", s.ID, s.Block)
		}
		prevBlock, prevEnd = s.Block, s.Range.End
	}
}

func TestInvariantsOnSample(t *testing.T) {
	src, err := os.ReadFile("../../testdata/sample.md")
	if err != nil {
		t.Fatal(err)
	}
	d := process(t, Options{TTS: true, Chunk: 40}, string(src), "sample.md")
	checkInvariants(t, string(src), d)
	if len(d.Chunks) == 0 {
		t.Error("expected chunks")
	}
}

func TestNonASCIIOffsetsAreBytes(t *testing.T) {
	src := "Café costs five euros. Naïve users like it — really. Third sentence here.\n"
	d := process(t, Options{}, src, "u.md")
	checkInvariants(t, src, d)
	if len(d.Segments) != 3 {
		t.Fatalf("want 3 sentences, got %d: %+v", len(d.Segments), d.Segments)
	}
	if d.Segments[2].Text != "Third sentence here." {
		t.Errorf("third = %q", d.Segments[2].Text)
	}
}

func TestEscapesEntitiesAndWrapping(t *testing.T) {
	src := "Escaped \\*stars\\* and &amp; an entity,\nwrapped across lines. Next one.\n"
	d := process(t, Options{}, src, "e.md")
	checkInvariants(t, src, d)
	if got := d.Segments[0].Plain; got != "Escaped *stars* and & an entity, wrapped across lines." {
		t.Errorf("plain = %q", got)
	}
}

func TestFrontMatterAndDivsDoNotLeak(t *testing.T) {
	src := "---\ntitle: T\nauthor: A\n---\n\n# Head {#id .c}\n\n::: warning\nBody text.\n:::\n"
	d := process(t, Options{}, src, "f.md")
	checkInvariants(t, src, d)
	if len(d.Segments) != 2 {
		t.Fatalf("want heading + paragraph, got %d: %+v", len(d.Segments), d.Segments)
	}
	if d.Segments[0].Plain != "Head" || d.Segments[0].Level != 1 {
		t.Errorf("heading = %+v", d.Segments[0])
	}
	if d.Segments[1].Plain != "Body text." {
		t.Errorf("para = %q", d.Segments[1].Plain)
	}
}

func TestSectionBreadcrumb(t *testing.T) {
	src := "# A\n\n## B\n\ntext\n\n## C\n\nmore\n\n# D\n\nlast\n"
	d := process(t, Options{}, src, "s.md")
	want := [][]string{{"A"}, {"A", "B"}, {"A", "B"}, {"A", "C"}, {"A", "C"}, {"D"}, {"D"}}
	if len(d.Segments) != len(want) {
		t.Fatalf("got %d segments", len(d.Segments))
	}
	for i, w := range want {
		if strings.Join(d.Segments[i].Section, "/") != strings.Join(w, "/") {
			t.Errorf("segment %d section = %v, want %v", i, d.Segments[i].Section, w)
		}
	}
}

func TestSkipAndCodeBlocks(t *testing.T) {
	src := "Intro.\n\n```sh\nmake all\n```\n\nOutro.\n"
	d := process(t, Options{TTS: true}, src, "c.md")
	var code *doc.Segment
	for i := range d.Segments {
		if d.Segments[i].Kind == doc.KindCodeBlock {
			code = &d.Segments[i]
		}
	}
	if code == nil || code.Lang != "sh" || code.Plain != "make all" || *code.TTS != "" {
		t.Fatalf("code segment = %+v", code)
	}
	d = process(t, Options{Skip: map[string]bool{"code_block": true}}, src, "c.md")
	for _, s := range d.Segments {
		if s.Kind == doc.KindCodeBlock {
			t.Error("code block not skipped")
		}
	}
}

func TestAnnotation(t *testing.T) {
	src := "Tazz moved to New York. It rained.\n"
	d := process(t, Options{POS: true, NER: true, Stem: true}, src, "a.md")
	var found bool
	for _, s := range d.Segments {
		if len(s.Tokens) == 0 {
			t.Errorf("%s has no tokens", s.ID)
		}
		for _, e := range s.Entities {
			if e.Text == "New York" {
				found = true
				if e.Range == nil || s.Plain[e.Range.Start:e.Range.End] != "New York" {
					t.Errorf("entity range wrong: %+v in %q", e.Range, s.Plain)
				}
			}
		}
	}
	if !found {
		t.Error("New York not recognised")
	}
}

func TestGolden(t *testing.T) {
	src, err := os.ReadFile("../../testdata/sample.md")
	if err != nil {
		t.Fatal(err)
	}
	d := process(t, Options{TTS: true, Chunk: 60}, string(src), "sample.md")
	got, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	golden := filepath.Join("..", "..", "testdata", "sample.golden.json")
	if *update {
		if err := os.WriteFile(golden, got, 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("%v (run: go test ./internal/pipeline -update)", err)
	}
	if string(got) != string(want) {
		t.Errorf("output differs from %s; rerun with -update after reviewing the diff", golden)
	}
}
