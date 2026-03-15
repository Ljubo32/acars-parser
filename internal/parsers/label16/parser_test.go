package label16

import (
	"math"
	"testing"

	"acars_parser/internal/acars"
)

func TestParsePOSAPosition(t *testing.T) {
	tests := []struct {
		name                string
		text                string
		wantLatitude        float64
		wantLongitude       float64
		wantFlightLevel     int
		wantReference       string
		wantCurrentWaypoint string
		wantCurrentETA      string
		wantNextWaypoint    string
		wantNextETA         string
		wantTemperature     string
		wantWind            string
		wantWindSpeed       int
		wantFuelOnBoard     int
		wantMach            float64
	}{
		{
			name:                "POSA Europe sample one",
			text:                "POSA1N42851E 16405,GIS40  ,092609,380,ROTAR  ,100331,,-58, 22, 306,844",
			wantLatitude:        42.851,
			wantLongitude:       16.405,
			wantFlightLevel:     380,
			wantReference:       "POSA",
			wantCurrentWaypoint: "GIS40",
			wantCurrentETA:      "09:26:09",
			wantNextWaypoint:    "ROTAR",
			wantNextETA:         "10:03:31",
			wantTemperature:     "-58",
			wantWind:            "22",
			wantWindSpeed:       22,
			wantFuelOnBoard:     306,
			wantMach:            0.844,
		},
		{
			name:                "POSA Europe sample two",
			text:                "POSA1N46594E 20528,TEGRI  ,072800,309,TEG-03 ,073508,,-52, 13, 808,827",
			wantLatitude:        46.594,
			wantLongitude:       20.528,
			wantFlightLevel:     309,
			wantReference:       "POSA",
			wantCurrentWaypoint: "TEGRI",
			wantCurrentETA:      "07:28:00",
			wantNextWaypoint:    "TEG-03",
			wantNextETA:         "07:35:08",
			wantTemperature:     "-52",
			wantWind:            "13",
			wantWindSpeed:       13,
			wantFuelOnBoard:     808,
			wantMach:            0.827,
		},
		{
			name:                "POSA with masked temperature",
			text:                "POSA1N45874E 25393,ROMAG  ,094845,370,ROM19  ,101423,,*****,20.14, 869,   0",
			wantLatitude:        45.874,
			wantLongitude:       25.393,
			wantFlightLevel:     370,
			wantReference:       "POSA",
			wantCurrentWaypoint: "ROMAG",
			wantCurrentETA:      "09:48:45",
			wantNextWaypoint:    "ROM19",
			wantNextETA:         "10:14:23",
			wantTemperature:     "*****",
			wantWind:            "20.14",
			wantWindSpeed:       0,
			wantFuelOnBoard:     869,
			wantMach:            0,
		},
	}

	parser := &Parser{}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := &acars.Message{
				ID:        acars.FlexInt64(101),
				Timestamp: "2026-03-14T00:00:00Z",
				Tail:      "TEST01",
				Label:     "16",
				Text:      tc.text,
			}

			parsed := parser.Parse(msg)
			if parsed == nil {
				t.Fatal("Parse() returned nil")
			}

			result, ok := parsed.(*Result)
			if !ok {
				t.Fatalf("Parse() returned %T, want *Result", parsed)
			}

			assertFloatEqual(t, "latitude", result.Latitude, tc.wantLatitude)
			assertFloatEqual(t, "longitude", result.Longitude, tc.wantLongitude)
			if result.FlightLevel != tc.wantFlightLevel {
				t.Errorf("FlightLevel = %d, want %d", result.FlightLevel, tc.wantFlightLevel)
			}
			if result.Reference != tc.wantReference {
				t.Errorf("Reference = %q, want %q", result.Reference, tc.wantReference)
			}
			if result.Waypoint != tc.wantReference {
				t.Errorf("Waypoint = %q, want %q", result.Waypoint, tc.wantReference)
			}
			if result.CurrentWaypoint != tc.wantCurrentWaypoint {
				t.Errorf("CurrentWaypoint = %q, want %q", result.CurrentWaypoint, tc.wantCurrentWaypoint)
			}
			if result.CurrentWaypointETA != tc.wantCurrentETA {
				t.Errorf("CurrentWaypointETA = %q, want %q", result.CurrentWaypointETA, tc.wantCurrentETA)
			}
			if result.NextWaypoint != tc.wantNextWaypoint {
				t.Errorf("NextWaypoint = %q, want %q", result.NextWaypoint, tc.wantNextWaypoint)
			}
			if result.NextWaypointETA != tc.wantNextETA {
				t.Errorf("NextWaypointETA = %q, want %q", result.NextWaypointETA, tc.wantNextETA)
			}
			if result.ETA != tc.wantNextETA {
				t.Errorf("ETA = %q, want %q", result.ETA, tc.wantNextETA)
			}
			if result.Temperature != tc.wantTemperature {
				t.Errorf("Temperature = %q, want %q", result.Temperature, tc.wantTemperature)
			}
			if result.Wind != tc.wantWind {
				t.Errorf("Wind = %q, want %q", result.Wind, tc.wantWind)
			}
			if result.WindSpeed != tc.wantWindSpeed {
				t.Errorf("WindSpeed = %d, want %d", result.WindSpeed, tc.wantWindSpeed)
			}
			if result.FuelOnBoard != tc.wantFuelOnBoard {
				t.Errorf("FuelOnBoard = %d, want %d", result.FuelOnBoard, tc.wantFuelOnBoard)
			}
			assertFloatEqual(t, "mach", result.Mach, tc.wantMach)

			if len(result.Waypoints) != 2 {
				t.Fatalf("len(Waypoints) = %d, want 2", len(result.Waypoints))
			}
			if result.Waypoints[0].Name != tc.wantCurrentWaypoint || result.Waypoints[0].ETA != tc.wantCurrentETA {
				t.Errorf("Waypoints[0] = %+v, want {%q %q}", result.Waypoints[0], tc.wantCurrentWaypoint, tc.wantCurrentETA)
			}
			if result.Waypoints[1].Name != tc.wantNextWaypoint || result.Waypoints[1].ETA != tc.wantNextETA {
				t.Errorf("Waypoints[1] = %+v, want {%q %q}", result.Waypoints[1], tc.wantNextWaypoint, tc.wantNextETA)
			}
		})
	}
}

