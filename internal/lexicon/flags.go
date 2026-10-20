package lexicon

import "github.com/spf13/cobra"

func AddLexiconFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.StringArray("lexicon", nil, "user lexicon YAML merged over the builtin one (repeatable; later files win)")
	flags.Bool("no-builtin-lexicon", false, "do not load the builtin lexicon")
}
