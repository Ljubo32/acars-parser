package loadsheet

import (
	"testing"

	"acars_parser/internal/acars"
)

func TestLoadsheetParseEmitsMsgType(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		ID:        acars.FlexInt64(42),
		Timestamp: "2026-03-20T00:00:00Z",
		Tail:      "9H-VJG",
		Label:     "C1",
		Text:      "LOADSHEET U21234/001 123ABC456 LTN DUB AIRCRAFT TYPE: A320 ZFW 62000 TOW 70100 LAW 64500 TOF 8100 TTL: 176 CREW: 2/4 EDNO 7",
	}

	if !parser.QuickCheck(msg.Text) {
		t.Fatal("QuickCheck() = false, want true")
	}

	parsed := parser.Parse(msg)
	if parsed == nil {
		t.Fatal("Parse() returned nil")
	}

	result, ok := parsed.(*Result)
	if !ok {
		t.Fatalf("Parse() returned %T, want *Result", parsed)
	}

	assertStringEqual(t, "msg_type", result.MsgType, "LOADSHEET")
	assertStringEqual(t, "tail", result.Tail, "9H-VJG")
	assertIntEqual(t, "zfw", result.ZFW, 62000)
	assertIntEqual(t, "tow", result.TOW, 70100)
	assertIntEqual(t, "law", result.LAW, 64500)
	assertIntEqual(t, "tof", result.TOF, 8100)
	assertIntEqual(t, "pax", result.PAX, 176)
	assertStringEqual(t, "crew", result.Crew, "2/4")
	assertStringEqual(t, "aircraft_type", result.AircraftType, "A320")
	assertStringEqual(t, "edition", result.Edition, "7")
	if result.MessageID() != 42 {
		t.Fatalf("message_id = %d, want 42", result.MessageID())
	}
}

func TestLoadsheetQuickCheckAcceptsSpacedKeyword(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		Label: "H1",
		Text:  "L O A D S H E E T FINAL EDNO 1 ZFW 60158 TOW 70000 TTL: 180",
	}

	if !parser.QuickCheck(msg.Text) {
		t.Fatal("QuickCheck() = false, want true for spaced LOADSHEET keyword")
	}

	parsed := parser.Parse(msg)
	if parsed == nil {
		t.Fatal("Parse() returned nil for spaced LOADSHEET keyword")
	}

	result, ok := parsed.(*Result)
	if !ok {
		t.Fatalf("Parse() returned %T, want *Result", parsed)
	}
	assertStringEqual(t, "msg_type", result.MsgType, "LOADSHEET")
	assertIntEqual(t, "zfw", result.ZFW, 60158)
	assertIntEqual(t, "tow", result.TOW, 70000)
	assertIntEqual(t, "pax", result.PAX, 180)
}

func TestLoadsheetLabelsAreGlobal(t *testing.T) {
	parser := &Parser{}
	if labels := parser.Labels(); len(labels) != 0 {
		t.Fatalf("Labels() = %v, want global parser with no labels", labels)
	}
}

func TestLoadsheetRejectsWithoutUsefulData(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		Label: "C1",
		Text:  "LOADSHEET HEADER ONLY",
	}

	if parsed := parser.Parse(msg); parsed != nil {
		t.Fatalf("Parse() returned %T, want nil", parsed)
	}
}

func assertStringEqual(t *testing.T, field, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %q, want %q", field, got, want)
	}
}

func assertIntEqual(t *testing.T, field string, got, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %d, want %d", field, got, want)
	}
}
