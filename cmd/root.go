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
		fs := cmd.Flags()
		var (
			format = viper.GetString("format")
			output = viper.GetString("output")
			pretty = viper.GetBool("pretty")
			pos    = viper.GetBool("pos")
			ner    = viper.GetBool("ner")
			stem   = viper.GetBool("stem")
			ttsOn  = viper.GetBool("tts")
			counts = viper.GetBool("counts")
			chunkN = viper.GetInt("chunk")

			dump      = viper.GetBool("dump-lexicon")
			lexFiles  = viper.GetStringSlice("lexicon")
			noBuiltin = viper.GetBool("no-builtin-lexicon")

			acronyms = viper.GetString("acronyms")
			handles  = viper.GetString("handles")
			inline   = viper.GetString("inline-code")
			blocks   = viper.GetString("code-blocks")
			urls     = viper.GetString("urls")
			math     = viper.GetString("math")
			skip     = viper.GetStringSlice("skip")
		)

		for _, c := range []struct {
			n, v string
			ok   []string
		}{
			{"format", format, []string{"md", "txt", "auto"}},
			{"output", output, []string{"json", "ndjson"}},
			{"acronyms", acronyms, []string{"spell", "expand", "keep"}},
			{"handles", handles, []string{"say", "strip", "keep"}},
			{"inline-code", inline, []string{"speak", "strip", "placeholder"}},
			{"code-blocks", blocks, []string{"strip", "placeholder"}},
			{"urls", urls, []string{"drop", "domain", "spell"}},
			{"math", math, []string{"drop", "placeholder"}},
		} {
			if err := oneOf(c.n, c.v, c.ok...); err != nil {
				return err
			}
		}

		if chunkN < 0 {
			return fmt.Errorf("--chunk must be positive")
		}

		lex := lexicon.Empty()
		if !noBuiltin {
			lex = lexicon.Builtin()
		}

		for _, f := range lexFiles {
			u, err := lexicon.Load(f)
			if err != nil {
				return err
			}

			lex.Merge(u)
		}

		if dump {
			b, err := lex.Dump()
			if err != nil {
				return err
			}

			_, err = os.Stdout.Write(b)
			return err
		}

		skipSet := map[string]bool{}
		for _, k := range skip {
			skipSet[strings.TrimSpace(k)] = true
		}

		proc, err := pipeline.New(pipeline.Options{
			Format: format,
			POS:    pos, NER: ner, Stem: stem,
			TTS:    ttsOn,
			Counts: counts,
			Chunk:  chunkN,
			Skip:   skipSet,
			TTSOptions: tts.Options{
				Lexicon:    lex,
				Acronyms:   acronyms,
				Handles:    handles,
				InlineCode: inline,
				CodeBlocks: blocks,
				URLs:       urls,
				Math:       math,
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

		return emit.Write(os.Stdout, docs, output == "ndjson", pretty)
	},
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
	RootCommand.Flags().StringArray("lexicon", nil, "user lexicon YAML merged over the builtin one (repeatable; later files win)")
	RootCommand.Flags().Bool("no-builtin-lexicon", false, "do not load the builtin lexicon")
	internal.AddAcryonymFlags(RootCommand)
	RootCommand.Flags().String("handles", "say", "@handles with --tts: say, strip or keep")
	RootCommand.Flags().String("inline-code", "speak", "inline code with --tts: speak, strip or placeholder")
	RootCommand.Flags().String("code-blocks", "strip", "code blocks with --tts: strip or placeholder")
	RootCommand.Flags().String("urls", "drop", "URLs with --tts: drop, domain or spell")
	RootCommand.Flags().String("math", "drop", "pandoc inline math with --tts: drop or placeholder")
	RootCommand.Flags().StringSlice("skip", nil, "segment kinds to omit, e.g. code_block,table_cell")
	RootCommand.Flags().Bool("dump-lexicon", false, "print the merged lexicon as YAML and exit")
}

func oneOf(name, val string, allowed ...string) error {
	if slices.Contains(allowed, val) {
		return nil
	}

	return fmt.Errorf("--%s=%q: want one of %s", name, val, strings.Join(allowed, ", "))
}
