package agfsr

import (
	"acars_parser/internal/acars"
	"strings"
	"testing"
)

func TestAGFSRHeadingGroundSpeedTemperature(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{Text: "AGFSR AC0042/15/16/YYZDEL/1438Z/702/4531.1N02459.2E/349/CRUISE/0526/0698/M60/037036/0294/121/483/0605/0646/----/----"}
	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("Expected *Result, got %T", res)
	}
	if result.Heading != 121 {
		t.Errorf("Expected heading 121, got %d", result.Heading)
	}
	if result.GroundSpeed != 483 {
		t.Errorf("Expected ground speed 483, got %d", result.GroundSpeed)
	}
	if result.Temperature != -60 {
		t.Errorf("Expected temperature -60, got %d", result.Temperature)
	}
	if result.MsgType != "AGFSR" {
		t.Errorf("Expected msg_type AGFSR, got %q", result.MsgType)
	}
}

// TestAGFSRNormalisesIATARoute verifies that the 6-character IATA route pair
// is split and converted to ICAO codes.  LTN and DUB are used because their
// ICAO equivalents (EGGW and EIDW) are present in the airports database and
// are verified by the airports package's own test suite.
func TestAGFSRNormalisesIATARoute(t *testing.T) {
	parser := &Parser{}
	// Route field "LTNDUB" → origin LTN (EGGW) + destination DUB (EIDW).
	msg := &acars.Message{Text: "AGFSR AC0123/01/01/LTNDUB/1200Z/738/5130.0N00200.0W/350/CRUISE/0400/0300/M50/270050/0490/090/500/1400/1500/----/----"}
	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("Expected *Result, got %T", res)
	}

	if result.Origin != "EGGW" {
		t.Errorf("Origin: got %q, want %q (IATA LTN → ICAO EGGW)", result.Origin, "EGGW")
	}
	if result.Destination != "EIDW" {
		t.Errorf("Destination: got %q, want %q (IATA DUB → ICAO EIDW)", result.Destination, "EIDW")
	}

	// Route should be reconstructed as an ICAO pair with a dash separator.
	if !strings.Contains(result.Route, "-") {
		t.Errorf("Route %q: expected a dash separator", result.Route)
	}
	if result.Route != "EGGW-EIDW" {
		t.Errorf("Route: got %q, want %q", result.Route, "EGGW-EIDW")
	}
}
