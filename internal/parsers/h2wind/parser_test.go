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

func TestParse02AInitialLayersAndRoutePoints(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{Text: "02A041419BKPRLSZHN42341E021019041359 151P127     197P125021001G     246P112000002G    /N42270E0205231012M052267008G    N42265E0204681228M097263009G"}

	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("expected *Result, got %T", res)
	}
	if result.Phase != "02A" {
		t.Fatalf("Phase = %q, want %q", result.Phase, "02A")
	}
	if len(result.InitialLayers) != 3 {
		t.Fatalf("InitialLayers len = %d, want 3", len(result.InitialLayers))
	}

	first := result.InitialLayers[0]
	if first.AltitudeFeet != 1510 || first.AltitudeMetres != 460 {
		t.Fatalf("first layer altitude = %d ft / %d m, want 1510 ft / 460 m", first.AltitudeFeet, first.AltitudeMetres)
	}
	if first.TemperatureC != 12.7 {
		t.Fatalf("first layer temperature = %v, want 12.7", first.TemperatureC)
	}
	if first.WindDirDeg != 0 || first.WindSpeedKt != 0 {
		t.Fatalf("first layer wind = %d/%d, want no wind data", first.WindDirDeg, first.WindSpeedKt)
	}

	second := result.InitialLayers[1]
	if second.AltitudeFeet != 1970 || second.AltitudeMetres != 600 {
		t.Fatalf("second layer altitude = %d ft / %d m, want 1970 ft / 600 m", second.AltitudeFeet, second.AltitudeMetres)
	}
	if second.TemperatureC != 12.5 {
		t.Fatalf("second layer temperature = %v, want 12.5", second.TemperatureC)
	}
	if second.WindDirDeg != 21 || second.WindSpeedKt != 1 || second.WindSpeedKmh != 2 {
		t.Fatalf("second layer wind = %d deg / %d kt / %d km/h, want 21 / 1 / 2", second.WindDirDeg, second.WindSpeedKt, second.WindSpeedKmh)
	}

	if len(result.Points) != 2 {
		t.Fatalf("Points len = %d, want 2", len(result.Points))
	}
	if result.Points[0].Latitude != 42.27 || result.Points[0].Longitude != 20.523 {
		t.Fatalf("first route point coords = %.3f, %.3f, want 42.270, 20.523", result.Points[0].Latitude, result.Points[0].Longitude)
	}
	if result.Points[0].Temperature != -5.2 || result.Points[0].WindDir != 267 || result.Points[0].WindSpeed != 8 {
		t.Fatalf("first route point weather = %.1f / %d / %d, want -5.2 / 267 / 8", result.Points[0].Temperature, result.Points[0].WindDir, result.Points[0].WindSpeed)
	}
}

func TestParse02DUses02ALayout(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{Text: "02D041419BKPRLSZHN42341E021019041359 197P125021001G     246P112000002G    /N42270E0205231012M052267008G"}

	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("expected *Result, got %T", res)
	}
	if result.Phase != "02D" {
		t.Fatalf("Phase = %q, want %q", result.Phase, "02D")
	}
	if len(result.InitialLayers) != 2 {
		t.Fatalf("InitialLayers len = %d, want 2", len(result.InitialLayers))
	}
	if len(result.Points) != 1 {
		t.Fatalf("Points len = %d, want 1", len(result.Points))
	}
	if result.InitialLayers[0].WindSpeedKmh != 2 {
		t.Fatalf("first 02D layer WindSpeedKmh = %d, want 2", result.InitialLayers[0].WindSpeedKmh)
	}
}
