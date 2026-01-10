package main

import (
	"acars_parser/internal/acars"
	"acars_parser/internal/parsers/adsc"
	"encoding/json"
	"fmt"
)

func main() {
	msg := &acars.Message{
		Label: "B6",
		Text:  "/NYCODYA.ADS.C-FGDT070EF0E6A6C28908B7001F0D0CCCCEB05B090885A90B1F6EB5060908800E35F0FE3FFC0F3749A33FFC0258",
	}
	parser := &adsc.Parser{}
	result := parser.Parse(msg).(*adsc.Result)
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
}
