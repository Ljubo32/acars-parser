package labelb3

import (
	"testing"

	"acars_parser/internal/acars"
)

func TestParseEmitsMsgType(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{Text: "QFA123-YSSY-GATE A12-YMML ATIS K -TYP/B738"}

	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("expected *Result, got %T", res)
	}
	if result.MsgType != "GATEINFO" {
		t.Fatalf("MsgType = %q, want %q", result.MsgType, "GATEINFO")
	}
}
