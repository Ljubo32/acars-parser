package main

import (
	"encoding/json"
	"fmt"
	"log"

	"acars_parser/internal/acars"
	_ "acars_parser/internal/parsers/label33"
	"acars_parser/internal/registry"
)

func main() {
	msg := &acars.Message{
		ID:        0,
		Timestamp: "2026-01-21T00:21:39.202853Z",
		Tail:      ".4X-EDI",
		Label:     "33",
		Text:      "2026-01-21,00:21:38,EGLL,LLBG,0318,N42572E018339,491,FL410,0150,-58,240, 28,RIN29  ,00:31,BON30  ,-28,848,252,483,0132,0128,0134,022,200126",
		Frequency: 136.975,
	}

	fmt.Println("Testing label33 parser...")
	fmt.Printf("Message Label: %s\n", msg.Label)
	fmt.Printf("Message Text: %s\n\n", msg.Text)

	registry.Default().Sort()
	results := registry.Default().Dispatch(msg)

	fmt.Printf("Number of results: %d\n\n", len(results))

	if len(results) == 0 {
		fmt.Println("No results returned!")
		return
	}

	for i, result := range results {
		fmt.Printf("Result %d:\n", i+1)
		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			log.Printf("Error marshaling result: %v", err)
			continue
		}
		fmt.Println(string(jsonData))
	}
}
