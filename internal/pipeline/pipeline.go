// Package pipeline wires the stages together: parse, segment, annotate,
// TTS-clean and chunk.
package pipeline

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/s0cks/texttool/internal/annotate"
	"github.com/s0cks/texttool/internal/chunk"
	"github.com/s0cks/texttool/internal/doc"
	"github.com/s0cks/texttool/internal/lexicon"
	"github.com/s0cks/texttool/internal/parse"
	"github.com/s0cks/texttool/internal/segment"
	"github.com/s0cks/texttool/internal/tts"
)

// Options selects the optional stages. Segmentation always runs.
type Options struct {
	Format string // md | txt | auto (default)

	POS, NER, Stem bool
	TTS            bool
	Counts         bool
	Chunk          int // token budget; 0 disables chunking
	Skip           map[string]bool

	TTSOptions tts.Options // Lexicon here also feeds sentence-splitter abbreviations
}

// Processor runs the pipeline. Build it once and reuse it across inputs.
type Processor struct {
	opt      Options
	splitter *segment.Splitter
	cleaner  *tts.Cleaner
}

// New builds a Processor.
func New(opt Options) (*Processor, error) {
	if opt.TTSOptions.Lexicon == nil {
		opt.TTSOptions.Lexicon = lexicon.Builtin()
	}
	opt.TTSOptions.Terminate = true
	sp, err := segment.New(opt.TTSOptions.Lexicon.Abbreviations())
	if err != nil {
		return nil, err
	}
	p := &Processor{opt: opt, splitter: sp}
	if opt.TTS {
		p.cleaner = tts.New(opt.TTSOptions)
	}
	return p, nil
}

// DetectFormat picks md or txt from a file name; anything unknown is markdown.
func DetectFormat(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".txt", ".text":
		return "txt"
	}
	return "md"
}

// Process runs the pipeline over one input.
func (p *Processor) Process(src []byte, name string) (*doc.Document, error) {
	format := p.opt.Format
	if format == "" || format == "auto" {
		format = DetectFormat(name)
	}

	var blocks []parse.Block
	var err error
	switch format {
	case "md":
		blocks, err = parse.Markdown(src)
	case "txt":
		blocks, err = parse.PlainText(src)
	default:
		return nil, fmt.Errorf("unknown format %q (want md, txt or auto)", format)
	}
	if err != nil {
		return nil, err
	}

	segs := p.splitter.Split(blocks, src, "")
	if len(p.opt.Skip) > 0 {
		kept := segs[:0]
		for _, s := range segs {
			if !p.opt.Skip[s.Kind] {
				kept = append(kept, s)
			}
		}
		segs = kept
	}

	if err := annotate.Run(segs, annotate.Options{POS: p.opt.POS, NER: p.opt.NER, Stem: p.opt.Stem}); err != nil {
		return nil, fmt.Errorf("annotate: %w", err)
	}
	if p.cleaner != nil {
		for i := range segs {
			s := p.cleaner.Clean(&segs[i])
			segs[i].TTS = &s
		}
	}
	if p.opt.Counts || p.opt.Chunk > 0 {
		for i := range segs {
			if segs[i].Kind != doc.KindCodeBlock {
				segs[i].TokenCount = chunk.Count(segs[i].Plain)
			}
		}
	}

	d := &doc.Document{Source: name, Format: format, Segments: segs}
	if p.opt.Chunk > 0 {
		d.Chunks = chunk.Build(segs, p.opt.Chunk, p.opt.TTS)
	}
	return d, nil
}
