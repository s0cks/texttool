package cmd

import (
	"fmt"
	"strings"

	"github.com/s0cks/texttool/internal/lexicon"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type LexiconFormatter = func(lex *lexicon.Lexicon) ([]byte, error)

func init() {
	defaultFormat := "yml"
	formatters := map[string]LexiconFormatter{
		"yml":   lexicon.WriteLexiconYaml,
		"yaml":  lexicon.WriteLexiconYaml,
		"json":  lexicon.WriteLexiconJson,
		"jsonl": lexicon.WriteLexiconJsonCompact,
	}
	formats := make([]string, 0, len(formatters))
	for k := range formatters {
		formats = append(formats, k)
	}

	c := &cobra.Command{
		Use:   "dump-lexicon",
		Short: "dump the entire lexicon to console",
		RunE: func(cmd *cobra.Command, args []string) error {
			lex, err := lexicon.LoadCli()
			if err != nil {
				return err
			}

			format := viper.GetString("format")
			formatter := formatters[format]

			bytes, err := formatter(lex)
			if err != nil {
				return err
			}

			fmt.Println(string(bytes))
			return err
		},
	}
	lexicon.AddLexiconFlags(c)
	c.Flags().String("format", defaultFormat, fmt.Sprintf("the output format, valid values are: %s", strings.Join(formats, ", ")))
	RootCommand.AddCommand(c)
}
