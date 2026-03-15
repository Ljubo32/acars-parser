// Package label16 parses Label 16 waypoint position messages.
package label16

import (
	"strconv"
	"strings"
	"sync"

	"acars_parser/internal/acars"
	"acars_parser/internal/patterns"
	"acars_parser/internal/registry"
)

// WaypointETA represents a waypoint reference paired with its ETA.
type WaypointETA struct {
	Name string `json:"name"`
	ETA  string `json:"eta,omitempty"`
}

// Result represents a waypoint position from label 16 messages.
type Result struct {
	MsgID              int64         `json:"message_id"`
	Timestamp          string        `json:"timestamp"`
	Tail               string        `json:"tail,omitempty"`
	Time               string        `json:"time,omitempty"`
	Flight             string        `json:"flight,omitempty"`
	Reference          string        `json:"reference,omitempty"`
	Waypoint           string        `json:"waypoint,omitempty"`
	CurrentWaypoint    string        `json:"current_waypoint,omitempty"`
	CurrentWaypointETA string        `json:"current_waypoint_eta,omitempty"`
	NextWaypoint       string        `json:"next_waypoint,omitempty"`
	NextWaypointETA    string        `json:"next_waypoint_eta,omitempty"`
	Waypoints          []WaypointETA `json:"waypoints,omitempty"`
	Latitude           float64       `json:"latitude"`
	Longitude          float64       `json:"longitude"`
	FlightLevel        int           `json:"flight_level,omitempty"`
	GroundSpeed        int           `json:"ground_speed,omitempty"`
	ETA                string        `json:"eta,omitempty"`
	Track              int           `json:"track,omitempty"`
	Temperature        string        `json:"temperature,omitempty"`
	Wind               string        `json:"wind,omitempty"`
	WindSpeed          int           `json:"wind_speed,omitempty"`
	FuelOnBoard        int           `json:"fuel_on_board,omitempty"`
	Mach               float64       `json:"mach,omitempty"`
}

func (r *Result) Type() string     { return "waypoint_position" }
func (r *Result) MessageID() int64 { return r.MsgID }

// Parser parses Label 16 waypoint position messages.
type Parser struct{}

// Grok compiler singleton.
var (
	grokCompiler *patterns.Compiler
	grokOnce     sync.Once
	grokErr      error
)

// getCompiler returns the singleton grok compiler.
func getCompiler() (*patterns.Compiler, error) {
	grokOnce.Do(func() {
		grokCompiler = patterns.NewCompiler(Formats, nil)
		grokErr = grokCompiler.Compile()
	})
	return grokCompiler, grokErr
}

func init() {
	registry.Register(&Parser{})
}

func (p *Parser) Name() string     { return "label16" }
func (p *Parser) Labels() []string { return []string{"16"} }
func (p *Parser) Priority() int    { return 100 }

func (p *Parser) QuickCheck(text string) bool {
	return true // Label check is sufficient for 16.
}

