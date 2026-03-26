package rep301

import (
	"testing"

	"acars_parser/internal/acars"
)

func TestREP301Parse(t *testing.T) {
	parser := &Parser{}
	text := "A320,009407,1,1,TB000000/REP301,00,00,1/76401\r\n02E04LGAVLSGG\r\nN43094E01636007573799M059266041GXXXX2100307GX\r\n"

	msg := &acars.Message{
		ID:        acars.FlexInt64(9407),
		Timestamp: "2026-03-14T07:57:30Z",
		Tail:      ".HB-JLR",
		Label:     "ZZ",
		Text:      text,
	}

	if !parser.QuickCheck(msg.Text) {
		t.Fatal("QuickCheck() = false, want true")
	}

	parsed := parser.Parse(msg)
	if parsed == nil {
		t.Fatal("Parse() returned nil")
	}

	result, ok := parsed.(*Result)
	if !ok {
		t.Fatalf("Parse() returned %T, want *Result", parsed)
	}

	assertStringEqual(t, "timestamp", result.Timestamp, "2026-03-14T07:57:30Z")
	assertStringEqual(t, "msg_type", result.MsgType, "REP301")
	assertStringEqual(t, "tail", result.Tail, ".HB-JLR")
	assertStringEqual(t, "route", result.Route, "LGAV-LSGG")
	assertStringEqual(t, "origin", result.Origin, "LGAV")
	assertStringEqual(t, "destination", result.Destination, "LSGG")
	assertStringEqual(t, "report_time", result.ReportTime, "07:57")
	assertFloatEqual(t, "latitude", result.Latitude, 43.094)
	assertFloatEqual(t, "longitude", result.Longitude, 16.360)
	assertFloatEqual(t, "flight_level", result.FlightLevel, 379.9)
	assertIntEqual(t, "temperature_c", result.TemperatureC, -59)
	assertIntEqual(t, "wind_direction", result.WindDirection, 266)
	assertIntEqual(t, "wind_speed_kts", result.WindSpeedKts, 41)
	assertIntEqual(t, "wind_speed_kmh", result.WindSpeedKmh, 76)
	assertStringEqual(t, "raw_data", result.RawData, text)
	if result.MessageID() != 9407 {
		t.Fatalf("message_id = %d, want 9407", result.MessageID())
	}
}

func TestREP301ParseRealWorldSamples(t *testing.T) {
	parser := &Parser{}
	tests := []struct {
		name          string
		messageID     int64
		tail          string
		text          string
		wantRoute     string
		wantOrigin    string
		wantDest      string
		wantLatitude  float64
		wantLongitude float64
		wantTime      string
		wantFL        float64
		wantTemp      int
		wantWindDir   int
		wantWindKts   int
		wantWindKmh   int
	}{
		{
			name:          "lgav_lybe_low_level",
			messageID:     3648,
			tail:          ".SX-DVX",
			text:          "A320,003648,1,1,TB000000/REP301,00,00,1/76401\r\n02E25LGAVLYBE\r\nN42294E02166711293799M053278078GXXXX2400B0YY,\r\n",
			wantRoute:     "LGAV-LYBE",
			wantOrigin:    "LGAV",
			wantDest:      "LYBE",
			wantLatitude:  42.294,
			wantLongitude: 21.667,
			wantTime:      "11:29",
			wantFL:        379.9,
			wantTemp:      -53,
			wantWindDir:   278,
			wantWindKts:   78,
			wantWindKmh:   144,
		},
		{
			name:          "hesh_lszh",
			messageID:     31391,
			tail:          ".HB-JLR",
			text:          "A320,031391,1,1,TB000000/REP301,00,00,1/76401\r\n02E25HESHLSZH\r\nN42614E02065512293598M053287114GXXXX2000B805Z\r\n",
			wantRoute:     "HESH-LSZH",
			wantOrigin:    "HESH",
			wantDest:      "LSZH",
			wantLatitude:  42.614,
			wantLongitude: 20.655,
			wantTime:      "12:29",
			wantFL:        359.8,
			wantTemp:      -53,
			wantWindDir:   287,
			wantWindKts:   114,
			wantWindKmh:   211,
		},
		{
			name:          "lgkl_essa",
			messageID:     5936,
			tail:          ".SX-DND",
			text:          "A320,005936,1,1,TB000000/REP301,00,00,1/76401\r\n02E25LGKLESSA\r\nN41734E02123814273757M054294108GXXXX2300J805Z\r\n",
			wantRoute:     "LGKL-ESSA",
			wantOrigin:    "LGKL",
			wantDest:      "ESSA",
			wantLatitude:  41.734,
			wantLongitude: 21.238,
			wantTime:      "14:27",
			wantFL:        375.7,
			wantTemp:      -54,
			wantWindDir:   294,
			wantWindKts:   108,
			wantWindKmh:   200,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			msg := &acars.Message{
				ID:        acars.FlexInt64(test.messageID),
				Timestamp: "2026-03-14T00:00:00Z",
				Tail:      test.tail,
				Label:     "H1",
				Text:      test.text,
			}

			parsed := parser.Parse(msg)
			if parsed == nil {
				t.Fatal("Parse() returned nil")
			}

			result, ok := parsed.(*Result)
			if !ok {
				t.Fatalf("Parse() returned %T, want *Result", parsed)
			}

			assertStringEqual(t, "msg_type", result.MsgType, "REP301")
			assertStringEqual(t, "tail", result.Tail, test.tail)
			assertStringEqual(t, "route", result.Route, test.wantRoute)
			assertStringEqual(t, "origin", result.Origin, test.wantOrigin)
			assertStringEqual(t, "destination", result.Destination, test.wantDest)
			assertStringEqual(t, "report_time", result.ReportTime, test.wantTime)
			assertFloatEqual(t, "latitude", result.Latitude, test.wantLatitude)
			assertFloatEqual(t, "longitude", result.Longitude, test.wantLongitude)
			assertFloatEqual(t, "flight_level", result.FlightLevel, test.wantFL)
			assertIntEqual(t, "temperature_c", result.TemperatureC, test.wantTemp)
			assertIntEqual(t, "wind_direction", result.WindDirection, test.wantWindDir)
			assertIntEqual(t, "wind_speed_kts", result.WindSpeedKts, test.wantWindKts)
			assertIntEqual(t, "wind_speed_kmh", result.WindSpeedKmh, test.wantWindKmh)
			if result.MessageID() != test.messageID {
				t.Fatalf("message_id = %d, want %d", result.MessageID(), test.messageID)
			}
		})
	}
}

func TestREP301RejectsInvalidMessages(t *testing.T) {
	parser := &Parser{}

	if parser.QuickCheck("A320,009407,1,1,TB000000/OTHER,00,00,1/76401") {
		t.Fatal("QuickCheck() = true, want false")
	}

	msg := &acars.Message{
		Label: "H1",
		Text:  "A320,009407,1,1,TB000000/REP301,00,00,1/76401 02E04LGAVLSGG BADPAYLOAD",
	}

	if parsed := parser.Parse(msg); parsed != nil {
		t.Fatalf("Parse() returned %T, want nil", parsed)
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
