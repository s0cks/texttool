package parse

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/s0cks/texttool/internal/doc"
)

var (
	reUnderline = regexp.MustCompile(`^\s*(=+|-+)\s*$`)
	reBullet    = regexp.MustCompile(`^(\s*)([-*+•]|\d+[.)])\s+`)
	reNumbered  = regexp.MustCompile(`^(\d+(?:\.\d+)*)[.)]?\s+(\p{Lu}.*)$`)
)

type textLine struct {
	start, end int // end excludes the newline
	text       string
}

func splitTextLines(src []byte) []textLine {
	var out []textLine
	s := string(src)
	pos := 0
	for pos <= len(s) {
		nl := strings.IndexByte(s[pos:], '\n')
		end := len(s)
		next := len(s) + 1
		if nl >= 0 {
			end = pos + nl
			next = end + 1
		}
		e := end
		if e > pos && s[e-1] == '\r' {
			e--
		}
		out = append(out, textLine{pos, e, s[pos:e]})
		pos = next
	}
	return out
}

// PlainText parses unformatted prose: paragraphs separated by blank lines,
// optional setext, numbered or ALL-CAPS headings, and bullet lists.
func PlainText(src []byte) ([]Block, error) {
	lines := splitTextLines(src)
	var blocks []Block
	var secs sections

	for i := 0; i < len(lines); {
		if strings.TrimSpace(lines[i].text) == "" {
			i++
			continue
		}
		j := i
		for j < len(lines) && strings.TrimSpace(lines[j].text) != "" {
			j++
		}
		group := lines[i:j]
		i = j

		// Setext heading: one title line plus an underline.
		if len(group) == 2 && reUnderline.MatchString(group[1].text) && len(strings.TrimSpace(group[1].text)) >= 3 {
			level := 2
			if strings.Contains(group[1].text, "=") {
				level = 1
			}
			if b, ok := plainBlock(doc.KindHeading, level, group[:1], 0, &secs); ok {
				blocks = append(blocks, b)
			}
			continue
		}
		if len(group) == 1 && !reBullet.MatchString(group[0].text) {
			if level := looseHeadingLevel(group[0].text); level > 0 {
				if b, ok := plainBlock(doc.KindHeading, level, group, 0, &secs); ok {
					blocks = append(blocks, b)
				}
				continue
			}
		}
		if len(group) == 1 {
			if level := numberedHeadingLevel(group[0].text); level > 0 {
				if b, ok := plainBlock(doc.KindHeading, level, group, 0, &secs); ok {
					blocks = append(blocks, b)
				}
				continue
			}
		}

		// Split the group into bullet items and ordinary paragraphs.
		start := 0
		kind := doc.KindParagraph
		flush := func(end int) {
			if end > start {
				if b, ok := plainBlock(kind, 0, group[start:end], 0, &secs); ok {
					blocks = append(blocks, b)
				}
			}
		}
		for k, l := range group {
			if reBullet.MatchString(l.text) {
				flush(k)
				start = k
				kind = doc.KindListItem
			}
		}
		flush(len(group))
	}
	return blocks, nil
}

// plainBlock joins lines into one block, stripping a leading bullet marker.
func plainBlock(kind string, level int, group []textLine, _ int, secs *sections) (Block, bool) {
	c := &collector{}
	for idx, l := range group {
		text := l.text
		off := 0
		if idx == 0 && kind == doc.KindListItem {
			if m := reBullet.FindString(text); m != "" {
				off = len(m)
				text = text[off:]
			}
		}
		trimmed := strings.TrimLeft(text, " \t")
		off += len(text) - len(trimmed)
		trimmed = strings.TrimRight(trimmed, " \t")
		if trimmed == "" {
			continue
		}
		c.addSpace()
		c.addRun(trimmed, l.start+off, l.start+off+len(trimmed))
	}
	c.finish()
	if len(c.plain) == 0 {
		return Block{}, false
	}
	b := Block{Kind: kind, Level: level, Plain: string(c.plain), Map: &c.m}
	if kind == doc.KindHeading {
		b.Section = secs.push(level, b.Plain)
	} else {
		b.Section = secs.path()
	}
	r, ok := c.m.Range(0, len(c.plain))
	if !ok {
		return Block{}, false
	}
	b.Range = r
	return b, true
}

func endsWithSentencePunct(s string) bool {
	s = strings.TrimSpace(s)
	return s == "" || strings.ContainsRune(".!?,;:", rune(s[len(s)-1]))
}

// looseHeadingLevel recognises short ALL-CAPS lines as top-level headings.
func looseHeadingLevel(line string) int {
	t := strings.TrimSpace(line)
	if len(t) < 3 || len(t) > 60 || endsWithSentencePunct(t) {
		return 0
	}
	letters := 0
	for _, r := range t {
		if unicode.IsLower(r) {
			return 0
		}
		if unicode.IsLetter(r) {
			letters++
		}
	}
	if letters < 3 {
		return 0
	}
	return 1
}

var smallWords = map[string]bool{"a": true, "an": true, "the": true, "of": true, "and": true, "or": true, "in": true, "on": true, "to": true, "for": true, "with": true}

// numberedHeadingLevel recognises "1.2 Title" lines. Multi-part numbering is
// always a heading; a bare "1. Title" needs a short title-cased line, since it
// could just as well be the first item of a numbered list.
func numberedHeadingLevel(line string) int {
	t := strings.TrimSpace(line)
	m := reNumbered.FindStringSubmatch(t)
	if m == nil || len(t) > 80 || endsWithSentencePunct(t) {
		return 0
	}
	parts := strings.Count(m[1], ".") + 1
	if parts > 1 {
		return parts
	}
	words := strings.Fields(m[2])
	if len(words) > 6 {
		return 0
	}
	for _, w := range words {
		r := []rune(w)[0]
		if !unicode.IsUpper(r) && !smallWords[strings.ToLower(w)] {
			return 0
		}
	}
	return 1
}
