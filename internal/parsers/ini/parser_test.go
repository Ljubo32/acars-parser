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

func TestINIParseIDFormatWithAFOrigin(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		ID:        2,
		Timestamp: "2026-05-04T00:00:00Z",
		Label:     "RA",
		Text:      "INI/ID80003A,BRK59,COLLINSAM123/MR0,0/AFKBWI,KJFK/TD011137,11373847",
	}

	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("Expected *Result, got %T", res)
	}
	if result.Format != "INI/ID" {
		t.Fatalf("Format = %q, want %q", result.Format, "INI/ID")
	}
	if result.Flight != "BRK59" {
		t.Fatalf("Flight = %q, want %q", result.Flight, "BRK59")
	}
	if result.Origin != "KBWI" {
		t.Fatalf("Origin = %q, want %q", result.Origin, "KBWI")
	}
	if result.Destination != "KJFK" {
		t.Fatalf("Destination = %q, want %q", result.Destination, "KJFK")
	}
	if result.DayOfMonth != 1 {
		t.Fatalf("DayOfMonth = %d, want %d", result.DayOfMonth, 1)
	}
	if result.ReportTime != "11:37" {
		t.Fatalf("ReportTime = %q, want %q", result.ReportTime, "11:37")
	}
}

func TestINIParseIDFormatWithNonAFOrigin(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		ID:        3,
		Timestamp: "2026-05-04T00:00:00Z",
		Label:     "RA",
		Text:      "INI/ID77189A,RCH858,AAM182301027/MR0,0/AFOJAM,ETAR/TD010545,0548041C",
	}

	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("Expected *Result, got %T", res)
	}
	if result.Flight != "RCH858" {
		t.Fatalf("Flight = %q, want %q", result.Flight, "RCH858")
	}
	if result.Origin != "OJAM" {
		t.Fatalf("Origin = %q, want %q", result.Origin, "OJAM")
	}
	if result.Destination != "ETAR" {
		t.Fatalf("Destination = %q, want %q", result.Destination, "ETAR")
	}
	if result.ReportTime != "05:45" {
		t.Fatalf("ReportTime = %q, want %q", result.ReportTime, "05:45")
	}
}

func TestINIParseIDFormatWithLongFlight(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		ID:        4,
		Timestamp: "2026-05-04T00:00:00Z",
		Label:     "RA",
		Text:      "INI/ID88197A,RCH4563,AVM8470W1030/MR1,0/AFUBBB,ETAR/TD310945,1025FEA6",
	}

	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("Expected *Result, got %T", res)
	}
	if result.Flight != "RCH4563" {
		t.Fatalf("Flight = %q, want %q", result.Flight, "RCH4563")
	}
	if result.Origin != "UBBB" {
		t.Fatalf("Origin = %q, want %q", result.Origin, "UBBB")
	}
	if result.Destination != "ETAR" {
		t.Fatalf("Destination = %q, want %q", result.Destination, "ETAR")
	}
	if result.DayOfMonth != 31 {
		t.Fatalf("DayOfMonth = %d, want %d", result.DayOfMonth, 31)
	}
	if result.ReportTime != "09:45" {
		t.Fatalf("ReportTime = %q, want %q", result.ReportTime, "09:45")
	}
}
