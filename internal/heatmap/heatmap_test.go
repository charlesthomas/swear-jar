package heatmap

import (
	"reflect"
	"testing"
	"time"

	"github.com/charlesthomas/swear-jar/internal/scrape"
)

var today = time.Date(2026, 11, 11, 15, 30, 0, 0, time.Local)

func date(s string) time.Time {
	t, _ := time.Parse(time.DateOnly, s)
	return t
}

func TestGrid(t *testing.T) {
	h := Build(scrape.Result{Days: map[string]scrape.Day{}}, today)
	if h.Weeks != 53 {
		t.Errorf("Weeks = %d, want 53", h.Weeks)
	}
	if len(h.Cells) != 52*7+4 {
		t.Errorf("len(Cells) = %d, want %d", len(h.Cells), 52*7+4)
	}
	first, last := h.Cells[0], h.Cells[len(h.Cells)-1]
	if !first.Date.Equal(date("2025-11-09")) || first.Col != 2 || first.Row != 2 {
		t.Errorf("first cell = %v col %d row %d", first.Date, first.Col, first.Row)
	}
	if !last.Date.Equal(date("2026-11-11")) || last.Col != 54 || last.Row != 5 {
		t.Errorf("last cell = %v col %d row %d", last.Date, last.Col, last.Row)
	}
	if len(h.MonthLabels) != 12 || h.MonthLabels[0] != (Label{Class: "mlabel", Col: 6, Row: 1, Text: "Dec"}) {
		t.Errorf("MonthLabels = %v", h.MonthLabels)
	}
	if len(h.DayLabels) != 3 || h.DayLabels[0] != (Label{Class: "dlabel", Col: 1, Row: 3, Text: "Mon"}) {
		t.Errorf("DayLabels = %v", h.DayLabels)
	}
}

func TestLevels(t *testing.T) {
	days := map[string]scrape.Day{}
	for i := 1; i <= 8; i++ {
		days[date("2026-01-01").AddDate(0, 0, i).Format(time.DateOnly)] = scrape.Day{N: i, Msgs: 1}
	}
	cuts := quartileCuts(days)
	if cuts != [3]int{3, 5, 7} {
		t.Fatalf("cuts = %v, want [3 5 7]", cuts)
	}
	for n, want := range map[int]int{0: 0, 1: 1, 3: 1, 4: 2, 5: 2, 6: 3, 7: 3, 8: 4} {
		if got := level(n, cuts); got != want {
			t.Errorf("level(%d) = %d, want %d", n, got, want)
		}
	}
	if quartileCuts(nil) != [3]int{1, 1, 1} {
		t.Errorf("empty cuts = %v, want [1 1 1]", quartileCuts(nil))
	}
}

func TestStatsStreakAndVocab(t *testing.T) {
	res := scrape.Result{Days: map[string]scrape.Day{
		"2025-11-08": {N: 9, Msgs: 4, Words: map[string]int{"old": 9}},
		"2026-11-05": {N: 3, Msgs: 1, Words: map[string]int{"a": 3}},
		"2026-11-09": {N: 2, Msgs: 1, Words: map[string]int{"a": 2}},
		"2026-11-10": {N: 1, Msgs: 2, Words: map[string]int{"b": 1}},
		"2026-11-11": {N: 0, Msgs: 1, Words: map[string]int{}},
		"2026-11-12": {N: 3, Msgs: 1, Words: map[string]int{"c": 3}},
	}}
	h := Build(res, today)
	if h.Total != 6 {
		t.Errorf("Total = %d, want 6", h.Total)
	}
	want := []Stat{
		{Value: "6", Label: "swears"},
		{Value: "3/4", Label: "active days with swearing"},
		{Value: "3", Label: "worst day (Nov 5)"},
		{Value: "120.0", Label: "per 100 messages"},
		{Value: "2", Label: "longest swearing streak (days)"},
	}
	if !reflect.DeepEqual(h.Stats, want) {
		t.Errorf("Stats = %v\nwant %v", h.Stats, want)
	}
	wantVocab := []Bar{{Word: "a", Count: 5, Pct: 100}, {Word: "b", Count: 1, Pct: 20}}
	if !reflect.DeepEqual(h.Vocab, wantVocab) {
		t.Errorf("Vocab = %v\nwant %v", h.Vocab, wantVocab)
	}
	for _, c := range h.Cells {
		if c.Date.Equal(date("2026-11-09")) {
			if c.N != 2 || !reflect.DeepEqual(c.Words, []WordCount{{Word: "a", Count: 2}}) {
				t.Errorf("Nov 9 cell = %+v", c)
			}
		}
	}
}

func TestEmptyStats(t *testing.T) {
	h := Build(scrape.Result{}, today)
	want := []Stat{
		{Value: "0", Label: "swears"},
		{Value: "0/0", Label: "active days with swearing"},
		{Value: "0", Label: "worst day"},
		{Value: "0", Label: "per 100 messages"},
		{Value: "0", Label: "longest swearing streak (days)"},
	}
	if !reflect.DeepEqual(h.Stats, want) {
		t.Errorf("Stats = %v\nwant %v", h.Stats, want)
	}
	if len(h.Vocab) != 0 {
		t.Errorf("Vocab = %v, want empty", h.Vocab)
	}
}
