package adsc

import (
	"encoding/json"
	"math"
	"testing"

	"acars_parser/internal/acars"
)

func TestADSCParser(t *testing.T) {
	// Test cases using real messages with valid CRCs.
	tests := []struct {
		name        string
		text        string
		wantType    string
		wantReg     string
		wantStation string
		wantLat     float64
		wantLon     float64
		wantAlt     int
		tolerance   float64
	}{
		{
			name:        "Basic report (F-GXLI)",
			text:        "/XYTGL7X.ADS.F-GXLI0725BFC82D8D46BC46CC1D0D25B0182C2CC745807725965029EF880A40B791",
			wantType:    "basic",
			wantReg:     "F-GXLI",
			wantStation: "XYTGL7X",
			wantLat:     53.08,
			wantLon:     8.01,
			wantAlt:     27588,
			tolerance:   0.1,
		},
		{
			name:        "Basic report (G-ZBKO)",
			text:        "/QUKAXBA.ADS.G-ZBKO072495A7EE7786F6A4D21F7A5D",
			wantType:    "basic",
			wantReg:     "G-ZBKO",
			wantStation: "QUKAXBA",
			wantLat:     51.45,
			wantLon:     -3.08,
			wantAlt:     28520,
			tolerance:   0.1,
		},
		{
			name:        "Basic report with flight prefix (N760GT)",
			text:        "F67A5Y0700/FUKJJYA.ADS.N760GT0724F34BA86989C3C98D1D17231AE3868D09C408AB0D24B2D3A348C9C4013F23B1DB9071C9C4000E54A0E140040F54F1A0C004D45D",
			wantType:    "basic",
			wantReg:     "N760GT",
			wantStation: "FUKJJYA",
			wantLat:     51.96,
			wantLon:     164.60,
			wantAlt:     39996,
			tolerance:   0.1,
		},
		{
			name:        "Basic report (F-GXLO)",
			text:        "/XYTGL7X.ADS.F-GXLO0725A2E02967884D24581D0D25665826E6484D0110254F0025F2884D00815F",
			wantType:    "basic",
			wantReg:     "F-GXLO",
			wantStation: "XYTGL7X",
			wantLat:     52.93,
			wantLon:     7.28,
			wantAlt:     34000,
			tolerance:   0.1,
		},
	}

	p := &Parser{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &acars.Message{
				Label: "B6",
				Text:  tt.text,
			}

			result := p.Parse(msg)
			if result == nil {
				t.Fatalf("Parse returned nil")
			}

			r, ok := result.(*Result)
			if !ok {
				t.Fatalf("Result is not *Result type")
			}

			if r.MessageType != tt.wantType {
				t.Errorf("MessageType = %q, want %q", r.MessageType, tt.wantType)
			}

			if r.Registration != tt.wantReg {
				t.Errorf("Registration = %q, want %q", r.Registration, tt.wantReg)
			}

			if r.GroundStation != tt.wantStation {
				t.Errorf("GroundStation = %q, want %q", r.GroundStation, tt.wantStation)
			}

			if tt.wantLat != 0 {
				if math.Abs(r.Latitude-tt.wantLat) > tt.tolerance {
					t.Errorf("Latitude = %f, want %f (±%f)", r.Latitude, tt.wantLat, tt.tolerance)
				}
			}

			if tt.wantLon != 0 {
				if math.Abs(r.Longitude-tt.wantLon) > tt.tolerance {
					t.Errorf("Longitude = %f, want %f (±%f)", r.Longitude, tt.wantLon, tt.tolerance)
				}
			}

			if tt.wantAlt != 0 {
				if r.Altitude < tt.wantAlt-100 || r.Altitude > tt.wantAlt+100 {
					t.Errorf("Altitude = %d, want %d (±100)", r.Altitude, tt.wantAlt)
				}
			}
		})
	}
}