func (p *Parser) Parse(msg *acars.Message) registry.Result {
	if msg.Text == "" {
		return nil
	}

	// Try grok-based parsing.
	compiler, err := getCompiler()
	if err != nil {
		return nil
	}

	match := compiler.Parse(msg.Text)
	if match == nil {
		return nil
	}

	result := &Result{
		MsgID:     int64(msg.ID),
		Timestamp: msg.Timestamp,
		Tail:      msg.Tail,
	}

	// Handle different format types.
	switch match.FormatName {
	case "csv_position", "csv_position_no_alt", "csv_position_extended":
		result.Latitude = patterns.ParseDecimalCoord(match.Captures["lat"], match.Captures["lat_dir"])
		result.Longitude = patterns.ParseDecimalCoord(match.Captures["lon"], match.Captures["lon_dir"])
		result.Time = match.Captures["time"]

		// Parse altitude (may have + or M prefix).
		if altStr := match.Captures["altitude"]; altStr != "" {
			altStr = strings.TrimPrefix(altStr, "+")
			altStr = strings.TrimPrefix(altStr, "M")
			if alt, err := strconv.Atoi(altStr); err == nil {
				if alt > 1000 {
					result.FlightLevel = alt / 100
				} else {
					result.FlightLevel = alt
				}
			}
		}

		// Parse speed.
		if speed, err := strconv.Atoi(match.Captures["speed"]); err == nil {
			result.GroundSpeed = speed
		}

		// Parse track.
		if track, err := strconv.Atoi(match.Captures["track"]); err == nil {
			result.Track = track
		}

	case "waypoint_position", "waypoint_position_prefixed":
		result.Waypoint = match.Captures["waypoint"]
		result.Latitude = patterns.ParseDecimalCoord(match.Captures["lat"], match.Captures["lat_dir"])
		result.Longitude = patterns.ParseDecimalCoord(match.Captures["lon"], match.Captures["lon_dir"])
		result.ETA = match.Captures["eta"]

		// Convert altitude to flight level.
		if alt, err := strconv.Atoi(match.Captures["altitude"]); err == nil {
			if alt > 1000 {
				result.FlightLevel = alt / 100
			} else {
				result.FlightLevel = alt
			}
		}

		// Parse ground speed.
		if gs, err := strconv.Atoi(match.Captures["ground_speed"]); err == nil {
			result.GroundSpeed = gs
		}

		// Parse track.
		if track, err := strconv.Atoi(match.Captures["track"]); err == nil {
			result.Track = track
		}

		// For prefixed format, extract and flatten the flight identifier.
		// Flattening removes leading zeros (e.g., "007K" -> "7K") to match ACARS envelope format.
		if airline := match.Captures["prefix_airline"]; airline != "" {
			flightNum := match.Captures["prefix_flight"]
			result.Flight = airline + strings.TrimLeft(flightNum, "0")
		}

	case "posa_position":
		result.Reference = match.Captures["reference"]
		result.Waypoint = result.Reference
		result.CurrentWaypoint = strings.TrimSpace(match.Captures["current_waypoint"])
		result.CurrentWaypointETA = formatETA(match.Captures["current_eta"])
		result.NextWaypoint = strings.TrimSpace(match.Captures["next_waypoint"])
		result.NextWaypointETA = formatETA(match.Captures["next_eta"])
		result.Waypoints = []WaypointETA{
			{Name: result.CurrentWaypoint, ETA: result.CurrentWaypointETA},
			{Name: result.NextWaypoint, ETA: result.NextWaypointETA},
		}
		result.ETA = result.NextWaypointETA
		result.Latitude = parsePOSADecimalCoord(match.Captures["lat"], match.Captures["lat_dir"])
		result.Longitude = parsePOSADecimalCoord(match.Captures["lon"], match.Captures["lon_dir"])
		result.Temperature = strings.TrimSpace(match.Captures["temperature"])
		result.Wind = strings.TrimSpace(match.Captures["wind"])

		if alt, err := strconv.Atoi(match.Captures["altitude"]); err == nil {
			if alt > 1000 {
				result.FlightLevel = alt / 100
			} else {
				result.FlightLevel = alt
			}
		}

		if windSpeed, err := strconv.Atoi(result.Wind); err == nil {
			result.WindSpeed = windSpeed
		}

		if fuelOnBoard, err := strconv.Atoi(strings.TrimSpace(match.Captures["fuel_on_board"])); err == nil {
			result.FuelOnBoard = fuelOnBoard
		}

		if mach, err := strconv.Atoi(match.Captures["mach"]); err == nil {
			result.Mach = float64(mach) / 1000.0
		}

	case "autpos":
		// AUTPOS has compact lat/lon format: N440853 W0915239 = N44°08'53" W091°52'39"
		result.Time = match.Captures["time"]
		result.Latitude = parseCompactCoord(match.Captures["lat"], match.Captures["lat_dir"])
		result.Longitude = parseCompactCoord(match.Captures["lon"], match.Captures["lon_dir"])

	default:
		return nil
	}

	// Validate we got coordinates.
	if result.Latitude == 0 && result.Longitude == 0 {
		return nil
	}

	return result
}

// parseCompactCoord parses compact format like "440853" (44°08'53") to decimal degrees.
func parseCompactCoord(coord, dir string) float64 {
	if coord == "" {
		return 0
	}

	var deg, min, sec float64

	switch len(coord) {
	case 6: // DDMMSS (latitude).
		deg, _ = strconv.ParseFloat(coord[0:2], 64)
		min, _ = strconv.ParseFloat(coord[2:4], 64)
		sec, _ = strconv.ParseFloat(coord[4:6], 64)
	case 7: // DDDMMSS (longitude).
		deg, _ = strconv.ParseFloat(coord[0:3], 64)
		min, _ = strconv.ParseFloat(coord[3:5], 64)
		sec, _ = strconv.ParseFloat(coord[5:7], 64)
	default:
		return 0
	}

	result := deg + min/60 + sec/3600

	if dir == "S" || dir == "W" {
		result = -result
	}

	return result
}

// parsePOSADecimalCoord parses POSA coordinates encoded as thousandths of a degree without a decimal point.
func parsePOSADecimalCoord(coord, dir string) float64 {
	if coord == "" {
		return 0
	}

	if strings.Contains(coord, ".") {
		return patterns.ParseDecimalCoord(coord, dir)
	}

	value, err := strconv.Atoi(coord)
	if err != nil {
		return 0
	}

	decimal := float64(value) / 1000.0
	if dir == "S" || dir == "W" {
		return -decimal
	}

	return decimal
}

// formatETA converts HHMMSS values to HH:MM:SS and leaves other formats unchanged.
func formatETA(value string) string {
	value = strings.TrimSpace(value)
	if len(value) != 6 {
		return value
	}

	for _, char := range value {
		if char < '0' || char > '9' {
			return value
		}
	}

	return value[0:2] + ":" + value[2:4] + ":" + value[4:6]
}
