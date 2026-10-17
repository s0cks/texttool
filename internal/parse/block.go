// Package parse turns markdown or plain text into structural blocks that carry
// both clean text and a map back to source byte offsets.
package parse

import (
	"regexp"
	"strings"

	"github.com/s0cks/texttool/internal/doc"
)

// Block is one structural unit of the input: a heading, a paragraph-like run
// of prose, a table cell or a code block.
type Block struct {
	Kind    string
	Level   int // heading level
	Lang    string
	Section []string // heading breadcrumb; for a heading it ends with the heading itself
	Plain   string
	Spans   []doc.Span
	Map     *Mapper   // nil for code blocks
	Range   doc.Range // source range of the whole block (code blocks, single-segment blocks)
}

// sections tracks the heading breadcrumb while walking a document.
type sections struct {
	levels []int
	titles []string
}

func (s *sections) push(level int, title string) []string {
	for len(s.levels) > 0 && s.levels[len(s.levels)-1] >= level {
		s.levels = s.levels[:len(s.levels)-1]
		s.titles = s.titles[:len(s.titles)-1]
	}
	s.levels = append(s.levels, level)
	s.titles = append(s.titles, title)
	return s.path()
}

func (s *sections) path() []string {
	if len(s.titles) == 0 {
		return nil
	}
	return append([]string(nil), s.titles...)
}

// collector accumulates plain text, a source map and inline spans.
type collector struct {
	plain []byte
	m     Mapper
	spans []doc.Span
}

// addRun appends text that came from source bytes [sStart, sEnd).
func (c *collector) addRun(text string, sStart, sEnd int) {
	if text == "" {
		return
	}
	p := len(c.plain)
	c.plain = append(c.plain, text...)
	c.m.add(p, len(c.plain), sStart, sEnd)
}

// addSynthetic appends text that has no source (e.g. a space joining two lines).
func (c *collector) addSynthetic(text string) {
	c.addRun(text, -1, -1)
}

// addSpace appends a joining space unless the text already ends in whitespace.
func (c *collector) addSpace() {
	if n := len(c.plain); n == 0 || c.plain[n-1] == ' ' {
		return
	}
	c.addSynthetic(" ")
}

// cut deletes plain bytes [a, b), keeping runs and spans consistent.
func (c *collector) cut(a, b int) {
	if b <= a {
		return
	}
	d := b - a
	var runs []run
	for _, r := range c.m.runs {
		switch {
		case r.pEnd <= a:
			runs = append(runs, r)
		case r.pStart >= b:
			r.pStart -= d
			r.pEnd -= d
			runs = append(runs, r)
		default:
			if r.pStart < a { // left remainder
				l := r
				l.pEnd = a
				if !l.synthetic() {
					l.sEnd = min(l.sStart+(a-r.pStart), r.sEnd)
				}
				runs = append(runs, l)
			}
			if r.pEnd > b { // right remainder
				rr := r
				rr.pStart = a
				rr.pEnd = a + (r.pEnd - b)
				if !rr.synthetic() {
					rr.sStart = min(r.sStart+(b-r.pStart), r.sEnd)
				}
				runs = append(runs, rr)
			}
		}
	}
	c.m.runs = runs
	shift := func(x int) int {
		switch {
		case x <= a:
			return x
		case x >= b:
			return x - d
		default:
			return a
		}
	}
	var spans []doc.Span
	for _, s := range c.spans {
		s.Start, s.End = shift(s.Start), shift(s.End)
		if s.End > s.Start {
			spans = append(spans, s)
		}
	}
	c.spans = spans
	c.plain = append(c.plain[:a], c.plain[b:]...)
}

var reAttr = regexp.MustCompile(`\s*\{(?:[#.][^{}\n]*|[A-Za-z_-]+=[^{}\n]*)\}`)

// finish strips pandoc attribute blocks and surrounding whitespace.
func (c *collector) finish() {
	// Delete matches back to front so earlier offsets stay valid.
	ms := reAttr.FindAllIndex(c.plain, -1)
	for i := len(ms) - 1; i >= 0; i-- {
		c.cut(ms[i][0], ms[i][1])
	}
	// Trim.
	lead := len(c.plain) - len(strings.TrimLeft(string(c.plain), " \t\r\n"))
	c.cut(0, lead)
	trail := len(c.plain) - len(strings.TrimRight(string(c.plain), " \t\r\n"))
	c.cut(len(c.plain)-trail, len(c.plain))
}
