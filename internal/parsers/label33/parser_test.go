package label33

import (
	"testing"

	"acars_parser/internal/acars"
)

func TestLabel33Parser(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		wantNil bool
		checkFn func(*testing.T, *Result)
	}{
		{
			name: "Sample 1 - LLBG to KJFK",
			text: "2026-01-21,09:32:54,LLBG,KJFK,0009,N43350E021400,497,FL340,0540,-61,184, 21,NIV37 ,09:48,LUL38 ,-31,850,297,481,-046,-045,-039,019,210126",
			checkFn: func(t *testing.T, r *Result) {
				if r.Date != "2026-01-21" {
					t.Errorf("Date = %v, want 2026-01-21", r.Date)
				}
				if r.Time != "09:32:54" {
					t.Errorf("Time = %v, want 09:32:54", r.Time)
				}
				if r.OriginICAO != "LLBG" {
					t.Errorf("OriginICAO = %v, want LLBG", r.OriginICAO)
				}
				if r.DestICAO != "KJFK" {
					t.Errorf("DestICAO = %v, want KJFK", r.DestICAO)
				}
				if r.GroundSpeed != 497 {
					t.Errorf("GroundSpeed = %v, want 497", r.GroundSpeed)
				}
				if r.FlightLevel != 340 {
					t.Errorf("FlightLevel = %v, want 340", r.FlightLevel)
				}
				if r.FuelOnBoard != 540 {
					t.Errorf("FuelOnBoard = %v, want 540", r.FuelOnBoard)
				}
				if r.Temperature != -61 {
					t.Errorf("Temperature = %v, want -61", r.Temperature)
				}
				if r.WindDir != 184 {
					t.Errorf("WindDir = %v, want 184", r.WindDir)
				}
				if r.WindSpeedKts != 21 {
					t.Errorf("WindSpeedKts = %v, want 21", r.WindSpeedKts)
				}
				if r.WindSpeedKmh != 38 { // 21 * 1.852 = 38.892 -> 38
					t.Errorf("WindSpeedKmh = %v, want 38", r.WindSpeedKmh)
				}
				if r.NextWaypoint != "NIV37" {
					t.Errorf("NextWaypoint = %v, want NIV37", r.NextWaypoint)
				}
				if r.NextWptETA != "09:48" {
					t.Errorf("NextWptETA = %v, want 09:48", r.NextWptETA)
				}
				if r.FollowWaypoint != "LUL38" {
					t.Errorf("FollowWaypoint = %v, want LUL38", r.FollowWaypoint)
				}
				// Check coordinates
				if r.Latitude < 43.58 || r.Latitude > 43.59 {
					t.Errorf("Latitude = %v, expected around 43.58", r.Latitude)
				}
				if r.Longitude < 21.66 || r.Longitude > 21.67 {
					t.Errorf("Longitude = %v, expected around 21.67", r.Longitude)
				}
			},
		},
		{
			name: "Sample 2 - LLBG to EGLL",
			text: "2026-01-21,09:46:55,LLBG,EGLL,0315,N42379E020382,485,FL360,0193,-63,187, 17,DOLEV ,09:49,BEDOX ,-33,844,281,476,-057,-056,-050,024,210126",
			checkFn: func(t *testing.T, r *Result) {
				if r.Date != "2026-01-21" {
					t.Errorf("Date = %v, want 2026-01-21", r.Date)
				}
				if r.Time != "09:46:55" {
					t.Errorf("Time = %v, want 09:46:55", r.Time)
				}
				if r.OriginICAO != "LLBG" {
					t.Errorf("OriginICAO = %v, want LLBG", r.OriginICAO)
				}
				if r.DestICAO != "EGLL" {
					t.Errorf("DestICAO = %v, want EGLL", r.DestICAO)
				}
				if r.GroundSpeed != 485 {
					t.Errorf("GroundSpeed = %v, want 485", r.GroundSpeed)
				}
				if r.FlightLevel != 360 {
					t.Errorf("FlightLevel = %v, want 360", r.FlightLevel)
				}
				if r.FuelOnBoard != 193 {
					t.Errorf("FuelOnBoard = %v, want 193", r.FuelOnBoard)
				}
				if r.Temperature != -63 {
					t.Errorf("Temperature = %v, want -63", r.Temperature)
				}
				if r.WindDir != 187 {
					t.Errorf("WindDir = %v, want 187", r.WindDir)
				}
				if r.WindSpeedKts != 17 {
					t.Errorf("WindSpeedKts = %v, want 17", r.WindSpeedKts)
				}
				if r.WindSpeedKmh != 31 { // 17 * 1.852 = 31.484 -> 31
					t.Errorf("WindSpeedKmh = %v, want 31", r.WindSpeedKmh)
				}
				if r.NextWaypoint != "DOLEV" {
					t.Errorf("NextWaypoint = %v, want DOLEV", r.NextWaypoint)
				}
				if r.NextWptETA != "09:49" {
					t.Errorf("NextWptETA = %v, want 09:49", r.NextWptETA)
				}
				if r.FollowWaypoint != "BEDOX" {
					t.Errorf("FollowWaypoint = %v, want BEDOX", r.FollowWaypoint)
				}
			},
		},
		{
			name: "Sample 3 - LLBG to LFPG (short format)",
			text: "2026-01-21,09:59:26,LLBG,LFPG,0323,N42055E021177,476,FL380, 18.2,-60,193, 8,DOLEV ,100718,",
			checkFn: func(t *testing.T, r *Result) {
				if r.Date != "2026-01-21" {
					t.Errorf("Date = %v, want 2026-01-21", r.Date)
				}
				if r.OriginICAO != "LLBG" {
					t.Errorf("OriginICAO = %v, want LLBG", r.OriginICAO)
				}
				if r.DestICAO != "LFPG" {
					t.Errorf("DestICAO = %v, want LFPG", r.DestICAO)
				}
				if r.GroundSpeed != 476 {
					t.Errorf("GroundSpeed = %v, want 476", r.GroundSpeed)
				}
				if r.FlightLevel != 380 {
					t.Errorf("FlightLevel = %v, want 380", r.FlightLevel)
				}
				// Fuel is 18.2, should parse as 18
				if r.FuelOnBoard != 18 {
					t.Errorf("FuelOnBoard = %v, want 18", r.FuelOnBoard)
				}
				if r.Temperature != -60 {
					t.Errorf("Temperature = %v, want -60", r.Temperature)
				}
				if r.WindDir != 193 {
					t.Errorf("WindDir = %v, want 193", r.WindDir)
				}
				if r.WindSpeedKts != 8 {
					t.Errorf("WindSpeedKts = %v, want 8", r.WindSpeedKts)
				}
				if r.NextWaypoint != "DOLEV" {
					t.Errorf("NextWaypoint = %v, want DOLEV", r.NextWaypoint)
				}
			},
		},
		{
			name: "Sample 4 - LLBG to EGLL (reference example)",
			text: "2026-01-21,10:01:56,LLBG,EGLL,0315,N44064E018441,492,FL360,0180,-65,161, 19,BEDOX ,10:23,SIMBA ,-36,846,282,474,-049,-048,-043,024,210126",
			checkFn: func(t *testing.T, r *Result) {
				if r.Date != "2026-01-21" {
					t.Errorf("Date = %v, want 2026-01-21", r.Date)
				}
				if r.Time != "10:01:56" {
					t.Errorf("Time = %v, want 10:01:56", r.Time)
				}
				if r.OriginICAO != "LLBG" {
					t.Errorf("OriginICAO = %v, want LLBG", r.OriginICAO)
				}
				if r.DestICAO != "EGLL" {
					t.Errorf("DestICAO = %v, want EGLL", r.DestICAO)
				}
				if r.GroundSpeed != 492 {
					t.Errorf("GroundSpeed = %v, want 492", r.GroundSpeed)
				}
				if r.FlightLevel != 360 {
					t.Errorf("FlightLevel = %v, want 360", r.FlightLevel)
				}
				if r.FuelOnBoard != 180 {
					t.Errorf("FuelOnBoard = %v, want 180", r.FuelOnBoard)
				}
				if r.Temperature != -65 {
					t.Errorf("Temperature = %v, want -65", r.Temperature)
				}
				if r.WindDir != 161 {
					t.Errorf("WindDir = %v, want 161", r.WindDir)
				}
				if r.WindSpeedKts != 19 {
					t.Errorf("WindSpeedKts = %v, want 19", r.WindSpeedKts)
				}
				// 19 * 1.852 = 35.188 km/h -> 35
				if r.WindSpeedKmh != 35 {
					t.Errorf("WindSpeedKmh = %v, want 35", r.WindSpeedKmh)
				}
				if r.NextWaypoint != "BEDOX" {
					t.Errorf("NextWaypoint = %v, want BEDOX", r.NextWaypoint)
				}
				if r.NextWptETA != "10:23" {
					t.Errorf("NextWptETA = %v, want 10:23", r.NextWptETA)
				}
				if r.FollowWaypoint != "SIMBA" {
					t.Errorf("FollowWaypoint = %v, want SIMBA", r.FollowWaypoint)
				}
				// Check coordinates parsing - N44064E018441
				// N44064 -> 44 degrees + 064/10 minutes = 44 + 6.4/60 = 44.107°
				// E018441 -> 18 degrees + 441/10 minutes = 18 + 44.1/60 = 18.735°
				if r.Latitude < 44.10 || r.Latitude > 44.11 {
					t.Errorf("Latitude = %v, expected around 44.107", r.Latitude)
				}
				if r.Longitude < 18.73 || r.Longitude > 18.74 {
					t.Errorf("Longitude = %v, expected around 18.735", r.Longitude)
				}
			},
		},
		{
			name:    "Invalid - too few fields",
			text:    "2026-01-21,09:32:54,LLBG",
			wantNil: true,
		},
		{
			name:    "Invalid - no date format",
			text:    "INVALID,09:32:54,LLBG,KJFK,0009",
			wantNil: true,
		},
	}

	parser := &Parser{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &acars.Message{
				ID:        123,
				Timestamp: "2026-01-21T10:00:00Z",
				Tail:      "TEST123",
				Label:     "33",
				Text:      tt.text,
			}

			result := parser.Parse(msg)
			if tt.wantNil {
				if result != nil {
					t.Errorf("Parse() returned result, want nil")
				}
				return
			}

			if result == nil {
				t.Fatalf("Parse() returned nil, want result")
			}

			r, ok := result.(*Result)
			if !ok {
				t.Fatalf("Parse() returned wrong type")
			}

			if r.MsgType != "position" {
				t.Errorf("MsgType = %v, want position", r.MsgType)
			}

			if tt.checkFn != nil {
				tt.checkFn(t, r)
			}
		})
	}
}

