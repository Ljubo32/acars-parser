package cpdlc

import (
	"encoding/hex"
	"math"
	"testing"

	"acars_parser/internal/acars"
)

func TestQuickCheck(t *testing.T) {
	parser := &Parser{}

	tests := []struct {
		name      string
		text      string
		wantMatch bool
	}{
		{
			name:      "CPDLC message",
			text:      "/PIKCPYA.AT1.F-GSQC214823E24092E7",
			wantMatch: true,
		},
		{
			name:      "Connect request",
			text:      "/NYCODYA.CR1.N784AV12345678",
			wantMatch: true,
		},
		{
			name:      "Connect confirm",
			text:      "/YQXD2YA.CC1.TC-LLH12345678",
			wantMatch: true,
		},
		{
			name:      "Disconnect",
			text:      "/KZDCAYA.DR1.N12345",
			wantMatch: true,
		},
		{
			name:      "Non-CPDLC",
			text:      "Some random ACARS text",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parser.QuickCheck(tt.text); got != tt.wantMatch {
				t.Errorf("QuickCheck() = %v, want %v", got, tt.wantMatch)
			}
		})
	}
}

func TestParse(t *testing.T) {
	parser := &Parser{}

	tests := []struct {
		name         string
		label        string
		text         string
		wantType     string
		wantDir      string
		wantElements int
		wantError    bool
	}{
		{
			name:         "Uplink contact with free text",
			label:        "AA",
			text:         "/BZVCAYA.AT1.5Y-KZGA181529D848336845A55972675391069C1EA38C2A42664F8F3E7208D6AD410C11F",
			wantType:     "cpdlc",
			wantDir:      "uplink",
			wantElements: 2,
		},
		{
			name:     "Connect request keeps AA uplink direction",
			label:    "AA",
			text:     "/NYCODYA.CR1.N784AVABCD1234",
			wantType: "connect_request",
			wantDir:  "uplink",
		},
		{
			name:         "Connect request with facility designation and TP4 table",
			label:        "AA",
			text:         "/BOMCAYA.CR1.A6-EQF0051D6830A30637A",
			wantType:     "connect_request",
			wantDir:      "uplink",
			wantElements: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &acars.Message{
				ID:        1,
				Label:     tt.label,
				Text:      tt.text,
				Timestamp: "2024-01-01T00:00:00Z",
			}

			result := parser.Parse(msg)
			if result == nil {
				t.Fatal("Parse() returned nil")
			}

			r := result.(*Result)
			if r.MessageType != tt.wantType {
				t.Errorf("MessageType = %v, want %v", r.MessageType, tt.wantType)
			}
			if r.Direction != tt.wantDir {
				t.Errorf("Direction = %v, want %v", r.Direction, tt.wantDir)
			}
			if tt.wantElements > 0 && len(r.Elements) != tt.wantElements {
				t.Errorf("Elements count = %d, want %d", len(r.Elements), tt.wantElements)
			}
			if tt.wantError && r.Error == "" {
				t.Error("Expected error but got none")
			}
		})
	}
}

func TestBitReader(t *testing.T) {
	// Test basic bit reading.
	data := []byte{0xAB, 0xCD} // 1010 1011 1100 1101
	br := NewBitReader(data)

	// Read 4 bits - should be 1010 = 10.
	v, err := br.ReadBits(4)
	if err != nil {
		t.Fatalf("ReadBits(4) error: %v", err)
	}
	if v != 10 {
		t.Errorf("ReadBits(4) = %d, want 10", v)
	}

	// Read 4 more bits - should be 1011 = 11.
	v, err = br.ReadBits(4)
	if err != nil {
		t.Fatalf("ReadBits(4) error: %v", err)
	}
	if v != 11 {
		t.Errorf("ReadBits(4) = %d, want 11", v)
	}

	// Read 8 more bits - should be 1100 1101 = 205.
	v, err = br.ReadBits(8)
	if err != nil {
		t.Fatalf("ReadBits(8) error: %v", err)
	}
	if v != 0xCD {
		t.Errorf("ReadBits(8) = %d, want 205", v)
	}

	// Should have 0 bits remaining.
	if br.Remaining() != 0 {
		t.Errorf("Remaining() = %d, want 0", br.Remaining())
	}
}

