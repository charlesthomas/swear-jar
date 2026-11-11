package svg

import (
	"bytes"
	"encoding/xml"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charlesthomas/swear-jar/internal/heatmap"
	"github.com/charlesthomas/swear-jar/internal/scrape"
)

var testHeatmap = heatmap.Build(scrape.Result{Files: 2, Sessions: 1, Days: map[string]scrape.Day{
	"2026-11-09": {N: 2, Msgs: 1, Words: map[string]int{"shit": 2}},
	"2026-11-10": {N: 1, Msgs: 2, Words: map[string]int{"damn": 1}},
}}, time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC))

func render(t *testing.T, th Theme) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Render(&buf, testHeatmap, th); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestRenderIsValidXML(t *testing.T) {
	for _, th := range Themes {
		d := xml.NewDecoder(strings.NewReader(render(t, th)))
		for {
			if _, err := d.Token(); err == io.EOF {
				break
			} else if err != nil {
				t.Fatalf("%s: invalid XML: %v", th.Name, err)
			}
		}
	}
}

func TestRenderContent(t *testing.T) {
	out := render(t, Light)
	for _, want := range []string{
		`width="802" height="`,
		`>3 swears in the last year</text>`,
		`>1 session · 2 transcripts</text>`,
		`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512" x="12" y="12" width="72" height="72">`,
		`<text x="96" y="53" font-size="14" font-weight="600" fill="#1f2328">3 swears in the last year</text>`,
		`<text x="103" y="103">Dec</text>`,
		`<text x="16" y="133">Mon</text>`,
		`<rect x="775" y="124" width="11" height="11" rx="2" fill="#ff8787"><title>2 swears on Mon, Nov 9, 2026 · 1 message sent</title></rect>`,
		`<title>No swears on Wed, Nov 11, 2026 · 0 messages sent</title>`,
		`>2/2</text>`,
		`<tspan x="653.6" y="284">longest swearing streak</tspan><tspan x="653.6" y="297">(days)</tspan>`,
		`<text x="16" y="328" font-size="11" fill="#656d76">Rendered with https://github.com/charlesthomas/swear-jar</text>`,
		`width="802" height="340"`,
		`fill="#ffffff" stroke="#d0d7de"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
	if n := strings.Count(out, "<title>"); n != len(testHeatmap.Cells) {
		t.Errorf("%d cell titles, want %d", n, len(testHeatmap.Cells))
	}
}

func TestRenderNeverShowsWords(t *testing.T) {
	for _, th := range Themes {
		out := render(t, th)
		for _, word := range []string{"shit", "sh*t", "damn", "d*mn"} {
			if strings.Contains(out, word) {
				t.Errorf("%s: output contains %q", th.Name, word)
			}
		}
	}
}

func TestDarkTheme(t *testing.T) {
	out := render(t, Dark)
	if !strings.Contains(out, `fill="#0d1117" stroke="#30363d"`) || strings.Contains(out, `fill="#ffffff" stroke="#d0d7de"`) {
		t.Error("dark output should use the dark background and no light colors")
	}
}

var update = flag.Bool("update", false, "rewrite golden files")

func TestGolden(t *testing.T) {
	for _, th := range Themes {
		got := render(t, th)
		path := filepath.Join("testdata", th.File)
		if *update {
			if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if got != string(want) {
			t.Errorf("%s differs from golden; rerun with -update if the change is intended", path)
		}
	}
}

func TestWrap(t *testing.T) {
	for in, want := range map[string]string{
		"swears":                         "swears",
		"active days with swearing":      "active days with|swearing",
		"longest swearing streak (days)": "longest swearing streak|(days)",
		"":                               "",
	} {
		if got := strings.Join(wrap(in, lineChars), "|"); got != want {
			t.Errorf("wrap(%q) = %q, want %q", in, got, want)
		}
	}
}
