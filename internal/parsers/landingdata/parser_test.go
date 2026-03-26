package landingdata

import (
	"testing"

	"acars_parser/internal/acars"
)

func TestParseEmitsMsgType(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{Text: "LANDING DATA HNL RW 08L\n12245 FT\n777-200 PW4077\n*FLAPS 30*\nTEMP 25C       ALT 29.94\nWIND 089/5 MAG\n421.6 - PLANNED LDG WT\n445.0 - STRUCTURAL\n580.0 - LM\nRWY DRY"}

	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("expected *Result, got %T", res)
	}
	if result.MsgType != "LANDINGDATA" {
		t.Fatalf("MsgType = %q, want %q", result.MsgType, "LANDINGDATA")
	}
}
