package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/charlesthomas/swear-jar/internal/heatmap"
	"github.com/charlesthomas/swear-jar/internal/html"
	"github.com/charlesthomas/swear-jar/internal/profile"
	"github.com/charlesthomas/swear-jar/internal/scrape"
	"github.com/charlesthomas/swear-jar/internal/svg"
	"github.com/charlesthomas/swear-jar/internal/swears"
)

var version = "dev"

func main() {
	root, err := scrape.DefaultRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, "swear-jar:", err)
		os.Exit(1)
	}
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "swear-jar:", err)
		os.Exit(1)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "swear-jar:", err)
		os.Exit(1)
	}
	list, err := wordList(cwd, home)
	if err != nil {
		fmt.Fprintln(os.Stderr, "swear-jar:", err)
		os.Exit(1)
	}
	os.Exit(run(os.Args[1:], root, list, time.Now(), os.Stdout, os.Stderr))
}

func wordList(cwd, home string) (*swears.List, error) {
	custom, _, err := swears.LoadCustom(cwd, home)
	if err != nil {
		return nil, err
	}
	return swears.New(append(custom, swears.Builtin()...)), nil
}

func run(args []string, root string, list *swears.List, now time.Time, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("swear-jar", flag.ContinueOnError)
	flags.SetOutput(stderr)
	htmlOut := flags.String("html", "", "write the heatmap page to `file` (- for stdout)")
	wordlist := flags.String("wordlist", "none", "words shown on the --html page: none, censored, or uncensored")
	svgOut := flags.String("svg", "", "write heatmap-light.svg and heatmap-dark.svg into `dir`")
	profileReadme := flags.Bool("profile-readme", false, "with --svg, also write a GitHub profile README snippet and print how to use it")
	jsonOut := flags.String("json", "", "write the scraped data as JSON to `file` (- for stdout)")
	showVersion := flags.Bool("version", false, "print the version and exit")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Fprintln(stdout, "swear-jar", version)
		return 0
	}
	usage := func(msg string) int {
		fmt.Fprintln(stderr, "swear-jar:", msg)
		flags.Usage()
		return 2
	}
	if flags.NArg() > 0 {
		return usage(fmt.Sprintf("unexpected arguments: %v", flags.Args()))
	}
	if *htmlOut == "" && *svgOut == "" && *jsonOut == "" {
		return usage("nothing to do; pass --html, --svg, or --json")
	}
	if *svgOut == "-" {
		return usage("--svg writes two files, so it needs a directory, not -")
	}
	wordlistSet := false
	flags.Visit(func(f *flag.Flag) {
		if f.Name == "wordlist" {
			wordlistSet = true
		}
	})
	if wordlistSet && *htmlOut == "" {
		return usage("--wordlist only applies to --html")
	}
	var show func(string) string
	switch *wordlist {
	case "none":
	case "censored":
		show = list.Censor
	case "uncensored":
		show = func(w string) string { return w }
	default:
		return usage(fmt.Sprintf("--wordlist must be none, censored, or uncensored, not %q", *wordlist))
	}
	if *htmlOut == "-" && *jsonOut == "-" {
		return usage("only one output can be written to stdout")
	}
	if *profileReadme && *svgOut == "" {
		return usage("--profile-readme only applies to --svg")
	}
	if *profileReadme && (*htmlOut == "-" || *jsonOut == "-") {
		return usage("--profile-readme prints instructions to stdout, so no other output can use it")
	}

	res, err := scrape.Scrape(root, list.Match)
	if err != nil {
		fmt.Fprintln(stderr, "swear-jar:", err)
		return 1
	}
	if *svgOut != "" {
		if err := os.MkdirAll(*svgOut, 0o755); err != nil {
			fmt.Fprintln(stderr, "swear-jar:", err)
			return 1
		}
	}
	if *jsonOut != "" {
		if err := writeOutput(*jsonOut, stdout, func(w io.Writer) error { return json.NewEncoder(w).Encode(res) }); err != nil {
			fmt.Fprintln(stderr, "swear-jar:", err)
			return 1
		}
	}
	h := heatmap.Build(res, now)
	if *htmlOut != "" {
		if err := writeOutput(*htmlOut, stdout, func(w io.Writer) error { return html.Render(w, h, show) }); err != nil {
			fmt.Fprintln(stderr, "swear-jar:", err)
			return 1
		}
	}
	if *svgOut != "" {
		for _, t := range svg.Themes {
			path := filepath.Join(*svgOut, t.File)
			if err := writeOutput(path, stdout, func(w io.Writer) error { return svg.Render(w, h, t) }); err != nil {
				fmt.Fprintln(stderr, "swear-jar:", err)
				return 1
			}
		}
	}
	if *profileReadme {
		snippetPath := filepath.Join(*svgOut, profile.SnippetFile)
		if err := writeOutput(snippetPath, stdout, func(w io.Writer) error { return profile.Snippet(w, h) }); err != nil {
			fmt.Fprintln(stderr, "swear-jar:", err)
			return 1
		}
		abs := func(name string) string {
			p, err := filepath.Abs(filepath.Join(*svgOut, name))
			if err != nil {
				return filepath.Join(*svgOut, name)
			}
			return p
		}
		light, dark, snip := abs(svg.Light.File), abs(svg.Dark.File), abs(profile.SnippetFile)
		if err := profile.Instructions(stdout, light, dark, snip); err != nil {
			fmt.Fprintln(stderr, "swear-jar:", err)
			return 1
		}
	}
	return 0
}

func writeOutput(path string, stdout io.Writer, write func(io.Writer) error) error {
	if path == "-" {
		return write(stdout)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := write(f); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
