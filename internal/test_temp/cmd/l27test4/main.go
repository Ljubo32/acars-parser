package main

import (
	"acars_parser/internal/acars"
	"acars_parser/internal/parsers/label27"
	"encoding/json"
	"fmt"
)

func main() {
	msg := &acars.Message{
		Label: "27",
		Text:  "POS01SDM6599 /18181749ULLIUWKD FUEL 66 TEMP- 56 WDIR27582 WSPD 46 LATN 57.034 LONE 43.416 ETA1832 TUR ALT 36977",
	}
	parser := &label27.Parser{}
	result := parser.Parse(msg)
	if result == nil {
		fmt.Println("Parser returned nil - message did not match")
		return
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
}
