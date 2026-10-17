// Package tts rewrites prose into text that speech models read cleanly: no
// markup, digits and symbols spelled out, acronyms pronounceable, punctuation
// that controls pauses, and normalized whitespace.
package tts

import (
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/s0cks/texttool/internal/doc"
	"github.com/s0cks/texttool/internal/lexicon"
)

// Options controls the cleaner. Empty policy strings take the default.
type Options struct {
	Lexicon    *lexicon.Lexicon // nil means the builtin lexicon
	Acronyms   string           // spell (default) | expand | keep
	Handles    string           // say (default) | strip | keep
	InlineCode string           // speak (default) | strip | placeholder
	CodeBlocks string           // strip (default) | placeholder
	URLs       string           // drop (default) | domain | spell
	Math       string           // drop (default) | placeholder
	Terminate  bool             // ensure every result ends in sentence punctuation
}

func (o *Options) defaults() {
	if o.Lexicon == nil {
		o.Lexicon = lexicon.Builtin()
	}
	set := func(p *string, def string) {
		if *p == "" {
			*p = def
		}
	}
	set(&o.Acronyms, "spell")
	set(&o.Handles, "say")
	set(&o.InlineCode, "speak")
	set(&o.CodeBlocks, "strip")
	set(&o.URLs, "drop")
	set(&o.Math, "drop")
}

type literal struct {
	key  string
	fold bool
}

// Cleaner applies the TTS rewrite. It is safe for concurrent use.
type Cleaner struct {
	opt Options

	acr     map[string]lexicon.Acronym // alphabetic acronym keys
	words   map[string]string          // alphabetic word keys, lower-case
	handles map[string]string

	litExact map[string]string // exact-case literal replacements
	litFold  map[string]string // case-insensitive literal replacements (lower-case keys)
	reExact  *regexp.Regexp
	reFold   *regexp.Regexp
}

// New builds a Cleaner from opt.
func New(opt Options) *Cleaner {
	opt.defaults()
	c := &Cleaner{
		opt:      opt,
		acr:      map[string]lexicon.Acronym{},
		words:    map[string]string{},
		handles:  opt.Lexicon.Handles,
		litExact: map[string]string{},
		litFold:  map[string]string{},
	}
	var exact, fold []string
	for k, a := range opt.Lexicon.Acronyms {
		if isAlpha(k) {
			c.acr[k] = a
			continue
		}
		if v, ok := c.resolve(a, k); ok {
			c.litExact[k] = v
			exact = append(exact, k)
		}
	}
	for k, v := range opt.Lexicon.Words {
		k = strings.ToLower(k)
		if isAlpha(k) {
			c.words[k] = v
			continue
		}
		c.litFold[k] = v
		fold = append(fold, k)
	}
	build := func(keys []string, ci bool) *regexp.Regexp {
		if len(keys) == 0 {
			return nil
		}
		sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
		q := make([]string, len(keys))
		for i, k := range keys {
			q[i] = regexp.QuoteMeta(k)
		}
		pre := ""
		if ci {
			pre = "(?i)"
		}
		return regexp.MustCompile(pre + strings.Join(q, "|"))
	}
	c.reExact = build(exact, false)
	c.reFold = build(fold, true)
	return c
}

