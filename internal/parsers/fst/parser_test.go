package fst

import (
	"acars_parser/internal/acars"
	"encoding/json"
	"testing"
)

func TestFST01FixedFormat(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		origin  string
		dest    string
		route   string
		lat     float64
		lon     float64
		fl      int
		temp    int
		windKts int
		windKmh int
		windDir int
		heading int
		track   int
		gsKts   int
		gsKmh   int
	}{
		{
			name:    "two_digit_wind_speed",
			input:   "FST01VTBSEGKKN465853E0210031360 197 825 M50C 1226229129746211600005180304",
			origin:  "VTBS",
			dest:    "EGKK",
			route:   "VTBS-EGKK",
			lat:     46.5853,
			lon:     21.0031,
			fl:      360,
			temp:    -50,
			windKts: 12,
			windKmh: 22,
			windDir: 262,
			heading: 297,
			track:   291,
			gsKts:   462,
			gsKmh:   855,
		},
		{
			name:    "single_digit_wind_speed",
			input:   "FST01VTBSEGKKN471493E0193150360 187 836 M52C 520529029546411600005180314",
			origin:  "VTBS",
			dest:    "EGKK",
			route:   "VTBS-EGKK",
			lat:     47.1493,
			lon:     19.3150,
			fl:      360,
			temp:    -52,
			windKts: 5,
			windKmh: 9,
			windDir: 205,
			heading: 295,
			track:   290,
			gsKts:   464,
			gsKmh:   859,
		},
		{
			name:    "combined_temperature_and_compact_block",
			input:   "FST01VHHHEGLLN465508E0219146380 169 904 M62C017071291293468  05380326",
			origin:  "VHHH",
			dest:    "EGLL",
			route:   "VHHH-EGLL",
			lat:     46.5508,
			lon:     21.9146,
			fl:      380,
			temp:    -62,
			windKts: 17,
			windKmh: 31,
			windDir: 251,
			heading: 293,
			track:   291,
			gsKts:   468,
			gsKmh:   866,
		},
		{
			name:    "three_digit_wind_speed_in_combined_block",
			input:   "FST01EGLLHKJKN445759E0162296370 461 139 M56C031080133131509  18211203",
			origin:  "EGLL",
			dest:    "HKJK",
			route:   "EGLL-HKJK",
			lat:     44.5759,
			lon:     16.2296,
			fl:      370,
			temp:    -56,
			windKts: 31,
			windKmh: 57,
			windDir: 260,
			heading: 131,
			track:   133,
			gsKts:   509,
			gsKmh:   942,
		},
		{
			name:    "suffix_before_temperature_still_uses_compact_heading_and_track",
			input:   "FST01EGLLOMAAN418071E0214075390 245 145M 57C 3828713613851713682504540050",
			origin:  "EGLL",
			dest:    "OMAA",
			route:   "EGLL-OMAA",
			lat:     41.8071,
			lon:     21.4075,
			fl:      390,
			temp:    57,
			windKts: 38,
			windKmh: 70,
			windDir: 287,
			heading: 138,
			track:   136,
			gsKts:   517,
			gsKmh:   957,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &Parser{}
			msg := &acars.Message{Text: tt.input}
			res := parser.Parse(msg)
			result, ok := res.(*Result)
			if !ok {
				t.Fatalf("Expected *Result, got %T", res)
			}

			assertStringEqual(t, "msg_type", result.MsgType, "FST")
			assertStringEqual(t, "origin", result.Origin, tt.origin)
			assertStringEqual(t, "destination", result.Destination, tt.dest)
			assertStringEqual(t, "route", result.Route, tt.route)
			assertFloatEqual(t, "latitude", result.Latitude, tt.lat)
			assertFloatEqual(t, "longitude", result.Longitude, tt.lon)
			assertIntEqual(t, "flight_level", result.FlightLevel, tt.fl)
			assertIntEqual(t, "temperature", result.Temperature, tt.temp)
			assertIntEqual(t, "wind_speed_kts", result.WindSpeedKts, tt.windKts)
			assertIntEqual(t, "wind_speed_kmh", result.WindSpeedKmh, tt.windKmh)
			assertIntEqual(t, "wind_direction", result.WindDirection, tt.windDir)
			assertIntEqual(t, "heading", result.Heading, tt.heading)
			assertIntEqual(t, "track", result.Track, tt.track)
			assertIntEqual(t, "ground_speed_kts", result.GroundSpeedKts, tt.gsKts)
			assertIntEqual(t, "ground_speed_kmh", result.GroundSpeedKmh, tt.gsKmh)
		})
	}
}

