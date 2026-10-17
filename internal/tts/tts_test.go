package tts

import "testing"

func clean(t *testing.T, opt Options, in string) string {
	t.Helper()
	opt.Terminate = true
	return New(opt).CleanText(in)
}

func TestNumbers(t *testing.T) {
	cases := []struct{ in, want string }{
		{"I have 50 cats", "I have fifty cats."},
		{"It was released in 1989.", "It was released in nineteen eighty-nine."},
		{"Back in 2005 and 2024", "Back in two thousand five and twenty twenty-four."},
		{"Founded in 1900 and 1905", "Founded in nineteen hundred and nineteen oh five."},
		{"The 1990s were fun", "The nineteen nineties were fun."},
		{"about 1500 users", "about one thousand five hundred users."},
		{"It costs $5", "It costs five dollars."},
		{"It costs $1.50 today", "It costs one dollar and fifty cents today."},
		{"Raised $1,250,000 total", "Raised one million two hundred fifty thousand dollars total."},
		{"A $5M round", "A five million dollars round."},
		{"50% fewer bugs", "fifty percent fewer bugs."},
		{"the 3rd and 21st", "the third and twenty-first."},
		{"pi is 3.14", "pi is three point one four."},
		{"version v1.2.3 shipped", "version version one point two point three shipped."},
		{"Meet at 3:30 pm", "Meet at three thirty p m."},
		{"Call 555-1234 now", "Call five five five, one two three four now."},
		{"Pages 10-20", "Pages ten to twenty."},
		{"From 2020-2024", "From twenty twenty to twenty twenty-four."},
		{"2 1/2 cups", "two one half cups."},
		{"on 2026-10-16", "on October sixteenth, twenty twenty-six."},
		{"a 5GB file and 10 ms", "a five gigabytes file and ten milliseconds."},
		{"It is -5 degrees", "It is minus five degrees."},
		{"Order #42", "Order number forty-two."},
		{"1,234 items", "one thousand two hundred thirty-four items."},
		{"zip 007", "zip zero zero seven."},
	}
	for _, c := range cases {
		if got := clean(t, Options{}, c.in); got != c.want {
			t.Errorf("%q\n  got  %q\n  want %q", c.in, got, c.want)
		}
	}
}

func TestSymbolsAndPunctuation(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Tom & Jerry", "Tom and Jerry."},
		{"Wait... what?", "Wait, what?"},
		{"Wait... What?", "Wait. What?"},
		{"And then...", "And then."},
		{"This — that", "This, that."},
		{"This -- that", "This, that."},
		{"trailing hyphen -", "trailing hyphen."},
		{"a   b \t c\n\nd", "a b c d."},
		{"Wow!!", "Wow!"},
		{"x >= 5", "x greater than or equal to five."},
		{"a < b", "a less than b."},
		{"Use C++ or C#", "Use C plus plus or C sharp."},
		{"and/or both", "and or both."},
		{"“quoted” text", `"quoted" text.`},
		{"Hello 😀 world", "Hello world."},
		{"see [1] and [2, 3]", "see and."},
		{"foo [^1] bar", "foo bar."},
	}
	for _, c := range cases {
		if got := clean(t, Options{}, c.in); got != c.want {
			t.Errorf("%q\n  got  %q\n  want %q", c.in, got, c.want)
		}
	}
}

func TestAcronymsAndHandles(t *testing.T) {
	cases := []struct {
		opt  Options
		in   string
		want string
	}{
		{Options{}, "Run the CLI now", "Run the C L I now."},
		{Options{Acronyms: "expand"}, "Run the CLI now", "Run the command line interface now."},
		{Options{Acronyms: "keep"}, "Run the CLI now", "Run the CLI now."},
		{Options{}, "Parse the JSON file", "Parse the jason file."},
		{Options{}, "Many APIs exist", "Many A P I's exist."},
		{Options{}, "The XYZ team", "The X Y Z team."},
		{Options{}, "NASA launched it", "NASA launched it."},
		{Options{}, "DO NOT PANIC", "do not panic."},
		{Options{}, "Ping @s0cks today", "Ping at s zero cks today."},
		{Options{Handles: "strip"}, "Ping @s0cks today", "Ping today."},
		{Options{Handles: "keep"}, "Ping @s0cks today", "Ping at s0cks today."},
		{Options{}, "Mail me at tazz@example.com", "Mail me at tazz at example dot com."},
		{Options{}, "e.g. this, i.e. that", "for example this, that is that."},
		{Options{}, "Install nginx and kubectl", "Install engine x and cube control."},
		{Options{}, "See https://github.com/tesserawm/vespa now", "See now."},
		{Options{URLs: "domain"}, "See https://github.com/tesserawm/vespa now", "See github dot com now."},
		{Options{URLs: "spell"}, "See https://github.com/tesserawm/vespa now", "See github dot com slash tesserawm slash vespa now."},
		{Options{}, "Cost is $5 and $10 total", "Cost is five dollars and ten dollars total."},
		{Options{}, "Solve $x^2 = 4$ now", "Solve now."},
	}
	for _, c := range cases {
		if got := clean(t, c.opt, c.in); got != c.want {
			t.Errorf("%q\n  got  %q\n  want %q", c.in, got, c.want)
		}
	}
}

func TestIdempotent(t *testing.T) {
	for _, in := range []string{
		"Run the CLI for $5 in 1989 — it's 50% faster... really!",
		"Ping @s0cks about the API at https://x.dev/a",
		"WARNING: DO NOT TOUCH",
	} {
		c := New(Options{Terminate: true})
		once := c.CleanText(in)
		twice := c.CleanText(once)
		if once != twice {
			t.Errorf("not idempotent:\n  once  %q\n  twice %q", once, twice)
		}
	}
}
