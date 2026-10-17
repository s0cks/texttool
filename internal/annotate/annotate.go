// Package annotate adds tokens, part-of-speech tags, stems and named entities.
// The default backend is github.com/jdkato/prose/v2.
package annotate

import (
	"sort"
	"strings"
	"unicode"

	"github.com/jdkato/prose/v2"
	"github.com/kljensen/snowball"
	"github.com/s0cks/texttool/internal/doc"
)

// Options selects which annotations to produce.
type Options struct {
	POS  bool
	NER  bool
	Stem bool
}

// Enabled reports whether any annotation was requested.
func (o Options) Enabled() bool { return o.POS || o.NER || o.Stem }

// Run annotates every non-code segment in place.
//
// prose reloads its models on each NewDocument call (about half a second), so
// the whole input is analysed in one call: segment text is joined with
// newlines, and tokens are then mapped back to segments by position.
func Run(segs []doc.Segment, opt Options) error {
	if !opt.Enabled() {
		return nil
	}
	var joined strings.Builder
	var idx, starts []int
	for i := range segs {
		if segs[i].Kind == doc.KindCodeBlock || segs[i].Plain == "" {
			continue
		}
		idx = append(idx, i)
		starts = append(starts, joined.Len())
		joined.WriteString(segs[i].Plain)
		joined.WriteByte('\n')
	}
	if len(idx) == 0 {
		return nil
	}
	text := joined.String()

	d, err := prose.NewDocument(text,
		prose.WithSegmentation(false),
		prose.WithTagging(true),
		prose.WithExtraction(opt.NER))
	if err != nil {
		return err
	}

	// segOf maps an offset in text to the index (into idx) of its segment.
	segOf := func(off int) int {
		j := sort.Search(len(starts), func(k int) bool { return starts[k] > off }) - 1
		return max(j, 0)
	}

	type entity struct {
		label      string
		first, end int // byte offsets into text; -1 when unknown
		words      []string
		seg        int
	}
	var cur *entity
	flush := func() {
		if cur == nil {
			return
		}
		s := &segs[idx[cur.seg]]
		e := doc.Entity{Label: cur.label}
		if cur.first >= 0 {
			base := starts[cur.seg]
			e.Text = text[cur.first:cur.end]
			e.Range = &doc.Range{Start: cur.first - base, End: cur.end - base}
		} else {
			e.Text = strings.Join(cur.words, " ")
		}
		s.Entities = append(s.Entities, e)
		cur = nil
	}

	cursor, lastSeg := 0, 0
	for _, t := range d.Tokens() {
		start, end := -1, -1
		if i := strings.Index(text[cursor:], t.Text); i >= 0 && strings.TrimSpace(text[cursor:cursor+i]) == "" {
			start = cursor + i
			end = start + len(t.Text)
			cursor = end
		}
		sj := lastSeg
		if start >= 0 {
			sj = segOf(start)
			lastSeg = sj
		}
		seg := &segs[idx[sj]]

		tok := doc.Token{Text: t.Text}
		if opt.POS {
			tok.POS = t.Tag
		}
		if opt.Stem && hasLetter(t.Text) {
			if st, err := snowball.Stem(strings.ToLower(t.Text), "english", true); err == nil {
				tok.Stem = st
			}
		}
		if start >= 0 {
			base := starts[sj]
			tok.Range = &doc.Range{Start: start - base, End: end - base}
			if end > base+len(seg.Plain) { // token spilled across a segment boundary
				tok.Range = nil
			}
		}
		if opt.POS || opt.Stem {
			seg.Tokens = append(seg.Tokens, tok)
		}

		if opt.NER {
			// Like prose's own Entities(), a run of consecutive non-"O" tokens
			// is one entity labelled by its first token ("Jane"/"Smith" are both
			// B-PERSON, so B- vs I- cannot be trusted to mark boundaries).
			label := strings.TrimPrefix(strings.TrimPrefix(t.Label, "B-"), "I-")
			switch {
			case t.Label == "" || t.Label == "O":
				flush()
			case cur == nil || cur.seg != sj:
				flush()
				cur = &entity{label: label, first: start, end: end, words: []string{t.Text}, seg: sj}
			default:
				cur.words = append(cur.words, t.Text)
				if start >= 0 && cur.first >= 0 {
					cur.end = end
				} else {
					cur.first = -1
				}
			}
		}
	}
	flush()
	return nil
}

func hasLetter(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}
