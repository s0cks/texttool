package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runCLI(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	var out, errb bytes.Buffer
	err := run(args, strings.NewReader(stdin), &out, &errb)
	return out.String(), err
}

func TestStdinJSONObject(t *testing.T) {
	out, err := runCLI(t, "# Title\n\nHello world. Second one.\n", "--tts")
	if err != nil {
		t.Fatal(err)
	}
	var d struct {
		Format   string `json:"format"`
		Segments []struct {
			Kind string  `json:"kind"`
			TTS  *string `json:"tts"`
		} `json:"segments"`
	}
	if err := json.Unmarshal([]byte(out), &d); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if d.Format != "md" || len(d.Segments) != 3 || d.Segments[0].Kind != "heading" || d.Segments[0].TTS == nil {
		t.Errorf("unexpected: %+v", d)
	}
}

func TestMultiFileNDJSONCarriesSource(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.md"), filepath.Join(dir, "b.txt")
	os.WriteFile(a, []byte("Alpha sentence.\n"), 0o644)
	os.WriteFile(b, []byte("Beta sentence.\n"), 0o644)
	out, err := runCLI(t, "", "--output", "ndjson", a, b)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 lines, got %d:\n%s", len(lines), out)
	}
	for i, want := range []string{a, b} {
		var r struct {
			Source string `json:"source"`
		}
		json.Unmarshal([]byte(lines[i]), &r)
		if r.Source != want {
			t.Errorf("line %d source = %q, want %q", i, r.Source, want)
		}
	}
}

func TestUserLexiconOverridesBuiltin(t *testing.T) {
	lex := filepath.Join(t.TempDir(), "lex.yaml")
	os.WriteFile(lex, []byte("acronyms:\n  CLI: { expand: command line tool }\nhandles:\n  \"@s0cks\": at socks\n"), 0o644)
	out, err := runCLI(t, "Use the CLI, thanks @s0cks.\n", "--tts", "--acronyms", "expand", "--lexicon", lex, "--output", "ndjson")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Use the command line tool, thanks at socks.") {
		t.Errorf("lexicon not applied:\n%s", out)
	}
}

func TestDumpLexiconAndValidation(t *testing.T) {
	out, err := runCLI(t, "", "--dump-lexicon")
	if err != nil || !strings.Contains(out, "JSON") {
		t.Fatalf("dump failed: %v\n%s", err, out)
	}
	out, err = runCLI(t, "", "--no-builtin-lexicon", "--dump-lexicon")
	if err != nil || strings.Contains(out, "JSON") {
		t.Fatalf("--no-builtin-lexicon still dumped builtin entries: %v", err)
	}
	if _, err := runCLI(t, "x", "--acronyms", "bogus"); err == nil {
		t.Error("expected an error for --acronyms=bogus")
	}
	if _, err := runCLI(t, "x", "--format", "rtf"); err == nil {
		t.Error("expected an error for --format=rtf")
	}
}
