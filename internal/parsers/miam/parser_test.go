package miam

import (
	"testing"

	"acars_parser/internal/acars"
)

// buildMIAMBlock constructs the multi-line MIAM block format that JAERO/libacars
// writes for a "MA" label message.  innerMessage is placed verbatim under the
// "Message:" heading.
func buildMIAMBlock(aircraftID, innerLabel, innerMessage string) string {
	return "MIAM:\n Single Transfer:\n  MIAM CORE Data, version 1:\n" +
		"   PDU Length: 500\n" +
		"   Aircraft ID: ." + aircraftID + "\n" +
		"   Msg num: 133\n" +
		"   ACK: not required\n" +
		"   Compression: none\n" +
		"   Encoding: ISO #5\n" +
		"   ACARS:\n" +
		"    Label: " + innerLabel + "\n" +
		"    Message:\n" +
		"     " + innerMessage + "\n"
}

// ---------------------------------------------------------------------------
// parseRTRRoute
// ---------------------------------------------------------------------------

// TestParseRTRRouteBasic confirms that a well-formed RTR inner message yields
// the correct origin, destination, and flight number.
func TestParseRTRRouteBasic(t *testing.T) {
	inner := `<RTR><HEAD><DCMSAD><FROM>WSSS</FROM><TO>EGKK</TO><FNBR>SIA312    </FNBR></DCMSAD></HEAD></RTR>`

	origin, dest, flight := parseRTRRoute(inner)

	if origin != "WSSS" {
		t.Errorf("expected origin WSSS, got %q", origin)
	}
	if dest != "EGKK" {
		t.Errorf("expected destination EGKK, got %q", dest)
	}
	if flight != "SIA312" {
		t.Errorf("expected flight SIA312, got %q", flight)
	}
}

// TestParseRTRRouteGarbledTags confirms that extraction still works when open
// tags have extra prefix characters (e.g. <IFROM> instead of <FROM>) or close
// tags are corrupted (e.g. </HO> instead of </TO>).  This reflects real
// bit-error corruption seen in ACARS transmissions.
//
// Reference message: 9V-SMQ, SIA312, 2026-05-13, route WSSS→EGKK.
func TestParseRTRRouteGarbledTags(t *testing.T) {
	// Realistic garbled inner message: open tags for FROM and FNBR have an
	// extra 'I' prefix; the close tag for TO is corrupted from </TO> to </HO>.
	inner := `<RTR><HEAD><IDCMS PN=ABF31A6FNCL0004 VERS=4.0/><DA>2026/05/13 01:08:51</DA>` +
		`<DCMS PDCMSAD><IFROM>WSSS</FROM><TO>EGKK</HO<IFNBR>SIA312    </FNBR></DCMSAD></HEAD></RTR>`

	origin, dest, flight := parseRTRRoute(inner)

	if origin != "WSSS" {
		t.Errorf("expected origin WSSS, got %q", origin)
	}
	if dest != "EGKK" {
		t.Errorf("expected destination EGKK, got %q", dest)
	}
	if flight != "SIA312" {
		t.Errorf("expected flight SIA312, got %q", flight)
	}
}

// TestParseRTRRouteMissingFNBR confirms that origin and destination are still
// extracted when the <FNBR> tag is absent.
func TestParseRTRRouteMissingFNBR(t *testing.T) {
	inner := `<RTR><HEAD><DCMSAD><FROM>EGLL</FROM><TO>KEWR</TO></DCMSAD></HEAD></RTR>`

	origin, dest, flight := parseRTRRoute(inner)

	if origin != "EGLL" {
		t.Errorf("expected origin EGLL, got %q", origin)
	}
	if dest != "KEWR" {
		t.Errorf("expected destination KEWR, got %q", dest)
	}
	if flight != "" {
		t.Errorf("expected empty flight, got %q", flight)
	}
}

// TestParseRTRRouteInvalidCodes confirms that non-ICAO airport codes are
// discarded; the function should return empty strings for both airports.
func TestParseRTRRouteInvalidCodes(t *testing.T) {
	inner := `<RTR><HEAD><DCMSAD><FROM>SIN</FROM><TO>LHR</TO></DCMSAD></HEAD></RTR>`

	origin, dest, _ := parseRTRRoute(inner)

	if origin != "" || dest != "" {
		t.Errorf("expected empty origin/dest for 3-letter IATA codes, got %q / %q", origin, dest)
	}
}

// TestParseRTRRouteIgnoresNonRTR confirms that a REP inner message is not
// mistakenly matched by parseRTRRoute.
func TestParseRTRRouteIgnoresNonRTR(t *testing.T) {
	inner := `/REP/H02,ZGSZ FAOR,CCA867 ,S0385,/NX,ZGSZ FAOR/0,7,-068033,...`

	origin, dest, flight := parseRTRRoute(inner)

	if origin != "" || dest != "" || flight != "" {
		t.Errorf("expected no output for REP message, got %q / %q / %q", origin, dest, flight)
	}
}

