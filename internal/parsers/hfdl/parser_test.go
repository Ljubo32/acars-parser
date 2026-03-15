package hfdl

import (
	"testing"

	"acars_parser/internal/acars"
)

func TestParserParsesSyntheticFrequencyDataMessage(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		Label:     "HFDL",
		Text:      "HFDL Frequency data GS 17",
		Timestamp: "2026-03-13T12:03:31.316679Z",
		Source:    "dumphfdl",
		Flight: &acars.Flight{
			ID:        "UAE7CN",
			Latitude:  40.077171,
			Longitude: 21.178934,
		},
		Airframe: &acars.Airframe{ICAO: "8963E6"},
	}

	parsed := parser.Parse(msg)
	result, ok := parsed.(*Result)
	if !ok || result == nil {
		t.Fatalf("expected HFDL result, got %T", parsed)
	}
	if result.FlightID != "UAE7CN" {
		t.Fatalf("flight_id = %q, want UAE7CN", result.FlightID)
	}
	if result.Latitude != 40.077171 || result.Longitude != 21.178934 {
		t.Fatalf("unexpected coords = %v,%v", result.Latitude, result.Longitude)
	}
	if result.ICAO != "8963E6" {
		t.Fatalf("icao = %q, want 8963E6", result.ICAO)
	}
	if result.HFNPDUType != "Frequency data" {
		t.Fatalf("hfnpdu_type = %q, want Frequency data", result.HFNPDUType)
	}
}

func TestParserParsesSyntheticPerformanceDataMessage(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		Label:     "HFDL",
		Text:      "HFDL Performance data GS 17",
		Timestamp: "2026-03-13T12:05:04.683998Z",
		Source:    "dumphfdl",
		Flight: &acars.Flight{
			ID:        "KQA116",
			Latitude:  31.324828,
			Longitude: 6.373456,
		},
		Airframe: &acars.Airframe{ICAO: "70605A"},
	}

	parsed := parser.Parse(msg)
	result, ok := parsed.(*Result)
	if !ok || result == nil {
		t.Fatalf("expected HFDL result, got %T", parsed)
	}
	if result.FlightID != "KQA116" {
		t.Fatalf("flight_id = %q, want KQA116", result.FlightID)
	}
	if result.HFNPDUType != "Performance data" {
		t.Fatalf("hfnpdu_type = %q, want Performance data", result.HFNPDUType)
	}
	if result.Latitude != 31.324828 || result.Longitude != 6.373456 {
		t.Fatalf("unexpected coords = %v,%v", result.Latitude, result.Longitude)
	}
}
