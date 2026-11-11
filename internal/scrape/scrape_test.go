package scrape

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
)

type dictEntry struct {
	Pattern  string `json:"pattern"`
	Word     string `json:"word"`
	Censored string `json:"censored"`
	Censor   *bool  `json:"censor"`
}

func pythonMatcher(t *testing.T, path string) Matcher {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var entries []dictEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		t.Fatal(err)
	}
	groups := make([]string, len(entries))
	for i, e := range entries {
		groups[i] = "(" + e.Pattern + ")"
	}
	rx := regexp.MustCompile(`(?i)\b(?:` + strings.Join(groups, "|") + `)\b`)
	return func(text string) []string {
		var out []string
		for _, loc := range rx.FindAllStringSubmatchIndex(text, -1) {
			word := strings.ToLower(text[loc[0]:loc[1]])
			for i, e := range entries {
				if loc[2*i+2] < 0 {
					continue
				}
				if e.Censor == nil || *e.Censor {
					word = regexp.MustCompile(`(?i)`+regexp.QuoteMeta(e.Word)).ReplaceAllLiteralString(word, e.Censored)
				}
				break
			}
			out = append(out, word)
		}
		return out
	}
}

func TestParityWithPython(t *testing.T) {
	loc, err := time.LoadLocation("America/Detroit")
	if err != nil {
		t.Fatal(err)
	}
	saved := time.Local
	time.Local = loc
	t.Cleanup(func() { time.Local = saved })

	root, err := filepath.Abs("testdata/parity/projects")
	if err != nil {
		t.Fatal(err)
	}
	res, err := Scrape(root, pythonMatcher(t, "testdata/parity/dictionary.json"))
	if err != nil {
		t.Fatal(err)
	}
	gotJSON, err := json.Marshal(res)
	if err != nil {
		t.Fatal(err)
	}
	wantJSON, err := os.ReadFile("testdata/parity/golden.json")
	if err != nil {
		t.Fatal(err)
	}
	var got, want any
	if err := json.Unmarshal(gotJSON, &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(wantJSON, &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Go scrape differs from Python golden\n got: %s\nwant: %s", gotJSON, wantJSON)
	}
}

func writeLines(t *testing.T, path string, lines ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScrapeFilters(t *testing.T) {
	saved := time.Local
	time.Local = time.UTC
	t.Cleanup(func() { time.Local = saved })

	root := t.TempDir()
	msg := func(fields, content string) string {
		return `{"type":"user","timestamp":"2026-11-10T12:00:00Z",` + fields + `"message":{"content":` + content + `}}`
	}
	writeLines(t, filepath.Join(root, "p", "a.jsonl"),
		msg(`"uuid":"1","sessionId":"s1",`, `"damn one"`),
		msg(`"uuid":"1","sessionId":"s1",`, `"damn duplicate uuid"`),
		msg(`"uuid":"2","sessionId":"s1","isMeta":true,`, `"damn meta"`),
		msg(`"uuid":"3","sessionId":"s1","isSidechain":true,`, `"damn sidechain"`),
		`{"type":"assistant","uuid":"4","sessionId":"s1","timestamp":"2026-11-10T12:00:00Z","message":{"content":"damn assistant"}}`,
		msg(`"uuid":"5","sessionId":"s2",`, `[{"type":"text","text":"damn"},{"type":"tool_result","text":"damn tool"},{"type":"text","text":"blocks"}]`),
		msg(`"sessionId":"s2",`, `"damn no uuid"`),
		msg(`"sessionId":"s2",`, `"damn no uuid again"`),
		msg(`"uuid":"6",`, `"damn no session"`),
		msg(`"uuid":"7","sessionId":"s3",`, `"<system-reminder>damn</system-reminder>clean"`),
		msg(`"uuid":"8","sessionId":"s3",`, `"   "`),
		`not json`,
		`{"type":"user","uuid":"9","sessionId":"s3","message":{"content":"damn no timestamp"}}`,
	)
	writeLines(t, filepath.Join(root, "p", "history.jsonl"), msg(`"uuid":"h","sessionId":"h",`, `"damn history"`))
	writeLines(t, filepath.Join(root, "node_modules", "x.jsonl"), msg(`"uuid":"n","sessionId":"n",`, `"damn pruned"`))
	writeLines(t, filepath.Join(root, "p", "notes.txt"), msg(`"uuid":"t","sessionId":"t",`, `"damn txt"`))

	match := func(text string) []string {
		var out []string
		for _, w := range strings.Fields(text) {
			if w == "damn" {
				out = append(out, w)
			}
		}
		return out
	}
	res, err := Scrape(root, match)
	if err != nil {
		t.Fatal(err)
	}
	want := Result{Files: 1, Sessions: 4, Messages: 6, Days: map[string]Day{
		"2026-11-10": {N: 5, Msgs: 6, Words: map[string]int{"damn": 5}},
	}}
	if !reflect.DeepEqual(res, want) {
		t.Errorf("Scrape =\n%+v\nwant\n%+v", res, want)
	}
}

func TestDefaultRoot(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", "/custom/config")
	if got, err := DefaultRoot(); err != nil || got != filepath.Join("/custom/config", "projects") {
		t.Errorf("with CLAUDE_CONFIG_DIR: DefaultRoot() = %q, %v", got, err)
	}
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if got, err := DefaultRoot(); err != nil || got != filepath.Join(home, ".claude", "projects") {
		t.Errorf("without CLAUDE_CONFIG_DIR: DefaultRoot() = %q, %v", got, err)
	}
}

func TestStripHarness(t *testing.T) {
	cases := map[string]string{
		"a<system-reminder>x</system-reminder>b":                          "a b",
		"<command-name>/c</command-name><command-args>y</command-args>z":  "  z",
		"<pasted_content id=\"1\">p</pasted_content>q":                    " q",
		"<system-reminder>unclosed":                                       "<system-reminder>unclosed",
		"<command-name>a</command-args>":                                  "<command-name>a</command-args>",
		"plain":                                                           "plain",
		"<pasted_content id=\"be5b\">fuck</pasted_content id=\"be5b\">ok": " ok",
		"<pasted_content id=\"1\">a</pasted_contentx>b</pasted_content>c": " c",
		"<system-reminder>a</system-reminder\n>b":                         " b",
	}
	for in, want := range cases {
		if got := stripHarness(in); got != want {
			t.Errorf("stripHarness(%q) = %q, want %q", in, got, want)
		}
	}
}
