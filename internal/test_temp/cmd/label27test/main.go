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
		Text:  "POS01AFL1866 /16180720UUEEUDYZ FUEL 140 TEMP- 32 WDIR26631 WSPD 36 LATN 55.164 LONE 38.545 ETA1013 TUR ALT 21728",
	}
	parser := &label27.Parser{}
	result := parser.Parse(msg)
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
}
