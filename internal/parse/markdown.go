package parse

import (
	"bytes"
	"html"
	"strings"

	"github.com/s0cks/texttool/internal/doc"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

type mdCtx struct {
	list     int
	quote    bool
	footnote bool
}

type mdParser struct {
	src    []byte // sanitized source; offsets equal the original's
	blocks []Block
	secs   sections
}

// Markdown parses CommonMark/GFM (plus footnotes and definition lists) into blocks.
func Markdown(src []byte) ([]Block, error) {
	prepared := Sanitize(src)
	md := goldmark.New(goldmark.WithExtensions(
		extension.Table,
		extension.Strikethrough,
		extension.Footnote,
		extension.DefinitionList,
		extension.Linkify,
	))
	root := md.Parser().Parse(text.NewReader(prepared))
	p := &mdParser{src: prepared}
	p.children(root, mdCtx{})
	return p.blocks, nil
}

func (p *mdParser) children(n ast.Node, c mdCtx) {
	for ch := n.FirstChild(); ch != nil; ch = ch.NextSibling() {
		p.node(ch, c)
	}
}

func (p *mdParser) node(n ast.Node, c mdCtx) {
	switch v := n.(type) {
	case *ast.Heading:
		col := p.collect(v)
		if len(col.plain) == 0 {
			return
		}
		title := string(col.plain)
		p.emit(Block{Kind: doc.KindHeading, Level: v.Level, Section: p.secs.push(v.Level, title)}, col)
	case *ast.Paragraph, *ast.TextBlock:
		kind := doc.KindParagraph
		switch {
		case c.footnote:
			kind = doc.KindFootnote
		case c.list > 0:
			kind = doc.KindListItem
		case c.quote:
			kind = doc.KindBlockquote
		}
		p.emit(Block{Kind: kind, Section: p.secs.path()}, p.collect(n))
	case *ast.FencedCodeBlock:
		lang := string(v.Language(p.src))
		p.code(v, lang)
	case *ast.CodeBlock:
		p.code(v, "")
	case *ast.HTMLBlock, *ast.ThematicBreak:
		return
	case *ast.List:
		c.list++
		p.children(v, c)
	case *ast.Blockquote:
		c.quote = true
		p.children(v, c)
	case *east.FootnoteList, *east.Footnote:
		c.footnote = true
		p.children(v, c)
	case *east.DefinitionTerm:
		p.emit(Block{Kind: doc.KindTerm, Section: p.secs.path()}, p.collect(v))
	case *east.TableCell:
		p.emit(Block{Kind: doc.KindTableCell, Section: p.secs.path()}, p.collect(v))
	default:
		p.children(n, c)
	}
}

func (p *mdParser) emit(b Block, col *collector) {
	if len(col.plain) == 0 {
		return
	}
	b.Plain = string(col.plain)
	b.Spans = col.spans
	b.Map = &col.m
	r, ok := col.m.Range(0, len(col.plain))
	if !ok {
		return
	}
	b.Range = r
	p.blocks = append(p.blocks, b)
}

func (p *mdParser) code(n ast.Node, lang string) {
	lines := n.Lines()
	if lines.Len() == 0 {
		return
	}
	start := lines.At(0).Start
	end := lines.At(lines.Len() - 1).Stop
	for end > start && (p.src[end-1] == '\n' || p.src[end-1] == '\r') {
		end--
	}
	if end <= start {
		return
	}
	p.blocks = append(p.blocks, Block{
		Kind:    doc.KindCodeBlock,
		Lang:    lang,
		Section: p.secs.path(),
		Plain:   string(p.src[start:end]),
		Range:   doc.Range{Start: start, End: end},
	})
}

// collect gathers the inline content under n into plain text with a source map.
func (p *mdParser) collect(n ast.Node) *collector {
	c := &collector{}
	p.inline(c, n, false)
	c.finish()
	return c
}

func (p *mdParser) inline(c *collector, n ast.Node, literal bool) {
	for ch := n.FirstChild(); ch != nil; ch = ch.NextSibling() {
		switch v := ch.(type) {
		case *ast.Text:
			seg := v.Segment
			raw := p.src[seg.Start:seg.Stop]
			raw = bytes.TrimRight(raw, "\r\n")
			if literal {
				c.addRun(strings.ReplaceAll(string(raw), "\n", " "), seg.Start, seg.Start+len(raw))
			} else {
				addDecoded(c, raw, seg.Start)
			}
			if v.SoftLineBreak() || v.HardLineBreak() {
				c.addSpace()
			}
		case *ast.String:
			c.addSynthetic(string(v.Value))
		case *ast.CodeSpan:
			start := len(c.plain)
			p.inline(c, v, true)
			if len(c.plain) > start {
				c.spans = append(c.spans, doc.Span{Kind: "code", Start: start, End: len(c.plain)})
			}
		case *ast.Link:
			start := len(c.plain)
			p.inline(c, v, literal)
			if len(c.plain) > start {
				c.spans = append(c.spans, doc.Span{Kind: "link", Start: start, End: len(c.plain), Dest: string(v.Destination)})
			}
		case *ast.AutoLink:
			start := len(c.plain)
			label := v.Label(p.src)
			s0 := -1
			if n := len(c.m.runs); n > 0 {
				if last := c.m.runs[n-1]; !last.synthetic() {
					if i := bytes.Index(p.src[last.sEnd:], label); i >= 0 {
						s0 = last.sEnd + i
					}
				}
			}
			if s0 >= 0 {
				c.addRun(string(label), s0, s0+len(label))
			} else {
				c.addSynthetic(string(label))
			}
			c.spans = append(c.spans, doc.Span{Kind: "url", Start: start, End: len(c.plain), Dest: string(v.URL(p.src))})
		case *ast.Image, *ast.RawHTML, *east.FootnoteLink, *east.FootnoteBacklink:
			continue
		default:
			p.inline(c, ch, literal)
		}
	}
}

func isASCIIPunct(b byte) bool {
	return (b >= '!' && b <= '/') || (b >= ':' && b <= '@') || (b >= '[' && b <= '`') || (b >= '{' && b <= '~')
}

// addDecoded appends raw markdown text, resolving backslash escapes and HTML
// entities while keeping the source map honest.
func addDecoded(c *collector, raw []byte, sStart int) {
	last := 0
	flush := func(upto int) {
		if upto > last {
			c.addRun(string(raw[last:upto]), sStart+last, sStart+upto)
		}
	}
	for i := 0; i < len(raw); {
		switch {
		case raw[i] == '\\' && i+1 < len(raw) && isASCIIPunct(raw[i+1]):
			flush(i)
			c.addRun(string(raw[i+1:i+2]), sStart+i+1, sStart+i+2)
			i += 2
			last = i
		case raw[i] == '&':
			end := -1
			for j := i + 1; j < len(raw) && j <= i+12; j++ {
				if raw[j] == ';' {
					end = j
					break
				}
				if raw[j] == ' ' || raw[j] == '&' {
					break
				}
			}
			if end > 0 {
				if dec := html.UnescapeString(string(raw[i : end+1])); dec != string(raw[i:end+1]) {
					flush(i)
					c.addRun(dec, sStart+i, sStart+end+1)
					i = end + 1
					last = i
					continue
				}
			}
			i++
		default:
			i++
		}
	}
	flush(len(raw))
}
