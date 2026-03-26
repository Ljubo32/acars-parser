package gateassign

import (
	"testing"

	"acars_parser/internal/acars"
)

func TestParseEmitsMsgType(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{Text: "GATE ASSIGNMENT: A12 PPOS:305 BAG BELT:206 NEXT LEG: LA3709 BPS-BSB"}

	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("expected *Result, got %T", res)
	}
	if result.MsgType != "GATEASSIGN" {
		t.Fatalf("MsgType = %q, want %q", result.MsgType, "GATEASSIGN")
	}
}