func isAlpha(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

// resolve picks the spoken form of an acronym under the active policy.
func (c *Cleaner) resolve(a lexicon.Acronym, key string) (string, bool) {
	if a.Say != "" {
		return a.Say, true
	}
	switch c.opt.Acronyms {
	case "keep":
		return "", false
	case "expand":
		if a.Expand != "" {
			return a.Expand, true
		}
	}
	if a.Spell != "" {
		return a.Spell, true
	}
	return spellLetters(key), true
}

func spellLetters(s string) string {
	var out []string
	for _, r := range s {
		if unicode.IsLetter(r) {
			out = append(out, string(unicode.ToUpper(r)))
		}
	}
	return strings.Join(out, " ")
}

// Clean returns the TTS text for a segment.
func (c *Cleaner) Clean(seg *doc.Segment) string {
	if seg.Kind == doc.KindCodeBlock {
		if c.opt.CodeBlocks == "placeholder" {
			return "Code omitted."
		}
		return ""
	}
	return c.clean(c.applySpans(seg.Plain, seg.Spans))
}

// CleanText cleans a bare string that has no span information.
func (c *Cleaner) CleanText(s string) string { return c.clean(s) }

// applySpans rewrites inline code and URL spans according to policy.
func (c *Cleaner) applySpans(plain string, spans []doc.Span) string {
	if len(spans) == 0 {
		return plain
	}
	sorted := append([]doc.Span(nil), spans...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Start < sorted[j].Start })
	var b strings.Builder
	pos := 0
	for _, sp := range sorted {
		if sp.Kind != "code" && sp.Kind != "url" {
			continue
		}
		if sp.Start < pos || sp.End > len(plain) {
			continue
		}
		b.WriteString(plain[pos:sp.Start])
		content := plain[sp.Start:sp.End]
		switch sp.Kind {
		case "code":
			switch c.opt.InlineCode {
			case "strip":
			case "placeholder":
				b.WriteString("code")
			default:
				b.WriteString(c.speakCode(content))
			}
		case "url":
			b.WriteString(urlWords(content, c.opt.URLs))
		}
		pos = sp.End
	}
	b.WriteString(plain[pos:])
	return b.String()
}

var (
	reDotBetween = regexp.MustCompile(`(\w)\.(\w)`)
	reCamel1     = regexp.MustCompile(`([a-z0-9])([A-Z])`)
	reCamel2     = regexp.MustCompile(`([A-Z]+)([A-Z][a-z])`)
)

func camelSplit(s string) string {
	s = reCamel1.ReplaceAllString(s, "$1 $2")
	return reCamel2.ReplaceAllString(s, "$1 $2")
}

// speakCode turns an inline code span into something sayable: identifiers are
// split into words, "." becomes "dot", empty call parens vanish.
func (c *Cleaner) speakCode(code string) string {
	code = strings.TrimSpace(code)
	if a, ok := c.acr[code]; ok {
		if v, ok := c.resolve(a, code); ok {
			return v
		}
	}
	if v, ok := c.words[strings.ToLower(code)]; ok {
		return v
	}
	code = strings.ReplaceAll(code, "()", "")
	code = reDotBetween.ReplaceAllString(code, "$1 dot $2")
	code = strings.NewReplacer("::", " ", "_", " ", "-", " ", "/", " slash ").Replace(code)
	return camelSplit(code)
}

var (
	reURL      = regexp.MustCompile(`(?i)\b(?:https?://|www\.)[^\s<>()\[\]"']+`)
	reEmail    = regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9-]+(?:\.[A-Za-z0-9-]+)+\b`)
	reHandle   = regexp.MustCompile(`(^|[^\w@./-])@([A-Za-z0-9_][A-Za-z0-9_-]*)`)
	reScheme   = regexp.MustCompile(`(?i)^[a-z][a-z0-9+.-]*://`)
	reTrailing = regexp.MustCompile(`[.,;:!?]+$`)
)

// urlWords renders a URL under the --urls policy.
func urlWords(u, policy string) string {
	if policy == "drop" {
		return ""
	}
	trail := reTrailing.FindString(u)
	u = strings.TrimSuffix(u, trail)
	u = reScheme.ReplaceAllString(u, "")
	if i := strings.IndexAny(u, "?#"); i >= 0 {
		u = u[:i]
	}
	host, path := u, ""
	if i := strings.IndexByte(u, '/'); i >= 0 {
		host, path = u[:i], u[i:]
	}
	if i := strings.LastIndexByte(host, '@'); i >= 0 {
		host = host[i+1:]
	}
	if i := strings.IndexByte(host, ':'); i >= 0 {
		host = host[:i]
	}
	host = strings.TrimPrefix(strings.ToLower(host), "www.")
	out := strings.ReplaceAll(host, ".", " dot ")
	if policy == "spell" {
		for _, seg := range strings.Split(path, "/") {
			if seg == "" {
				continue
			}
			seg = strings.NewReplacer("-", " ", "_", " ").Replace(strings.ReplaceAll(seg, ".", " dot "))
			out += " slash " + seg
		}
	}
	return out + trail
}

