package lexicon

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/spf13/viper"
)

//go:embed builtin.bin
var builtinLexicon []byte

type Acronym struct {
	Say    string `yaml:"say"`
	Spell  string `yaml:"spell"`
	Expand string `yaml:"expand"`
}

func (a *Acronym) UnmarshalYAML(b []byte) error {
	var s string
	if err := yaml.Unmarshal(b, &s); err == nil {
		*a = Acronym{Say: s}
		return nil
	}

	type plain Acronym
	var p plain
	if err := yaml.Unmarshal(b, &p); err != nil {
		return err
	}

	*a = Acronym(p)
	return nil
}

type Lexicon struct {
	Acronyms map[string]Acronym `yaml:"acronyms"`
	Words    map[string]string  `yaml:"words"`
	Handles  map[string]string  `yaml:"handles"`
}

func empty() *Lexicon {
	return &Lexicon{Acronyms: map[string]Acronym{}, Words: map[string]string{}, Handles: map[string]string{}}
}

func WriteLexiconJsonCompact(lex *Lexicon) ([]byte, error) {
	return json.Marshal(lex)
}

func WriteLexiconJson(lex *Lexicon) ([]byte, error) {
	return json.MarshalIndent(lex, "", "  ")
}

func parseJson(b []byte) (*Lexicon, error) {
	l := empty()
	if err := json.Unmarshal(b, l); err != nil {
		return nil, err
	}

	l.Normalize()
	return l, nil
}

func WriteLexiconYaml(lex *Lexicon) ([]byte, error) {
	return yaml.Marshal(lex)
}

func parseYaml(b []byte) (*Lexicon, error) {
	l := empty()
	if err := yaml.Unmarshal(b, l); err != nil {
		return nil, err
	}

	l.Normalize()
	return l, nil
}

func parse(path string) (*Lexicon, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	ext := filepath.Ext(path)
	switch ext {
	case "yaml":
		return parseYaml(b)

	case "yml":
		return parseYaml(b)

	case "json":
		return parseJson(b)

	case "bin":
		return parseFbs(b)

	default:
		return nil, fmt.Errorf("invalid manifest path `%s`", path)
	}
}

func Builtin() *Lexicon {
	l, err := parseFbs(builtinLexicon)
	if err != nil {
		panic(fmt.Sprintf("lexicon: builtin.bin is invalid: %v", err))
	}

	return l
}

func Empty() *Lexicon {
	return empty()
}

func Load(path string) (*Lexicon, error) {
	l, err := parse(path)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	return l, nil
}

func LoadCli() (*Lexicon, error) {
	lexFiles := viper.GetStringSlice("lexicon")
	noBuiltin := viper.GetBool("no-builtin-lexicon")

	lex := Empty()
	if !noBuiltin {
		lex = Builtin()
	}

	for _, f := range lexFiles {
		u, err := Load(f)
		if err != nil {
			return nil, err
		}

		lex.Merge(u)
	}

	return lex, nil
}

func (l *Lexicon) Merge(o *Lexicon) {
	maps.Copy(l.Acronyms, o.Acronyms)
	for k, v := range o.Words {
		l.Words[strings.ToLower(k)] = v
	}

	for k, v := range o.Handles {
		l.Handles[strings.ToLower(k)] = v
	}
}

func (l *Lexicon) Normalize() {
	w := make(map[string]string, len(l.Words))
	for k, v := range l.Words {
		w[strings.ToLower(k)] = v
	}

	l.Words = w
	h := make(map[string]string, len(l.Handles))
	for k, v := range l.Handles {
		h[strings.ToLower(k)] = v
	}

	l.Handles = h

	if l.Acronyms == nil {
		l.Acronyms = map[string]Acronym{}
	}

	if l.Words == nil {
		l.Words = map[string]string{}
	}

	if l.Handles == nil {
		l.Handles = map[string]string{}
	}
}

func (l *Lexicon) Abbreviations() []string {
	var out []string
	for k := range l.Acronyms {
		if before, ok := strings.CutSuffix(k, "."); ok {
			out = append(out, before)
		}
	}

	for k := range l.Words {
		if before, ok := strings.CutSuffix(k, "."); ok {
			out = append(out, before)
		}
	}

	sort.Strings(out)
	return out
}
