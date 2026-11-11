package swears

import (
	"reflect"
	"testing"
)

func TestCensorWord(t *testing.T) {
	for word, want := range map[string]string{
		"fuck":         "f*ck",
		"shit":         "sh*t",
		"damn":         "d*mn",
		"goddamn":      "g*ddamn",
		"asshole":      "*sshole",
		"Ass":          "*ss",
		"wtf":          "wtf",
		"rhythm":       "rhythm",
		"jesus christ": "j*sus christ",
	} {
		if got := CensorWord(word); got != want {
			t.Errorf("CensorWord(%q) = %q, want %q", word, got, want)
		}
	}
}

func TestMatch(t *testing.T) {
	l := New(Builtin())
	for text, want := range map[string][]string{
		"this is FUCKING broken":                      {"fucking"},
		"fucked, fucks, fuck":                         {"fucked", "fucks", "fuck"},
		"motherfucker, clusterfuck":                   {"motherfucker", "clusterfuck"},
		"bullshit and batshit and shitty":             {"bullshit", "batshit", "shitty"},
		"god damn it, goddamnit, damned":              {"damn", "goddamnit", "damned"},
		"Hell, hello, shell, hell's":                  {"hell", "hell"},
		"ass, class, assessment, bass, asses":         {"ass"},
		"assholes":                                    {"assholes"},
		"wtf ffs stfu omfg":                           {"wtf", "ffs", "stfu", "omfg"},
		"bloody bloodymary":                           {"bloody"},
		"bitches, bastards, crappy, pissed, dickhead": {"bitches", "bastards", "crappy", "pissed", "dickhead"},
		"perfectly clean text":                        nil,
	} {
		if got := l.Match(text); !reflect.DeepEqual(got, want) {
			t.Errorf("Match(%q) = %q, want %q", text, got, want)
		}
	}
}

func TestCensor(t *testing.T) {
	l := New(Builtin())
	for word, want := range map[string]string{
		"fucking":      "f*cking",
		"motherfucker": "motherf*cker",
		"bullshit":     "bullsh*t",
		"goddamnit":    "godd*mnit",
		"asshole":      "*sshole",
		"ass":          "*ss",
		"bloody":       "bl*ody",
		"wtf":          "wtf",
		"omfg":         "*mfg",
		"unknown":      "*nknown",
	} {
		if got := l.Censor(word); got != want {
			t.Errorf("Censor(%q) = %q, want %q", word, got, want)
		}
	}
}

func TestCensorOverride(t *testing.T) {
	l := New([]Entry{{Word: "frak", Suffix: true, Censored: "fr@k"}})
	if got := l.Censor("frakking"); got != "fr@kking" {
		t.Errorf("Censor(frakking) = %q, want fr@kking", got)
	}
}
