//go:build ignore
// +build ignore

package main

import (
	"encoding/hex"
	"fmt"

	"acars_parser/internal/acars"
	"acars_parser/internal/parsers/adsc"
)

func main() {
	// Test ADSC poruka
	text := "/XYTGL7X.ADS.F-GXLO0725A2E02967884D24581D0D25665826E6484D0110254F0025F2884D00815F"

	msg := &acars.Message{
		ID:    1,
		Label: "B6",
		Text:  text,
	}

	parser := &adsc.Parser{}
	result := parser.Parse(msg)

	if result == nil {
		fmt.Println("❌ Parsiranje nije uspelo!")
		return
	}

	r, ok := result.(*adsc.Result)
	if !ok {
		fmt.Println("❌ Tip rezultata nije ispravan!")
		return
	}

	fmt.Println("OK ADSC Poruka uspesno parsirana!")
	fmt.Printf("\nRezultat:\n")
	fmt.Printf("  Registration: %s\n", r.Registration)
	fmt.Printf("  Ground Station: %s\n", r.GroundStation)
	fmt.Printf("  Latitude: %.6f\n", r.Latitude)
	fmt.Printf("  Longitude: %.6f\n", r.Longitude)
	fmt.Printf("  Altitude: %d ft (očekivano: 34000 ft)\n", r.Altitude)
	fmt.Printf("  Report Time: %.1f sec\n", r.ReportTime)

	// Dekodujmo payload rucno da vidimo sta se desava
	fmt.Println("\nAnaliza payload-a:")

	// Izvuci hex deo iz poruke
	parts := []byte(text)
	hexStart := -1
	for i, c := range parts {
		if (c >= '0' && c <= '9') || (c >= 'A' && c <= 'F') {
			hexStart = i
			break
		}
	}

	if hexStart >= 0 {
		hexStr := string(parts[hexStart:])
		payload, err := hex.DecodeString(hexStr)
		if err == nil && len(payload) >= 10 {
			fmt.Printf("  Hex payload (svi bajtovi): %X\n", payload)
			fmt.Printf("  Prvih 10 bajtova: %X\n", payload[:10])

			// Ručna analiza bitova
			bits := uint64(0)
			for i := 0; i < 8; i++ {
				bits = (bits << 8) | uint64(payload[i])
			}

			fmt.Printf("  Prvih 8 bajtova (bits): %064b\n", bits)
			fmt.Printf("  Prvih 8 bajtova (hex):  %016X\n", bits)

			// Probajmo razne pozicije za altitude (12 bita)
			for offset := 40; offset <= 46; offset++ {
				altRaw := uint32((bits >> (64 - offset - 12)) & 0xFFF)
				alt := int(altRaw) * 16
				alt64 := int(altRaw) * 64
				fmt.Printf("  Offset %d: raw=0x%03X (%d) → ×16=%d, ×64=%d\n",
					offset, altRaw, altRaw, alt, alt64)
				if alt >= 33500 && alt <= 34500 {
					fmt.Printf("    ✓ MATCH sa ×16!\n")
				}
				if alt64 >= 33500 && alt64 <= 34500 {
					fmt.Printf("    ✓ MATCH sa ×64!\n")
				}
			}
		}
	}
}
