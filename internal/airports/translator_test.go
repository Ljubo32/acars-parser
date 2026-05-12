package airports

import "testing"

func TestParseAirportJSONSkipsNullCodes(t *testing.T) {
	lookup, err := parseAirportJSON([]byte(`[
		{"iata":"LTN","icao":"EGGW"},
		{"iata":null,"icao":"EIDW"},
		{"iata":"DUB","icao":null},
		{"iata":"BAD","icao":2136}
	]`))
	if err != nil {
		t.Fatalf("parseAirportJSON() error = %v", err)
	}

	if got := lookup["LTN"]; got != "EGGW" {
		t.Fatalf("lookup[LTN] = %q, want %q", got, "EGGW")
	}
	if _, ok := lookup["DUB"]; ok {
		t.Fatal("lookup[DUB] present, want skipped null ICAO entry")
	}
	if _, ok := lookup["BAD"]; ok {
		t.Fatal("lookup[BAD] present, want skipped numeric ICAO entry")
	}
}

func TestNormaliseCode(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "iata maps to icao", input: "LTN", want: "EGGW"},
		{name: "iata lower case maps to icao", input: "dub", want: "EIDW"},
		{name: "icao stays unchanged", input: "eggw", want: "EGGW"},
		{name: "unknown code stays upper case", input: "zzz", want: "ZZZ"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := NormaliseCode(test.input); got != test.want {
				t.Fatalf("NormaliseCode(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}