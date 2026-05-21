package label83

import (
	"math"
	"testing"

	"acars_parser/internal/acars"
)

func TestLabel83FlightLevel(t *testing.T) {
	parser := &Parser{}
	tests := []struct {
		name   string
		text   string
		expect int
	}{
		{
			name:   "FL370 from 370465",
			text:   "001PR16121136N5102.0E02023.0370465",
			expect: 370,
		},
		{
			name:   "FL221 from 221388",
			text:   "001PR16132212N4319.9E01941.52213880012",
			expect: 221,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := &acars.Message{Text: tc.text}
			res := parser.Parse(msg)
			result, ok := res.(*Result)
			if !ok {
				t.Fatalf("Expected *Result, got %T", res)
			}
			if result.FlightLevel != tc.expect {
				t.Errorf("Expected flight level %d, got %d", tc.expect, result.FlightLevel)
			}
	// Altitude field is removed, so nothing to check here
		})
	}
}

// TestLabel83CSVPosition covers CSV-format position messages (ICAO,ICAO,HHMMSS,…)
// which arrive without a "ZSPD" prefix.  Key differences from the existing ZSPD
// tests: signed coordinates carry an internal space (e.g. "- 29.34"), and the
// flight level must be derived from feet by dividing by 100 (not 1000).
func TestLabel83CSVPosition(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		origin      string
		dest        string
		lat         float64
		lon         float64
		flightLevel int
		heading     int
	}{
		{
			// Western longitude encoded with an internal space after the minus sign.
			name:        "west_lon_with_space",
			text:        "KORD,EGLL,130317, 56.01,- 29.34,39001,265,  93.2, 16400",
			origin:      "KORD",
			dest:        "EGLL",
			lat:         56.01,
			lon:         -29.34,
			flightLevel: 390,
			heading:     265,
		},
		{
			// Eastern longitude, positive — verifies the existing ZSPD path is
			// unaffected by the pattern change.
			name:        "east_lon_positive",
			text:        "WSSS,EGLL,130107, 41.12,  47.94,38002,272,- 87.7, 30800",
			origin:      "WSSS",
			dest:        "EGLL",
			lat:         41.12,
			lon:         47.94,
			flightLevel: 380,
			heading:     272,
		},
	}

	parser := &Parser{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &acars.Message{Text: tt.text, Label: "83"}
			res := parser.Parse(msg)
			result, ok := res.(*Result)
			if !ok {
				t.Fatalf("expected *Result, got %T", res)
			}
			if result.Origin != tt.origin {
				t.Errorf("origin = %q, want %q", result.Origin, tt.origin)
			}
			if result.Destination != tt.dest {
				t.Errorf("dest = %q, want %q", result.Destination, tt.dest)
			}
			if math.Abs(result.Latitude-tt.lat) > 0.001 {
				t.Errorf("latitude = %.4f, want %.4f", result.Latitude, tt.lat)
			}
			if math.Abs(result.Longitude-tt.lon) > 0.001 {
				t.Errorf("longitude = %.4f, want %.4f", result.Longitude, tt.lon)
			}
			if result.FlightLevel != tt.flightLevel {
				t.Errorf("flight_level = %d, want %d", result.FlightLevel, tt.flightLevel)
			}
			if result.Heading != tt.heading {
				t.Errorf("heading = %d, want %d", result.Heading, tt.heading)
			}
		})
	}
}

// TestLabel83POSRPT covers the slash-delimited POSRPT position report format.
func TestLabel83POSRPT(t *testing.T) {
	const text = "3N01 POSRPT 0182/SKBO/LEMD .N783AV/03A03:40/WPY N34W0/NWYP MANOX " +
		"/HDG  64.80/MCH    .86/POS N29304 W029014/FL 40000/TAS 495" +
		"/SAT - 52/SWND    67/DWND - 103/FOB 19500/ETA 0624"

	parser := &Parser{}
	msg := &acars.Message{Text: text, Label: "83"}
	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("expected *Result, got %T", res)
	}

	if result.MessageType != "POSRPT" {
		t.Errorf("message_type = %q, want \"POSRPT\"", result.MessageType)
	}
	if result.Origin != "SKBO" {
		t.Errorf("origin = %q, want \"SKBO\"", result.Origin)
	}
	if result.Destination != "LEMD" {
		t.Errorf("destination = %q, want \"LEMD\"", result.Destination)
	}
	if result.Tail != "N783AV" {
		t.Errorf("tail = %q, want \"N783AV\"", result.Tail)
	}
	if result.ReportTime != "03:40" {
		t.Errorf("report_time = %q, want \"03:40\"", result.ReportTime)
	}

	// N29304 → N 29°30.4′ = 29 + 30.4/60 ≈ 29.507°N
	if math.Abs(result.Latitude-29.507) > 0.001 {
		t.Errorf("latitude = %.4f, want ≈29.507", result.Latitude)
	}
	// W029014 → W 029°01.4′ = -(29 + 1.4/60) ≈ -29.023°W
	if math.Abs(result.Longitude-(-29.023)) > 0.001 {
		t.Errorf("longitude = %.4f, want ≈-29.023", result.Longitude)
	}

	if result.FlightLevel != 400 {
		t.Errorf("flight_level = %d, want 400", result.FlightLevel)
	}
	// HDG 64.80 rounds to 65.
	if result.Heading != 65 {
		t.Errorf("heading = %d, want 65", result.Heading)
	}
	// TAS 495 kts stored as ground speed.
	if math.Abs(result.GroundSpeed-495) > 0.1 {
		t.Errorf("ground_speed = %.1f, want 495", result.GroundSpeed)
	}
}
