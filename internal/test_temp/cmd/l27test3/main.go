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
		Text:  "POS01AFL637 /17171847VTSPUNNT FUEL 145 TEMP- 55 WDIR34204 WSPD 27 LATN51.595 LONE089.709 ETA1957 TUR ALT 37992",
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
