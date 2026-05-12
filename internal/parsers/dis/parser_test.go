package dis

import (
	"testing"

	"acars_parser/internal/acars"
)

func TestDISParseOFPInfo(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		ID:        1,
		Timestamp: "2026-05-06T00:00:00Z",
		Label:     "RA",
		Text:      "QUDXBEGEK~1DIS01111636\r\nOFP INFO\r\nEK512 DXB-DEL A6ENM: CURRENT OFP NUMBER 15/0/0",
	}

	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("Expected *Result, got %T", res)
	}
	if result.MsgType != "DIS" {
		t.Fatalf("MsgType = %q, want %q", result.MsgType, "DIS")
	}
	if result.Format != "DIS01" {
		t.Fatalf("Format = %q, want %q", result.Format, "DIS01")
	}
	if result.Flight != "UAE512" {
		t.Fatalf("Flight = %q, want %q", result.Flight, "UAE512")
	}
	if result.Route != "DXB-DEL" {
		t.Fatalf("Route = %q, want %q", result.Route, "DXB-DEL")
	}
	if result.Aircraft != "A6ENM" {
		t.Fatalf("Aircraft = %q, want %q", result.Aircraft, "A6ENM")
	}
	if result.OFPNumber != "15/0/0" {
		t.Fatalf("OFPNumber = %q, want %q", result.OFPNumber, "15/0/0")
	}
	if result.DayOfMonth != 11 {
		t.Fatalf("DayOfMonth = %d, want %d", result.DayOfMonth, 11)
	}
	if result.ReportTime != "16:36" {
		t.Fatalf("ReportTime = %q, want %q", result.ReportTime, "16:36")
	}
}

func TestDISRejectsOtherDISMessages(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{Label: "RA", Text: "QUDXBEGEK~1DIS01090001 LIDO WX"}
	if res := parser.Parse(msg); res != nil {
		t.Fatalf("expected nil parse result, got %T", res)
	}
}

func TestDISParseLoadsheetAcceptAck(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		ID:        2,
		Timestamp: "2026-05-06T00:00:00Z",
		Label:     "RA",
		Text:      "QUDXBEGEK~1DIS01110429\r\nLDSHT ACCEPT ACK\r\nFLIGHT NUMBER: EK0651/UAE7P\r\nSECTOR: VCBI-OMDB\r\nFLIGHT DATE: 11",
	}

	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("Expected *Result, got %T", res)
	}
	if result.MsgType != "DIS" {
		t.Fatalf("MsgType = %q, want %q", result.MsgType, "DIS")
	}
	if result.Format != "DIS01" {
		t.Fatalf("Format = %q, want %q", result.Format, "DIS01")
	}
	if result.Flight != "UAE7P" {
		t.Fatalf("Flight = %q, want %q", result.Flight, "UAE7P")
	}
	if result.Route != "VCBI-OMDB" {
		t.Fatalf("Route = %q, want %q", result.Route, "VCBI-OMDB")
	}
	if result.DayOfMonth != 11 {
		t.Fatalf("DayOfMonth = %d, want %d", result.DayOfMonth, 11)
	}
	if result.ReportTime != "04:29" {
		t.Fatalf("ReportTime = %q, want %q", result.ReportTime, "04:29")
	}
	if result.OFPNumber != "" {
		t.Fatalf("OFPNumber = %q, want empty", result.OFPNumber)
	}
	if result.Aircraft != "" {
		t.Fatalf("Aircraft = %q, want empty", result.Aircraft)
	}
}

func TestDISParseFlightSummaryAck(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		ID:        3,
		Timestamp: "2026-05-06T00:00:00Z",
		Label:     "RA",
		Text:      "QUDXBEGEK~1DIS01111650\r\nFLT SUMM ACK\r\nFI EK0809/AN A6-ECO\r\nFLIGHT SUMMARY ACK\r\nRECEIVED\r\nFLIGHT NUMBER: UAE809\r\nSECTOR: OMDB-OEMA\r\nTAKEOFF PILOT: 464134\r\nLANDING PILOT: 446555",
	}

	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("Expected *Result, got %T", res)
	}
	if result.MsgType != "DIS" {
		t.Fatalf("MsgType = %q, want %q", result.MsgType, "DIS")
	}
	if result.Format != "DIS01" {
		t.Fatalf("Format = %q, want %q", result.Format, "DIS01")
	}
	if result.Flight != "UAE809" {
		t.Fatalf("Flight = %q, want %q", result.Flight, "UAE809")
	}
	if result.Route != "OMDB-OEMA" {
		t.Fatalf("Route = %q, want %q", result.Route, "OMDB-OEMA")
	}
	if result.DayOfMonth != 11 {
		t.Fatalf("DayOfMonth = %d, want %d", result.DayOfMonth, 11)
	}
	if result.ReportTime != "16:50" {
		t.Fatalf("ReportTime = %q, want %q", result.ReportTime, "16:50")
	}
	if result.OFPNumber != "" {
		t.Fatalf("OFPNumber = %q, want empty", result.OFPNumber)
	}
	if result.Aircraft != "" {
		t.Fatalf("Aircraft = %q, want empty", result.Aircraft)
	}
}

func TestDISParseOFPInfoNormalisesFlight(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		ID:        4,
		Timestamp: "2026-05-06T00:00:00Z",
		Label:     "RA",
		Text:      "QUDXBEGEK~1DIS01020222\r\nOFP INFO\r\nEK55 DXB-DUS A6EGP: CURRENT OFP NUMBER 15/0/0",
	}

	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("Expected *Result, got %T", res)
	}
	if result.Flight != "UAE55" {
		t.Fatalf("Flight = %q, want %q", result.Flight, "UAE55")
	}
	if result.Route != "DXB-DUS" {
		t.Fatalf("Route = %q, want %q", result.Route, "DXB-DUS")
	}
	if result.Aircraft != "A6EGP" {
		t.Fatalf("Aircraft = %q, want %q", result.Aircraft, "A6EGP")
	}
	if result.OFPNumber != "15/0/0" {
		t.Fatalf("OFPNumber = %q, want %q", result.OFPNumber, "15/0/0")
	}
	if result.ReportTime != "02:22" {
		t.Fatalf("ReportTime = %q, want %q", result.ReportTime, "02:22")
	}
}
