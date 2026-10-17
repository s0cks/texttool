package tts

import (
	"regexp"
	"strings"
	"unicode"
)

// operatorReplacer handles multi-character operators before single angle
// brackets and equals signs are read on their own.
var operatorReplacer = strings.NewReplacer(
	"->", " to ", "=>", " to ", "→", " to ", "⇒", " to ",
	"<=", " less than or equal to ", "≤", " less than or equal to ",
	">=", " greater than or equal to ", "≥", " greater than or equal to ",
	"!=", " not equal to ", "≠", " not equal to ",
	"==", " equals ",
)

var symbolReplacer = strings.NewReplacer(
	"=", " equals ",
	"&", " and ", "@", " at ", "+", " plus ", "%", " percent ",
	"/", " slash ", "~", " about ",
	"×", " times ", "÷", " divided by ", "±", " plus or minus ", "≈", " approximately ",
	"∞", " infinity ", "§", " section ", "©", " copyright ", "π", " pi ", "√", " square root of ",
	"µ", " micro ", "°", " degrees ",
	"€", " euros ", "£", " pounds ", "¥", " yen ", "$", " ",
	"®", " ", "™", " ", "¶", " ",
	"•", ", ", "·", ", ", "|", ", ",
	"#", " ", "*", " ", "_", " ", "^", " ", "\\", " ", "`", " ",
	"[", " ", "]", " ", "{", " ", "}", " ",
)

var reAngle = regexp.MustCompile(`[<>]`)

func isEmoji(r rune) bool {
	return (r >= 0x1F000 && r <= 0x1FAFF) || (r >= 0x2600 && r <= 0x27BF) || (r >= 0x2B00 && r <= 0x2BFF) ||
		(r >= 0x2190 && r <= 0x21FF) || (r >= 0xE000 && r <= 0xF8FF)
}

// expandSymbols spells out the symbols a speech model would skip or mangle.
func expandSymbols(s string) string {
	s = operatorReplacer.Replace(s)
	s = replaceIdx(s, reAngle, func(s string, m []int) (string, bool) {
		before := m[0] == 0 || s[m[0]-1] == ' '
		after := m[1] >= len(s) || s[m[1]] == ' '
		if !before && !after {
			return " ", true // an HTML-ish bracket, not a comparison
		}
		if s[m[0]] == '<' {
			return " less than ", true
		}
		return " greater than ", true
	})
	s = symbolReplacer.Replace(s)
	return strings.Map(func(r rune) rune {
		if isEmoji(r) {
			return -1
		}
		return r
	}, s)
}

var (
	reEllipsis    = regexp.MustCompile(`\s*\.{3,}\s*`)
	reEmDash      = regexp.MustCompile(`\s*[—–]+\s*`)
	reSpacedDash  = regexp.MustCompile(`\s+-{1,2}\s+|\s*--+\s*`)
	reTrailHyphen = regexp.MustCompile(`[\s\-–—]+$`)
	reLeadHyphen  = regexp.MustCompile(`^[\s\-–—]+`)
	reHyphenPunct = regexp.MustCompile(`-+\s*([.,;:!?])`)
	reMixedPunct  = regexp.MustCompile(`([,;:])\s*([.!?])`)
	reAfterEnd    = regexp.MustCompile(`([.!?])\s*([,;:])`)
	reRepeat      = regexp.MustCompile(`([,;:!?.])(?:\s*[,;:])+|\.{2,}|!{2,}|\?{2,}`)
)

// standardizePunctuation replaces ellipses, em dashes and stray hyphens with
// explicit commas and periods so phrasing is controlled by plain punctuation.
func standardizePunctuation(s string) string {
	s = replaceIdx(s, reEllipsis, func(s string, m []int) (string, bool) {
		rest := strings.TrimLeft(s[m[1]:], " ")
		switch {
		case rest == "":
			return ".", true
		case unicode.IsLower([]rune(rest)[0]) || rest[0] == ',':
			return ", ", true
		default:
			return ". ", true
		}
	})
	s = reEmDash.ReplaceAllString(s, ", ")
	s = reSpacedDash.ReplaceAllString(s, ", ")
	s = reLeadHyphen.ReplaceAllString(s, "")
	s = reTrailHyphen.ReplaceAllString(s, "")
	s = reHyphenPunct.ReplaceAllString(s, "$1")
	s = reMixedPunct.ReplaceAllString(s, "$2")
	s = reAfterEnd.ReplaceAllString(s, "$1")
	s = reRepeat.ReplaceAllStringFunc(s, func(m string) string {
		return string(m[0])
	})
	return s
}

var (
	reSpaceBefore = regexp.MustCompile(`\s+([,.;:!?])`)
	reNoSpace     = regexp.MustCompile(`([,;:!?])(\p{L})`)
	reLoneQuote   = regexp.MustCompile(`^\s*"\s*$`)
)

// fixSpacing tidies spaces around punctuation after the rewrite.
func fixSpacing(s string) string {
	s = reEmptyPar.ReplaceAllString(s, "")
	s = reSpaceBefore.ReplaceAllString(s, "$1")
	s = reNoSpace.ReplaceAllString(s, "$1 $2")
	s = strings.Join(strings.Fields(s), " ")
	if reLoneQuote.MatchString(s) {
		return ""
	}
	return s
}

// terminate makes sure the text ends with sentence punctuation, which gives
// speech models a clear pause at the end of every segment.
func terminate(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimRight(s, ",;:- ")
	if s == "" {
		return s
	}
	trimmed := strings.TrimRight(s, `)"'`)
	if trimmed == "" {
		return s
	}
	switch trimmed[len(trimmed)-1] {
	case '.', '!', '?':
		return s
	}
	return s + "."
}
