package fst

import (
	"testing"

	"acars_parser/internal/acars"
)

func TestParser(t *testing.T) {
	testCases := []struct {
		name string
		text string
		want struct {
			origin        string
			destination   string
			latitude      float64
			longitude     float64
			flightLevel   int
			groundSpeed   int
			speedUnit     string
			speedType     string
			temperature   int
			windSpeed     int
			windDirection int
		}
	}{
		{
			name: "FST with 7-digit longitude and wind data (KMH conversion)",
			text: "FST01EGLLWSSSN452140E0249275330 854 242 M54C 6235410711950911600009590004",
			want: struct {
				origin        string
				destination   string
				latitude      float64
				longitude     float64
				flightLevel   int
				groundSpeed   int
				speedUnit     string
				speedType     string
				temperature   int
				windSpeed     int
				windDirection int
			}{
				origin:        "EGLL",
				destination:   "WSSS",
				latitude:      45.3567, // N 45° 21.40' = 45.3567°
				longitude:     24.9275, // E 024.9275° (decimal format)
				flightLevel:   330,
				groundSpeed:   854, // 854 KM/H stays as-is
				speedUnit:     "kmh",
				speedType:     "KMH",
				temperature:   -54,
				windSpeed:     62,
				windDirection: 354,
			},
		},
		{
			name: "FST with Ground Speed in knots",
			text: "FST01EGLLLFPGN452140E0249275330 485 242 M54C 623541",
			want: struct {
				origin        string
				destination   string
				latitude      float64
				longitude     float64
				flightLevel   int
				groundSpeed   int
				speedUnit     string
				speedType     string
				temperature   int
				windSpeed     int
				windDirection int
			}{
				origin:        "EGLL",
				destination:   "LFPG",
				latitude:      45.3567,
				longitude:     24.9275,
				flightLevel:   330,
				groundSpeed:   485, // Already in knots (450-500 range)
				speedUnit:     "knots",
				speedType:     "GS",
				temperature:   -54,
				windSpeed:     62,
				windDirection: 354,
			},
		},
		{
			name: "FST with IAS (Indicated Airspeed)",
			text: "FST01EGLLLFPGN452140E0249275330 235 242 M54C 623541",
			want: struct {
				origin        string
				destination   string
				latitude      float64
				longitude     float64
				flightLevel   int
				groundSpeed   int
				speedUnit     string
				speedType     string
				temperature   int
				windSpeed     int
				windDirection int
			}{
				origin:        "EGLL",
				destination:   "LFPG",
				latitude:      45.3567,
				longitude:     24.9275,
				flightLevel:   330,
				groundSpeed:   242, // Second field is GS
				speedUnit:     "knots",
				speedType:     "IAS+GS",
				temperature:   -54,
				windSpeed:     62,
				windDirection: 354,
			},
		},
		{
			name: "FST EGLL to OTHH with IAS",
			text: "FST01EGLLOTHHN444207E0250872390 292 161 M64C 4330711512051411600020141628",
			want: struct {
				origin        string
				destination   string
				latitude      float64
				longitude     float64
				flightLevel   int
				groundSpeed   int
				speedUnit     string
				speedType     string
				temperature   int
				windSpeed     int
				windDirection int
			}{
				origin:        "EGLL",
				destination:   "OTHH",
				latitude:      44.7012, // N 44° 42.07' = 44 + 42.07/60
				longitude:     25.0872, // E 025.0872° (decimal format)
				flightLevel:   390,
				groundSpeed:   161, // Second field is GS
				speedUnit:     "knots",
				speedType:     "IAS+GS", // 292 is IAS, 161 is GS
				temperature:   -64,
				windSpeed:     43,
				windDirection: 307,
			},
		},
		{
			name: "FST EGLL to OBBI with IAS",
			text: "FST01EGLLOBBIN467315E0193244350 289 124 M64C 6230011612051411600021071650",
			want: struct {
				origin        string
				destination   string
				latitude      float64
				longitude     float64
				flightLevel   int
				groundSpeed   int
				speedUnit     string
				speedType     string
				temperature   int
				windSpeed     int
				windDirection int
			}{
				origin:        "EGLL",
				destination:   "OBBI",
				latitude:      46.7315, // N 46.7315° (decimal format)
				longitude:     19.3244, // E 019.3244° (decimal format)
				flightLevel:   350,
				groundSpeed:   124, // Second field is GS
				speedUnit:     "knots",
				speedType:     "IAS+GS", // 289 is IAS, 124 is GS
				temperature:   -64,
				windSpeed:     62,
				windDirection: 300,
			},
		},
		{
			name: "FST WSSS to EGLL with IAS+KMH concatenated (newer aircraft)",
			text: "FST01WSSSEGLLN466873E0206998360 2001084 M52C 5431429329643811600005570349",
			want: struct {
				origin        string
				destination   string
				latitude      float64
				longitude     float64
				flightLevel   int
				groundSpeed   int
				speedUnit     string
				speedType     string
				temperature   int
				windSpeed     int
				windDirection int
			}{
				origin:        "WSSS",
				destination:   "EGLL",
				latitude:      46.6873,
				longitude:     20.6998,
				flightLevel:   360,
				groundSpeed:   1084, // KMH stays as-is
				speedUnit:     "kmh",
				speedType:     "IAS+KMH", // 2001084 = IAS 200 + KMH 1084 (newer aircraft format)
				temperature:   -52,
				windSpeed:     54,
				windDirection: 314,
			},
		},
		{
			name: "FST OBBI to EGLL with IAS+GS separate (older aircraft)",
			text: "FST01OBBIEGLLN453619E0230053400 192 312 M49C 6328629129843211600006220349",
			want: struct {
				origin        string
				destination   string
				latitude      float64
				longitude     float64
				flightLevel   int
				groundSpeed   int
				speedUnit     string
				speedType     string
				temperature   int
				windSpeed     int
				windDirection int
			}{
				origin:        "OBBI",
				destination:   "EGLL",
				latitude:      45.60316666666667, // N 45° 36.19' = 45 + 36.19/60
				longitude:     23.0053,
				flightLevel:   400,
				groundSpeed:   312, // Direct GS value
				speedUnit:     "knots",
				speedType:     "IAS+GS", // 192 312 = IAS 192 + GS 312 (older aircraft format)
				temperature:   -49,
				windSpeed:     63,
				windDirection: 286,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msg := &acars.Message{
				ID:    1,
				Label: "15",
				Text:  tc.text,
			}

			parser := &Parser{}
			result := parser.Parse(msg)
			if result == nil {
				t.Fatal("Failed to parse FST message")
			}

			fst, ok := result.(*Result)
			if !ok {
				t.Fatal("Result is not FST Result")
			}

			if fst.Origin != tc.want.origin {
				t.Errorf("Origin = %v, want %v", fst.Origin, tc.want.origin)
			}
			if fst.Destination != tc.want.destination {
				t.Errorf("Destination = %v, want %v", fst.Destination, tc.want.destination)
			}
			if abs(fst.Latitude-tc.want.latitude) > 0.01 {
				t.Errorf("Latitude = %v, want %v", fst.Latitude, tc.want.latitude)
			}
			if abs(fst.Longitude-tc.want.longitude) > 0.01 {
				t.Errorf("Longitude = %v, want %v", fst.Longitude, tc.want.longitude)
			}
			if fst.FlightLevel != tc.want.flightLevel {
				t.Errorf("FlightLevel = %v, want %v", fst.FlightLevel, tc.want.flightLevel)
			}
			if fst.GroundSpeed != tc.want.groundSpeed {
				t.Errorf("GroundSpeed = %v, want %v", fst.GroundSpeed, tc.want.groundSpeed)
			}
			if fst.SpeedUnit != tc.want.speedUnit {
				t.Errorf("SpeedUnit = %v, want %v", fst.SpeedUnit, tc.want.speedUnit)
			}
			if fst.SpeedType != tc.want.speedType {
				t.Errorf("SpeedType = %v, want %v", fst.SpeedType, tc.want.speedType)
			}
			if fst.Temperature != tc.want.temperature {
				t.Errorf("Temperature = %v, want %v", fst.Temperature, tc.want.temperature)
			}
			if fst.WindSpeed != tc.want.windSpeed {
				t.Errorf("WindSpeed = %v, want %v", fst.WindSpeed, tc.want.windSpeed)
			}
			if fst.WindDirection != tc.want.windDirection {
				t.Errorf("WindDirection = %v, want %v", fst.WindDirection, tc.want.windDirection)
			}
		})
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func TestParseCoord(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  float64
	}{
		{
			name:  "6-digit latitude N452140",
			input: "452140",
			want:  45.3567, // 45° 21.40' = 45 + 21.40/60
		},
		{
			name:  "7-digit longitude E0249275",
			input: "0249275",
			want:  24.9275, // 024.9275° decimal format
		},
		{
			name:  "5-digit latitude 51420",
			input: "51420",
			want:  51.7, // 51° 42.0' = 51 + 42.0/60
		},
		{
			name:  "6-digit longitude with leading zero (DDMMTT)",
			input: "031200",
			want:  3.2, // 03° 12.00' = 3 + 12.0/60
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCoord(tt.input)
			if abs(got-tt.want) > 0.01 {
				t.Errorf("parseCoord(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
