package cpdlc

import (
	"encoding/hex"
	"testing"
)

// TestDecodePositionReportOOSFC verifies decoding of a real-world dM48 POSITION REPORT
// from OO-SFC (Brussels Airlines) over central Africa.
//
// This message exercises opt[10-13]: speedground, verticalChange, trackAngle, trueHeading.
//
// Expected output (per libacars reference and bit-level trace):
//   - Position: 8°36'N, 22°27.7'E
//   - Time at position: 22:36
//   - Altitude: FL360
//   - Next: KITRA ETA 22:45, ERESA, dest ETA 05:10
//   - Temp: -46°C, Winds: 165°/9kts, Speed: M.82
//   - Speed ground: 490 kts
//   - Vertical change: down, 0 ft/min
//   - Track angle: 318° true
//   - True heading: 318° true
//   - Reported waypoint: ONUDA at 22:32, FL360
func TestDecodePositionReportOOSFC(t *testing.T) {
	// The last two bytes are the FCS and must be trimmed before decoding.
	raw := "20DA4D0C3D9F3B885A11645569329424B9352941B5A245A5169C129444A404EAD5019EE7A24F9D56241B4194A09FE1"
	data, _ := hex.DecodeString(raw)
	data = data[:len(data)-2]

	d := NewDecoder(data, DirectionDownlink)
	msg, err := d.Decode()
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if len(msg.Elements) < 1 {
		t.Fatalf("Expected at least 1 element, got %d", len(msg.Elements))
	}
	elem := msg.Elements[0]
	pr, ok := elem.Data.(*PositionReport)
	if !ok || pr == nil {
		t.Fatalf("Element[0].Data is %T, want *PositionReport", elem.Data)
	}

	// Current position: 8°36'N, 22°27.7'E.
	if pr.PosCurrent == nil || pr.PosCurrent.Latitude == nil || pr.PosCurrent.Longitude == nil {
		t.Fatal("PosCurrent lat/lon is nil")
	}
	if *pr.PosCurrent.Latitude < 8.55 || *pr.PosCurrent.Latitude > 8.65 {
		t.Errorf("Latitude: got %.4f, want ~8.6", *pr.PosCurrent.Latitude)
	}
	if *pr.PosCurrent.Longitude < 22.45 || *pr.PosCurrent.Longitude > 22.48 {
		t.Errorf("Longitude: got %.4f, want ~22.46", *pr.PosCurrent.Longitude)
	}

	// Time at current position: 22:36.
	if pr.TimeAtPosCurrent == nil || pr.TimeAtPosCurrent.Hours != 22 || pr.TimeAtPosCurrent.Minutes != 36 {
		t.Errorf("TimeAtPosCurrent: got %v, want 22:36", pr.TimeAtPosCurrent)
	}

	// Altitude: FL360.
	if pr.Alt == nil || pr.Alt.Type != "flight_level" || pr.Alt.Value != 360 {
		t.Errorf("Alt: got %v, want FL360", pr.Alt)
	}

	// Next fix: KITRA ETA 22:45.
	if pr.NextFix == nil || pr.NextFix.Name != "KITRA" {
		t.Errorf("NextFix: got %v, want KITRA", pr.NextFix)
	}
	if pr.EtaAtFixNext == nil || pr.EtaAtFixNext.Hours != 22 || pr.EtaAtFixNext.Minutes != 45 {
		t.Errorf("EtaAtFixNext: got %v, want 22:45", pr.EtaAtFixNext)
	}

	// Next+1 fix: ERESA, destination ETA 05:10.
	if pr.NextNextFix == nil || pr.NextNextFix.Name != "ERESA" {
		t.Errorf("NextNextFix: got %v, want ERESA", pr.NextNextFix)
	}
	if pr.EtaAtDest == nil || pr.EtaAtDest.Hours != 5 || pr.EtaAtDest.Minutes != 10 {
		t.Errorf("EtaAtDest: got %v, want 05:10", pr.EtaAtDest)
	}

	// Temperature: -46°C.
	if pr.Temp == nil || pr.Temp.Type != "C" || pr.Temp.Value != -46 {
		t.Errorf("Temp: got %v, want -46C", pr.Temp)
	}

	// Winds: 165°, 9 kts.
	if pr.Winds == nil || pr.Winds.Direction != 165 {
		t.Errorf("Winds.Direction: got %v, want 165", pr.Winds)
	}
	if pr.Winds == nil || pr.Winds.Speed == nil || pr.Winds.Speed.Value != 9 {
		t.Errorf("Winds.Speed: got %v, want 9 kts", pr.Winds.Speed)
	}

	// Speed: Mach 0.82.
	if pr.Speed == nil || pr.Speed.Type != "mach" || pr.Speed.Value != 82 {
		t.Errorf("Speed: got %v, want mach 0.82", pr.Speed)
	}

	// Speed ground: 490 kts (opt[10] — FANSGroundSpeedKnots direct integer).
	if pr.SpeedGround == nil {
		t.Fatal("SpeedGround is nil")
	}
	if pr.SpeedGround.Type != "knots" || pr.SpeedGround.Value != 490 {
		t.Errorf("SpeedGround: got %+v, want {knots 490}", pr.SpeedGround)
	}

	// Vertical change: down, 0 ft/min (opt[11]).
	if pr.VertChange == nil {
		t.Fatal("VertChange is nil")
	}
	if pr.VertChange.Direction != "down" {
		t.Errorf("VertChange.Direction: got %q, want \"down\"", pr.VertChange.Direction)
	}
	if pr.VertChange.Rate == nil || pr.VertChange.Rate.Value != 0 {
		t.Errorf("VertChange.Rate: got %v, want 0 ft/min", pr.VertChange.Rate)
	}

	// Track angle: 318° true (opt[12]).
	if pr.TrackAngle == nil {
		t.Fatal("TrackAngle is nil")
	}
	if pr.TrackAngle.Value != 318 || pr.TrackAngle.Magnetic {
		t.Errorf("TrackAngle: got {Value:%d Magnetic:%v}, want {318 false}", pr.TrackAngle.Value, pr.TrackAngle.Magnetic)
	}

	// True heading: 318° true (opt[13]).
	if pr.TrueHeading == nil {
		t.Fatal("TrueHeading is nil")
	}
	if pr.TrueHeading.Value != 318 || pr.TrueHeading.Magnetic {
		t.Errorf("TrueHeading: got {Value:%d Magnetic:%v}, want {318 false}", pr.TrueHeading.Value, pr.TrueHeading.Magnetic)
	}

	// Reported waypoint: ONUDA at 22:32, FL360 (opt[16-18]).
	if pr.ReportedWptPos == nil || pr.ReportedWptPos.Name != "ONUDA" {
		t.Errorf("ReportedWptPos: got %v, want ONUDA", pr.ReportedWptPos)
	}
	if pr.ReportedWptTime == nil || pr.ReportedWptTime.Hours != 22 || pr.ReportedWptTime.Minutes != 32 {
		t.Errorf("ReportedWptTime: got %v, want 22:32", pr.ReportedWptTime)
	}
	if pr.ReportedWptAlt == nil || pr.ReportedWptAlt.Type != "flight_level" || pr.ReportedWptAlt.Value != 360 {
		t.Errorf("ReportedWptAlt: got %v, want FL360", pr.ReportedWptAlt)
	}
}

