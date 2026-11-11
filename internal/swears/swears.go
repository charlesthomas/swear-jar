package swears

import (
	"regexp"
	"strings"
)

type Entry struct {
	Word     string
	Prefix   bool
	Suffix   bool
	Censored string
}

type List struct {
	entries []Entry
	rx      *regexp.Regexp
}

func New(entries []Entry) *List {
	groups := make([]string, len(entries))
	for i, e := range entries {
		g := regexp.QuoteMeta(strings.ToLower(e.Word))
		if e.Prefix {
			g = `\w*` + g
		}
		if e.Suffix {
			g += `\w*`
		}
		groups[i] = "(" + g + ")"
	}
	return &List{
		entries: entries,
		rx:      regexp.MustCompile(`(?i)\b(?:` + strings.Join(groups, "|") + `)\b`),
	}
}

func (l *List) Match(text string) []string {
	var out []string
	for _, loc := range l.rx.FindAllStringIndex(text, -1) {
		out = append(out, strings.ToLower(text[loc[0]:loc[1]]))
	}
	return out
}

func (l *List) Censor(word string) string {
	e, ok := l.entry(word)
	if !ok {
		return CensorWord(word)
	}
	base := strings.ToLower(e.Word)
	i := strings.Index(word, base)
	if i < 0 {
		return CensorWord(word)
	}
	censored := e.Censored
	if censored == "" {
		censored = CensorWord(base)
	}
	return word[:i] + censored + word[i+len(base):]
}

func (l *List) entry(word string) (Entry, bool) {
	loc := l.rx.FindStringSubmatchIndex(word)
	if loc == nil || loc[0] != 0 || loc[1] != len(word) {
		return Entry{}, false
	}
	for i := range l.entries {
		if loc[2*i+2] >= 0 {
			return l.entries[i], true
		}
	}
	return Entry{}, false
}

func CensorWord(word string) string {
	i := strings.IndexAny(word, "aeiouAEIOU")
	if i < 0 {
		return word
	}
	return word[:i] + "*" + word[i+1:]
}
