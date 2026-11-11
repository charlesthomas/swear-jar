package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charlesthomas/swear-jar/internal/scrape"
	"github.com/charlesthomas/swear-jar/internal/swears"
)

var now = time.Date(2026, 11, 11, 18, 0, 0, 0, time.UTC)

func fixtureRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "p")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	line := `{"type":"user","uuid":"u1","sessionId":"s1","timestamp":"2026-11-11T15:00:00Z","message":{"content":"well damn"}}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "s.jsonl"), []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func runArgs(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(args, fixtureRoot(t), swears.New(swears.Builtin()), now, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func decode(t *testing.T, raw []byte) scrape.Result {
	t.Helper()
	var res scrape.Result
	if err := json.Unmarshal(raw, &res); err != nil {
		t.Fatalf("invalid JSON %q: %v", raw, err)
	}
	return res
}

func checkResult(t *testing.T, res scrape.Result) {
	t.Helper()
	if res.Files != 1 || res.Sessions != 1 || res.Messages != 1 || len(res.Days) != 1 {
		t.Fatalf("result = %+v", res)
	}
	for _, d := range res.Days {
		if d.N != 1 || d.Words["damn"] != 1 {
			t.Errorf("day = %+v", d)
		}
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestJSONToStdout(t *testing.T) {
	code, stdout, stderr := runArgs(t, "--json", "-")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	checkResult(t, decode(t, []byte(stdout)))
}

func TestJSONToFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	code, stdout, stderr := runArgs(t, "--json", path)
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	checkResult(t, decode(t, []byte(readFile(t, path))))
}

func TestHTMLWordlistModes(t *testing.T) {
	for mode, check := range map[string]func(string) bool{
		"": func(out string) bool {
			return !strings.Contains(out, "Vocabulary") && !strings.Contains(out, "d*mn") && !strings.Contains(out, ">damn")
		},
		"none": func(out string) bool {
			return !strings.Contains(out, "Vocabulary") && !strings.Contains(out, "d*mn") && !strings.Contains(out, ">damn")
		},
		"censored": func(out string) bool {
			return strings.Contains(out, "<span>d*mn</span>") && !strings.Contains(out, ">damn")
		},
		"uncensored": func(out string) bool {
			return strings.Contains(out, "<span>damn</span>") && !strings.Contains(out, "d*mn")
		},
	} {
		args := []string{"--html", "-"}
		if mode != "" {
			args = append(args, "--wordlist", mode)
		}
		code, stdout, stderr := runArgs(t, args...)
		if code != 0 {
			t.Fatalf("%v: exit %d, stderr %q", args, code, stderr)
		}
		if !strings.Contains(stdout, "<h1>Swear Jar</h1>") || !strings.Contains(stdout, "1 swear in the last year") {
			t.Errorf("%v: page missing heading or total", args)
		}
		if !check(stdout) {
			t.Errorf("%v: wrong word list in page", args)
		}
	}
}

func TestHTMLAndJSONTogether(t *testing.T) {
	dir := t.TempDir()
	htmlPath, jsonPath := filepath.Join(dir, "index.html"), filepath.Join(dir, "data.json")
	code, _, stderr := runArgs(t, "--html", htmlPath, "--json", jsonPath, "--wordlist", "censored")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	if !strings.Contains(readFile(t, htmlPath), "<span>d*mn</span>") {
		t.Error("html file missing censored word")
	}
	checkResult(t, decode(t, []byte(readFile(t, jsonPath))))
}

func TestSVG(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "out")
	htmlPath := filepath.Join(dir, "index.html")
	code, stdout, stderr := runArgs(t, "--svg", dir, "--html", htmlPath)
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	for _, name := range []string{"heatmap-light.svg", "heatmap-dark.svg"} {
		out := readFile(t, filepath.Join(dir, name))
		if !strings.HasPrefix(out, "<svg ") || !strings.Contains(out, "1 swear in the last year") {
			t.Errorf("%s: unexpected content", name)
		}
		if strings.Contains(out, "damn") || strings.Contains(out, "d*mn") {
			t.Errorf("%s: contains swear text", name)
		}
	}
	if !strings.Contains(readFile(t, htmlPath), "<h1>Swear Jar</h1>") {
		t.Error("html not written alongside svg")
	}
}

func TestProfileReadme(t *testing.T) {
	dir := t.TempDir()
	code, stdout, stderr := runArgs(t, "--svg", dir, "--profile-readme")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	snippet := readFile(t, filepath.Join(dir, "swear-jar.md"))
	if !strings.Contains(snippet, `srcset="./heatmap-dark.svg"`) || strings.Contains(snippet, "damn") {
		t.Errorf("snippet = %q", snippet)
	}
	for _, name := range []string{"heatmap-light.svg", "heatmap-dark.svg", "swear-jar.md"} {
		if !strings.Contains(stdout, "  "+filepath.Join(dir, name)+"\n") {
			t.Errorf("instructions missing path for %s:\n%s", name, stdout)
		}
	}
}

func TestVersion(t *testing.T) {
	code, stdout, stderr := runArgs(t, "--version")
	if code != 0 || stdout != "swear-jar dev\n" || stderr != "" {
		t.Errorf("exit %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

func TestWordList(t *testing.T) {
	cwd, home := t.TempDir(), t.TempDir()
	l, err := wordList(cwd, home)
	if err != nil {
		t.Fatal(err)
	}
	if got := l.Match("gaslighting, damn"); len(got) != 1 || got[0] != "damn" {
		t.Errorf("built-ins only: Match = %q", got)
	}

	if err := os.WriteFile(filepath.Join(home, ".swears.json"), []byte(`["gaslight"]`), 0o644); err != nil {
		t.Fatal(err)
	}
	l, err = wordList(cwd, home)
	if err != nil {
		t.Fatal(err)
	}
	if got := l.Match("gaslighting, damn"); len(got) != 2 || got[0] != "gaslighting" || got[1] != "damn" {
		t.Errorf("with custom: Match = %q", got)
	}

	if err := os.WriteFile(filepath.Join(cwd, ".swears.json"), []byte(`{`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := wordList(cwd, home); err == nil {
		t.Error("malformed cwd file: want error")
	}
}

func TestUsageErrors(t *testing.T) {
	for _, tc := range []struct {
		args []string
		msg  string
	}{
		{nil, "nothing to do"},
		{[]string{"extra"}, "unexpected arguments"},
		{[]string{"--bogus"}, "not defined"},
		{[]string{"--wordlist", "censored"}, "nothing to do"},
		{[]string{"--json", "-", "--wordlist", "none"}, "--wordlist only applies to --html"},
		{[]string{"--svg", "out", "--wordlist", "censored"}, "--wordlist only applies to --html"},
		{[]string{"--svg", "-"}, "--svg writes two files"},
		{[]string{"--html", "x.html", "--profile-readme"}, "--profile-readme only applies to --svg"},
		{[]string{"--svg", "out", "--json", "-", "--profile-readme"}, "no other output can use it"},
		{[]string{"--html", "-", "--wordlist", "spicy"}, "--wordlist must be none, censored, or uncensored"},
		{[]string{"--html", "-", "--json", "-"}, "only one output can be written to stdout"},
	} {
		code, stdout, stderr := runArgs(t, tc.args...)
		if code != 2 {
			t.Errorf("%q: exit %d, want 2", tc.args, code)
		}
		if stdout != "" {
			t.Errorf("%q: stdout = %q, want empty", tc.args, stdout)
		}
		if !strings.Contains(stderr, tc.msg) || !strings.Contains(stderr, "-html") {
			t.Errorf("%q: stderr missing %q or usage: %q", tc.args, tc.msg, stderr)
		}
	}
}