func (c *Cleaner) urls(s string) string {
	return reURL.ReplaceAllStringFunc(s, func(m string) string { return urlWords(m, c.opt.URLs) })
}

func emails(s string) string {
	return reEmail.ReplaceAllStringFunc(s, func(m string) string {
		i := strings.LastIndexByte(m, '@')
		local := strings.NewReplacer(".", " dot ", "_", " ", "-", " ", "+", " plus ").Replace(m[:i])
		return local + " at " + strings.ReplaceAll(m[i+1:], ".", " dot ")
	})
}

// nameWords splits a handle or identifier into pronounceable pieces; digits
// are left for the number pass.
func nameWords(name string) string {
	name = strings.NewReplacer("_", " ", "-", " ").Replace(name)
	name = camelSplit(name)
	var b strings.Builder
	var prev rune
	for _, r := range name {
		if prev != 0 && prev != ' ' && r != ' ' && unicode.IsDigit(r) != unicode.IsDigit(prev) {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
		prev = r
	}
	return b.String()
}

func (c *Cleaner) handle(s string) string {
	if c.opt.Handles == "keep" {
		return s
	}
	return replaceIdx(s, reHandle, func(s string, m []int) (string, bool) {
		prefix, name := group(s, m, 1), group(s, m, 2)
		if v, ok := c.handles["@"+strings.ToLower(name)]; ok {
			return prefix + v, true
		}
		if c.opt.Handles == "strip" {
			return prefix, true
		}
		return prefix + "at " + nameWords(name), true
	})
}

var (
	reFootRef   = regexp.MustCompile(`\[\^[^\]]*\]`)
	reCite      = regexp.MustCompile(`\[-?@[^\]]*\]`)
	reNumCite   = regexp.MustCompile(`\[\d+(?:\s*[,–-]\s*\d+)*\]`)
	reMath      = regexp.MustCompile(`\$[^$\s][^$\n]*?[^$\s]\$|\$[^$\s]\$`)
	reEmptyPar  = regexp.MustCompile(`\(\s*\)`)
	reAttrInTTS = regexp.MustCompile(`\{[^{}\n]*\}`)
)

func (c *Cleaner) stripLeftovers(s string) string {
	s = reFootRef.ReplaceAllString(s, "")
	s = reCite.ReplaceAllString(s, "")
	s = reNumCite.ReplaceAllString(s, "")
	s = replaceIdx(s, reMath, func(s string, m []int) (string, bool) {
		inner := s[m[0]+1 : m[1]-1]
		if !strings.ContainsAny(inner, `\^_={}`) {
			return "", false // currency like "$5 and $10", not math
		}
		if c.opt.Math == "placeholder" {
			return "formula", true
		}
		return "", true
	})
	return reAttrInTTS.ReplaceAllString(s, "")
}

var charFix = strings.NewReplacer(
	"\u00a0", " ", "\u200b", "", "\u200d", "", "\ufeff", "", "\u00ad", "", "\ufe0f", "",
	"\u2018", "'", "\u2019", "'", "\u201c", `"`, "\u201d", `"`,
	"\u2026", "...", "\r\n", " ", "\n", " ", "\r", " ", "\t", " ",
)

func (c *Cleaner) clean(s string) string {
	s = charFix.Replace(s)
	allCaps := isAllCaps(s)
	s = c.stripLeftovers(s)
	s = c.urls(s)
	s = emails(s)
	s = c.handle(s)
	s = c.literals(s)
	s = expandNumbers(s)
	s = c.wordsPass(s, allCaps)
	s = expandSymbols(s)
	s = standardizePunctuation(s)
	s = strings.Join(strings.Fields(s), " ")
	s = fixSpacing(s)
	if c.opt.Terminate {
		s = terminate(s)
	}
	return s
}

func isAllCaps(s string) bool {
	run, maxRun := 0, 0
	for _, r := range s {
		switch {
		case unicode.IsLower(r):
			return false
		case unicode.IsUpper(r):
			run++
			maxRun = max(maxRun, run)
		default:
			run = 0
		}
	}
	return maxRun >= 3
}

func alnumByte(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b >= 0x80
}

// literals applies lexicon entries whose keys contain non-letters ("C++",
// "CI/CD", "e.g.", "x86"), which the word pass cannot see.
func (c *Cleaner) literals(s string) string {
	apply := func(s string, re *regexp.Regexp, table map[string]string, fold bool) string {
		if re == nil {
			return s
		}
		return replaceIdx(s, re, func(s string, m []int) (string, bool) {
			key := s[m[0]:m[1]]
			if alnumByte(key[0]) && m[0] > 0 && alnumByte(s[m[0]-1]) {
				return "", false
			}
			if alnumByte(key[len(key)-1]) && m[1] < len(s) && alnumByte(s[m[1]]) {
				return "", false
			}
			if fold {
				key = strings.ToLower(key)
			}
			v, ok := table[key]
			if !ok {
				return "", false
			}
			return " " + v + " ", true
		})
	}
	s = apply(s, c.reExact, c.litExact, false)
	return apply(s, c.reFold, c.litFold, true)
}

var reLetters = regexp.MustCompile(`\p{L}+`)

var commonWords = func() map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields(`a am an and any are as at be but by can did do for from get go had has have he her him his how i if in is it its let me my no not now of off oh on one or our out say see she so than that the them then they this to too two up us use was we were what when who why will with yes yet you your all new old day way may note none once same such take tell time very here just like long make many much need only over also back more most must some into been`) {
		m[w] = true
	}
	return m
}()

