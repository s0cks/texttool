// Package chunk counts tokens and groups segments into budgeted chunks.
package chunk

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/s0cks/texttool/internal/doc"
)

var reToken = regexp.MustCompile(`\w+|[^\w\s]`)

// Count approximates the token count of s as words plus punctuation marks.
// It is deterministic and cheap, not a model tokenizer; budget accordingly.
func Count(s string) int { return len(reToken.FindAllStringIndex(s, -1)) }

// Build groups consecutive segments of one section into chunks of at most
// budget tokens. A single segment larger than the budget becomes its own
// chunk. Code blocks are skipped. When useTTS is set, chunk text is the TTS
// text instead of the plain text.
func Build(segs []doc.Segment, budget int, useTTS bool) []doc.Chunk {
	var chunks []doc.Chunk
	var cur *doc.Chunk
	var parts []string
	lastBlock := -1

	flush := func() {
		if cur == nil {
			return
		}
		cur.Text = strings.Join(parts, "")
		cur.ID = fmt.Sprintf("c%d", len(chunks))
		chunks = append(chunks, *cur)
		cur, parts, lastBlock = nil, nil, -1
	}
	for _, s := range segs {
		if s.Kind == doc.KindCodeBlock {
			continue
		}
		text := s.Plain
		if useTTS && s.TTS != nil {
			text = *s.TTS
		}
		if strings.TrimSpace(text) == "" {
			continue
		}
		n := s.TokenCount
		if n == 0 {
			n = Count(s.Plain)
		}
		if cur != nil && (!sameSection(cur.Section, s.Section) || cur.TokenCount+n > budget) {
			flush()
		}
		if cur == nil {
			cur = &doc.Chunk{Type: "chunk", Section: s.Section, Range: doc.Range{Start: s.Range.Start}}
		} else if s.Block != lastBlock {
			parts = append(parts, "\n\n")
		} else {
			parts = append(parts, " ")
		}
		parts = append(parts, text)
		cur.Range.End = s.Range.End
		cur.SegmentIDs = append(cur.SegmentIDs, s.ID)
		cur.TokenCount += n
		lastBlock = s.Block
	}
	flush()
	return chunks
}

func sameSection(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
