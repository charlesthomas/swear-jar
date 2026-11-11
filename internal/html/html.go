package html

import (
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/charlesthomas/swear-jar/assets"
	"github.com/charlesthomas/swear-jar/internal/heatmap"
)

//go:embed page.html.tmpl
var pageSource string

var page = template.Must(template.New("page").Funcs(template.FuncMap{
	"plural":   plural,
	"count":    count,
	"longDate": longDate,
	"words":    words,
	"logo":     func() template.HTML { return template.HTML(assets.Logo(0, 0, 96)) },
	"repoURL":  func() string { return assets.RepoURL },
}).Parse(pageSource))

type view struct {
	heatmap.Heatmap
	ShowWords bool
}

func Render(w io.Writer, h heatmap.Heatmap, show func(string) string) error {
	v := view{Heatmap: h, ShowWords: show != nil}
	if show != nil {
		cells := make([]heatmap.Cell, len(h.Cells))
		for i, c := range h.Cells {
			c.Words = relabel(c.Words, show)
			cells[i] = c
		}
		v.Cells = cells
		v.Vocab = relabelBars(h.Vocab, show)
	} else {
		v.Vocab = nil
	}
	return page.Execute(w, v)
}

func relabel(ws []heatmap.WordCount, show func(string) string) []heatmap.WordCount {
	counts := map[string]int{}
	for _, w := range ws {
		counts[show(w.Word)] += w.Count
	}
	out := make([]heatmap.WordCount, 0, len(counts))
	for w, c := range counts {
		out = append(out, heatmap.WordCount{Word: w, Count: c})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Word < out[j].Word
	})
	return out
}

func relabelBars(bars []heatmap.Bar, show func(string) string) []heatmap.Bar {
	ws := make([]heatmap.WordCount, len(bars))
	for i, b := range bars {
		ws[i] = heatmap.WordCount{Word: b.Word, Count: b.Count}
	}
	ws = relabel(ws, show)
	out := make([]heatmap.Bar, len(ws))
	for i, w := range ws {
		out[i] = heatmap.Bar{Word: w.Word, Count: w.Count, Pct: 100 * float64(w.Count) / float64(ws[0].Count)}
	}
	return out
}

func plural(n int, word string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, word)
	}
	return fmt.Sprintf("%d %ss", n, word)
}

func count(n int) string {
	if n == 0 {
		return "No swears"
	}
	return plural(n, "swear")
}

func longDate(t time.Time) string {
	return t.Format("Mon, Jan 2, 2006")
}

func words(ws []heatmap.WordCount) string {
	parts := make([]string, len(ws))
	for i, w := range ws {
		parts[i] = fmt.Sprintf("%s×%d", w.Word, w.Count)
	}
	return strings.Join(parts, ", ")
}
