package h2wind

import (
	"testing"

	"acars_parser/internal/acars"
)

func TestParseEmitsMsgType(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{Text: "02A291829EDDKLSZHN50529E007101291809   6M005   48P002290008G"}

	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("expected *Result, got %T", res)
	}
	if result.MsgType != "H2WIND" {
		t.Fatalf("MsgType = %q, want %q", result.MsgType, "H2WIND")
	}
}
