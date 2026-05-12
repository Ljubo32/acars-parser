package abs

import (
	"testing"

	"acars_parser/internal/acars"
)

func TestABSParsesInlineRouteHint(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{Text: "ABS001DA_T       EBBRLTFJ537\r\n 44391  196361318035340-49-285104XA0IH-0190220000--\r\n"}

	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("Expected *Result, got %T", res)
	}

	if result.BlockID != "ABS001DA_T" {
		t.Fatalf("BlockID = %q, want %q", result.BlockID, "ABS001DA_T")
	}
	if result.Origin != "EBBR" {
		t.Fatalf("Origin = %q, want %q", result.Origin, "EBBR")
	}
	if result.Destination != "LTFJ" {
		t.Fatalf("Destination = %q, want %q", result.Destination, "LTFJ")
	}
	if result.Route != "EBBR-LTFJ" {
		t.Fatalf("Route = %q, want %q", result.Route, "EBBR-LTFJ")
	}
	if result.Level != "537" {
		t.Fatalf("Level = %q, want %q", result.Level, "537")
	}
	assertFloatClose(t, "latitude", result.Latitude, 44.391)
	assertFloatClose(t, "longitude", result.Longitude, 19.636)
	if result.AltitudeFt != 35340 {
		t.Fatalf("AltitudeFt = %d, want %d", result.AltitudeFt, 35340)
	}
	if result.Temperature != -49 {
		t.Fatalf("Temperature = %d, want %d", result.Temperature, -49)
	}
	if len(result.Positions) != 1 {
		t.Fatalf("Positions len = %d, want %d", len(result.Positions), 1)
	}
}

func TestABSParsesRouteHintFromNextLine(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{Text: "H1 HEADER\r\nABS001DA_T\r\n1234EGLLRJTT350 EXTRA\r\n"}

	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("Expected *Result, got %T", res)
	}

	if result.Route != "EGLL-RJTT" {
		t.Fatalf("Route = %q, want %q", result.Route, "EGLL-RJTT")
	}
	if result.Level != "350" {
		t.Fatalf("Level = %q, want %q", result.Level, "350")
	}
}

func TestABSParsesSplitRouteAndLevelFromSameLine(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{Text: "ABS001DA_T       EGSSLTAC 29\r\n 46835  185191751035003-55-058006X209B,0140170000--\r\n"}

	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("Expected *Result, got %T", res)
	}

	if result.Route != "EGSS-LTAC" {
		t.Fatalf("Route = %q, want %q", result.Route, "EGSS-LTAC")
	}
	if result.Level != "29" {
		t.Fatalf("Level = %q, want %q", result.Level, "29")
	}
	assertFloatClose(t, "latitude", result.Latitude, 46.835)
	assertFloatClose(t, "longitude", result.Longitude, 18.519)
	if result.AltitudeFt != 35003 {
		t.Fatalf("AltitudeFt = %d, want %d", result.AltitudeFt, 35003)
	}
	if result.Temperature != -55 {
		t.Fatalf("Temperature = %d, want %d", result.Temperature, -55)
	}
}

func TestABSParsesMultiPositionBlock(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{Text: "ABS011DA_        LQSAESGG4\r\n 43924  182691349006127-04-035012XYOXZC0660940000--\r\n 43990  182391350009095-06-052014XZ:700++++++0000--\r\n 44065  182051351010924-083037020XZ:700++++++0000--\r\n 44149  181661352013383-13-016016XZ:700++++++0000--\r\n 44237  181251353015901-19-328016XZ:700++++++0000--\r\n 44326  180771354018264-238328021XZ:700++++++0000--\r\n"}

	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("Expected *Result, got %T", res)
	}

	if result.Route != "LQSA-ESGG" {
		t.Fatalf("Route = %q, want %q", result.Route, "LQSA-ESGG")
	}
	if result.Level != "" {
		t.Fatalf("Level = %q, want empty", result.Level)
	}
	if len(result.Positions) != 6 {
		t.Fatalf("Positions len = %d, want %d", len(result.Positions), 6)
	}
	assertFloatClose(t, "first latitude", result.Positions[0].Latitude, 43.924)
	assertFloatClose(t, "first longitude", result.Positions[0].Longitude, 18.269)
	if result.Positions[0].AltitudeFt != 6127 {
		t.Fatalf("First AltitudeFt = %d, want %d", result.Positions[0].AltitudeFt, 6127)
	}
	if result.Positions[0].TemperatureC != -4 {
		t.Fatalf("First TemperatureC = %d, want %d", result.Positions[0].TemperatureC, -4)
	}
	assertFloatClose(t, "last latitude", result.Positions[5].Latitude, 44.326)
	assertFloatClose(t, "last longitude", result.Positions[5].Longitude, 18.077)
	if result.Positions[5].AltitudeFt != 18264 {
		t.Fatalf("Last AltitudeFt = %d, want %d", result.Positions[5].AltitudeFt, 18264)
	}
	if result.Positions[5].TemperatureC != -23 {
		t.Fatalf("Last TemperatureC = %d, want %d", result.Positions[5].TemperatureC, -23)
	}
}

func TestABSReturnsNilWithoutRouteOrPosition(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{Text: "ABS001DA_T\r\nNO POSITION DATA\r\n"}

	if res := parser.Parse(msg); res != nil {
		t.Fatalf("Expected nil result, got %T", res)
	}
}

func assertFloatClose(t *testing.T, field string, got, want float64) {
	t.Helper()
	if got < want-0.0001 || got > want+0.0001 {
		t.Fatalf("%s = %.6f, want %.3f", field, got, want)
	}
}