package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/s0cks/texttool/internal"
	"github.com/s0cks/texttool/internal/doc"
	"github.com/s0cks/texttool/internal/emit"
	"github.com/s0cks/texttool/internal/lexicon"
	"github.com/s0cks/texttool/internal/pipeline"
	"github.com/s0cks/texttool/internal/tts"
)

type runOpts struct {
	Format string
	Output string
	Pretty bool
	Pos    bool
	Ner    bool
	Stem   bool
	Tts    bool
	Counts bool
	ChunkN int

	Acronyms string
	Inline   string
	Blocks   string
	Urls     string
	Math     string
	Skip     []string
}

var RootCommand = &cobra.Command{
	Use:   "texttool",
	Short: "segment prose into JSON and prepare it for LLMs, classifiers and TTS",
	Long: `texttool [flags] [file ...]      reads os.Stdin when no file (or "-") is given

Segmentation always runs. Every segment carries a byte range into the source,
its heading breadcrumb ("section") and its kind. Other steps are enabled by flags.`,
	Args: cobra.ExactArgs(1),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		viper.AutomaticEnv()
		viper.EnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
		viper.SetEnvPrefix("TEXTTOOL")
		return viper.BindPFlags(cmd.Flags())
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := runOpts{
			Format: viper.GetString("format"),
			Output: viper.GetString("output"),
			Pretty: viper.GetBool("pretty"),

			Pos:    viper.GetBool("pos"),
			Ner:    viper.GetBool("ner"),
			Stem:   viper.GetBool("stem"),
			Tts:    viper.GetBool("tts"),
			Counts: viper.GetBool("counts"),
			ChunkN: viper.GetInt("chunk"),

			Acronyms: viper.GetString("acronyms"),
			Inline:   viper.GetString("inline-code"),
			Blocks:   viper.GetString("code-blocks"),
			Urls:     viper.GetString("urls"),
			Math:     viper.GetString("math"),
			Skip:     viper.GetStringSlice("skip"),
		}

		for _, c := range []struct {
			n, v string
			ok   []string
		}{
			{"format", opts.Format, []string{"md", "txt", "auto"}},
			{"output", opts.Output, []string{"json", "ndjson"}},
			{"inline-code", opts.Inline, []string{"speak", "strip", "placeholder"}},
			{"code-blocks", opts.Blocks, []string{"strip", "placeholder"}},
			{"urls", opts.Urls, []string{"drop", "domain", "spell"}},
			{"math", opts.Math, []string{"drop", "placeholder"}},
		} {
			if err := oneOf(c.n, c.v, c.ok...); err != nil {
				return err
			}
		}

		return run(&opts, args)
	},
}

func run(opts *runOpts, args []string) error {
	if opts.ChunkN < 0 {
		return fmt.Errorf("--chunk must be positive")
	}

	lex, err := lexicon.LoadCli()
	if err != nil {
		return err
	}

	skipSet := map[string]bool{}
	for _, k := range opts.Skip {
		skipSet[strings.TrimSpace(k)] = true
	}

	proc, err := pipeline.New(pipeline.Options{
		Format: opts.Format,
		POS:    opts.Pos, NER: opts.Ner, Stem: opts.Stem,
		TTS:    opts.Tts,
		Counts: opts.Counts,
		Chunk:  opts.ChunkN,
		Skip:   skipSet,
		TTSOptions: tts.Options{
			Lexicon:    lex,
			Acronyms:   opts.Acronyms,
			Handles:    string(internal.GetHandlesMode()),
			InlineCode: opts.Inline,
			CodeBlocks: opts.Blocks,
			URLs:       opts.Urls,
			Math:       opts.Math,
		},
	})
	if err != nil {
		return err
	}

	if len(args) == 0 {
		args = []string{"-"}
	}

	var docs []*doc.Document
	for _, in := range args {
		var src []byte
		var name string
		var rerr error
		if in == "-" {
			src, rerr = io.ReadAll(os.Stdin)
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

	return emit.Write(os.Stdout, docs, opts.Output == "ndjson", opts.Pretty)
}

func Execute(ctx context.Context) error {
	return RootCommand.ExecuteContext(ctx)
}

func init() {
	RootCommand.Flags().String("format", "auto", "input format: md, txt or auto (by extension; os.Stdin is md)")
	RootCommand.Flags().String("output", "json", "output: json (object, or array for several files) or ndjson (one record per line)")
	RootCommand.Flags().Bool("pretty", false, "indent JSON output")
	RootCommand.Flags().Bool("pos", false, "add part-of-speech tags per token")
	RootCommand.Flags().Bool("ner", false, "add named entities")
	RootCommand.Flags().Bool("stem", false, "add Snowball stems per token")
	RootCommand.Flags().Bool("tts", false, "add a text-to-speech-cleaned `tts` field to every segment")
	RootCommand.Flags().Bool("counts", false, "add an approximate token_count to every segment")
	RootCommand.Flags().Int("chunk", 0, "group segments into chunks of at most N tokens (same section only)")
	lexicon.AddLexiconFlags(RootCommand)
	internal.AddAcryonymFlags(RootCommand)
	internal.AddHandlesFlags(RootCommand)
	RootCommand.Flags().String("inline-code", "speak", "inline code with --tts: speak, strip or placeholder")
	RootCommand.Flags().String("code-blocks", "strip", "code blocks with --tts: strip or placeholder")
	RootCommand.Flags().String("urls", "drop", "URLs with --tts: drop, domain or spell")
	RootCommand.Flags().String("math", "drop", "pandoc inline math with --tts: drop or placeholder")
	RootCommand.Flags().StringSlice("skip", nil, "segment kinds to omit, e.g. code_block,table_cell")
}

func oneOf(name, val string, allowed ...string) error {
	if slices.Contains(allowed, val) {
		return nil
	}

	return fmt.Errorf("--%s=%q: want one of %s", name, val, strings.Join(allowed, ", "))
}