func TestConstrainedInt(t *testing.T) {
	// Test constrained integer reading.
	// Range 0-7 needs 3 bits.
	data := []byte{0b10100000} // 101 = 5 in first 3 bits.
	br := NewBitReader(data)

	v, err := br.ReadConstrainedInt(0, 7)
	if err != nil {
		t.Fatalf("ReadConstrainedInt error: %v", err)
	}
	if v != 5 {
		t.Errorf("ReadConstrainedInt(0,7) = %d, want 5", v)
	}
}

func TestIsValidHex(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"ABCD1234", true},
		{"abcd1234", true},
		{"0123456789ABCDEF", true},
		{"ABC", false},  // Odd length.
		{"GHIJ", false}, // Invalid chars.
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := isValidHex(tt.input); got != tt.want {
				t.Errorf("isValidHex(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSplitRegistrationAndData(t *testing.T) {
	tests := []struct {
		input   string
		wantReg string
		wantHex string
	}{
		{"F-GSQC214823E24092E7", "F-GSQC", "214823E24092E7"},
		{"N784AV22C823E840FBCE", "N784AV", "22C823E840FBCE"},
		// Full message from database (hex must be even length).
		{"TC-LLH2148242A526A48934D049A6820CE4106AD49F360D48B1104D8B4E9C18F150549E821CF9D1A4D29A821D089321A0873E754830EA20AF26A48411E0CE8916920893E6C5A7524C39201", "TC-LLH", "2148242A526A48934D049A6820CE4106AD49F360D48B1104D8B4E9C18F150549E821CF9D1A4D29A821D089321A0873E754830EA20AF26A48411E0CE8916920893E6C5A7524C39201"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			gotReg, gotHex := splitRegistrationAndData(tt.input)
			if gotReg != tt.wantReg {
				t.Errorf("reg = %q, want %q", gotReg, tt.wantReg)
			}
			if gotHex != tt.wantHex {
				t.Errorf("hex = %q, want %q", gotHex, tt.wantHex)
			}
		})
	}
}

func TestDecodeElementID(t *testing.T) {
	// Test that a known-good AA uplink sample decodes to the expected first element ID.
	parser := &Parser{}

	msg := &acars.Message{
		ID:        1,
		Label:     "AA",
		Text:      "/BZVCAYA.AT1.5Y-KZGA181529D848336845A55972675391069C1EA38C2A42664F8F3E7208D6AD410C11F",
		Timestamp: "2024-01-01T00:00:00Z",
	}

	result := parser.Parse(msg)
	if result == nil {
		t.Fatal("Parse() returned nil")
	}

	r := result.(*Result)
	if r.MessageType != "cpdlc" {
		t.Fatalf("MessageType = %v, want cpdlc", r.MessageType)
	}

	if len(r.Elements) == 0 {
		t.Fatal("No elements decoded")
	}

	elem := r.Elements[0]
	if elem.ID != 118 {
		t.Errorf("Element ID = %d, want 118", elem.ID)
	}

	// Verify the label matches.
	if elem.Label != "AT [position] CONTACT [icaounitname] [frequency]" {
		t.Errorf("Element Label = %q, want 'AT [position] CONTACT [icaounitname] [frequency]'", elem.Label)
	}

	t.Logf("Decoded element: ID=%d, Label=%s, Text=%s", elem.ID, elem.Label, elem.Text)
}