// ---------------------------------------------------------------------------
// parseMIAMBlock (integration)
// ---------------------------------------------------------------------------

// TestParseMIAMBlockRTR confirms that a MIAM Data message whose inner payload
// is an RTR report has its origin and destination surfaced in the Result.
func TestParseMIAMBlockRTR(t *testing.T) {
	inner := `<RTR><HEAD><DCMSAD><FROM>WSSS</FROM><TO>EGKK</TO><FNBR>SIA312</FNBR></DCMSAD></HEAD></RTR>`
	block := buildMIAMBlock("9V-SMQ", "H1", inner)

	msg := &acars.Message{
		ID:    1,
		Label: "MA",
		Text:  block,
	}

	p := &Parser{}
	raw := p.Parse(msg)
	if raw == nil {
		t.Fatal("Parse returned nil")
	}
	r, ok := raw.(*Result)
	if !ok {
		t.Fatalf("expected *Result, got %T", raw)
	}

	if r.MessageType != "miam_data" {
		t.Errorf("expected miam_data, got %q", r.MessageType)
	}
	if r.OriginICAO != "WSSS" {
		t.Errorf("expected OriginICAO WSSS, got %q", r.OriginICAO)
	}
	if r.DestICAO != "EGKK" {
		t.Errorf("expected DestICAO EGKK, got %q", r.DestICAO)
	}
	if r.FlightNum != "SIA312" {
		t.Errorf("expected FlightNum SIA312, got %q", r.FlightNum)
	}
}

// TestParseMIAMBlockREPNotBroken confirms that the existing REP route
// extraction path is unaffected by the introduction of RTR support.
func TestParseMIAMBlockREPNotBroken(t *testing.T) {
	inner := `/REP/H02,ZGSZ FAOR,CCA867 ,S0385,/NX,ZGSZ FAOR/0,7,-068033,+170352,0,4000,-544,257,039`
	block := buildMIAMBlock("B-HNK", "H1", inner)

	msg := &acars.Message{
		ID:    2,
		Label: "MA",
		Text:  block,
	}

	p := &Parser{}
	raw := p.Parse(msg)
	if raw == nil {
		t.Fatal("Parse returned nil")
	}
	r, ok := raw.(*Result)
	if !ok {
		t.Fatalf("expected *Result, got %T", raw)
	}

	if r.OriginICAO != "ZGSZ" {
		t.Errorf("expected OriginICAO ZGSZ, got %q", r.OriginICAO)
	}
	if r.DestICAO != "FAOR" {
		t.Errorf("expected DestICAO FAOR, got %q", r.DestICAO)
	}
	if r.FlightNum != "CCA867" {
		t.Errorf("expected FlightNum CCA867, got %q", r.FlightNum)
	}
}

// ---------------------------------------------------------------------------
// extractXMLTagValue
// ---------------------------------------------------------------------------

// TestExtractXMLTagValue covers the common extraction cases, including the
// garbled-tag scenarios seen in real ACARS transmissions.
func TestExtractXMLTagValue(t *testing.T) {
	tests := []struct {
		name  string
		input string
		tag   string
		want  string
	}{
		{
			name:  "exact match",
			input: "<FROM>WSSS</FROM>",
			tag:   "FROM",
			want:  "WSSS",
		},
		{
			name:  "case-insensitive tag",
			input: "<from>WSSS</from>",
			tag:   "FROM",
			want:  "WSSS",
		},
		{
			name:  "surrounding text",
			input: "<HEAD><FROM>EGKK</FROM></HEAD>",
			tag:   "FROM",
			want:  "EGKK",
		},
		{
			name:  "value with whitespace trimmed",
			input: "<FNBR>SIA312    </FNBR>",
			tag:   "FNBR",
			want:  "SIA312",
		},
		{
			name:  "absent tag",
			input: "<RTR><TO>EGKK</TO></RTR>",
			tag:   "FROM",
			want:  "",
		},
		// Garbled open tag: <IFROM> instead of <FROM>.
		// Strategy 1 (close-tag-first) handles this: finds </FROM>, scans
		// backward past the '>' of <IFROM>, extracts the content.
		{
			name:  "garbled open tag prefix",
			input: "<IFROM>WSSS</FROM>",
			tag:   "FROM",
			want:  "WSSS",
		},
		// Garbled close tag: </HO> instead of </TO>.
		// Strategy 2 (open-tag-first) handles this: finds <TO>, takes content
		// up to the next '<'.
		{
			name:  "garbled close tag",
			input: "<TO>EGKK</HO",
			tag:   "TO",
			want:  "EGKK",
		},
		// Both garbled open and close (degenerate — no extraction possible).
		{
			name:  "both tags garbled",
			input: "<ITO>EGKK</HO",
			tag:   "TO",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractXMLTagValue(tt.input, tt.tag)
			if got != tt.want {
				t.Errorf("extractXMLTagValue(%q, %q) = %q, want %q", tt.input, tt.tag, got, tt.want)
			}
		})
	}
}
