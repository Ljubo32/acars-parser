package airlines

import "testing"

func TestParseCSV(t *testing.T) {
	lookup, err := parseCSV("iata,icao\nTK,THY\nFR,RYR\n2C,CMA\n2G,HUA\n07,ORG\n")
	if err != nil {
		t.Fatalf("parseCSV returned error: %v", err)
	}

	if got := lookup["TK"]; got != "THY" {
		t.Fatalf("lookup[TK] = %q, want %q", got, "THY")
	}
	if got := lookup["FR"]; got != "RYR" {
		t.Fatalf("lookup[FR] = %q, want %q", got, "RYR")
	}
	if got := lookup["2C"]; got != "CMA" {
		t.Fatalf("lookup[2C] = %q, want %q", got, "CMA")
	}
	if got := lookup["2G"]; got != "HUA" {
		t.Fatalf("lookup[2G] = %q, want %q", got, "HUA")
	}
	if got := lookup["07"]; got != "ORG" {
		t.Fatalf("lookup[07] = %q, want %q", got, "ORG")
	}
}

func TestTranslateFlight(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "iata prefix translated", input: "TK011", want: "THY11"},
		{name: "alphanumeric prefix translated", input: "2C0573", want: "CMA573"},
		{name: "second alphanumeric prefix translated", input: "2G0421", want: "HUA421"},
		{name: "iata prefix lower case", input: "tk011", want: "THY11"},
		{name: "suffix keeps letters", input: "AEE01BS", want: "AEE1BS"},
		{name: "suffix may be alphanumeric", input: "TVS07K2", want: "TVS7K2"},
		{name: "icao also drops leading zeros", input: "THY011", want: "THY11"},
		{name: "unknown prefix also drops leading zeros", input: "ZZ011", want: "ZZ11"},
		{name: "numeric prefix translated when present in csv", input: "07123", want: "ORG123"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := TranslateFlight(test.input); got != test.want {
				t.Fatalf("TranslateFlight(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}