func hasVowel(s string) bool {
	return strings.ContainsAny(strings.ToLower(s), "aeiouy")
}

func isUpperASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 'A' || s[i] > 'Z' {
			return false
		}
	}
	return s != ""
}

// wordsPass resolves acronyms and pronunciation overrides token by token.
func (c *Cleaner) wordsPass(s string, allCaps bool) string {
	return replaceIdx(s, reLetters, func(s string, m []int) (string, bool) {
		tok := s[m[0]:m[1]]
		next := byte(0)
		if m[1] < len(s) {
			next = s[m[1]]
		}
		if a, ok := c.acr[tok]; ok {
			if v, ok := c.resolve(a, tok); ok {
				return v, true
			}
			return "", false
		}
		// Plural acronyms: "APIs" -> "A P I's".
		if n := len(tok); n > 2 && tok[n-1] == 's' && isUpperASCII(tok[:n-1]) {
			if a, ok := c.acr[tok[:n-1]]; ok {
				if v, ok := c.resolve(a, tok[:n-1]); ok {
					if i := strings.LastIndexByte(v, ' '); i >= 0 && len(v)-i-1 == 1 {
						return v + "'s", true
					}
					return v + "s", true
				}
			}
		}
		if v, ok := c.words[strings.ToLower(tok)]; ok {
			return v, true
		}
		if !isUpperASCII(tok) || len(tok) < 2 {
			return "", false
		}
		if allCaps {
			return strings.ToLower(tok), true
		}
		if len(tok) > 6 || c.opt.Acronyms == "keep" || commonWords[strings.ToLower(tok)] {
			return "", false
		}
		if next == '\'' && m[1]+1 < len(s) && unicode.IsLetter(rune(s[m[1]+1])) {
			return "", false // contraction such as DON'T
		}
		if len(tok) >= 4 && hasVowel(tok) {
			return "", false // word-like (NASA, NATO)
		}
		return spellLetters(tok), true
	})
}