// TestDecodePositionReportHBJND verifies decoding of a real-world dM48 POSITION REPORT
// from HB-JND (Swiss A321neo) en route over the Atlantic at ~02:55 UTC.
//
// Expected output (per libacars reference):
//   - Position: 10°59.6'N, 020°11.6'W
//   - Time at position: 02:55
//   - Altitude: FL320
//   - Next: GANDO ETA 03:47, ERETU, dest ETA 08:23
//   - Temp: -34°C, Winds: 283°/16kts, Speed: M.83
//   - Reported waypoint: TURUP at 02:50, FL320
func TestDecodePositionReportHBJND(t *testing.T) {
	raw := "A48B7C8C3D903B8A951141D22DF244247833A24F1DE245A516A5542E5D1A086B0952AD2AB405964884323A7520CE8926747410ACA08D3E920A716643833104391161CB413E72070E1B31A3D5"
	data, _ := hex.DecodeString(raw)
	data = data[:len(data)-2]

	d := NewDecoder(data, DirectionDownlink)
	msg, err := d.Decode()
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if msg.Direction != DirectionDownlink {
		t.Errorf("Direction: got %v, want DirectionDownlink", msg.Direction)
	}
	if len(msg.Elements) < 1 {
		t.Fatalf("Expected at least 1 element, got %d", len(msg.Elements))
	}
	elem := msg.Elements[0]
	pr, ok := elem.Data.(*PositionReport)
	if !ok || pr == nil {
		t.Fatalf("Element[0].Data is %T, want *PositionReport", elem.Data)
	}

	// Current position: ~10.99°N, ~20.19°W.
	if pr.PosCurrent == nil || pr.PosCurrent.Latitude == nil || pr.PosCurrent.Longitude == nil {
		t.Fatal("PosCurrent lat/lon is nil")
	}
	if *pr.PosCurrent.Latitude < 10.9 || *pr.PosCurrent.Latitude > 11.1 {
		t.Errorf("Latitude: got %.4f, want ~10.99", *pr.PosCurrent.Latitude)
	}
	if *pr.PosCurrent.Longitude > -20.0 || *pr.PosCurrent.Longitude < -20.3 {
		t.Errorf("Longitude: got %.4f, want ~-20.19", *pr.PosCurrent.Longitude)
	}

	// Time at current position: 02:55.
	if pr.TimeAtPosCurrent == nil || pr.TimeAtPosCurrent.Hours != 2 || pr.TimeAtPosCurrent.Minutes != 55 {
		t.Errorf("TimeAtPosCurrent: got %v, want 02:55", pr.TimeAtPosCurrent)
	}

	// Altitude: FL320.
	if pr.Alt == nil || pr.Alt.Type != "flight_level" || pr.Alt.Value != 320 {
		t.Errorf("Alt: got %v, want FL320", pr.Alt)
	}

	// Next fix: GANDO.
	if pr.NextFix == nil || pr.NextFix.Name != "GANDO" {
		t.Errorf("NextFix: got %v, want GANDO", pr.NextFix)
	}

	// ETA at next fix: 03:47.
	if pr.EtaAtFixNext == nil || pr.EtaAtFixNext.Hours != 3 || pr.EtaAtFixNext.Minutes != 47 {
		t.Errorf("EtaAtFixNext: got %v, want 03:47", pr.EtaAtFixNext)
	}

	// Next+1 fix: ERETU.
	if pr.NextNextFix == nil || pr.NextNextFix.Name != "ERETU" {
		t.Errorf("NextNextFix: got %v, want ERETU", pr.NextNextFix)
	}

	// ETA at destination: 08:23.
	if pr.EtaAtDest == nil || pr.EtaAtDest.Hours != 8 || pr.EtaAtDest.Minutes != 23 {
		t.Errorf("EtaAtDest: got %v, want 08:23", pr.EtaAtDest)
	}

	// Temperature: -34°C.
	if pr.Temp == nil || pr.Temp.Type != "C" || pr.Temp.Value != -34 {
		t.Errorf("Temp: got %v, want -34C", pr.Temp)
	}

	// Winds: 283°, 16 kts.
	if pr.Winds == nil || pr.Winds.Direction != 283 {
		t.Errorf("Winds.Direction: got %v, want 283", pr.Winds)
	}
	if pr.Winds == nil || pr.Winds.Speed == nil || pr.Winds.Speed.Value != 16 {
		t.Errorf("Winds.Speed: got %v, want 16 kts", pr.Winds.Speed)
	}

	// Speed: Mach 0.83.
	if pr.Speed == nil || pr.Speed.Type != "mach" {
		t.Errorf("Speed.Type: got %v, want mach", pr.Speed)
	}

	// Reported waypoint: TURUP.
	if pr.ReportedWptPos == nil || pr.ReportedWptPos.Name != "TURUP" {
		t.Errorf("ReportedWptPos: got %v, want TURUP", pr.ReportedWptPos)
	}

	// Reported waypoint time: 02:50.
	if pr.ReportedWptTime == nil || pr.ReportedWptTime.Hours != 2 || pr.ReportedWptTime.Minutes != 50 {
		t.Errorf("ReportedWptTime: got %v, want 02:50", pr.ReportedWptTime)
	}

	// Reported waypoint altitude: FL320.
	if pr.ReportedWptAlt == nil || pr.ReportedWptAlt.Type != "flight_level" || pr.ReportedWptAlt.Value != 320 {
		t.Errorf("ReportedWptAlt: got %v, want FL320", pr.ReportedWptAlt)
	}
}
