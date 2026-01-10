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
		Text:  "POS01SU0245 /18181727FSIAUUEE FUEL 14 TEMP-55 WDIR25381 WSPD53 LATN 54.567 LONE 38.387 ETA1813 TUR ALT 36221",
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
