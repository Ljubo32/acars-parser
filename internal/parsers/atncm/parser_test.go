package atncm

import (
	"testing"

	"acars_parser/internal/acars"
)

func TestATNCMParsesRouteFromFlightMetadata(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		Label: "ATNCM",
		Text:  "ATN CM LOGON EVA061 VTBS-LOWW",
		Flight: &acars.Flight{
			ID:                 "EVA061",
			DepartingAirport:   "VTBS",
			DestinationAirport: "LOWW",
		},
	}

	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("Expected *Result, got %T", res)
	}
	if result.FlightID != "EVA061" {
		t.Fatalf("FlightID = %q, want %q", result.FlightID, "EVA061")
	}
	if result.Route != "VTBS-LOWW" {
		t.Fatalf("Route = %q, want %q", result.Route, "VTBS-LOWW")
	}
	if result.HumanReadableText() != "ATN CM LOGON EVA061 VTBS-LOWW" {
		t.Fatalf("HumanReadableText() = %q, want %q", result.HumanReadableText(), "ATN CM LOGON EVA061 VTBS-LOWW")
	}
}

func TestATNCMReturnsNilWithoutRoute(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{Label: "ATNCM", Text: "ATN CM LOGON", Flight: &acars.Flight{ID: "EVA061"}}
	if res := parser.Parse(msg); res != nil {
		t.Fatalf("Expected nil result, got %T", res)
	}
}
