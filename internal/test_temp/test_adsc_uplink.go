package test_temp

import (
	"encoding/json"
	"fmt"
	"strings"

	"acars_parser/internal/acars"
	"acars_parser/internal/parsers/adsc"
	"acars_parser/internal/parsers/envelope"
)

// TestADSCUplink tests the ADSC parser with uplink (ground-to-air) message.
func TestADSCUplink() {
	// Example uplink message: /DXBEGEK.ADS.A6-EDK07550BCAA4E2
	msg := &acars.Message{
		ID:        1,
		Timestamp: "2026-01-10 12:00:00",
		Label:     "A6", // Uplink label
		Text:      "/DXBEGEK.ADS.A6-EDK07550BCAA4E2",
	}

	fmt.Println("=== Testing Uplink Message (Label A6) ===")
	fmt.Println("ADSC Parser:")
	testADSC(msg)
	fmt.Println("\nEnvelope Parser (should NOT show FL202):")
	testEnvelope(msg)

	// Example downlink message (Label B6)
	msg2 := &acars.Message{
		ID:        2,
		Timestamp: "2026-01-10 12:05:00",
		Label:     "B6",                           // Downlink label
		Text:      "/TEST.ADS.A6-EDK07550BCAA4E2", // Using same hex for test
	}

	fmt.Println("\n=== Testing Downlink Message (Label B6) ===")
	fmt.Println("ADSC Parser:")
	testADSC(msg2)
	fmt.Println("\nEnvelope Parser (may show altitude):")
	testEnvelope(msg2)
}

func testADSC(msg *acars.Message) {
	parser := &adsc.Parser{}
	result := parser.Parse(msg)

	if result == nil {
		fmt.Println("  Parser returned nil")
		return
	}

	hexPart := msg.Text[strings.Index(msg.Text, ".ADS.")+11:]
	fmt.Printf("  Hex: %s\n", hexPart)

	jsonBytes, _ := json.MarshalIndent(result, "  ", "  ")
	fmt.Printf("  %s\n", string(jsonBytes))
}

func testEnvelope(msg *acars.Message) {
	parser := &envelope.Parser{}
	result := parser.Parse(msg)

	if result == nil {
		fmt.Println("  Parser returned nil")
		return
	}

	jsonBytes, _ := json.MarshalIndent(result, "  ", "  ")
	fmt.Printf("  %s\n", string(jsonBytes))
}