func TestDecodePositionReportDM48(t *testing.T) {
	// Raw hex payload (incl. FCS at the end in the ACARS message; our parser trims 2 bytes already).
	// This is a downlink (label BA) CPDLC dM48 POSITION REPORT and should decode to a populated PositionReport.
	rawHex := "20B2C90C3D903BAE2D1141ECCB325824E8B4A249686255AD06655B3041390B6B09360D693499564B009A26"

	b, err := hex.DecodeString(rawHex)
	if err != nil {
		t.Fatalf("hex decode: %v", err)
	}
	if len(b) < 3 {
		t.Fatalf("payload too short")
	}
	// Trim FCS (2 bytes) like the parser does.
	b = b[:len(b)-2]

	d := NewDecoder(b, DirectionDownlink)
	msg, err := d.Decode()
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if msg == nil {
		t.Fatalf("nil msg")
	}
	if msg.Header.MsgID != 1 {
		t.Fatalf("unexpected header: %+v", msg.Header)
	}
	if msg.Header.Timestamp == nil || msg.Header.Timestamp.Hours != 12 || msg.Header.Timestamp.Minutes != 44 {
		t.Fatalf("unexpected header timestamp: %+v", msg.Header.Timestamp)
	}
	if len(msg.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(msg.Elements))
	}
	el := msg.Elements[0]
	if el.ID != 48 {
		t.Fatalf("expected element 48, got %d", el.ID)
	}
	pr, ok := el.Data.(*PositionReport)
	if !ok || pr == nil {
		t.Fatalf("expected PositionReport data, got %T", el.Data)
	}

	// Spot-check key fields against libacars reference decode.
	if pr.PosCurrent == nil || pr.PosCurrent.Latitude == nil || pr.PosCurrent.Longitude == nil {
		t.Fatalf("missing pos_current lat/lon: %+v", pr.PosCurrent)
	}
	if math.Abs(*pr.PosCurrent.Latitude-46.3) > 0.01 || math.Abs(*pr.PosCurrent.Longitude-20.205) > 0.01 {
		t.Fatalf("unexpected pos_current lat/lon: %v,%v", *pr.PosCurrent.Latitude, *pr.PosCurrent.Longitude)
	}
	if pr.TimeAtPosCurrent == nil || pr.TimeAtPosCurrent.Hours != 12 || pr.TimeAtPosCurrent.Minutes != 44 {
		t.Fatalf("unexpected time_at_pos_current: %+v", pr.TimeAtPosCurrent)
	}
	if pr.Alt == nil || pr.Alt.Type != "flight_level" || pr.Alt.Value != 330 {
		t.Fatalf("unexpected alt: %+v", pr.Alt)
	}
	if pr.NextFix == nil || pr.NextFix.Name != "NERDI" {
		t.Fatalf("unexpected next_fix: %+v", pr.NextFix)
	}
	if pr.NextNextFix == nil || pr.NextNextFix.Name != "UVALU" {
		t.Fatalf("unexpected next_next_fix: %+v", pr.NextNextFix)
	}
	if pr.Temp == nil || pr.Temp.Type != "C" || math.Abs(pr.Temp.Value-(-48.0)) > 0.001 {
		t.Fatalf("unexpected temp: %+v", pr.Temp)
	}
	if pr.Winds == nil || pr.Winds.Direction != 314 || pr.Winds.Speed == nil || pr.Winds.Speed.Type != "kts" || pr.Winds.Speed.Value != 22 {
		t.Fatalf("unexpected winds: %+v", pr.Winds)
	}
	if pr.Speed == nil || pr.Speed.Type != "mach" || pr.Speed.Value != 83 {
		t.Fatalf("unexpected speed: %+v", pr.Speed)
	}
	if pr.ReportedWptPos == nil || pr.ReportedWptPos.Name != "MAVIR" {
		t.Fatalf("unexpected reported_wpt_pos: %+v", pr.ReportedWptPos)
	}
	if pr.ReportedWptTime == nil || pr.ReportedWptTime.Hours != 12 || pr.ReportedWptTime.Minutes != 42 {
		t.Fatalf("unexpected reported_wpt_time: %+v", pr.ReportedWptTime)
	}
	if pr.ReportedWptAlt == nil || pr.ReportedWptAlt.Type != "flight_level" || pr.ReportedWptAlt.Value != 330 {
		t.Fatalf("unexpected reported_wpt_alt: %+v", pr.ReportedWptAlt)
	}
}

