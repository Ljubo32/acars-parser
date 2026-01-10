package main

import (
	"acars_parser/internal/acars"
	_ "acars_parser/internal/parsers"
	"acars_parser/internal/registry"
	"encoding/json"
	"fmt"
)

func main() {
	messages := []string{
		"POS01AFL1866 /16180720UUEEUDYZ FUEL 140 TEMP- 32 WDIR26631 WSPD 36 LATN 55.164 LONE 38.545 ETA1013 TUR ALT 21728",
		"POS01SU0245 /18181727FSIAUUEE FUEL 14 TEMP-55 WDIR25381 WSPD53 LATN 54.567 LONE 38.387 ETA1813 TUR ALT 36221",
	}

	reg := registry.Default()

	for i, text := range messages {
		fmt.Printf("\n=== Message %d ===\n", i+1)
		msg := &acars.Message{Label: "27", Text: text}
		result := reg.DispatchFirst(msg)
		if result == nil {
			fmt.Println("No parser matched")
			continue
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(data))
	}
}
