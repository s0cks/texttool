package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/s0cks/texttool/internal/lexicon"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func writeLexiconFile(lex *lexicon.Lexicon) error {
	var filename = viper.GetString("target")
	if !strings.HasSuffix(filename, ".bin") {
		filename = filename + ".bin"
	}

	bytes, err := lexicon.WriteLexiconFbs(lex)
	if err != nil {
		return fmt.Errorf("failed to write Lexicon to fbs")
	}

	if err := os.WriteFile(filename, bytes, 0644); err != nil {
		return fmt.Errorf("failed to write Lexicon file to %s: %v", filename, err)
	}

	return nil
}

func init() {
	c := &cobra.Command{
		Use: "make-lexicon sources...",
		RunE: func(cmd *cobra.Command, args []string) error {
			baseLex, err := lexicon.LoadCli()
			if err != nil {
				return err
			}

			out := lexicon.Empty()
			if viper.GetBool("include-builtin") {
				out.Merge(lexicon.Builtin())
			} else if viper.GetBool("include-all") {
				out.Merge(baseLex)
			}

			for _, file := range viper.GetStringSlice("include") {
				fileLex, err := lexicon.Load(file)
				if err != nil {
					return err
				}

				out.Merge(fileLex)
			}

			return writeLexiconFile(out)
		},
	}
	c.Flags().Bool("include-builtin", false, "whether or not to include the builtin lexicon")
	c.Flags().Bool("include-all", false, "whether or not to include all loaded lexicons")
	c.Flags().StringArray("include", nil, "the extra lexicons to include")
	c.Flags().String("target", "lexicon.bin", "the target filename to generate")
	c.MarkFlagsMutuallyExclusive("include-builtin", "include-all")
	RootCommand.AddCommand(c)
}
