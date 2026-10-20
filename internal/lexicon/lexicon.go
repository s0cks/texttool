// Package lexicon holds pronunciation data for acronyms, words and handles:
// a builtin set embedded in the binary plus any user-supplied files.
package lexicon

import (
	_ "embed"
	"fmt"
	"maps"
	"os"
	"sort"
	"strings"

	"github.com/goccy/go-yaml"
)

//go:embed builtin.yaml
var builtinYAML []byte

// Acronym describes how an acronym or abbreviation is read aloud.
type Acronym struct {
	Say    string `yaml:"say"`
	Spell  string `yaml:"spell"`
	Expand string `yaml:"expand"`
}

// UnmarshalYAML accepts either a mapping or a bare string (shorthand for say).
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

// Lexicon is the merged pronunciation data.
type Lexicon struct {
	Acronyms map[string]Acronym `yaml:"acronyms"`
	Words    map[string]string  `yaml:"words"`
	Handles  map[string]string  `yaml:"handles"`
}

func empty() *Lexicon {
	return &Lexicon{Acronyms: map[string]Acronym{}, Words: map[string]string{}, Handles: map[string]string{}}
}

func parse(b []byte) (*Lexicon, error) {
	l := empty()
	if err := yaml.Unmarshal(b, l); err != nil {
		return nil, err
	}
	if l.Acronyms == nil {
		l.Acronyms = map[string]Acronym{}
	}
	if l.Words == nil {
		l.Words = map[string]string{}
	}
	if l.Handles == nil {
		l.Handles = map[string]string{}
	}
	l.Normalize()
	return l, nil
}

// Builtin returns the lexicon embedded in the binary.
func Builtin() *Lexicon {
	l, err := parse(builtinYAML)
	if err != nil {
		panic(fmt.Sprintf("lexicon: builtin.yaml is invalid: %v", err))
	}
	return l
}

// Empty returns a lexicon with no entries.
func Empty() *Lexicon { return empty() }

// Load reads a user lexicon file.
func Load(path string) (*Lexicon, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	l, err := parse(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return l, nil
}

// Merge copies o's entries over l's, so o wins on conflict.
func (l *Lexicon) Merge(o *Lexicon) {
	maps.Copy(l.Acronyms, o.Acronyms)
	for k, v := range o.Words {
		l.Words[strings.ToLower(k)] = v
	}
	for k, v := range o.Handles {
		l.Handles[strings.ToLower(k)] = v
	}
}

// Normalize lower-cases the keys of Words and Handles.
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
}

// Abbreviations lists dotted entries ("e.g.", "Dr.") so the sentence splitter
// can avoid breaking after them.
func (l *Lexicon) Abbreviations() []string {
	var out []string
	for k := range l.Acronyms {
		if strings.HasSuffix(k, ".") {
			out = append(out, strings.TrimSuffix(k, "."))
		}
	}
	for k := range l.Words {
		if strings.HasSuffix(k, ".") {
			out = append(out, strings.TrimSuffix(k, "."))
		}
	}
	sort.Strings(out)
	return out
}

// Dump renders the merged lexicon as YAML.
func (l *Lexicon) Dump() ([]byte, error) {
	return yaml.Marshal(l)
}
