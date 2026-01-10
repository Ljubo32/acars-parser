package main

import (
	"acars_parser/internal/acars"
	_ "acars_parser/internal/parsers"
	"acars_parser/internal/registry"
	"encoding/json"
	"fmt"
)

func main() {
	msg := &acars.Message{
		Label: "27",
		Text:  "POS01AFL1866 /16180720UUEEUDYZ FUEL 140 TEMP- 32 WDIR26631 WSPD 36 LATN 55.164 LONE 38.545 ETA1013 TUR ALT 21728",
	}
	reg := registry.Default()
	result := reg.DispatchFirst(msg)
	if result == nil {
		fmt.Println("No parser matched")
		return
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println("Parser Type:", result.Type())
	fmt.Println(string(data))
}
