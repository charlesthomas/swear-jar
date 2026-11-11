package profile

import (
	_ "embed"
	"fmt"
	"io"
	"text/template"

	"github.com/charlesthomas/swear-jar/assets"
	"github.com/charlesthomas/swear-jar/internal/heatmap"
	"github.com/charlesthomas/swear-jar/internal/svg"
)

const SnippetFile = "swear-jar.md"

//go:embed snippet.md.tmpl
var snippetSource string

//go:embed instructions.txt.tmpl
var instructionsSource string

var (
	snippet      = template.Must(template.New("snippet").Parse(snippetSource))
	instructions = template.Must(template.New("instructions").Parse(instructionsSource))
)

func Snippet(w io.Writer, h heatmap.Heatmap) error {
	heading := fmt.Sprintf("%d swears in the last year", h.Total)
	if h.Total == 1 {
		heading = "1 swear in the last year"
	}
	return snippet.Execute(w, struct {
		Light, Dark, Heading string
		Stats                []heatmap.Stat
		RepoURL              string
	}{svg.Light.File, svg.Dark.File, heading, h.Stats, assets.RepoURL})
}

func Instructions(w io.Writer, light, dark, snippetPath string) error {
	return instructions.Execute(w, struct{ Light, Dark, Snippet string }{light, dark, snippetPath})
}