func TestCoordinateParsing(t *testing.T) {
	tests := []struct {
		name      string
		coord     string
		wantLat   float64
		wantLon   float64
		tolerance float64
	}{
		{
			name:      "Sample 1 coords",
			coord:     "N43350E021400",
			wantLat:   43.583,
			wantLon:   21.667,
			tolerance: 0.01,
		},
		{
			name:      "Sample 2 coords",
			coord:     "N42379E020382",
			wantLat:   42.632,
			wantLon:   20.637,
			tolerance: 0.01,
		},
		{
			name:      "Sample 3 coords",
			coord:     "N42055E021177",
			wantLat:   42.092,
			wantLon:   21.295,
			tolerance: 0.01,
		},
		{
			name:      "Sample 4 coords",
			coord:     "N44064E018441",
			wantLat:   44.107,
			wantLon:   18.735,
			tolerance: 0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lat, lon := parseCoordinates(tt.coord)

			if lat < tt.wantLat-tt.tolerance || lat > tt.wantLat+tt.tolerance {
				t.Errorf("Latitude = %v, want %v (±%v)", lat, tt.wantLat, tt.tolerance)
			}
			if lon < tt.wantLon-tt.tolerance || lon > tt.wantLon+tt.tolerance {
				t.Errorf("Longitude = %v, want %v (±%v)", lon, tt.wantLon, tt.tolerance)
			}

			t.Logf("Parsed %s: lat=%.4f, lon=%.4f", tt.coord, lat, lon)
		})
	}
}
