package labelb2

import (
	"testing"

	"acars_parser/internal/acars"
)

func TestParseEmitsMsgType(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{Text: "DAL123 CLRD TO EGLL 50N030W F350 M84"}

	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("expected *Result, got %T", res)
	}
	if result.MsgType != "OCEANIC_CLEARANCE" {
		t.Fatalf("MsgType = %q, want %q", result.MsgType, "OCEANIC_CLEARANCE")
	}
}
