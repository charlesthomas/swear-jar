package heatmap

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/charlesthomas/swear-jar/internal/scrape"
)

type WordCount struct {
	Word  string
	Count int
}

type Cell struct {
	Col, Row int
	Level    int
	N, Msgs  int
	Date     time.Time
	Words    []WordCount
}

type Label struct {
	Class    string
	Col, Row int
	Text     string
}

type Stat struct {
	Value string
	Label string
}

type Bar struct {
	Word  string
	Count int
	Pct   float64
}

type Heatmap struct {
	Total       int
	Sessions    int
	Files       int
	Weeks       int
	Generated   time.Time
	DayLabels   []Label
	MonthLabels []Label
	Cells       []Cell
	Stats       []Stat
	Vocab       []Bar
}

func Build(res scrape.Result, today time.Time) Heatmap {
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)
	start := today.AddDate(0, 0, -364)
	start = start.AddDate(0, 0, -int(start.Weekday()))
	weeks := int(today.Sub(start).Hours()/24)/7 + 1
	days := map[string]scrape.Day{}
	first, last := start.Format(time.DateOnly), today.Format(time.DateOnly)
	for k, d := range res.Days {
		if k >= first && k <= last {
			days[k] = d
		}
	}
	cuts := quartileCuts(days)

	h := Heatmap{Sessions: res.Sessions, Files: res.Files, Weeks: weeks, Generated: today}
	for i, text := range []string{"", "Mon", "", "Wed", "", "Fri", ""} {
		if text != "" {
			h.DayLabels = append(h.DayLabels, Label{Class: "dlabel", Col: 1, Row: i + 2, Text: text})
		}
	}

	streak, best := 0, 0
	lastMonth := time.Month(0)
	for w := 0; w < weeks; w++ {
		for d := 0; d < 7; d++ {
			day := start.AddDate(0, 0, w*7+d)
			if d == 0 && day.Month() != lastMonth && day.Day() <= 7 {
				lastMonth = day.Month()
				h.MonthLabels = append(h.MonthLabels, Label{Class: "mlabel", Col: w + 2, Row: 1, Text: day.Format("Jan")})
			}
			if day.After(today) {
				continue
			}
			rec := days[day.Format(time.DateOnly)]
			h.Total += rec.N
			if rec.N > 0 {
				streak++
			} else {
				streak = 0
			}
			best = max(best, streak)
			h.Cells = append(h.Cells, Cell{
				Col: w + 2, Row: d + 2,
				Level: level(rec.N, cuts),
				N:     rec.N, Msgs: rec.Msgs,
				Date:  day,
				Words: sortedWords(rec.Words),
			})
		}
	}

	h.Stats = stats(days, h.Total, best)
	h.Vocab = vocab(days)
	return h
}

func quartileCuts(days map[string]scrape.Day) [3]int {
	var counts []int
	for _, d := range days {
		if d.N > 0 {
			counts = append(counts, d.N)
		}
	}
	sort.Ints(counts)
	q := func(p float64) int {
		if len(counts) == 0 {
			return 1
		}
		return counts[min(len(counts)-1, int(p*float64(len(counts))))]
	}
	return [3]int{q(0.25), q(0.5), q(0.75)}
}

func level(n int, cuts [3]int) int {
	if n == 0 {
		return 0
	}
	l := 1
	for _, c := range cuts {
		if n > c {
			l++
		}
	}
	return l
}

func sortedWords(words map[string]int) []WordCount {
	out := make([]WordCount, 0, len(words))
	for w, c := range words {
		out = append(out, WordCount{Word: w, Count: c})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Word < out[j].Word
	})
	return out
}

func stats(days map[string]scrape.Day, total, best int) []Stat {
	keys := make([]string, 0, len(days))
	for k := range days {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	active, swore, messages := 0, 0, 0
	worstDay, worstN := "", 0
	for _, k := range keys {
		d := days[k]
		if d.Msgs > 0 {
			active++
			messages += d.Msgs
		}
		if d.N > 0 {
			swore++
		}
		if d.N > worstN {
			worstDay, worstN = k, d.N
		}
	}

	worst := Stat{Value: "0", Label: "worst day"}
	if worstN > 0 {
		t, _ := time.Parse(time.DateOnly, worstDay)
		worst = Stat{Value: strconv.Itoa(worstN), Label: fmt.Sprintf("worst day (%s)", t.Format("Jan 2"))}
	}
	per100 := "0"
	if messages > 0 {
		per100 = fmt.Sprintf("%.1f", 100*float64(total)/float64(messages))
	}
	return []Stat{
		{Value: strconv.Itoa(total), Label: "swears"},
		{Value: fmt.Sprintf("%d/%d", swore, active), Label: "active days with swearing"},
		worst,
		{Value: per100, Label: "per 100 messages"},
		{Value: strconv.Itoa(best), Label: "longest swearing streak (days)"},
	}
}

func vocab(days map[string]scrape.Day) []Bar {
	totals := map[string]int{}
	for _, d := range days {
		for w, c := range d.Words {
			totals[w] += c
		}
	}
	words := sortedWords(totals)
	top := 1
	if len(words) > 0 {
		top = words[0].Count
	}
	bars := make([]Bar, len(words))
	for i, wc := range words {
		bars[i] = Bar{Word: wc.Word, Count: wc.Count, Pct: 100 * float64(wc.Count) / float64(top)}
	}
	return bars
}
