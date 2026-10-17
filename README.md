# texttool

Segment markdown or plain text into JSON with byte ranges, and optionally
annotate it (POS, NER, stems) or rewrite it for text-to-speech models. One
command; segmentation always runs and flags switch the other steps on.

Built for the pipeline `typst => pandoc => markdown`, with plain text as the
fallback. English only (American number reading).

```
texttool [flags] [file ...]        # no file, or "-", reads stdin

texttool doc.md                          # segments + headings + ranges
texttool --ner --pos doc.md              # + named entities, POS tags
texttool --tts doc.md                    # + TTS-cleaned text per segment
texttool --tts --chunk 400 --output ndjson doc.md   # chunks for an LLM
```

## Output

Each **segment** is one sentence, heading, code block or table cell:

```json
{
  "type": "segment", "id": "s7", "block": 3, "kind": "paragraph",
  "section": ["Getting Started", "Install"],
  "range": {"start": 353, "end": 417},
  "text":  "The tool costs $5 and was released in **1989**.",
  "plain": "The tool costs $5 and was released in 1989.",
  "tts":   "The tool costs five dollars and was released in nineteen eighty-nine."
}
```

- `range` is a **byte** range into the original input, and `text` is exactly
  `src[start:end]` (a test enforces this). `plain` is the same sentence with
  markup stripped and hard-wrapped lines joined.
- `kind`: `heading` (with `level`), `paragraph`, `list_item`, `blockquote`,
  `footnote`, `term`, `table_cell`, `code_block` (with `lang`).
- `section` is the heading breadcrumb; for a heading it ends with itself.
- `tokens`, `entities`: ranges are byte offsets into `plain`, not the source.
- `--chunk N` adds `{"type":"chunk", ...}` records: consecutive segments from
  one section, at most N tokens, with the range they cover.

`--output json` (default) writes one object (an array for several files).
`--output ndjson` writes one record per line.

## Flags

| Flag | Meaning |
|------|---------|
| `--format md\|txt\|auto` | auto picks by extension; stdin is markdown |
| `--pos` `--ner` `--stem` | tokens with POS tags, entities, Snowball stems |
| `--tts` | add the `tts` field |
| `--counts` | add an approximate `token_count` (words + punctuation, not BPE) |
| `--chunk N` | group into chunks of at most N tokens |
| `--lexicon FILE` | user lexicon, repeatable, later wins |
| `--no-builtin-lexicon` | skip the embedded lexicon |
| `--dump-lexicon` | print the merged lexicon and exit |
| `--skip kinds` | omit kinds, e.g. `code_block,table_cell` |
| `--acronyms spell\|expand\|keep` | default `spell`: `CLI` -> `C L I` |
| `--handles say\|strip\|keep` | default `say`: `@s0cks` -> `at s zero cks` |
| `--inline-code speak\|strip\|placeholder` | default `speak` |
| `--code-blocks strip\|placeholder` | default `strip` |
| `--urls drop\|domain\|spell` | default `drop` |
| `--math drop\|placeholder` | pandoc `$...$` math, default `drop` |
| `--pretty` | indent JSON |

## What `--tts` does

Runs per segment, on text that markdown parsing already stripped (so brackets,
`*`, `**`, backticks and link targets never reach the cleaner):

1. Footnote marks, `[@cite]`, `[1]`, math, pandoc `{...}` attributes are dropped.
2. URLs, emails and `@handles` are handled per policy.
3. Lexicon entries that contain symbols or digits (`C++`, `CI/CD`, `e.g.`, `x86`).
4. Numbers, in an order that avoids misreads: ISO dates, versions, times,
   currency (`$1.50` -> one dollar and fifty cents), percent, units (`5GB`),
   ordinals, decades, ranges (`10-20`), phone-style chains, fractions, decimals,
   years (`1989` -> nineteen eighty-nine), then plain integers.
   A 4-digit number is read as a year unless the neighbouring word says it is
   a quantity ("about 1500", "1500 users").
5. Acronyms and pronunciation overrides from the lexicon. Unknown ALL-CAPS
   tokens of 2-3 letters (or 4-6 without vowels) are spelled; common words and
   pronounceable ones (NASA) are left alone; an ALL-CAPS segment is lower-cased.
6. Symbols (`& @ + = < > / ~ ° ×` ...) become words; emoji are removed.
7. Punctuation: em/en dashes and spaced hyphens become commas, `...` becomes a
   comma or period, trailing hyphens and doubled punctuation are removed.
8. Whitespace collapsed to single spaces, and every result ends in `.`, `!` or `?`
   so speech models get a pause at each segment boundary.

Paragraph breaks are not collapsed away: they are segment boundaries, which is
where a TTS pipeline wants its pauses.

## Lexicon

`internal/lexicon/builtin.yaml` is embedded in the binary. See
`examples/lexicon.yaml` for the user format (`acronyms`, `words`, `handles`).
Per acronym: `say` (always used), `spell`, `expand` (used by
`--acronyms expand`). A bare string is shorthand for `say`. Dotted entries such
as `e.g.` and `Dr.` also teach the sentence splitter not to break after them.

## Build

```
go mod tidy          # first time; fills in go.sum
go build ./cmd/texttool
go test ./...
go test ./internal/pipeline -update   # after reviewing a golden diff
```

## Known limits

- **NER quality is modest.** The backend is `jdkato/prose/v2` (perceptron
  trained on CoNLL-style data): it labels PERSON/GPE-style entities, will call
  `CLI` a person, and treats capitalized headings as names. It is behind the
  `annotate.Run` seam so an ONNX backend can replace it later.
- prose reloads its models on every `NewDocument`, about 0.4 s. texttool calls
  it once per input file, not per sentence, so process several files in one run.
- `--skip` leaves gaps in segment ids; ids always refer to positions in the full parse.
- Entity decoding (`&amp;`) and escapes are range-exact at the character level;
  a range that ends inside a multi-character entity snaps to the entity's edge.
- A heading such as `1.2 Details` is read "one point two Details".
- If your text ever comes from PDF extraction instead of pandoc, add an unwrap
  step first (joining hard-wrapped lines, removing soft hyphens); not built yet.
