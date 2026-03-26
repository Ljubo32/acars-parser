package ini

import (
	"testing"

	"acars_parser/internal/acars"
)

func TestINIParse(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		ID:        1,
		Timestamp: "2026-03-20T00:00:00Z",
		Label:     "RA",
		Text:      "QUDXBEGEK~1INI01091501 UAE810 /09/OEMA/OMDB/398948/616142/616142/ /",
	}

	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("Expected *Result, got %T", res)
	}
	if result.MsgType != "INI" {
		t.Fatalf("MsgType = %q, want %q", result.MsgType, "INI")
	}
	if result.Format != "INI01" {
		t.Fatalf("Format = %q, want %q", result.Format, "INI01")
	}
	if result.Flight != "UAE810" {
		t.Fatalf("Flight = %q, want %q", result.Flight, "UAE810")
	}
	if result.DayOfMonth != 9 {
		t.Fatalf("DayOfMonth = %d, want %d", result.DayOfMonth, 9)
	}
	if result.ReportTime != "15:01" {
		t.Fatalf("ReportTime = %q, want %q", result.ReportTime, "15:01")
	}
	if result.Origin != "OEMA" {
		t.Fatalf("Origin = %q, want %q", result.Origin, "OEMA")
	}
	if result.Destination != "OMDB" {
		t.Fatalf("Destination = %q, want %q", result.Destination, "OMDB")
	}
}

func TestINIRejectsInvalidMessages(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{Label: "RA", Text: "QUDXBEGEK~1DIS01090001 LIDO WX"}
	if res := parser.Parse(msg); res != nil {
		t.Fatalf("expected nil parse result, got %T", res)
	}
}
