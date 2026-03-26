package eb00

import (
	"testing"

	"acars_parser/internal/acars"
)

func TestEB00Parse(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		registration  string
		msgNo         int
		route         string
		lat           float64
		lon           float64
		reportTime    string
		altitudeFt    int
		altitudeM     int
		temperatureC  float64
		windDirection int
		windSpeedKts  int
		windSpeedKmh  int
	}{
		{
			name:          "user_sample",
			text:          "EB0032AA_ D-ABPR VABBEDDF 44 45570 0236890440 35998-633193015090W/X014022",
			registration:  "D-ABPR",
			msgNo:         44,
			route:         "VABB-EDDF",
			lat:           45.570,
			lon:           23.689,
			reportTime:    "04:40",
			altitudeFt:    35998,
			altitudeM:     10972,
			temperatureC:  -63.3,
			windDirection: 193,
			windSpeedKts:  15,
			windSpeedKmh:  27,
		},
		{
			name:          "inline_registration",
			text:          "EB0032AA_D-ABPR VABBEDDF 44 45570 0236890440 35998-633193015090W/X014022",
			registration:  "D-ABPR",
			msgNo:         44,
			route:         "VABB-EDDF",
			lat:           45.570,
			lon:           23.689,
			reportTime:    "04:40",
			altitudeFt:    35998,
			altitudeM:     10972,
			temperatureC:  -63.3,
			windDirection: 193,
			windSpeedKts:  15,
			windSpeedKmh:  27,
		},
		{
			name:          "combined_route_and_message_number",
			text:          "EB0032AA_VN-A868 VVTSEDDF127 47204 0210920526 39996-572269020010GMY004005",
			registration:  "VN-A868",
			msgNo:         127,
			route:         "VVTS-EDDF",
			lat:           47.204,
			lon:           21.092,
			reportTime:    "05:26",
			altitudeFt:    39996,
			altitudeM:     12190,
			temperatureC:  -57.2,
			windDirection: 269,
			windSpeedKts:  20,
			windSpeedKmh:  37,
		},
	}

	parser := &Parser{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &acars.Message{ID: 1, Timestamp: "2026-03-14T00:00:00Z", Tail: "envelope-tail", Label: "H1", Text: tt.text}
			res := parser.Parse(msg)
			result, ok := res.(*Result)
			if !ok {
				t.Fatalf("Expected *Result, got %T", res)
			}

			assertStringEqual(t, "tail", result.Tail, tt.registration)
			assertStringEqual(t, "msg_type", result.MsgType, "EBSB")
			assertStringEqual(t, "registration", result.Registration, tt.registration)
			assertIntEqual(t, "msg_no", result.MsgNo, tt.msgNo)
			assertStringEqual(t, "route", result.Route, tt.route)
			assertStringEqual(t, "report_time", result.ReportTime, tt.reportTime)
			assertFloatEqual(t, "latitude", result.Latitude, tt.lat)
			assertFloatEqual(t, "longitude", result.Longitude, tt.lon)
			assertIntEqual(t, "altitude_ft", result.AltitudeFt, tt.altitudeFt)
			assertIntEqual(t, "altitude_m", result.AltitudeM, tt.altitudeM)
			assertTempEqual(t, "temperature_c", result.TemperatureC, tt.temperatureC)
			assertIntEqual(t, "wind_direction", result.WindDirection, tt.windDirection)
			assertIntEqual(t, "wind_speed_kts", result.WindSpeedKts, tt.windSpeedKts)
			assertIntEqual(t, "wind_speed_kmh", result.WindSpeedKmh, tt.windSpeedKmh)
		})
	}
}

func TestEB00RejectsInvalidMessages(t *testing.T) {
	parser := &Parser{}
	tests := []string{
		"EB00 broken message",
		"EB0032AA_ D-ABPR VABBEDDF XX 45570 0236890440 35998-633193015090W/X014022",
		"EB0032AA_ D-ABPR VABBEDDF 44 45570 0236892460 35998-633193015090W/X014022",
		"SB0122BA_F-GZNG LFPOFMEE195 42703 0184101832 31001-550356015010GMY012015",
	}

	for _, text := range tests {
		msg := &acars.Message{Label: "H1", Text: text}
		if res := parser.Parse(msg); res != nil {
			t.Fatalf("expected nil parse result for %q, got %T", text, res)
		}
	}
}

func assertStringEqual(t *testing.T, field, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %q, want %q", field, got, want)
	}
}

func assertIntEqual(t *testing.T, field string, got, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %d, want %d", field, got, want)
	}
}

func assertFloatEqual(t *testing.T, field string, got, want float64) {
	t.Helper()
	if got < want-0.0001 || got > want+0.0001 {
		t.Fatalf("%s = %.6f, want %.3f", field, got, want)
	}
}

func assertTempEqual(t *testing.T, field string, got, want float64) {
	t.Helper()
	if got < want-0.0001 || got > want+0.0001 {
		t.Fatalf("%s = %.1f, want %.1f", field, got, want)
	}
}
