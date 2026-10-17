// Command texttool segments markdown or plain text into JSON with source
// ranges and optionally annotates it (POS, NER, stems) or rewrites it for
// text-to-speech models.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/s0cks/texttool/internal/doc"
	"github.com/s0cks/texttool/internal/emit"
	"github.com/s0cks/texttool/internal/lexicon"
	"github.com/s0cks/texttool/internal/pipeline"
	"github.com/s0cks/texttool/internal/tts"
	"github.com/spf13/pflag"
)

const usage = `texttool - segment prose into JSON and prepare it for LLMs, classifiers and TTS

Usage:
  texttool [flags] [file ...]      reads stdin when no file (or "-") is given

Segmentation always runs. Every segment carries a byte range into the source,
its heading breadcrumb ("section") and its kind. Other steps are enabled by flags.

Flags:
`

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "texttool:", err)
		os.Exit(1)
	}
}

func oneOf(name, val string, allowed ...string) error {
	for _, a := range allowed {
		if val == a {
			return nil
		}
	}
	return fmt.Errorf("--%s=%q: want one of %s", name, val, strings.Join(allowed, ", "))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fs := pflag.NewFlagSet("texttool", pflag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprint(stderr, usage)
		fs.PrintDefaults()
	}

	var (
		format    = fs.String("format", "auto", "input format: md, txt or auto (by extension; stdin is md)")
		output    = fs.String("output", "json", "output: json (object, or array for several files) or ndjson (one record per line)")
		pretty    = fs.Bool("pretty", false, "indent JSON output")
		pos       = fs.Bool("pos", false, "add part-of-speech tags per token")
		ner       = fs.Bool("ner", false, "add named entities")
		stem      = fs.Bool("stem", false, "add Snowball stems per token")
		ttsOn     = fs.Bool("tts", false, "add a text-to-speech-cleaned `tts` field to every segment")
		counts    = fs.Bool("counts", false, "add an approximate token_count to every segment")
		chunkN    = fs.Int("chunk", 0, "group segments into chunks of at most N tokens (same section only)")
		lexFiles  = fs.StringArray("lexicon", nil, "user lexicon YAML merged over the builtin one (repeatable; later files win)")
		noBuiltin = fs.Bool("no-builtin-lexicon", false, "do not load the builtin lexicon")
		acronyms  = fs.String("acronyms", "spell", "unknown/known acronyms with --tts: spell, expand or keep")
		handles   = fs.String("handles", "say", "@handles with --tts: say, strip or keep")
		inline    = fs.String("inline-code", "speak", "inline code with --tts: speak, strip or placeholder")
		blocks    = fs.String("code-blocks", "strip", "code blocks with --tts: strip or placeholder")
		urls      = fs.String("urls", "drop", "URLs with --tts: drop, domain or spell")
		math      = fs.String("math", "drop", "pandoc inline math with --tts: drop or placeholder")
		skip      = fs.StringSlice("skip", nil, "segment kinds to omit, e.g. code_block,table_cell")
		dump      = fs.Bool("dump-lexicon", false, "print the merged lexicon as YAML and exit")
		help      = fs.BoolP("help", "h", false, "show this help")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *help {
		fs.Usage()
		return nil
	}

	for _, c := range []struct {
		n, v string
		ok   []string
	}{
		{"format", *format, []string{"md", "txt", "auto"}},
		{"output", *output, []string{"json", "ndjson"}},
		{"acronyms", *acronyms, []string{"spell", "expand", "keep"}},
		{"handles", *handles, []string{"say", "strip", "keep"}},
		{"inline-code", *inline, []string{"speak", "strip", "placeholder"}},
		{"code-blocks", *blocks, []string{"strip", "placeholder"}},
		{"urls", *urls, []string{"drop", "domain", "spell"}},
		{"math", *math, []string{"drop", "placeholder"}},
	} {
		if err := oneOf(c.n, c.v, c.ok...); err != nil {
			return err
		}
	}
	if *chunkN < 0 {
		return fmt.Errorf("--chunk must be positive")
	}

	lex := lexicon.Empty()
	if !*noBuiltin {
		lex = lexicon.Builtin()
	}
	for _, f := range *lexFiles {
		u, err := lexicon.Load(f)
		if err != nil {
			return err
		}
		lex.Merge(u)
	}
	if *dump {
		b, err := lex.Dump()
		if err != nil {
			return err
		}
		_, err = stdout.Write(b)
		return err
	}

	skipSet := map[string]bool{}
	for _, k := range *skip {
		skipSet[strings.TrimSpace(k)] = true
	}

	proc, err := pipeline.New(pipeline.Options{
		Format: *format,
		POS:    *pos, NER: *ner, Stem: *stem,
		TTS:    *ttsOn,
		Counts: *counts,
		Chunk:  *chunkN,
		Skip:   skipSet,
		TTSOptions: tts.Options{
			Lexicon:    lex,
			Acronyms:   *acronyms,
			Handles:    *handles,
			InlineCode: *inline,
			CodeBlocks: *blocks,
			URLs:       *urls,
			Math:       *math,
		},
	})
	if err != nil {
		return err
	}

	inputs := fs.Args()
	if len(inputs) == 0 {
		inputs = []string{"-"}
	}
	var docs []*doc.Document
	for _, in := range inputs {
		var src []byte
		var name string
		var rerr error
		if in == "-" {
			src, rerr = io.ReadAll(stdin)
		} else {
			name = in
			src, rerr = os.ReadFile(in)
		}
		if rerr != nil {
			return rerr
		}
		d, err := proc.Process(src, name)
		if err != nil {
			return fmt.Errorf("%s: %w", in, err)
		}
		docs = append(docs, d)
	}
	return emit.Write(stdout, docs, *output == "ndjson", *pretty)
}
