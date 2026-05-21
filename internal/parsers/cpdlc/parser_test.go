package cpdlc

import (
	"encoding/hex"
	"encoding/json"
	"math"
	"strings"
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
			name:      "CPDLC message without trailing IMI separator",
			text:      "/OAKODYA.AT1RPC87862616402A49AACE830A64541199B960822558A82A4F41529418D1A4C35882945A2827CE411A4CC8A03B1E",
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

func TestParseAAUplinkRouteClearanceSequence(t *testing.T) {
	parser := &Parser{}

	msg := &acars.Message{
		ID:        1,
		Label:     "AA",
		Text:      "/FIHCAYA.AT1.A6-ECQA0A3A093C4A926641A00180052E3C90213C913B093A0CC9F4EB2E4CEA7220D383D471952374A2D09F4AA208B4EA20971E4A0974E0A0833A220A926641A000207",
		Timestamp: "2026-04-17T08:58:02Z",
	}

	result := parser.Parse(msg)
	if result == nil {
		t.Fatal("Parse() returned nil")
	}

	r := result.(*Result)
	if r.Error != "" {
		t.Fatalf("Parse() error = %q", r.Error)
	}

	if len(r.Elements) != 4 {
		t.Fatalf("Elements count = %d, want 4", len(r.Elements))
	}

	if r.Header == nil || r.Header.Timestamp == nil || r.Header.Timestamp.String() != "08:58:02" {
		t.Fatalf("Header timestamp = %#v, want 08:58:02", r.Header)
	}

	if r.Elements[0].ID != 79 {
		t.Fatalf("First element ID = %d, want 79", r.Elements[0].ID)
	}
	firstData, ok := r.Elements[0].Data.(map[string]interface{})
	if !ok {
		t.Fatalf("first element data type = %T, want map[string]interface{}", r.Elements[0].Data)
	}
	firstPos, ok := firstData["position"].(*Position)
	if !ok || firstPos == nil || firstPos.Name != "TILAP" {
		t.Fatalf("unexpected first position: %#v", firstData["position"])
	}
	firstRoute, ok := firstData["route_clearance"].(*RouteClearance)
	if !ok || firstRoute == nil {
		t.Fatalf("unexpected first route clearance: %#v", firstData["route_clearance"])
	}
	if len(firstRoute.RouteInformation) != 1 || firstRoute.RouteInformation[0].Position == nil || firstRoute.RouteInformation[0].Position.Name != "KGI" {
		t.Fatalf("unexpected first route information: %#v", firstRoute.RouteInformation)
	}
	if len(firstRoute.RouteInfoAdditional) != 0 {
		t.Fatalf("unexpected first route info additional: %#v", firstRoute.RouteInfoAdditional)
	}
	if r.Elements[1].ID != 19 {
		t.Fatalf("Second element ID = %d, want 19", r.Elements[1].ID)
	}
	if r.Elements[2].ID != 118 {
		t.Fatalf("Third element ID = %d, want 118", r.Elements[2].ID)
	}
	if r.Elements[3].ID != 169 {
		t.Fatalf("Fourth element ID = %d, want 169", r.Elements[3].ID)
	}
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
	wantText := "FANS-1/A CPDLC MESSAGE:\n CPDLC UPLINK MESSAGE:\n  HEADER:\n   MSG ID: 3\n   TIMESTAMP: 00:21:10\n  MESSAGE DATA:\n   AT AMPER CONTACT KINSHASA 126.100 MHz\n   LOGON FZZA"
	if r.FormattedText != wantText {
		t.Fatalf("unexpected formatted text: %q", r.FormattedText)
	}
}

func TestParseUplinkContactHFWithFreeTextRegression(t *testing.T) {
	parser := &Parser{}

	msg := &acars.Message{
		ID:        1,
		Label:     "AA",
		Text:      "/MRUCAYA.AT1.ZS-SXDA286F95D991C968D99B366F146C152354E2C39F3A241A5650488C823528B46AC59D0ECA06AD99B405FFF",
		Timestamp: "2026-03-20T00:00:00Z",
	}

	result := parser.Parse(msg)
	if result == nil {
		t.Fatal("Parse() returned nil")
	}

	r := result.(*Result)
	if r.Error != "" {
		t.Fatalf("unexpected parse error: %s", r.Error)
	}
	if r.Header == nil || r.Header.MsgID != 5 {
		t.Fatalf("unexpected header: %+v", r.Header)
	}
	if r.Header.Timestamp == nil || r.Header.Timestamp.Hours != 1 || r.Header.Timestamp.Minutes != 47 || r.Header.Timestamp.Seconds == nil || *r.Header.Timestamp.Seconds != 37 {
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
	if !ok || position == nil || position.Latitude == nil || position.Longitude == nil {
		t.Fatalf("unexpected position: %#v", data["position"])
	}
	if math.Abs(*position.Latitude-(-35.0)) > 0.0001 || math.Abs(*position.Longitude-75.0) > 0.0001 {
		t.Fatalf("unexpected position lat/lon: %v,%v", *position.Latitude, *position.Longitude)
	}
	unit, ok := data["unit"].(string)
	if !ok || unit != "YMMM" {
		t.Fatalf("unexpected unit: %#v", data["unit"])
	}
	unitType, ok := data["unit_type"].(string)
	if !ok || unitType != "control" {
		t.Fatalf("unexpected unit_type: %#v", data["unit_type"])
	}
	freq, ok := data["frequency"].(*Frequency)
	if !ok || freq == nil || freq.Type != "hf" || math.Abs(freq.Value-13306.0) > 0.0001 {
		t.Fatalf("unexpected frequency: %#v", data["frequency"])
	}

	second := r.Elements[1]
	if second.ID != 169 {
		t.Fatalf("expected second element 169, got %d", second.ID)
	}
	freeText, ok := second.Data.(*FreeText)
	if !ok || freeText == nil || freeText.Text != "SECONDARY HF FREQUENCY 5634" {
		t.Fatalf("unexpected freetext: %#v", second.Data)
	}
	wantText := "FANS-1/A CPDLC MESSAGE:\n CPDLC UPLINK MESSAGE:\n  HEADER:\n   MSG ID: 5\n   TIMESTAMP: 01:47:37\n  MESSAGE DATA:\n   AT -35.0000,75.0000 CONTACT YMMM 13306 kHz\n   SECONDARY HF FREQUENCY 5634"
	if r.FormattedText != wantText {
		t.Fatalf("unexpected formatted text: %q", r.FormattedText)
	}
}

func TestParseUplinkErrorInformationWithFreeTextRegression(t *testing.T) {
	parser := &Parser{}

	msg := &acars.Message{
		ID:        1,
		Label:     "AA",
		Text:      "/MGQCAYA.AT1.A6-EBRE0827C849F0152530E844990D04E9F51041AD064CC830A6454106A20A9224D341524CD8AB6AD38A82B4F930E28C406",
		Timestamp: "2026-03-21T00:00:00Z",
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
	if r.Header.MsgRef == nil || *r.Header.MsgRef != 1 {
		t.Fatalf("unexpected msg ref: %+v", r.Header)
	}
	if r.Header.Timestamp == nil || r.Header.Timestamp.Hours != 7 || r.Header.Timestamp.Minutes != 50 || r.Header.Timestamp.Seconds == nil || *r.Header.Timestamp.Seconds != 4 {
		t.Fatalf("unexpected header timestamp: %+v", r.Header.Timestamp)
	}
	if len(r.Elements) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(r.Elements))
	}

	first := r.Elements[0]
	if first.ID != 159 {
		t.Fatalf("expected first element 159, got %d", first.ID)
	}
	errorInfo, ok := first.Data.(*ErrorInfo)
	if !ok || errorInfo == nil {
		t.Fatalf("unexpected first element data: %#v", first.Data)
	}
	if errorInfo.Code != 0 || errorInfo.Desc != "applicationError" {
		t.Fatalf("unexpected error info: %#v", errorInfo)
	}

	second := r.Elements[1]
	if second.ID != 169 {
		t.Fatalf("expected second element 169, got %d", second.ID)
	}
	freeText, ok := second.Data.(*FreeText)
	if !ok || freeText == nil || freeText.Text != "CPDLC NOT AVAILABLE AT THIS TIME-USE VOICE" {
		t.Fatalf("unexpected freetext: %#v", second.Data)
	}
}

func TestParseUplinkFreeTextDotlessAT1Regression(t *testing.T) {
	parser := &Parser{}

	msg := &acars.Message{
		ID:        1,
		Label:     "AA",
		Text:      "/OAKODYA.AT1RPC87862616402A49AACE830A64541199B960822558A82A4F41529418D1A4C35882945A2827CE411A4CC8A03B1E",
		Timestamp: "2026-03-21T00:00:00Z",
	}

	result := parser.Parse(msg)
	if result == nil {
		t.Fatal("Parse() returned nil")
	}

	r := result.(*Result)
	if r.Error != "" {
		t.Fatalf("unexpected parse error: %s", r.Error)
	}
	if r.MessageType != "cpdlc" {
		t.Fatalf("MessageType = %q, want %q", r.MessageType, "cpdlc")
	}
	if r.Registration != "RPC8786" {
		t.Fatalf("Registration = %q, want %q", r.Registration, "RPC8786")
	}
	if r.Header == nil || r.Header.MsgID != 12 {
		t.Fatalf("unexpected header: %+v", r.Header)
	}
	if r.Header.MsgRef != nil {
		t.Fatalf("unexpected msg ref: %+v", r.Header)
	}
	if r.Header.Timestamp == nil || r.Header.Timestamp.Hours != 5 || r.Header.Timestamp.Minutes != 36 || r.Header.Timestamp.Seconds == nil || *r.Header.Timestamp.Seconds != 0 {
		t.Fatalf("unexpected header timestamp: %+v", r.Header.Timestamp)
	}
	if len(r.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(r.Elements))
	}

	first := r.Elements[0]
	if first.ID != 169 {
		t.Fatalf("expected first element 169, got %d", first.ID)
	}
	freeText, ok := first.Data.(*FreeText)
	if !ok || freeText == nil || freeText.Text != "UNABLE F390 DUE TO TRAFFIC, REQ ON FILE" {
		t.Fatalf("unexpected freetext: %#v", first.Data)
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
	wantText := "FANS-1/A CPDLC MESSAGE:\n CPDLC UPLINK MESSAGE:\n  HEADER:\n   MSG ID: 3\n   TIMESTAMP: 00:37:01\n  MESSAGE DATA:\n   SQUAWK 0410"
	if r.FormattedText != wantText {
		t.Fatalf("unexpected formatted text: %q", r.FormattedText)
	}
}

func TestParseUplinkClimbToReachAltitudeByTime(t *testing.T) {
	parser := &Parser{}

	msg := &acars.Message{
		ID:        1,
		Label:     "AA",
		Text:      "/BOMCAYA.AT1.B-20CJ008D6529E000E454",
		Timestamp: "2026-03-17T00:00:00Z",
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
	if len(r.Elements) != 1 {
		t.Fatalf("expected 1 element, got %d", len(r.Elements))
	}

	first := r.Elements[0]
	if first.ID != 26 {
		t.Fatalf("expected first element 26, got %d", first.ID)
	}
	if first.Label != "CLIMB TO REACH [altitude] BY [time]" {
		t.Fatalf("unexpected first label: %q", first.Label)
	}
	data, ok := first.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("expected first element compound data, got %T", first.Data)
	}
	altitude, ok := data["altitude"].(*Altitude)
	if !ok || altitude == nil || altitude.Type != "flight_level" || altitude.Value != 360 {
		t.Fatalf("unexpected altitude: %#v", data["altitude"])
	}
	altitudeType, ok := data["type"].(string)
	if !ok || altitudeType != "flight_level" {
		t.Fatalf("unexpected top-level altitude type: %#v", data["type"])
	}
	altitudeValue, ok := data["value"].(int)
	if !ok || altitudeValue != 360 {
		t.Fatalf("unexpected top-level altitude value: %#v", data["value"])
	}
	timeValue, ok := data["time"].(*Time)
	if !ok || timeValue == nil || timeValue.Hours != 15 || timeValue.Minutes != 0 {
		t.Fatalf("unexpected time: %#v", data["time"])
	}
	if first.Text != "CLIMB TO REACH FL360 BY 15:00" {
		t.Fatalf("unexpected first text: %q", first.Text)
	}
	wantText := "FANS-1/A CPDLC MESSAGE:\n CPDLC UPLINK MESSAGE:\n  HEADER:\n   MSG ID: 1\n  MESSAGE DATA:\n   CLIMB TO REACH FL360 BY 15:00"
	if r.FormattedText != wantText {
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
	wantText := "FANS-1/A CPDLC MESSAGE:\n CPDLC UPLINK MESSAGE:\n  HEADER:\n   MSG ID: 0\n  MESSAGE DATA:\n   VABF labelA"
	if r.FormattedText != wantText {
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
	if len(routeClearance.RouteInformation) != 1 || routeClearance.RouteInformation[0].Kind != "airway" || routeClearance.RouteInformation[0].Airway != "UG862" {
		t.Fatalf("unexpected route information: %#v", routeClearance.RouteInformation)
	}
}

func TestParseUplinkRouteClearanceWithCoordinateRouteInformationRegression(t *testing.T) {
	parser := &Parser{}

	msg := &acars.Message{
		ID:        1,
		Label:     "AA",
		Text:      "/MELCAYA.AT1.A6-EOEA5B3A3EA482945A53EAD48A82A4F41420D283326459882AC18AE1854410A2C8933A209EACCD9B30021243E621169A4122C7834A7828B4E804AD266D5A6129D585566849D3E3C9A11D664DDC8483320D89E066CC07F89",
		Timestamp: "2026-03-21T00:00:00Z",
	}

	result := parser.Parse(msg)
	if result == nil {
		t.Fatal("Parse() returned nil")
	}

	r := result.(*Result)
	if r.Error != "" {
		t.Fatalf("unexpected parse error: %s", r.Error)
	}
	if r.Header == nil || r.Header.MsgID != 11 {
		t.Fatalf("unexpected header: %+v", r.Header)
	}
	if r.Header.Timestamp == nil || r.Header.Timestamp.Hours != 12 || r.Header.Timestamp.Minutes != 58 || r.Header.Timestamp.Seconds == nil || *r.Header.Timestamp.Seconds != 15 {
		t.Fatalf("unexpected header timestamp: %+v", r.Header.Timestamp)
	}
	if len(r.Elements) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(r.Elements))
	}

	first := r.Elements[0]
	if first.ID != 169 {
		t.Fatalf("expected first element 169, got %d", first.ID)
	}
	freeText, ok := first.Data.(*FreeText)
	if !ok || freeText == nil || freeText.Text != "REROUTE TO PARALLEL UAE80T BEHIND" {
		t.Fatalf("unexpected freetext: %#v", first.Data)
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
	if !ok || position == nil || position.Name != "YMML" {
		t.Fatalf("unexpected position: %#v", data["position"])
	}
	routeClearance, ok := data["route_clearance"].(*RouteClearance)
	if !ok || routeClearance == nil {
		t.Fatalf("unexpected route clearance: %#v", data["route_clearance"])
	}
	if len(routeClearance.RouteInformation) != 10 {
		t.Fatalf("len(route information) = %d, want 10; got %#v", len(routeClearance.RouteInformation), routeClearance.RouteInformation)
	}

	firstRoute := routeClearance.RouteInformation[0]
	if firstRoute.Kind != "latlon" || firstRoute.Position == nil || firstRoute.Position.Latitude == nil || firstRoute.Position.Longitude == nil {
		t.Fatalf("unexpected first route element: %#v", firstRoute)
	}
	if math.Abs(*firstRoute.Position.Latitude-(-15.0)) > 0.0001 || math.Abs(*firstRoute.Position.Longitude-98.0) > 0.0001 {
		t.Fatalf("unexpected first route lat/lon: %v,%v", *firstRoute.Position.Latitude, *firstRoute.Position.Longitude)
	}

	secondRoute := routeClearance.RouteInformation[1]
	if secondRoute.Kind != "latlon" || secondRoute.Position == nil || secondRoute.Position.Latitude == nil || secondRoute.Position.Longitude == nil {
		t.Fatalf("unexpected second route element: %#v", secondRoute)
	}
	if math.Abs(*secondRoute.Position.Latitude-(-22.0)) > 0.0001 || math.Abs(*secondRoute.Position.Longitude-105.0) > 0.0001 {
		t.Fatalf("unexpected second route lat/lon: %v,%v", *secondRoute.Position.Latitude, *secondRoute.Position.Longitude)
	}

	wantPublished := []string{"EGARO", "ESP", "VIMUS", "SUBUM", "NOGIP", "ALAXO", "ML"}
	gotPublished := []string{}
	for _, routeInfo := range routeClearance.RouteInformation {
		if routeInfo.Kind == "published_identifier" && routeInfo.Position != nil {
			gotPublished = append(gotPublished, routeInfo.Position.Name)
		}
	}
	if strings.Join(gotPublished, ",") != strings.Join(wantPublished, ",") {
		t.Fatalf("unexpected published identifiers: got %v want %v", gotPublished, wantPublished)
	}

	airwayCount := 0
	for _, routeInfo := range routeClearance.RouteInformation {
		if routeInfo.Kind == "airway" {
			airwayCount++
			if routeInfo.Airway != "V279" {
				t.Fatalf("unexpected airway: %#v", routeInfo)
			}
		}
	}
	if airwayCount != 1 {
		t.Fatalf("airwayCount = %d, want 1; route=%#v", airwayCount, routeClearance.RouteInformation)
	}
}

// TestParseBADownlinkAssignedRoute verifies that a label "BA" (downlink) ASSIGNED ROUTE
// message decodes correctly and produces the expected hierarchical formatted text.
// The input is a real-world ACARS capture; expected output matches the libacars format.
func TestParseBADownlinkAssignedRoute(t *testing.T) {
	parser := &Parser{}

	msg := &acars.Message{
		ID:        1,
		Label:     "BA",
		Text:      "/PIKCPYA.AT1.D-AIKQ20C6D44A13A972A34B542E94EAD665A10694390D1459D1439360524D81E9360A24D4328CA5265C19966F026CA0649CD5BB28232A4FA1429D6998A1B771516745AB3FD465C21C07",
		Timestamp: "2026-01-01T17:45:00Z",
	}

	result := parser.Parse(msg)
	if result == nil {
		t.Fatal("Parse() returned nil")
	}

	r := result.(*Result)
	if r.Error != "" {
		t.Fatalf("unexpected parse error: %s", r.Error)
	}
	if r.Direction != "downlink" {
		t.Fatalf("expected downlink, got %q", r.Direction)
	}
	if r.Registration != "D-AIKQ" {
		t.Fatalf("expected registration D-AIKQ, got %q", r.Registration)
	}
	if r.Header == nil || r.Header.MsgID != 1 {
		t.Fatalf("unexpected header: %+v", r.Header)
	}
	if r.Header.Timestamp == nil || r.Header.Timestamp.Hours != 17 || r.Header.Timestamp.Minutes != 45 {
		t.Fatalf("unexpected header timestamp: %+v", r.Header.Timestamp)
	}
	if len(r.Elements) == 0 {
		t.Fatal("expected at least one element")
	}

	// The first element must be an ASSIGNED ROUTE downlink (dM40).
	first := r.Elements[0]
	if first.ID != 40 {
		t.Fatalf("expected element ID 40 (ASSIGNED ROUTE), got %d", first.ID)
	}
	rc, ok := first.Data.(*RouteClearance)
	if !ok || rc == nil {
		t.Fatalf("expected *RouteClearance data, got %T", first.Data)
	}

	// Verify the fixed fields that are well-established in this message.
	if rc.AirportDestination != "KJFK" {
		t.Fatalf("expected destination KJFK, got %q", rc.AirportDestination)
	}
	if rc.RunwayArrival == nil || rc.RunwayArrival.Direction != 22 {
		t.Fatalf("unexpected arrival runway: %+v", rc.RunwayArrival)
	}
	if rc.RunwayArrival.Configuration != "left" {
		t.Fatalf("expected left configuration, got %q", rc.RunwayArrival.Configuration)
	}
	if rc.ProcedureApproach == nil || rc.ProcedureApproach.Name != "RNVY" {
		t.Fatalf("unexpected approach procedure: %+v", rc.ProcedureApproach)
	}
	if len(rc.RouteInformation) == 0 {
		t.Fatal("expected route information")
	}

	// Log the raw JSON struct for debugging the procedure decode.
	rawJSON, _ := json.MarshalIndent(rc, "", "  ")
	t.Logf("RouteClearance JSON:\n%s", string(rawJSON))

	// Verify the formatted text contains the expected hierarchical content.
	if r.FormattedText == "" {
		t.Fatal("FormattedText is empty")
	}
	mustContain := []string{
		"FANS-1/A CPDLC MESSAGE:",
		"CPDLC DOWNLINK MESSAGE:",
		"MSG ID: 1",
		"ASSIGNED ROUTE [ROUTECLEARANCE]",
		"DESTINATION AIRPORT: KJFK",
		"ARRIVAL RUNWAY:",
		"RUNWAY DIRECTION: 22",
		"RUNWAY CONFIGURATION: LEFT",
		"APPROACH PROCEDURE:",
		"PROCEDURE NAME: RNVY",
		"ROUTE:",
	}
	for _, s := range mustContain {
		if !strings.Contains(r.FormattedText, s) {
			t.Errorf("FormattedText missing %q\nGot:\n%s", s, r.FormattedText)
		}
	}

	t.Logf("Formatted text:\n%s", r.FormattedText)
}

// TestParseBADownlinkAssignedRouteWithAdditionalConstraints verifies that a label "BA"
// (downlink) ASSIGNED ROUTE message containing a FANSRouteInformationAdditional block
// (the waypointSpeedAltitudeSequence optional field) decodes correctly.  The message
// also carries a trailing dM80 element (DEVIATING ... OF ROUTE) to confirm that bit
// consumption by the additional block does not corrupt subsequent elements.
//
// The input is a real-world capture: /PIKCPYA.AT1.G-TUIH ...
func TestParseBADownlinkAssignedRouteWithAdditionalConstraints(t *testing.T) {
	parser := &Parser{}

	msg := &acars.Message{
		ID:        2,
		Label:     "BA",
		Text:      "/PIKCPYA.AT1.G-TUIHAE0E910A3BB97366428B1E846215B324CCA6C998B0A95586C63852720CC9C829094EACE9F5024B8B4E4D809066CC8310642A5561C5B8239057E610F123599D169080C1E22499EC593120489124CF9EE1C90244874ABB26600038A000207200",
		Timestamp: "2026-05-01T10:00:00Z",
	}

	result := parser.Parse(msg)
	if result == nil {
		t.Fatal("Parse() returned nil")
	}

	r := result.(*Result)
	if r.Error != "" {
		t.Fatalf("unexpected parse error: %s", r.Error)
	}
	if r.Direction != "downlink" {
		t.Fatalf("expected downlink, got %q", r.Direction)
	}
	if r.Registration != "G-TUIH" {
		t.Fatalf("expected registration G-TUIH, got %q", r.Registration)
	}

	// The message must decode to exactly two elements: dM40 then dM80.
	if len(r.Elements) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(r.Elements))
	}

	// -- Element 0: dM40 ASSIGNED ROUTE with route-information-additional --

	el0 := r.Elements[0]
	if el0.ID != 40 {
		t.Fatalf("expected element[0] ID 40 (ASSIGNED ROUTE), got %d", el0.ID)
	}
	rc, ok := el0.Data.(*RouteClearance)
	if !ok || rc == nil {
		t.Fatalf("expected *RouteClearance data for element[0], got %T", el0.Data)
	}
	if rc.AirportDeparture != "KMLB" {
		t.Fatalf("expected departure KMLB, got %q", rc.AirportDeparture)
	}
	if rc.AirportDestination != "EGPF" {
		t.Fatalf("expected destination EGPF, got %q", rc.AirportDestination)
	}

	// Verify that origin/destination are lifted to the top-level Result fields so
	// that the state extractor and viewer can read them without navigating elements.
	if r.Origin != "KMLB" {
		t.Fatalf("expected Result.Origin KMLB, got %q", r.Origin)
	}
	if r.Destination != "EGPF" {
		t.Fatalf("expected Result.Destination EGPF, got %q", r.Destination)
	}

	// The additional block must contain exactly 4 waypoint-speed-altitude entries.
	if len(rc.RouteInfoAdditional) != 4 {
		t.Fatalf("expected 4 RouteInfoAdditional entries, got %d", len(rc.RouteInfoAdditional))
	}

	// Spot-check each additional waypoint position name and altitude tolerance.
	type expectedAdditional struct {
		fixName   string
		tolerance string
	}
	expectedAdditionals := []expectedAdditional{
		{"FYNER", "at"},
		{"IO121", "atorabove"},
		{"IOO89", "atorabove"},
		{"RW23", "at"},
	}
	for i, exp := range expectedAdditionals {
		entry := rc.RouteInfoAdditional[i]
		if entry.Position == nil {
			t.Errorf("additional[%d]: Position is nil", i)
			continue
		}
		if entry.Position.Name != exp.fixName {
			t.Errorf("additional[%d]: expected fix %q, got %q", i, exp.fixName, entry.Position.Name)
		}
		if len(entry.Altitudes) == 0 {
			t.Errorf("additional[%d]: no altitudes decoded", i)
			continue
		}
		if entry.Altitudes[0].Tolerance != exp.tolerance {
			t.Errorf("additional[%d]: expected tolerance %q, got %q", i, exp.tolerance, entry.Altitudes[0].Tolerance)
		}
	}

	// -- Element 1: dM80 DEVIATING ... OF ROUTE --

	el1 := r.Elements[1]
	if el1.ID != 80 {
		t.Fatalf("expected element[1] ID 80 (DEVIATING ... OF ROUTE), got %d", el1.ID)
	}

	rawJSON, _ := json.MarshalIndent(rc, "", "  ")
	t.Logf("RouteClearance JSON:\n%s", string(rawJSON))
	t.Logf("Formatted text:\n%s", r.FormattedText)
}
