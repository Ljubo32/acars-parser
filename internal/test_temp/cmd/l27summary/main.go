package main

import (
	"acars_parser/internal/acars"
	_ "acars_parser/internal/parsers"
	"acars_parser/internal/registry"
	"fmt"
)

func main() {
	messages := []struct{ text, desc string }{
		{"POS01AFL1866 /16180720UUEEUDYZ FUEL 140 TEMP- 32 WDIR26631 WSPD 36 LATN 55.164 LONE 38.545 ETA1013 TUR ALT 21728", "AFL flight level"},
		{"POS01SU0245 /18181727FSIAUUEE FUEL 14 TEMP-55 WDIR25381 WSPD53 LATN 54.567 LONE 38.387 ETA1813 TUR ALT 36221", "2-letter airline code"},
		{"POS01AFL637 /17171847VTSPUNNT FUEL 145 TEMP- 55 WDIR34204 WSPD 27 LATN51.595 LONE089.709 ETA1957 TUR ALT 37992", "No spaces in LAT/LON"},
		{"POS01SDM6599 /18181749ULLIUWKD FUEL 66 TEMP- 56 WDIR27582 WSPD 46 LATN 57.034 LONE 43.416 ETA1832 TUR ALT 36977", "3-letter airline code"},
	}

	reg := registry.Default()

	for i, m := range messages {
		fmt.Printf("\n=== Message %d: %s ===\n", i+1, m.desc)
		msg := &acars.Message{Label: "27", Text: m.text}
		result := reg.DispatchFirst(msg)
		if result == nil {
			fmt.Println(" No parser matched")
			continue
		}
		fmt.Println(" Parsed successfully")
	}
}
