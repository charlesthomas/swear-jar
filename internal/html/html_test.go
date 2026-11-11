package html

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/charlesthomas/swear-jar/internal/heatmap"
	"github.com/charlesthomas/swear-jar/internal/scrape"
)

var testResult = scrape.Result{Files: 2, Sessions: 3, Days: map[string]scrape.Day{
	"2026-11-09": {N: 2, Msgs: 1, Words: map[string]int{"sh*t": 2}},
	"2026-11-10": {N: 1, Msgs: 2, Words: map[string]int{"<b>": 1}},
}}

func render(t *testing.T, show func(string) string) string {
	t.Helper()
	var buf bytes.Buffer
	if err := Render(&buf, heatmap.Build(testResult, time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC)), show); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func identity(s string) string { return s }

func TestRender(t *testing.T) {
	out := render(t, identity)
	for _, want := range []string{
		`<strong>3 swears in the last year</strong><span class="num">3 sessions · 2 transcripts</span>`,
		`style="--weeks: 53"`,
		`<div class="mlabel" style="grid-column: 6; grid-row: 1">Dec</div>`,
		`<div class="cell" data-l="2" style="grid-column: 54; grid-row: 3"><span class="tip"><b>2 swears</b> on Mon, Nov 9, 2026 · 1 message sent<span class="w">sh*t×2</span></span></div>`,
		`<b>No swears</b> on Wed, Nov 11, 2026 · 0 messages sent<span class="w"></span>`,
		`<span class="w">&lt;b&gt;×1</span>`,
		`<div class="stat"><b>2/2</b><small>active days with swearing</small></div>`,
		`<span>sh*t</span><div class="bar" style="width: 100.0%"></div><span class="num">2</span>`,
		`<div class="bar" style="width: 50.0%"></div>`,
		`Generated 2026-11-11 from`,
		`<title>Swear Jar</title>`,
		`<h1>Swear Jar</h1>`,
		`<header><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512" x="0" y="0" width="96" height="96">`,
		`Rendered with <a href="https://github.com/charlesthomas/swear-jar">swear-jar</a>.</footer>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
	if strings.Contains(out, "ZgotmplZ") {
		t.Error("output contains ZgotmplZ, so the template escaper rejected a value")
	}
}

func TestRenderWithoutWords(t *testing.T) {
	out := render(t, nil)
	for _, unwanted := range []string{"Vocabulary", `<span class="w">`, "sh*t", "&lt;b&gt;", `class="bar"`} {
		if strings.Contains(out, unwanted) {
			t.Errorf("output contains %q", unwanted)
		}
	}
	for _, want := range []string{
		`<b>2 swears</b> on Mon, Nov 9, 2026 · 1 message sent</span></div>`,
		`<div class="stat"><b>3</b><small>swears</small></div>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestRenderRelabelMerges(t *testing.T) {
	out := render(t, func(string) string { return "same" })
	for _, want := range []string{
		`<span class="w">same×2</span>`,
		`<span>same</span><div class="bar" style="width: 100.0%"></div><span class="num">3</span>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
	if strings.Count(out, "<span>same</span>") != 1 {
		t.Error("relabeled words should merge into one vocabulary bar")
	}
}
