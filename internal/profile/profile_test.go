package profile

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charlesthomas/swear-jar/internal/heatmap"
	"github.com/charlesthomas/swear-jar/internal/scrape"
)

var update = flag.Bool("update", false, "rewrite golden files")

func golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name)
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
		t.Errorf("%s differs from golden; rerun with -update if the change is intended\n got:\n%s", path, got)
	}
}

func TestSnippet(t *testing.T) {
	h := heatmap.Build(scrape.Result{Days: map[string]scrape.Day{
		"2026-11-09": {N: 2, Msgs: 1, Words: map[string]int{"shit": 2}},
	}}, time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC))
	var buf bytes.Buffer
	if err := Snippet(&buf, h); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), "shit") || strings.Contains(buf.String(), "sh*t") {
		t.Error("snippet contains swear text")
	}
	golden(t, "swear-jar.md", buf.String())
}

func TestSnippetSingular(t *testing.T) {
	var buf bytes.Buffer
	if err := Snippet(&buf, heatmap.Heatmap{Total: 1}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `alt="Swear Jar: 1 swear in the last year"`) {
		t.Errorf("snippet = %q", buf.String())
	}
}

func TestInstructions(t *testing.T) {
	var buf bytes.Buffer
	if err := Instructions(&buf, "/o/heatmap-light.svg", "/o/heatmap-dark.svg", "/o/swear-jar.md"); err != nil {
		t.Fatal(err)
	}
	golden(t, "instructions.txt", buf.String())
}
