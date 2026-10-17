// Package doc defines the data model shared by every pipeline stage.
package doc

// Segment kinds.
const (
	KindHeading    = "heading"
	KindParagraph  = "paragraph"
	KindListItem   = "list_item"
	KindBlockquote = "blockquote"
	KindFootnote   = "footnote"
	KindTerm       = "term"
	KindTableCell  = "table_cell"
	KindCodeBlock  = "code_block"
)

// Range is a half-open byte range [Start, End).
type Range struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// Span marks an inline construct inside a segment's Plain text.
// Start and End are byte offsets into Plain.
type Span struct {
	Kind  string // "code", "link" or "url"
	Start int
	End   int
	Dest  string
}

// Token is one annotated token. Range, when present, is a byte range into the
// owning segment's Plain text.
type Token struct {
	Text  string `json:"t"`
	POS   string `json:"pos,omitempty"`
	Stem  string `json:"stem,omitempty"`
	Range *Range `json:"range,omitempty"`
}

// Entity is a named entity. Range, when present, is a byte range into Plain.
type Entity struct {
	Text  string `json:"text"`
	Label string `json:"label"`
	Range *Range `json:"range,omitempty"`
}

// Segment is the atomic output record: one sentence of prose, one heading,
// one code block, one table cell.
type Segment struct {
	Type       string   `json:"type"` // always "segment"
	ID         string   `json:"id"`
	Source     string   `json:"source,omitempty"`
	Block      int      `json:"block"`
	Kind       string   `json:"kind"`
	Level      int      `json:"level,omitempty"`
	Lang       string   `json:"lang,omitempty"`
	Section    []string `json:"section,omitempty"`
	Range      Range    `json:"range"`
	Text       string   `json:"text"`  // exact source bytes at Range
	Plain      string   `json:"plain"` // markup stripped, whitespace unwrapped
	TTS        *string  `json:"tts,omitempty"`
	Tokens     []Token  `json:"tokens,omitempty"`
	Entities   []Entity `json:"entities,omitempty"`
	TokenCount int      `json:"token_count,omitempty"`

	Spans []Span `json:"-"`
}

// Chunk groups consecutive segments from one section under a token budget.
type Chunk struct {
	Type       string   `json:"type"` // always "chunk"
	ID         string   `json:"id"`
	Source     string   `json:"source,omitempty"`
	Section    []string `json:"section,omitempty"`
	Range      Range    `json:"range"`
	SegmentIDs []string `json:"segments"`
	TokenCount int      `json:"token_count"`
	Text       string   `json:"text"` // plain text, or TTS text when --tts is on
}

// Document is the result of processing one input.
type Document struct {
	Source   string    `json:"source,omitempty"`
	Format   string    `json:"format"`
	Segments []Segment `json:"segments"`
	Chunks   []Chunk   `json:"chunks,omitempty"`
}
