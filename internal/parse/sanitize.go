package parse

import (
	"bytes"
	"regexp"
)

var (
	reFrontKey  = regexp.MustCompile(`^[A-Za-z_][\w-]*\s*:`)
	reDivFence  = regexp.MustCompile(`^\s{0,3}:{3,}`)
	reCodeFence = regexp.MustCompile("^\\s{0,3}(`{3,}|~{3,})")
)

// Sanitize blanks out pandoc-only constructs goldmark would misread: YAML
// metadata blocks and fenced div markers. Bytes are replaced with spaces and
// newlines are kept, so every offset in the result equals the original offset.
func Sanitize(src []byte) []byte {
	out := append([]byte(nil), src...)
	blank := func(a, b int) {
		for i := a; i < b; i++ {
			if out[i] != '\n' && out[i] != '\r' {
				out[i] = ' '
			}
		}
	}
	type line struct{ start, end int } // end excludes the newline
	var lines []line
	for pos := 0; pos < len(src); {
		nl := bytes.IndexByte(src[pos:], '\n')
		if nl < 0 {
			lines = append(lines, line{pos, len(src)})
			break
		}
		lines = append(lines, line{pos, pos + nl})
		pos += nl + 1
	}

	i := 0
	// YAML front matter: "---", a key line, ..., then "---" or "...".
	if len(lines) > 2 && string(bytes.TrimRight(src[lines[0].start:lines[0].end], " \r")) == "---" &&
		reFrontKey.Match(src[lines[1].start:lines[1].end]) {
		for j := 1; j < len(lines) && j < 400; j++ {
			t := string(bytes.TrimRight(src[lines[j].start:lines[j].end], " \r"))
			if t == "---" || t == "..." {
				blank(lines[0].start, lines[j].end)
				i = j + 1
				break
			}
		}
	}

	var fence []byte
	for ; i < len(lines); i++ {
		l := src[lines[i].start:lines[i].end]
		if m := reCodeFence.FindSubmatch(l); m != nil {
			f := m[1]
			switch {
			case fence == nil:
				fence = f
			case f[0] == fence[0] && len(f) >= len(fence):
				fence = nil
			}
			continue
		}
		if fence == nil && reDivFence.Match(l) {
			blank(lines[i].start, lines[i].end)
		}
	}
	return out
}
