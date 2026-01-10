//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"fmt"

	"acars_parser/internal/acars"
	"acars_parser/internal/parsers/h1"
)

func main() {
	// Test POS poruke
	messages := []string{
		"POSN45209E023245,INVED,105718,330,LUGEB,112508,UDROS,M59,33450,1808/TS105718,010126904E",
		"POSN45318E024415,DINRO,044704,380,IRL-24,045003,GAN-25,M65,31458,410/TS044704,1003288E36",
	}

	for i, text := range messages {
		fmt.Printf("\n=== Test %d ===\n", i+1)
		msg := &acars.Message{
			ID:    acars.FlexInt64(i + 1),
			Label: "H1",
			Text:  text,
		}

		parser := &h1.H1PosParser{}
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

		fmt.Println("✅ H1 POS Poruka uspešno parsirana!")
		fmt.Println("\n📨 Originalna poruka:")
		fmt.Println(text)
		fmt.Println("\n📊 Parsirani podaci:")
		fmt.Println(string(jsonData))
	}
}