func TestParseUplinkContactWithFreeText(t *testing.T) {
	parser := &Parser{}

	msg := &acars.Message{
		ID:        1,
		Label:     "AA",
		Text:      "/BZVCAYA.AT1.5Y-KZGA181529D848336845A55972675391069C1EA38C2A42664F8F3E7208D6AD410C11F",
		Timestamp: "2026-03-09T00:21:14Z",
	}

	result := parser.Parse(msg)
	if result == nil {
		t.Fatal("Parse() returned nil")
	}

	r := result.(*Result)
	if r.Error != "" {
		t.Fatalf("unexpected parse error: %s", r.Error)
	}
	if r.Header == nil || r.Header.Timestamp == nil || r.Header.Timestamp.Seconds == nil {
		t.Fatalf("missing header timestamp seconds: %+v", r.Header)
	}
	if *r.Header.Timestamp.Seconds != 10 {
		t.Fatalf("unexpected header timestamp: %+v", r.Header.Timestamp)
	}
	if len(r.Elements) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(r.Elements))
	}

	first := r.Elements[0]
	if first.ID != 118 {
		t.Fatalf("expected first element 118, got %d", first.ID)
	}
	data, ok := first.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected first element compound data, got %T", first.Data)
	}
	position, ok := data["position"].(*Position)
	if !ok || position == nil || position.Name != "AMPER" {
		t.Fatalf("unexpected position: %#v", data["position"])
	}
	unit, ok := data["unit"].(string)
	if !ok || unit != "KINSHASA" {
		t.Fatalf("unexpected unit: %#v", data["unit"])
	}
	unitType, ok := data["unit_type"].(string)
	if !ok || unitType != "control" {
		t.Fatalf("unexpected unit_type: %#v", data["unit_type"])
	}
	freq, ok := data["frequency"].(*Frequency)
	if !ok || freq == nil || freq.Type != "vhf" || math.Abs(freq.Value-126.100) > 0.0001 {
		t.Fatalf("unexpected frequency: %#v", data["frequency"])
	}

	second := r.Elements[1]
	if second.ID != 169 {
		t.Fatalf("expected second element 169, got %d", second.ID)
	}
	freeText, ok := second.Data.(*FreeText)
	if !ok || freeText == nil || freeText.Text != "LOGON FZZA" {
		t.Fatalf("unexpected freetext: %#v", second.Data)
	}
	if r.FormattedText != "LOGON FZZA" {
		t.Fatalf("unexpected formatted text: %q", r.FormattedText)
	}
}

func TestParseUplinkSquawkBeaconCode(t *testing.T) {
	parser := &Parser{}

	msg := &acars.Message{
		ID:        1,
		Label:     "AA",
		Text:      "/JNBCAYA.AT1.ZS-SXD2182505EC420D8B4",
		Timestamp: "2026-03-09T00:37:04Z",
	}

	result := parser.Parse(msg)
	if result == nil {
		t.Fatal("Parse() returned nil")
	}

	r := result.(*Result)
	if r.Error != "" {
		t.Fatalf("unexpected parse error: %s", r.Error)
	}
	if r.Header == nil || r.Header.MsgID != 3 {
		t.Fatalf("unexpected header: %+v", r.Header)
	}
	if r.Header.Timestamp == nil || r.Header.Timestamp.Hours != 0 || r.Header.Timestamp.Minutes != 37 || r.Header.Timestamp.Seconds == nil || *r.Header.Timestamp.Seconds != 1 {
		t.Fatalf("unexpected header timestamp: %+v", r.Header.Timestamp)
	}
	if len(r.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(r.Elements))
	}

	first := r.Elements[0]
	if first.ID != 123 {
		t.Fatalf("expected first element 123, got %d", first.ID)
	}
	if first.Label != "SQUAWK [beaconcode]" {
		t.Fatalf("unexpected first label: %q", first.Label)
	}
	beacon, ok := first.Data.(*BeaconCode)
	if !ok || beacon == nil || beacon.Code != "0410" {
		t.Fatalf("unexpected beacon data: %#v", first.Data)
	}
	if first.Text != "SQUAWK 0410" {
		t.Fatalf("unexpected first text: %q", first.Text)
	}
	if r.FormattedText != "SQUAWK 0410" {
		t.Fatalf("unexpected formatted text: %q", r.FormattedText)
	}
}

