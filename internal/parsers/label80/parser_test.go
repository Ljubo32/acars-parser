package label80

import (
	"acars_parser/internal/acars"
	"testing"
)

// TestPOSFullFields tests the full POS message format that includes wind,
// temperature, TAS and ETA fields in addition to the base position fields.
func TestPOSFullFields(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		Text: "POSHAAB/DGAA/LATN08406/LONE037312/ALT282/FOB32171/TME0636/WND -34 7/OAT-24/TAS469/ETA1121",
	}

	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("Expected *Result, got %T", res)
	}

	if result.MsgType != "POS" {
		t.Errorf("MsgType: got %q, want %q", result.MsgType, "POS")
	}
	if result.OriginICAO != "HAAB" {
		t.Errorf("OriginICAO: got %q, want %q", result.OriginICAO, "HAAB")
	}
	if result.DestICAO != "DGAA" {
		t.Errorf("DestICAO: got %q, want %q", result.DestICAO, "DGAA")
	}

	// Latitude: N 08.406° → 8.406
	const wantLat = 8.406
	if result.Latitude < wantLat-0.001 || result.Latitude > wantLat+0.001 {
		t.Errorf("Latitude: got %f, want ~%f", result.Latitude, wantLat)
	}

	// Longitude: E 037.312° → 37.312
	const wantLon = 37.312
	if result.Longitude < wantLon-0.001 || result.Longitude > wantLon+0.001 {
		t.Errorf("Longitude: got %f, want ~%f", result.Longitude, wantLon)
	}

	if result.Altitude != 282 {
		t.Errorf("Altitude: got %d, want 282 (FL282)", result.Altitude)
	}
	if result.FuelOnBoard != 32171 {
		t.Errorf("FuelOnBoard: got %d, want 32171", result.FuelOnBoard)
	}
	if result.ReportTime != "06:36" {
		t.Errorf("ReportTime: got %q, want %q", result.ReportTime, "06:36")
	}

	// WND -34 7 → direction = 360 + (-34) = 326°, speed = 7 kt → 13 km/h
	if result.WindDir != 326 {
		t.Errorf("WindDir: got %d, want 326 (offset -34 from 360)", result.WindDir)
	}
	if result.WindSpeed != 7 {
		t.Errorf("WindSpeed: got %d, want 7", result.WindSpeed)
	}
	// 7 kt × 1.852 = 12.964 → rounded to 13 km/h
	if result.WindSpeedKmh != 13 {
		t.Errorf("WindSpeedKmh: got %d, want 13 (7 kt converted)", result.WindSpeedKmh)
	}

	if result.OAT != -24 {
		t.Errorf("OAT: got %d, want -24", result.OAT)
	}
	if result.TAS != 469 {
		t.Errorf("TAS: got %d, want 469 kt", result.TAS)
	}
	// 469 kt × 1.852 = 868.588 → rounded to 869 km/h
	if result.TASKmh != 869 {
		t.Errorf("TASKmh: got %d, want 869 (469 kt converted)", result.TASKmh)
	}
	if result.ETA != "11:21" {
		t.Errorf("ETA: got %q, want %q", result.ETA, "11:21")
	}
}

// TestPOSPositiveWindDir tests that a positive WND offset is used as the
// actual wind direction without modification.
func TestPOSPositiveWindDir(t *testing.T) {
	parser := &Parser{}
	// WND 170 12 → direction = 170°, speed = 12 kt
	msg := &acars.Message{
		Text: "POSZSPD/HAAB/LATN09298/LONE042494/ALT400/FOB11998/TME0308/WND 170 12/OAT-55/TAS488/ETA0345",
	}

	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("Expected *Result, got %T", res)
	}

	if result.WindDir != 170 {
		t.Errorf("WindDir: got %d, want 170 (positive offset used directly)", result.WindDir)
	}
	if result.WindSpeed != 12 {
		t.Errorf("WindSpeed: got %d, want 12", result.WindSpeed)
	}
	// 12 kt × 1.852 = 22.224 → rounded to 22 km/h
	if result.WindSpeedKmh != 22 {
		t.Errorf("WindSpeedKmh: got %d, want 22 (12 kt converted)", result.WindSpeedKmh)
	}
}

// TestPOSShortForm tests the base POS message format that has no TME/WND/OAT/TAS/ETA.
// These shorter messages are common in the log files.
func TestPOSShortForm(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		Text: "POSDGAA/HAAB/LATN07533/LONE033290/ALT370/FOB37963",
	}

	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("Expected *Result, got %T", res)
	}

	if result.MsgType != "POS" {
		t.Errorf("MsgType: got %q, want %q", result.MsgType, "POS")
	}
	if result.OriginICAO != "DGAA" {
		t.Errorf("OriginICAO: got %q, want %q", result.OriginICAO, "DGAA")
	}
	if result.DestICAO != "HAAB" {
		t.Errorf("DestICAO: got %q, want %q", result.DestICAO, "HAAB")
	}

	// Latitude: N 07.533°
	const wantLat = 7.533
	if result.Latitude < wantLat-0.001 || result.Latitude > wantLat+0.001 {
		t.Errorf("Latitude: got %f, want ~%f", result.Latitude, wantLat)
	}
	if result.Altitude != 370 {
		t.Errorf("Altitude: got %d, want 370 (FL370)", result.Altitude)
	}
	if result.FuelOnBoard != 37963 {
		t.Errorf("FuelOnBoard: got %d, want 37963", result.FuelOnBoard)
	}
	// Optional fields should be absent (zero value).
	if result.ReportTime != "" {
		t.Errorf("ReportTime: got %q, want empty", result.ReportTime)
	}
	if result.TAS != 0 {
		t.Errorf("TAS: got %d, want 0 (not present)", result.TAS)
	}
}
