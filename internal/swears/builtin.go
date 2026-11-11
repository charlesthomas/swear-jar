package swears

func Builtin() []Entry {
	return []Entry{
		{Word: "ass"},
		{Word: "asshole", Suffix: true},
		{Word: "bastard", Suffix: true},
		{Word: "bitch", Suffix: true},
		{Word: "bloody"},
		{Word: "crap", Suffix: true},
		{Word: "damn", Prefix: true, Suffix: true},
		{Word: "dick", Suffix: true},
		{Word: "ffs"},
		{Word: "fuck", Prefix: true, Suffix: true},
		{Word: "hell"},
		{Word: "omfg"},
		{Word: "piss", Suffix: true},
		{Word: "shit", Prefix: true, Suffix: true},
		{Word: "stfu"},
		{Word: "wtf"},
	}
}
