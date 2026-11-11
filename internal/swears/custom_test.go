package swears

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestExampleFile(t *testing.T) {
	data, err := os.ReadFile("../../.swears.example.json")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := ParseCustom(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Censored != "" {
			t.Errorf("%q: example file should not set censored", e.Word)
		}
	}
	if got := New(entries).Match("a bobcat chased a bunnyhop past the otters"); !reflect.DeepEqual(got, []string{"bobcat", "bunnyhop"}) {
		t.Errorf("Match = %q", got)
	}
}

func TestParseCustom(t *testing.T) {
	got, err := ParseCustom([]byte(`["gaslight", {"word": "jesus christ"}, {"word": "frak", "censored": "fr@k"}]`))
	if err != nil {
		t.Fatal(err)
	}
	want := []Entry{
		{Word: "gaslight", Suffix: true},
		{Word: "jesus christ", Suffix: true},
		{Word: "frak", Suffix: true, Censored: "fr@k"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ParseCustom = %+v\nwant %+v", got, want)
	}
	if got, err := ParseCustom([]byte(`[]`)); err != nil || len(got) != 0 {
		t.Errorf("empty array = %v, %v", got, err)
	}
}

func TestParseCustomErrors(t *testing.T) {
	for _, in := range []string{
		``,
		`null`,
		`{"word": "frak"}`,
		`"frak"`,
		`["frak",]`,
		`[""]`,
		`["  "]`,
		`[null]`,
		`[42]`,
		`[{}]`,
		`[{"word": ""}]`,
		`[{"word": 42}]`,
		`[{"word": "frak", "censored": ""}]`,
		`[{"word": "frak", "censored": 1}]`,
		`[{"word": "frak", "pattern": "frak\\w*"}]`,
		`[{"word": "frak", "prefix": "yes"}]`,
		`[{"word": "frak", "suffix": 1}]`,
		`[{"word": "frak", "censor": false}]`,
	} {
		if got, err := ParseCustom([]byte(in)); err == nil {
			t.Errorf("ParseCustom(%q) = %+v, want error", in, got)
		}
	}
}

func writeCustom(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".swears.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadCustom(t *testing.T) {
	cwd, home := t.TempDir(), t.TempDir()

	entries, path, err := LoadCustom(cwd, home)
	if err != nil || entries != nil || path != "" {
		t.Errorf("neither file: %v, %q, %v", entries, path, err)
	}

	writeCustom(t, home, `["fromhome"]`)
	entries, path, err = LoadCustom(cwd, home)
	if err != nil || len(entries) != 1 || entries[0].Word != "fromhome" || path != filepath.Join(home, ".swears.json") {
		t.Errorf("home only: %v, %q, %v", entries, path, err)
	}

	writeCustom(t, cwd, `["fromcwd"]`)
	entries, path, err = LoadCustom(cwd, home)
	if err != nil || len(entries) != 1 || entries[0].Word != "fromcwd" || path != filepath.Join(cwd, ".swears.json") {
		t.Errorf("both files, cwd wins: %v, %q, %v", entries, path, err)
	}

	writeCustom(t, cwd, `{"broken": true}`)
	_, _, err = LoadCustom(cwd, home)
	if err == nil || !strings.Contains(err.Error(), filepath.Join(cwd, ".swears.json")) {
		t.Errorf("malformed cwd file: err = %v, want error naming the file", err)
	}
}

func TestCustomOptions(t *testing.T) {
	custom, err := ParseCustom([]byte(`[
		"bunny",
		{"word": "cat", "prefix": true, "suffix": false},
		{"word": "otter", "suffix": false},
		{"word": "wombat", "censored": "wombat"},
		{"word": "hedgehog", "censored": "h*dg*h*g"},
		"red panda"
	]`))
	if err != nil {
		t.Fatal(err)
	}
	l := New(custom)
	got := l.Match("Bunnies? No: bunnyhop, bunny. Bobcat, cat, category, cats. Otter, otters. Wombats, hedgehogs, Red Panda.")
	want := []string{"bunnyhop", "bunny", "bobcat", "cat", "otter", "wombats", "hedgehogs", "red panda"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Match = %q\nwant %q", got, want)
	}
	for word, want := range map[string]string{
		"bunnyhop":  "b*nnyhop",
		"bobcat":    "bobc*t",
		"otter":     "*tter",
		"wombats":   "wombats",
		"hedgehogs": "h*dg*h*gs",
		"red panda": "r*d panda",
	} {
		if got := l.Censor(word); got != want {
			t.Errorf("Censor(%q) = %q, want %q", word, got, want)
		}
	}
}

func TestCustomMatching(t *testing.T) {
	custom, err := ParseCustom([]byte(`["gaslight", "jesus christ", {"word": "fuck", "censored": "f**k"}]`))
	if err != nil {
		t.Fatal(err)
	}
	l := New(append(custom, Builtin()...))
	got := l.Match("Jesus Christ, stop gaslighting me, fucking hell")
	want := []string{"jesus christ", "gaslighting", "fucking", "hell"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Match = %q, want %q", got, want)
	}
	for word, want := range map[string]string{
		"gaslighting":  "g*slighting",
		"jesus christ": "j*sus christ",
		"fucking":      "f**king",
		"hell":         "h*ll",
	} {
		if got := l.Censor(word); got != want {
			t.Errorf("Censor(%q) = %q, want %q", word, got, want)
		}
	}
}
