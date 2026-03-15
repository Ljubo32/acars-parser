package label83

import (
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
