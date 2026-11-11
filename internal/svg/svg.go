package svg

import (
	_ "embed"
	"fmt"
	"io"
	"math"
	"strings"
	"text/template"

	"github.com/charlesthomas/swear-jar/assets"
	"github.com/charlesthomas/swear-jar/internal/heatmap"
)

//go:embed heatmap.svg.tmpl
var source string

var tmpl = template.Must(template.New("svg").Funcs(template.FuncMap{
	"sub": func(a, b int) int { return a - b },
}).Parse(source))

type Theme struct {
	Name, File                   string
	Bg, Fg, Muted, Border, Panel string
	Levels                       [5]string
}

var (
	Light = Theme{
		Name: "light", File: "heatmap-light.svg",
		Bg: "#ffffff", Fg: "#1f2328", Muted: "#656d76", Border: "#d0d7de", Panel: "#f6f8fa",
		Levels: [5]string{"#ebedf0", "#ffc9c9", "#ff8787", "#e03131", "#8b0000"},
	}
	Dark = Theme{
		Name: "dark", File: "heatmap-dark.svg",
		Bg: "#0d1117", Fg: "#e6edf3", Muted: "#8d96a0", Border: "#30363d", Panel: "#161b22",
		Levels: [5]string{"#161b22", "#4a1416", "#8f1d22", "#d6333a", "#ff6b6b"},
	}
	Themes = []Theme{Light, Dark}
)

const (
	pad       = 16
	labelW    = 28
	gap       = 3
	step      = 14
	monthH    = 15
	tileGap   = 12
	tileH     = 68
	lineChars = 24
	logoSize  = 72
)

type text struct {
	X, Y int
	Text string
}

type rect struct {
	X, Y  int
	Fill  string
	Title string
}

type tile struct {
	X, W   float64
	Y, H   int
	TextX  float64
	ValueY int
	Value  string
	Lines  []line
}

type line struct {
	X    float64
	Y    int
	Text string
}

type view struct {
	Theme              Theme
	Width, Height      int
	Left, Right, HeadY int
	HeadX, AttrY       int
	Logo, RepoURL      string
	Heading, Counts    string
	Labels             []text
	Cells              []rect
	Legend             []rect
	LessX, LegendTextY int
	Stats              []tile
}

func Render(w io.Writer, h heatmap.Heatmap, t Theme) error {
	gridW := labelW + h.Weeks*step
	v := view{
		Theme: t,
		Width: gridW + 2*pad,
		Left:  pad, Right: pad + gridW, HeadY: pad - 4 + logoSize/2 + 5,
		HeadX:   pad + logoSize + 8,
		Logo:    assets.Logo(pad-4, pad-4, logoSize),
		RepoURL: assets.RepoURL,
		Heading: plural(h.Total, "swear") + " in the last year",
		Counts:  fmt.Sprintf("%s · %s", plural(h.Sessions, "session"), plural(h.Files, "transcript")),
	}
	gridTop := pad - 4 + logoSize + 8
	cellX := func(col int) int { return pad + labelW + gap + (col-2)*step }
	cellY := func(row int) int { return gridTop + monthH + gap + (row-2)*step }

	for _, l := range h.MonthLabels {
		v.Labels = append(v.Labels, text{X: cellX(l.Col), Y: gridTop + 11, Text: l.Text})
	}
	for _, l := range h.DayLabels {
		v.Labels = append(v.Labels, text{X: pad, Y: cellY(l.Row) + 9, Text: l.Text})
	}
	for _, c := range h.Cells {
		v.Cells = append(v.Cells, rect{
			X: cellX(c.Col), Y: cellY(c.Row),
			Fill:  t.Levels[c.Level],
			Title: fmt.Sprintf("%s on %s · %s sent", count(c.N), c.Date.Format("Mon, Jan 2, 2006"), plural(c.Msgs, "message")),
		})
	}

	legendY := cellY(8) + 11 + 10
	v.LegendTextY = legendY + 9
	firstLegendX := v.Right - 32 - 5*step + gap
	for i, fill := range t.Levels {
		v.Legend = append(v.Legend, rect{X: firstLegendX + i*step, Y: legendY, Fill: fill})
	}
	v.LessX = firstLegendX - 4

	statsY := legendY + 11 + tileGap
	if n := len(h.Stats); n > 0 {
		tileW := float64(gridW-(n-1)*tileGap) / float64(n)
		for i, s := range h.Stats {
			x := float64(pad) + float64(i)*(tileW+tileGap)
			tl := tile{X: r1(x), W: r1(tileW), Y: statsY, H: tileH, TextX: r1(x + 12), ValueY: statsY + 28, Value: s.Value}
			for j, l := range wrap(s.Label, lineChars) {
				tl.Lines = append(tl.Lines, line{X: r1(x + 12), Y: statsY + 46 + j*13, Text: l})
			}
			v.Stats = append(v.Stats, tl)
		}
		statsY += tileH
	}
	v.AttrY = statsY + 22
	v.Height = v.AttrY + pad - 4
	return tmpl.Execute(w, v)
}

func r1(f float64) float64 {
	return math.Round(f*10) / 10
}

func wrap(s string, width int) []string {
	var lines []string
	cur := ""
	for _, word := range strings.Fields(s) {
		if cur != "" && len(cur)+1+len(word) > width {
			lines = append(lines, cur)
			cur = word
			continue
		}
		if cur != "" {
			cur += " "
		}
		cur += word
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
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
