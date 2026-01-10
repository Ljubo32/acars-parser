//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"fmt"

	"acars_parser/internal/acars"
	"acars_parser/internal/parsers/fst"
)

func main() {
	// Test poruke
	messages := []string{
		"FST01WSSSEGLLN448644E0251403360 2361048 M49C 6028429530243611600005570319", // IAS 236 + KMH 1048 (treba ostati 1048 kmh)
		"FST01WSSSEGLLN466873E0206998360 2001084 M52C 5431429329643811600005570349", // Noviji avion: IAS 200 + KMH 1084
		"FST01OBBIEGLLN453619E0230053400 192 312 M49C 6328629129843211600006220349", // Stariji avion: IAS 192 + GS 312 knots
	}

	for i, text := range messages {
		fmt.Printf("\n=== Test %d ===\n", i+1)
		msg := &acars.Message{
			ID:    acars.FlexInt64(i + 1),
			Label: "15",
			Text:  text,
		}

		parser := &fst.Parser{}
		result := parser.Parse(msg)

		if result == nil {
			fmt.Printf("❌ Parsiranje nije uspelo za: %s\n", text)
			continue
		}

		// Konvertuj u JSON za lepši prikaz
		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Printf("❌ Greška pri konverziji u JSON: %v\n", err)
			continue
		}

		fmt.Println("✅ FST Poruka uspešno parsirana!")
		fmt.Println("\n📨 Originalna poruka:")
		fmt.Println(text)
		fmt.Println("\n📊 Parsirani podaci:")
		fmt.Println(string(jsonData))
	}
}