func TestFST01SuffixVariantDoesNotReuseLegacyHeadingHeuristics(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{Text: "FST01EGLLOMAAN418071E0214075390 245 145M 57C 3828713613851713682504540050"}
	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("Expected *Result, got %T", res)
	}

	assertIntEqual(t, "heading", result.Heading, 138)
	assertIntEqual(t, "track", result.Track, 136)
	assertIntEqual(t, "unknown1", result.Unknown1, 0)
}

func TestFST01CombinedTemperatureDecodesA350WindDirection(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{Text: "FST01EGLLRJTTN448927E0252277370 692 203 M51C045098113111537 21461106"}
	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("Expected *Result, got %T", res)
	}

	assertStringEqual(t, "route", result.Route, "EGLL-RJTT")
	assertIntEqual(t, "wind_direction", result.WindDirection, 278)
	assertIntEqual(t, "track", result.Track, 113)
	assertIntEqual(t, "heading", result.Heading, 111)
	assertIntEqual(t, "ground_speed_kts", result.GroundSpeedKts, 537)
}

func TestFSTLegacyFormatStillParses(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{Text: "FST01EGLCEIDWN51420W00049317803270072M020C014331258256370"}
	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("Expected *Result, got %T", res)
	}

	assertStringEqual(t, "msg_type", result.MsgType, "FST")
	assertStringEqual(t, "origin", result.Origin, "EGLC")
	assertStringEqual(t, "destination", result.Destination, "EIDW")
	assertStringEqual(t, "route", result.Route, "EGLC-EIDW")
	assertFloatEqual(t, "latitude", result.Latitude, 51.4200)
	assertFloatEqual(t, "longitude", result.Longitude, -0.4931)
}

func TestFSTFiveDigitLatitudeKeepsCorrectScale(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{Text: "FST01HECAEGLLN46976E0170812380  54  80     27013      441  20011809"}
	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("Expected *Result, got %T", res)
	}

	assertStringEqual(t, "route", result.Route, "HECA-EGLL")
	assertFloatEqual(t, "latitude", result.Latitude, 46.9760)
	assertFloatEqual(t, "longitude", result.Longitude, 17.0812)
	assertIntEqual(t, "flight_level", result.FlightLevel, 380)
}

func TestFSTJSONHidesRedundantFields(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{Text: "FST01HECAEGLLN424045E0198495400 193 146 M54C 1828231732347511600010110755"}
	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("Expected *Result, got %T", res)
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	assertMapHasKey(t, got, "msg_type")
	assertMapHasKey(t, got, "route")
	assertMapHasKey(t, got, "ground_speed_kts")
	assertMapHasKey(t, got, "ground_speed_kmh")
	assertMapHasKey(t, got, "wind_speed_kts")
	assertMapHasKey(t, got, "wind_speed_kmh")
	assertMapLacksKey(t, got, "message_id")
	assertMapLacksKey(t, got, "origin")
	assertMapLacksKey(t, got, "destination")
	assertMapLacksKey(t, got, "ground_speed")
	assertMapLacksKey(t, got, "wind_speed")
}

func assertIntEqual(t *testing.T, field string, got, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %d, want %d", field, got, want)
	}
}

func assertStringEqual(t *testing.T, field, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %q, want %q", field, got, want)
	}
}

func assertFloatEqual(t *testing.T, field string, got, want float64) {
	t.Helper()
	if got < want-0.0001 || got > want+0.0001 {
		t.Fatalf("%s = %.6f, want %.4f", field, got, want)
	}
}

func assertMapHasKey(t *testing.T, m map[string]interface{}, key string) {
	t.Helper()
	if _, ok := m[key]; !ok {
		t.Fatalf("expected key %q in marshalled JSON", key)
	}
}

func assertMapLacksKey(t *testing.T, m map[string]interface{}, key string) {
	t.Helper()
	if _, ok := m[key]; ok {
		t.Fatalf("did not expect key %q in marshalled JSON", key)
	}
}
