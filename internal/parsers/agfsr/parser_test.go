package agfsr

import (
	"acars_parser/internal/acars"
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
}