func TestParseClassicWaypointPositionStillWorks(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		ID:        acars.FlexInt64(102),
		Timestamp: "2026-03-14T00:00:00Z",
		Tail:      "TEST02",
		Label:     "16",
		Text:      "BEGLA  ,N 47.555,E 18.028,40025,490,1934,030",
	}

	parsed := parser.Parse(msg)
	if parsed == nil {
		t.Fatal("Parse() returned nil")
	}

	result, ok := parsed.(*Result)
	if !ok {
		t.Fatalf("Parse() returned %T, want *Result", parsed)
	}

	if result.Waypoint != "BEGLA" {
		t.Errorf("Waypoint = %q, want %q", result.Waypoint, "BEGLA")
	}
	assertFloatEqual(t, "latitude", result.Latitude, 47.555)
	assertFloatEqual(t, "longitude", result.Longitude, 18.028)
	if result.FlightLevel != 400 {
		t.Errorf("FlightLevel = %d, want 400", result.FlightLevel)
	}
	if result.GroundSpeed != 490 {
		t.Errorf("GroundSpeed = %d, want 490", result.GroundSpeed)
	}
	if result.ETA != "1934" {
		t.Errorf("ETA = %q, want %q", result.ETA, "1934")
	}
	if result.Track != 30 {
		t.Errorf("Track = %d, want 30", result.Track)
	}
}

func assertFloatEqual(t *testing.T, field string, got, want float64) {
	t.Helper()

	if math.Abs(got-want) > 0.0001 {
		t.Errorf("%s = %.6f, want %.6f", field, got, want)
	}
}