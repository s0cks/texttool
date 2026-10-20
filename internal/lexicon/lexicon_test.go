package lexicon

import "testing"

func TestBuiltinParses(t *testing.T) {
	l := Builtin()
	l.Normalize()
	if l.Acronyms["JSON"].Say != "jason" {
		t.Errorf("JSON say = %q", l.Acronyms["JSON"].Say)
	}

	if l.Acronyms["CLI"].Expand != "command line interface" {
		t.Errorf("CLI expand = %q", l.Acronyms["CLI"].Expand)
	}

	if l.Words["c++"] != "C plus plus" {
		t.Errorf("c++ = %q", l.Words["c++"])
	}

	found := false
	for _, a := range l.Abbreviations() {
		if a == "e.g" {
			found = true
		}
	}

	if !found {
		t.Error("e.g. missing from abbreviations")
	}
}

func TestShorthandAndMerge(t *testing.T) {
	l := Builtin()
	u, err := parseYaml([]byte("acronyms:\n  CLI: command line tool\n  FOO: {spell: eff oh oh}\nhandles:\n  \"@S0cks\": at socks\n"))
	if err != nil {
		t.Fatal(err)
	}

	l.Merge(u)
	if l.Acronyms["CLI"].Say != "command line tool" {
		t.Errorf("shorthand not applied: %+v", l.Acronyms["CLI"])
	}

	if l.Acronyms["FOO"].Spell != "eff oh oh" {
		t.Errorf("FOO = %+v", l.Acronyms["FOO"])
	}

	if l.Handles["@s0cks"] != "at socks" {
		t.Errorf("handles = %v", l.Handles)
	}
}
