package tts

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// replaceIdx rewrites every match of re using fn, which sees the whole string
// and the match's submatch indices (for context checks). Returning ok=false
// leaves the match untouched.
func replaceIdx(s string, re *regexp.Regexp, fn func(s string, m []int) (string, bool)) string {
	ms := re.FindAllStringSubmatchIndex(s, -1)
	if len(ms) == 0 {
		return s
	}
	var b strings.Builder
	last := 0
	for _, m := range ms {
		rep, ok := fn(s, m)
		if !ok {
			continue
		}
		b.WriteString(s[last:m[0]])
		b.WriteString(rep)
		last = m[1]
	}
	b.WriteString(s[last:])
	return b.String()
}

// group returns submatch i of match m, or "" if it did not participate.
func group(s string, m []int, i int) string {
	if m[2*i] < 0 {
		return ""
	}
	return s[m[2*i]:m[2*i+1]]
}

func isWordByte(b byte) bool {
	return b == '_' || (b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b >= 0x80
}

type unit struct{ one, many string }

var units = map[string]unit{
	"KB": {"kilobyte", "kilobytes"}, "MB": {"megabyte", "megabytes"}, "GB": {"gigabyte", "gigabytes"},
	"TB": {"terabyte", "terabytes"}, "PB": {"petabyte", "petabytes"},
	"kb": {"kilobyte", "kilobytes"}, "mb": {"megabyte", "megabytes"}, "gb": {"gigabyte", "gigabytes"},
	"tb":  {"terabyte", "terabytes"},
	"KiB": {"kibibyte", "kibibytes"}, "MiB": {"mebibyte", "mebibytes"}, "GiB": {"gibibyte", "gibibytes"},
	"ms": {"millisecond", "milliseconds"}, "kg": {"kilogram", "kilograms"}, "mg": {"milligram", "milligrams"},
	"km": {"kilometer", "kilometers"}, "cm": {"centimeter", "centimeters"}, "mm": {"millimeter", "millimeters"},
	"lb": {"pound", "pounds"}, "lbs": {"pounds", "pounds"}, "oz": {"ounce", "ounces"},
	"ft": {"foot", "feet"}, "px": {"pixel", "pixels"}, "dB": {"decibel", "decibels"},
	"Hz": {"hertz", "hertz"}, "kHz": {"kilohertz", "kilohertz"}, "MHz": {"megahertz", "megahertz"},
	"GHz": {"gigahertz", "gigahertz"}, "kW": {"kilowatt", "kilowatts"},
	"fps": {"frames per second", "frames per second"}, "mph": {"miles per hour", "miles per hour"},
	"rpm": {"revolutions per minute", "revolutions per minute"},
}

type currency struct{ one, many, minorOne, minorMany string }

var currencies = map[string]currency{
	"$": {"dollar", "dollars", "cent", "cents"},
	"€": {"euro", "euros", "cent", "cents"},
	"£": {"pound", "pounds", "penny", "pence"},
	"¥": {"yen", "yen", "", ""},
}

var magnitudes = map[string]string{
	"k": "thousand", "K": "thousand", "M": "million", "B": "billion",
	"thousand": "thousand", "million": "million", "billion": "billion", "trillion": "trillion",
}

var (
	reISODate   = regexp.MustCompile(`\b(\d{4})-(\d{2})-(\d{2})\b`)
	reVersion   = regexp.MustCompile(`\b([vV]?)(\d+(?:\.\d+){2,})\b`)
	reTimeAP    = regexp.MustCompile(`\b(1[0-2]|0?[1-9])(?::([0-5]\d))?\s?([AaPp])\.?[Mm]\b\.?`)
	reTime24    = regexp.MustCompile(`\b([01]?\d|2[0-3]):([0-5]\d)\b`)
	reCurrency  = regexp.MustCompile(`([$€£¥])\s?(\d{1,3}(?:,\d{3})+|\d+)(\.\d+)?(?:\s?(thousand|million|billion|trillion|[kKMB])\b)?`)
	rePercent   = regexp.MustCompile(`(\d[\d,]*(?:\.\d+)?)\s?%`)
	reUnit      = regexp.MustCompile(`(\d[\d,]*(?:\.\d+)?)\s?(` + unitAlternation() + `)\b`)
	reMagnitude = regexp.MustCompile(`\b(\d+(?:\.\d+)?)\s?([kK]|[MB])\b`)
	reOrdinal   = regexp.MustCompile(`\b(\d+)(?:st|nd|rd|th)\b`)
	reDecade    = regexp.MustCompile(`\b(1[1-9]\d0|20\d0)'?s\b`)
	reChain     = regexp.MustCompile(`\b\d+(?:[-–]\d+)+\b`)
	reSlashNum  = regexp.MustCompile(`\b\d+(?:/\d+)+\b`)
	reDegree    = regexp.MustCompile(`(\d)\s?°\s?([CcFf])?\b`)
	reNumberSgn = regexp.MustCompile(`#(\d+)\b`)
	reNumPlus   = regexp.MustCompile(`(\d)\+`)
	reDecimal   = regexp.MustCompile(`\b(\d[\d,]*)\.(\d+)\b`)
	reYear      = regexp.MustCompile(`\b(1[1-9]\d\d|20\d\d)\b`)
	reNegative  = regexp.MustCompile(`(^|[\s(\[])[-−](\d)`)
	reCommaInt  = regexp.MustCompile(`\b\d{1,3}(?:,\d{3})+\b`)
	reInt       = regexp.MustCompile(`\b\d+\b`)
)

func unitAlternation() string {
	keys := make([]string, 0, len(units))
	for k := range units {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	return strings.Join(keys, "|")
}

var notYearBefore = map[string]bool{
	"about": true, "approximately": true, "around": true, "over": true, "under": true, "than": true,
	"least": true, "most": true, "only": true, "just": true, "nearly": true, "almost": true,
	"roughly": true, "another": true, "every": true, "each": true, "all": true, "these": true,
	"those": true, "other": true, "some": true, "many": true, "few": true, "several": true,
	"total": true, "exactly": true, "top": true, "first": true, "last": true, "next": true,
}

var notYearAfter = map[string]bool{
	"people": true, "users": true, "words": true, "bytes": true, "items": true, "times": true,
	"years": true, "days": true, "hours": true, "minutes": true, "seconds": true, "dollars": true,
	"euros": true, "units": true, "steps": true, "lines": true, "files": true, "rows": true,
	"cells": true, "pages": true, "requests": true, "calls": true, "tokens": true, "characters": true,
	"samples": true, "points": true, "miles": true, "meters": true, "feet": true, "kilometers": true,
	"percent": true, "cents": true, "calories": true, "customers": true, "employees": true,
	"members": true, "bits": true, "pixels": true, "hits": true, "visits": true, "downloads": true,
	"stars": true, "commits": true, "records": true, "messages": true, "events": true, "errors": true,
}

func wordBefore(s string, i int) string {
	j := i
	for j > 0 && s[j-1] == ' ' {
		j--
	}
	k := j
	for k > 0 && ((s[k-1] >= 'a' && s[k-1] <= 'z') || (s[k-1] >= 'A' && s[k-1] <= 'Z')) {
		k--
	}
	return strings.ToLower(s[k:j])
}

func wordAfter(s string, i int) string {
	j := i
	for j < len(s) && s[j] == ' ' {
		j++
	}
	k := j
	for k < len(s) && ((s[k] >= 'a' && s[k] <= 'z') || (s[k] >= 'A' && s[k] <= 'Z')) {
		k++
	}
	return strings.ToLower(s[j:k])
}

func plural(amount string, u unit) string {
	if amount == "1" {
		return u.one
	}
	return u.many
}

// expandNumbers rewrites digits, currency, percentages, units, dates, times
// and versions as words. The passes run in an order where earlier passes
// consume the notation later passes would misread.
func expandNumbers(s string) string {
	s = reISODate.ReplaceAllStringFunc(s, func(m string) string {
		p := reISODate.FindStringSubmatch(m)
		mo := atoi(p[2])
		if mo < 1 || mo > 12 || atoi(p[3]) < 1 || atoi(p[3]) > 31 {
			return m
		}
		return months[mo] + " " + ordinal(int64(atoi(p[3]))) + ", " + p[1]
	})
	s = reVersion.ReplaceAllStringFunc(s, func(m string) string {
		p := reVersion.FindStringSubmatch(m)
		parts := strings.Split(p[2], ".")
		for i, x := range parts {
			parts[i] = intWords(x)
		}
		out := strings.Join(parts, " point ")
		if p[1] != "" {
			out = "version " + out
		}
		return out
	})
	s = reTimeAP.ReplaceAllStringFunc(s, func(m string) string {
		p := reTimeAP.FindStringSubmatch(m)
		out := cardinal(int64(atoi(p[1])))
		if p[2] != "" && p[2] != "00" {
			if p[2][0] == '0' {
				out += " oh " + ones[p[2][1]-'0']
			} else {
				out += " " + cardinal(int64(atoi(p[2])))
			}
		}
		return out + " " + strings.ToLower(p[3]) + " m"
	})
	s = reTime24.ReplaceAllStringFunc(s, func(m string) string {
		p := reTime24.FindStringSubmatch(m)
		h, mi := atoi(p[1]), atoi(p[2])
		out := cardinal(int64(h))
		switch {
		case mi == 0:
			return out + " o'clock"
		case mi < 10:
			return out + " oh " + ones[mi]
		default:
			return out + " " + cardinal(int64(mi))
		}
	})
	s = reCurrency.ReplaceAllStringFunc(s, func(m string) string {
		p := reCurrency.FindStringSubmatch(m)
		cur := currencies[p[1]]
		whole := strings.ReplaceAll(p[2], ",", "")
		frac := strings.TrimPrefix(p[3], ".")
		if mag := magnitudes[p[4]]; mag != "" {
			amount := whole
			if frac != "" {
				amount += "." + frac
			}
			return numberWords(amount) + " " + mag + " " + cur.many
		}
		unitWord := cur.many
		if whole == "1" {
			unitWord = cur.one
		}
		if frac == "" || cur.minorOne == "" {
			if frac != "" {
				return numberWords(whole+"."+frac) + " " + cur.many
			}
			return intWords(whole) + " " + unitWord
		}
		if len(frac) == 1 {
			frac += "0"
		}
		if len(frac) > 2 {
			return numberWords(whole+"."+frac) + " " + cur.many
		}
		cents := atoi(frac)
		centWord := cur.minorMany
		if cents == 1 {
			centWord = cur.minorOne
		}
		w := intWords(whole)
		if whole == "0" {
			return cardinal(int64(cents)) + " " + centWord
		}
		if cents == 0 {
			return w + " " + unitWord
		}
		return w + " " + unitWord + " and " + cardinal(int64(cents)) + " " + centWord
	})
	s = rePercent.ReplaceAllStringFunc(s, func(m string) string {
		p := rePercent.FindStringSubmatch(m)
		return numberWords(p[1]) + " percent"
	})
	s = reUnit.ReplaceAllStringFunc(s, func(m string) string {
		p := reUnit.FindStringSubmatch(m)
		return numberWords(p[1]) + " " + plural(strings.ReplaceAll(p[1], ",", ""), units[p[2]])
	})
	s = reMagnitude.ReplaceAllStringFunc(s, func(m string) string {
		p := reMagnitude.FindStringSubmatch(m)
		return numberWords(p[1]) + " " + magnitudes[p[2]]
	})
	s = reOrdinal.ReplaceAllStringFunc(s, func(m string) string {
		p := reOrdinal.FindStringSubmatch(m)
		n, err := strconv.ParseInt(p[1], 10, 64)
		if err != nil || len(p[1]) > 12 {
			return m
		}
		return ordinal(n)
	})
	s = reDecade.ReplaceAllStringFunc(s, func(m string) string {
		p := reDecade.FindStringSubmatch(m)
		return pluralizeLast(yearWords(atoi(p[1])))
	})
	s = reChain.ReplaceAllStringFunc(s, chainWords)
	s = reSlashNum.ReplaceAllStringFunc(s, slashWords)
	s = reDegree.ReplaceAllStringFunc(s, func(m string) string {
		p := reDegree.FindStringSubmatch(m)
		switch strings.ToLower(p[2]) {
		case "c":
			return p[1] + " degrees Celsius"
		case "f":
			return p[1] + " degrees Fahrenheit"
		}
		return p[1] + " degrees"
	})
	s = reNumberSgn.ReplaceAllString(s, "number $1")
	s = reNumPlus.ReplaceAllString(s, "$1 plus")
	s = reDecimal.ReplaceAllStringFunc(s, func(m string) string {
		p := reDecimal.FindStringSubmatch(m)
		return numberWords(p[1] + "." + p[2])
	})
	s = replaceIdx(s, reYear, func(s string, m []int) (string, bool) {
		if notYearBefore[wordBefore(s, m[0])] || notYearAfter[wordAfter(s, m[1])] {
			return "", false
		}
		if m[0] > 0 && s[m[0]-1] == ',' || m[1] < len(s) && s[m[1]] == ',' && m[1]+1 < len(s) && s[m[1]+1] >= '0' && s[m[1]+1] <= '9' {
			return "", false
		}
		return yearWords(atoi(s[m[0]:m[1]])), true
	})
	s = reNegative.ReplaceAllString(s, "${1}minus ${2}")
	s = reCommaInt.ReplaceAllStringFunc(s, intWords)
	s = reInt.ReplaceAllStringFunc(s, intWords)
	return s
}

// chainWords handles hyphenated digit chains: ranges, phone numbers, IDs.
func chainWords(m string) string {
	parts := strings.FieldsFunc(m, func(r rune) bool { return r == '-' || r == '–' })
	if len(parts) == 2 && len(parts[0]) <= 4 && len(parts[1]) <= 4 &&
		!(len(parts[0]) == 3 && len(parts[1]) == 4) && atoi(parts[0]) < atoi(parts[1]) {
		return parts[0] + " to " + parts[1] // digits stay so the year pass can still read them
	}
	words := make([]string, len(parts))
	phone := len(parts) >= 2
	for i, p := range parts {
		words[i] = digitsWords(p)
		if len(p) < 2 || len(p) > 4 {
			phone = false
		}
	}
	if phone {
		return strings.Join(words, ", ")
	}
	for i, p := range parts {
		words[i] = intWords(p)
	}
	return strings.Join(words, " dash ")
}

var fractions = map[string]string{
	"1/2": "one half", "1/3": "one third", "2/3": "two thirds", "1/4": "one quarter",
	"3/4": "three quarters", "1/8": "one eighth", "1/5": "one fifth",
}

// slashWords handles digit/digit: fractions, otherwise "x slash y" reads.
func slashWords(m string) string {
	if f, ok := fractions[m]; ok {
		return f
	}
	parts := strings.Split(m, "/")
	if len(parts) == 2 {
		return intWords(parts[0]) + " over " + intWords(parts[1])
	}
	for i, p := range parts {
		parts[i] = intWords(p)
	}
	return strings.Join(parts, " slash ")
}