func TestParseConnectRequestFacilityDesignationTP4Table(t *testing.T) {
	parser := &Parser{}

	msg := &acars.Message{
		ID:        1,
		Label:     "AA",
		Text:      "/BOMCAYA.CR1.A6-EQF0051D6830A30637A",
		Timestamp: "2026-03-09T00:47:04Z",
	}

	result := parser.Parse(msg)
	if result == nil {
		t.Fatal("Parse() returned nil")
	}

	r := result.(*Result)
	if r.Error != "" {
		t.Fatalf("unexpected parse error: %s", r.Error)
	}
	if r.MessageType != "connect_request" {
		t.Fatalf("unexpected message type: %q", r.MessageType)
	}
	if r.Header == nil || r.Header.MsgID != 0 {
		t.Fatalf("unexpected header: %+v", r.Header)
	}
	if len(r.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(r.Elements))
	}
	first := r.Elements[0]
	if first.ID != 163 {
		t.Fatalf("expected first element 163, got %d", first.ID)
	}
	if first.Label != "[icaofacilitydesignation] [tp4table]" {
		t.Fatalf("unexpected first label: %q", first.Label)
	}
	data, ok := first.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map data, got %T", first.Data)
	}
	facilityDesignation, ok := data["facility_designation"].(string)
	if !ok || facilityDesignation != "VABF" {
		t.Fatalf("unexpected facility designation: %#v", data["facility_designation"])
	}
	tp4Table, ok := data["tp4_table"].(string)
	if !ok || tp4Table != "labelA" {
		t.Fatalf("unexpected TP4 table: %#v", data["tp4_table"])
	}
	if first.Text != "VABF labelA" {
		t.Fatalf("unexpected first text: %q", first.Text)
	}
	if r.FormattedText != "VABF labelA" {
		t.Fatalf("unexpected formatted text: %q", r.FormattedText)
	}
}

func TestParseUplinkMaintainAndClearedViaRoute(t *testing.T) {
	parser := &Parser{}

	msg := &acars.Message{
		ID:        1,
		Label:     "AA",
		Text:      "/BZVCAYA.AT1.5Y-KZEA0C45784F2D02789066D08B4803012558EE1B32000DEB0",
		Timestamp: "2026-03-09T17:05:34Z",
	}

	result := parser.Parse(msg)
	if result == nil {
		t.Fatal("Parse() returned nil")
	}

	r := result.(*Result)
	if r.Error != "" {
		t.Fatalf("unexpected parse error: %s", r.Error)
	}
	if r.Header == nil || r.Header.MsgID != 1 {
		t.Fatalf("unexpected header: %+v", r.Header)
	}
	if r.Header.Timestamp == nil || r.Header.Timestamp.Hours != 17 || r.Header.Timestamp.Minutes != 5 || r.Header.Timestamp.Seconds == nil || *r.Header.Timestamp.Seconds != 30 {
		t.Fatalf("unexpected header timestamp: %+v", r.Header.Timestamp)
	}
	if len(r.Elements) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(r.Elements))
	}

	first := r.Elements[0]
	if first.ID != 19 {
		t.Fatalf("expected first element 19, got %d", first.ID)
	}
	altitude, ok := first.Data.(*Altitude)
	if !ok || altitude == nil || altitude.Type != "flight_level" || altitude.Value != 390 {
		t.Fatalf("unexpected altitude: %#v", first.Data)
	}

	second := r.Elements[1]
	if second.ID != 79 {
		t.Fatalf("expected second element 79, got %d", second.ID)
	}
	data, ok := second.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected second element map data, got %T", second.Data)
	}
	position, ok := data["position"].(*Position)
	if !ok || position == nil || position.Name != "AMPER" {
		t.Fatalf("unexpected position: %#v", data["position"])
	}
	routeClearance, ok := data["route_clearance"].(*RouteClearance)
	if !ok || routeClearance == nil {
		t.Fatalf("unexpected route clearance: %#v", data["route_clearance"])
	}
	if len(routeClearance.RouteInformation) != 1 || routeClearance.RouteInformation[0] != "UG862" {
		t.Fatalf("unexpected route information: %#v", routeClearance.RouteInformation)
	}
}
