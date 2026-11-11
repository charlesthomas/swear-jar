package scrape

import (
	"bufio"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Day struct {
	N     int            `json:"n"`
	Msgs  int            `json:"msgs"`
	Words map[string]int `json:"words"`
}

type Result struct {
	Files    int            `json:"files"`
	Sessions int            `json:"sessions"`
	Messages int            `json:"messages"`
	Days     map[string]Day `json:"days"`
}

type Matcher func(text string) []string

var pruned = map[string]bool{"node_modules": true, ".Trash": true, "Caches": true, ".git": true}

var openTag = regexp.MustCompile(`<(system-reminder|command-\w+|local-command-\w+|pasted_content)[^>]*>`)

const missing = "\x00missing"

type entry struct {
	Type        string  `json:"type"`
	IsMeta      bool    `json:"isMeta"`
	IsSidechain bool    `json:"isSidechain"`
	UUID        *string `json:"uuid"`
	SessionID   *string `json:"sessionId"`
	Timestamp   string  `json:"timestamp"`
	Message     struct {
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

type block struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func Files(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == root {
				return err
			}
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if path != root && pruned[d.Name()] {
				return fs.SkipDir
			}
			return nil
		}
		name := d.Name()
		if strings.HasSuffix(name, ".jsonl") && name != "history.jsonl" {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func DefaultRoot() (string, error) {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "projects"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "projects"), nil
}

func Scrape(root string, match Matcher) (Result, error) {
	files, err := Files(root)
	if err != nil {
		return Result{}, err
	}
	s := scraper{
		match:    match,
		seen:     map[string]bool{},
		sessions: map[string]bool{},
		days:     map[string]*Day{},
	}
	for _, path := range files {
		s.file(path)
	}
	res := Result{Files: len(files), Sessions: len(s.sessions), Days: map[string]Day{}}
	for day, d := range s.days {
		res.Messages += d.Msgs
		res.Days[day] = *d
	}
	return res, nil
}

type scraper struct {
	match    Matcher
	seen     map[string]bool
	sessions map[string]bool
	days     map[string]*Day
}

func (s *scraper) file(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	r := bufio.NewReader(f)
	for {
		line, err := r.ReadBytes('\n')
		if len(line) > 0 {
			s.line(line)
		}
		if err != nil {
			return
		}
	}
}

func (s *scraper) line(line []byte) {
	var e entry
	if json.Unmarshal(line, &e) != nil {
		return
	}
	if e.Type != "user" || e.IsMeta || e.IsSidechain {
		return
	}
	if e.UUID != nil {
		if s.seen[*e.UUID] {
			return
		}
		s.seen[*e.UUID] = true
	}
	text, ok := content(e.Message.Content)
	if !ok || strings.TrimSpace(text) == "" {
		return
	}
	text = stripHarness(text)
	if e.Timestamp == "" {
		return
	}
	t, err := time.Parse(time.RFC3339Nano, e.Timestamp)
	if err != nil {
		return
	}
	day := t.Local().Format(time.DateOnly)
	d := s.days[day]
	if d == nil {
		d = &Day{Words: map[string]int{}}
		s.days[day] = d
	}
	d.Msgs++
	s.sessions[key(e.SessionID)] = true
	for _, w := range s.match(text) {
		d.N++
		d.Words[w]++
	}
}

func key(p *string) string {
	if p == nil {
		return missing
	}
	return *p
}

func content(raw json.RawMessage) (string, bool) {
	if len(raw) == 0 {
		return "", false
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return text, true
	}
	var items []json.RawMessage
	if json.Unmarshal(raw, &items) != nil {
		return "", false
	}
	var parts []string
	for _, item := range items {
		var b block
		if json.Unmarshal(item, &b) != nil || b.Type != "text" {
			continue
		}
		parts = append(parts, b.Text)
	}
	return strings.Join(parts, " "), true
}

func stripHarness(s string) string {
	var b strings.Builder
	copied, search := 0, 0
	for {
		loc := openTag.FindStringSubmatchIndex(s[search:])
		if loc == nil {
			break
		}
		start, end := search+loc[0], search+loc[1]
		after := closingEnd(s, end, s[search+loc[2]:search+loc[3]])
		if after < 0 {
			search = start + 1
			continue
		}
		b.WriteString(s[copied:start])
		b.WriteString(" ")
		copied = after
		search = copied
	}
	b.WriteString(s[copied:])
	return b.String()
}

func closingEnd(s string, from int, name string) int {
	prefix := "</" + name
	for {
		i := strings.Index(s[from:], prefix)
		if i < 0 {
			return -1
		}
		rest := from + i + len(prefix)
		if rest < len(s) && s[rest] == '>' {
			return rest + 1
		}
		if rest < len(s) && strings.ContainsRune(" \t\r\n", rune(s[rest])) {
			if j := strings.IndexByte(s[rest:], '>'); j >= 0 {
				return rest + j + 1
			}
			return -1
		}
		from = rest
	}
}
