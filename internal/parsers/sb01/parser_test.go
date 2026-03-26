package sb01

import (
	"testing"

	"acars_parser/internal/acars"
)

func TestSB01Parse(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		registration  string
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
			text:          "SB0122BA_F-GZNG LFPOFMEE195 42703 0184101832 31001-550356015010GMY012015",
			registration:  "F-GZNG",
			route:         "LFPO-FMEE",
			lat:           42.703,
			lon:           18.410,
			reportTime:    "18:32",
			altitudeFt:    31001,
			altitudeM:     9449,
			temperatureC:  -55.0,
			windDirection: 356,
			windSpeedKts:  15,
			windSpeedKmh:  27,
		},
		{
			name:          "turkish_sample",
			text:          "SB0122BA_TC-JJY  KJFKLTFM563\r\n 44897 0182871231 37007-480281087010/9W014017",
			registration:  "TC-JJY",
			route:         "KJFK-LTFM",
			lat:           44.897,
			lon:           18.287,
			reportTime:    "12:31",
			altitudeFt:    37007,
			altitudeM:     11279,
			temperatureC:  -48.0,
			windDirection: 281,
			windSpeedKts:  87,
			windSpeedKmh:  161,
		},
		{
			name:          "emirates_sample",
			text:          "SB0122BA_A6-EQC  OMDBLSGG418\r\n 44287 0226401530 37992-450272078090W/X015021",
			registration:  "A6-EQC",
			route:         "OMDB-LSGG",
			lat:           44.287,
			lon:           22.640,
			reportTime:    "15:30",
			altitudeFt:    37992,
			altitudeM:     11579,
			temperatureC:  -45.0,
			windDirection: 272,
			windSpeedKts:  78,
			windSpeedKmh:  144,
		},
		{
			name:          "spaced_route_single_digit_block",
			text:          "SB0122BA_F-GSQF LFPGRJTT 7 45265 0210782247 28993-5000090170H8FXW038051",
			registration:  "F-GSQF",
			route:         "LFPG-RJTT",
			lat:           45.265,
			lon:           21.078,
			reportTime:    "22:47",
			altitudeFt:    28993,
			altitudeM:     8837,
			temperatureC:  -50.0,
			windDirection: 9,
			windSpeedKts:  17,
			windSpeedKmh:  31,
		},
		{
			name:          "spaced_route_two_digit_block",
			text:          "SB0122BA_F-GSPP LFPGVIDP 46 43683 0226401211 35001-590287019010GMY011014",
			registration:  "F-GSPP",
			route:         "LFPG-VIDP",
			lat:           43.683,
			lon:           22.640,
			reportTime:    "12:11",
			altitudeFt:    35001,
			altitudeM:     10668,
			temperatureC:  -59.0,
			windDirection: 287,
			windSpeedKts:  19,
			windSpeedKmh:  35,
		},
	}

	parser := &Parser{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &acars.Message{ID: 1, Timestamp: "2026-03-13T00:00:00Z", Tail: "envelope-tail", Label: "H1", Text: tt.text}
			res := parser.Parse(msg)
			result, ok := res.(*Result)
			if !ok {
				t.Fatalf("Expected *Result, got %T", res)
			}

			assertStringEqual(t, "msg_type", result.MsgType, "EBSB")
			assertStringEqual(t, "registration", result.Registration, tt.registration)
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

func TestSB01RejectsInvalidMessages(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{Label: "H1", Text: "SB01 broken message"}
	if res := parser.Parse(msg); res != nil {
		t.Fatalf("expected nil parse result, got %T", res)
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