func TestDecodeCoordinate(t *testing.T) {
	// 21-bit coordinate encoding: MSB weight is 90°, range is approximately ±180°.
	// Value 0x080000 (bit 19 set) = 90°, 0x100000 (bit 20 set) = -180° (sign bit).
	tests := []struct {
		name      string
		raw       uint32
		want      float64
		tolerance float64
	}{
		{"Zero", 0, 0, 0.001},
		{"Positive 90°", 0x080000, 90.0, 0.01},
		{"Negative 90°", 0x180000, -90.0, 0.01},
		{"Max positive ~180°", 0x0FFFFF, 180.0, 0.01},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeCoordinate(tt.raw)
			if math.Abs(got-tt.want) > tt.tolerance {
				t.Errorf("decodeCoordinate(0x%X) = %f, want %f", tt.raw, got, tt.want)
			}
		})
	}
}

func TestParseUplinkPeriodicContractInterval(t *testing.T) {
	p := &Parser{}
	msg := &acars.Message{
		ID:        1,
		Timestamp: "2026-03-15T00:00:00Z",
		Label:     "A6",
		Text:      "/BOMCAYA.ADS.A6-ECW07010BD90D0110011501364FAF",
	}

	result := p.Parse(msg)
	if result == nil {
		t.Fatalf("Parse returned nil")
	}

	r, ok := result.(*Result)
	if !ok {
		t.Fatalf("Result is not *Result type")
	}

	if r.MessageType != "uplink_contract_request" {
		t.Fatalf("MessageType = %q, want %q", r.MessageType, "uplink_contract_request")
	}

	if r.ContractRequest == nil {
		t.Fatalf("ContractRequest is nil")
	}

	if r.ContractRequest.ContractNum != 1 {
		t.Fatalf("ContractNum = %d, want 1", r.ContractRequest.ContractNum)
	}

	if r.ContractRequest.IntervalSecs != 1664 {
		payload, _ := json.Marshal(r)
		t.Fatalf("IntervalSecs = %d, want 1664; result=%s", r.ContractRequest.IntervalSecs, string(payload))
	}

	if r.ContractRequest.Kind != "periodic" {
		payload, _ := json.Marshal(r)
		t.Fatalf("Kind = %q, want %q; result=%s", r.ContractRequest.Kind, "periodic", string(payload))
	}

	if len(r.ContractRequest.Groups) != 4 {
		payload, _ := json.Marshal(r)
		t.Fatalf("len(Groups) = %d, want 4; result=%s", len(r.ContractRequest.Groups), string(payload))
	}

	if got := r.ContractRequest.Groups[1].Name; got != "Predicted route" {
		payload, _ := json.Marshal(r)
		t.Fatalf("Groups[1].Name = %q, want %q; result=%s", got, "Predicted route", string(payload))
	}

	if r.ContractRequest.Groups[1].Modulus == nil || *r.ContractRequest.Groups[1].Modulus != 1 {
		payload, _ := json.Marshal(r)
		t.Fatalf("Groups[1].Modulus = %v, want 1; result=%s", r.ContractRequest.Groups[1].Modulus, string(payload))
	}

	if r.ContractRequest.Groups[3].ProjectionMins == nil || *r.ContractRequest.Groups[3].ProjectionMins != 54 {
		payload, _ := json.Marshal(r)
		t.Fatalf("Groups[3].ProjectionMins = %v, want 54; result=%s", r.ContractRequest.Groups[3].ProjectionMins, string(payload))
	}
}

