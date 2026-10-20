package tts

import (
	"strconv"
	"strings"
)

var ones = []string{"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine",
	"ten", "eleven", "twelve", "thirteen", "fourteen", "fifteen", "sixteen", "seventeen", "eighteen", "nineteen"}

var tens = []string{"", "", "twenty", "thirty", "forty", "fifty", "sixty", "seventy", "eighty", "ninety"}

var scales = []struct {
	n    int64
	name string
}{
	{1_000_000_000_000, "trillion"},
	{1_000_000_000, "billion"},
	{1_000_000, "million"},
	{1_000, "thousand"},
}

// cardinal spells out n in American English ("one hundred twenty-three").
func cardinal(n int64) string {
	switch {
	case n < 0:
		return "minus " + cardinal(-n)

	case n < 20:
		return ones[n]

	case n < 100:
		if n%10 == 0 {
			return tens[n/10]
		}

		return tens[n/10] + "-" + ones[n%10]

	case n < 1000:
		s := ones[n/100] + " hundred"
		if r := n % 100; r > 0 {
			s += " " + cardinal(r)
		}

		return s

	}

	for _, sc := range scales {
		if n >= sc.n {
			s := cardinal(n/sc.n) + " " + sc.name
			if r := n % sc.n; r > 0 {
				s += " " + cardinal(r)
			}

			return s
		}
	}

	return strconv.FormatInt(n, 10)
}

var irregularOrdinal = map[string]string{
	"one": "first", "two": "second", "three": "third", "five": "fifth",
	"eight": "eighth", "nine": "ninth", "twelve": "twelfth",
}

// ordinal spells out the ordinal form of n ("twenty-first").
func ordinal(n int64) string {
	c := cardinal(n)
	i := strings.LastIndexAny(c, " -")
	head, last := c[:i+1], c[i+1:]
	if o, ok := irregularOrdinal[last]; ok {
		return head + o
	}

	if strings.HasSuffix(last, "y") {
		return head + last[:len(last)-1] + "ieth"
	}

	return head + last + "th"
}

// digitsWords reads a digit string one digit at a time ("zero zero seven").
func digitsWords(d string) string {
	out := make([]string, 0, len(d))
	for _, r := range d {
		if r >= '0' && r <= '9' {
			out = append(out, ones[r-'0'])
		}
	}

	return strings.Join(out, " ")
}

// intWords reads a digit string as a cardinal. Leading-zero strings and very
// long strings (IDs, phone numbers) are read digit by digit instead.
func intWords(d string) string {
	d = strings.ReplaceAll(d, ",", "")
	if d == "" {
		return ""
	}

	if len(d) > 15 || (len(d) > 1 && d[0] == '0') {
		return digitsWords(d)
	}

	n, err := strconv.ParseInt(d, 10, 64)
	if err != nil {
		return digitsWords(d)
	}

	return cardinal(n)
}

// numberWords reads an integer or decimal ("3.14" -> "three point one four").
func numberWords(s string) string {
	s = strings.ReplaceAll(s, ",", "")
	if before, after, ok := strings.Cut(s, "."); ok {
		whole, frac := before, after
		if whole == "" {
			whole = "0"
		}

		if frac == "" {
			return intWords(whole)
		}

		return intWords(whole) + " point " + digitsWords(frac)
	}

	return intWords(s)
}

// yearWords reads a four-digit year ("1989" -> "nineteen eighty-nine").
func yearWords(y int) string {
	hi, lo := y/100, y%100
	switch {
	case y == 2000:
		return "two thousand"

	case y > 2000 && y < 2010:
		return "two thousand " + ones[lo]

	case lo == 0:
		return cardinal(int64(hi)) + " hundred"

	case lo < 10:
		return cardinal(int64(hi)) + " oh " + ones[lo]

	default:
		return cardinal(int64(hi)) + " " + cardinal(int64(lo))

	}
}

// pluralizeLast pluralizes the final word of a phrase ("ninety" -> "nineties").
func pluralizeLast(s string) string {
	if strings.HasSuffix(s, "y") {
		return s[:len(s)-1] + "ies"
	}

	return s + "s"
}

var months = []string{"", "January", "February", "March", "April", "May", "June", "July",
	"August", "September", "October", "November", "December"}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
