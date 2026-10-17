package parse

import (
	"sort"

	"github.com/s0cks/texttool/internal/doc"
)

// run ties a stretch of plain text to the source bytes it came from.
// sStart < 0 marks synthetic text (a joining space) with no source.
type run struct {
	pStart, pEnd int
	sStart, sEnd int
}

func (r run) synthetic() bool { return r.sStart < 0 }
func (r run) equalLen() bool  { return r.pEnd-r.pStart == r.sEnd-r.sStart }

// Mapper converts byte offsets in a block's plain text back to source offsets.
type Mapper struct {
	runs []run
}

func (m *Mapper) add(pStart, pEnd, sStart, sEnd int) {
	if pEnd <= pStart {
		return
	}
	m.runs = append(m.runs, run{pStart, pEnd, sStart, sEnd})
}

// Start maps the plain offset a (the first byte of a range) to a source offset.
func (m *Mapper) Start(a int) (int, bool) {
	i := sort.Search(len(m.runs), func(i int) bool { return m.runs[i].pEnd > a })
	for ; i < len(m.runs); i++ {
		r := m.runs[i]
		if r.synthetic() {
			continue
		}
		if r.pStart > a {
			return r.sStart, true // a sat in a gap; snap forward
		}
		off := a - r.pStart
		if r.equalLen() {
			return r.sStart + off, true
		}
		if off == 0 {
			return r.sStart, true
		}
		return min(r.sStart+off, r.sEnd-1), true
	}
	return 0, false
}

// End maps the plain offset b (one past the last byte of a range) to a source offset.
func (m *Mapper) End(b int) (int, bool) {
	i := sort.Search(len(m.runs), func(i int) bool { return m.runs[i].pEnd >= b })
	if i == len(m.runs) {
		i = len(m.runs) - 1
	}
	for ; i >= 0; i-- {
		r := m.runs[i]
		if r.synthetic() {
			continue
		}
		if r.pStart >= b {
			continue // b sat in a gap before this run; snap back to an earlier one
		}
		if b >= r.pEnd {
			return r.sEnd, true
		}
		off := b - r.pStart
		if r.equalLen() {
			return r.sStart + off, true
		}
		return min(r.sStart+off, r.sEnd), true
	}
	return 0, false
}

// Range maps the plain range [a, b) to a source range.
func (m *Mapper) Range(a, b int) (doc.Range, bool) {
	s, ok1 := m.Start(a)
	e, ok2 := m.End(b)
	if !ok1 || !ok2 || e <= s {
		return doc.Range{}, false
	}
	return doc.Range{Start: s, End: e}, true
}