func TestParseUplinkEventContractGroups(t *testing.T) {
	p := &Parser{}
	msg := &acars.Message{
		ID:        2,
		Timestamp: "2026-03-15T00:00:00Z",
		Label:     "A6",
		Text:      "/MLECAYA.ADS.A6-EPP080312341300660041140A50E6A8",
	}

	result := p.Parse(msg)
	if result == nil {
		t.Fatalf("Parse returned nil")
	}

	r, ok := result.(*Result)
	if !ok {
		t.Fatalf("Result is not *Result type")
	}

	if r.ContractRequest == nil {
		payload, _ := json.Marshal(r)
		t.Fatalf("ContractRequest is nil; result=%s", string(payload))
	}

	if r.ContractRequest.Kind != "event" {
		payload, _ := json.Marshal(r)
		t.Fatalf("Kind = %q, want %q; result=%s", r.ContractRequest.Kind, "event", string(payload))
	}

	if len(r.ContractRequest.Groups) != 4 {
		payload, _ := json.Marshal(r)
		t.Fatalf("len(Groups) = %d, want 4; result=%s", len(r.ContractRequest.Groups), string(payload))
	}

	vertSpeed := r.ContractRequest.Groups[0]
	if vertSpeed.ThresholdFPM == nil || *vertSpeed.ThresholdFPM != 3328 {
		payload, _ := json.Marshal(r)
		t.Fatalf("ThresholdFPM = %v, want 3328; result=%s", vertSpeed.ThresholdFPM, string(payload))
	}
	if vertSpeed.HigherThan == nil || !*vertSpeed.HigherThan {
		payload, _ := json.Marshal(r)
		t.Fatalf("HigherThan = %v, want true; result=%s", vertSpeed.HigherThan, string(payload))
	}

	altitudeRange := r.ContractRequest.Groups[1]
	if altitudeRange.FloorAlt == nil || *altitudeRange.FloorAlt != 260 {
		payload, _ := json.Marshal(r)
		t.Fatalf("FloorAlt = %v, want 260; result=%s", altitudeRange.FloorAlt, string(payload))
	}
	if altitudeRange.CeilingAlt == nil || *altitudeRange.CeilingAlt != 408 {
		payload, _ := json.Marshal(r)
		t.Fatalf("CeilingAlt = %v, want 408; result=%s", altitudeRange.CeilingAlt, string(payload))
	}

	waypoint := r.ContractRequest.Groups[2]
	if !waypoint.ReportWaypointChanges {
		payload, _ := json.Marshal(r)
		t.Fatalf("ReportWaypointChanges = %v, want true; result=%s", waypoint.ReportWaypointChanges, string(payload))
	}

	lateralDeviation := r.ContractRequest.Groups[3]
	if lateralDeviation.ThresholdNM == nil || *lateralDeviation.ThresholdNM != 10 {
		payload, _ := json.Marshal(r)
		t.Fatalf("ThresholdNM = %v, want 10; result=%s", lateralDeviation.ThresholdNM, string(payload))
	}
}

func TestParseDownlinkAirRefMachScale(t *testing.T) {
	p := &Parser{}
	msg := &acars.Message{
		Label: "B6",
		Text:  "/CCUCAYA.ADS.9V-SKU0720A690827089C409C41D172182086E8349C404B00C4C9073C388201176CD750E69E8F580000F6971AB4000100BA7BE5A0D220350615C89C407B522ACF84DFD49C4000826",
	}

	result := p.Parse(msg)
	if result == nil {
		t.Fatal("Parse returned nil")
	}

	r, ok := result.(*Result)
	if !ok {
		t.Fatal("Result is not *Result type")
	}

	if r.AirRef == nil {
		payload, _ := json.Marshal(r)
		t.Fatalf("AirRef is nil; result=%s", string(payload))
	}

	if math.Abs(r.AirRef.Mach-0.8545) > 0.0001 {
		payload, _ := json.Marshal(r)
		t.Fatalf("AirRef.Mach = %.4f, want 0.8545; result=%s", r.AirRef.Mach, string(payload))
	}

	if math.Abs(r.AirRef.Heading-296.542969) > 0.01 {
		payload, _ := json.Marshal(r)
		t.Fatalf("AirRef.Heading = %.6f, want 296.542969; result=%s", r.AirRef.Heading, string(payload))
	}

	if r.ADSCFlightID != "SIA308" {
		payload, _ := json.Marshal(r)
		t.Fatalf("ADSCFlightID = %q, want %q; result=%s", r.ADSCFlightID, "SIA308", string(payload))
	}
}
