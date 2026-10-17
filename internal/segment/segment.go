// Package segment splits parsed blocks into sentence-level segments.
package segment

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/neurosnap/sentences"
	"github.com/neurosnap/sentences/data"
	"github.com/neurosnap/sentences/english"
	"github.com/s0cks/texttool/internal/doc"
	"github.com/s0cks/texttool/internal/parse"
)

// Splitter turns blocks into segments with source ranges.
type Splitter struct {
	tok *sentences.DefaultSentenceTokenizer
}

// New builds a Splitter. abbrevs are extra abbreviations (e.g. "e.g", "vs")
// that must not end a sentence; a trailing period is ignored.
func New(abbrevs []string) (*Splitter, error) {
	b, err := data.Asset("data/english.json")
	if err != nil {
		return nil, err
	}
	training, err := sentences.LoadTraining(b)
	if err != nil {
		return nil, err
	}
	for _, a := range abbrevs {
		a = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(a), "."))
		if a != "" {
			training.AbbrevTypes.Add(a)
		}
	}
	tok, err := english.NewSentenceTokenizer(training)
	if err != nil {
		return nil, err
	}
	return &Splitter{tok: tok}, nil
}

// Split converts blocks into segments. src must be the original input bytes.
func (s *Splitter) Split(blocks []parse.Block, src []byte, source string) []doc.Segment {
	var out []doc.Segment
	add := func(bi int, b parse.Block, plain string, r doc.Range, spans []doc.Span) {
		if r.Start < 0 || r.End > len(src) || r.End <= r.Start {
			return
		}
		out = append(out, doc.Segment{
			Type:    "segment",
			ID:      fmt.Sprintf("s%d", len(out)),
			Source:  source,
			Block:   bi,
			Kind:    b.Kind,
			Level:   b.Level,
			Lang:    b.Lang,
			Section: b.Section,
			Range:   r,
			Text:    string(src[r.Start:r.End]),
			Plain:   plain,
			Spans:   spans,
		})
	}

	for bi, b := range blocks {
		switch b.Kind {
		case doc.KindHeading, doc.KindCodeBlock, doc.KindTableCell:
			add(bi, b, b.Plain, b.Range, b.Spans)
			continue
		}
		for _, sent := range s.tok.Tokenize(b.Plain) {
			a, e := trimBounds(b.Plain, sent.Start, sent.End)
			if e <= a {
				continue
			}
			r, ok := b.Map.Range(a, e)
			if !ok {
				continue
			}
			add(bi, b, b.Plain[a:e], r, clipSpans(b.Spans, a, e))
		}
	}
	return out
}

// trimBounds shrinks [a, e) to exclude surrounding whitespace.
func trimBounds(s string, a, e int) (int, int) {
	if a < 0 {
		a = 0
	}
	if e > len(s) {
		e = len(s)
	}
	for a < e && unicode.IsSpace(rune(s[a])) {
		a++
	}
	for e > a && unicode.IsSpace(rune(s[e-1])) {
		e--
	}
	return a, e
}

// clipSpans returns the spans overlapping [a, e), rebased to start at a.
func clipSpans(spans []doc.Span, a, e int) []doc.Span {
	var out []doc.Span
	for _, sp := range spans {
		if sp.End <= a || sp.Start >= e {
			continue
		}
		sp.Start = max(sp.Start, a) - a
		sp.End = min(sp.End, e) - a
		out = append(out, sp)
	}
	return out
}
