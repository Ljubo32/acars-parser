// Package label33 parses Label 33 position report messages.
package label33

import (
	"fmt"
	"strconv"
	"strings"

	"acars_parser/internal/acars"
	"acars_parser/internal/airports"
	"acars_parser/internal/registry"
)

// Result represents a position report from label 33 messages.
type Result struct {
	MsgID          int64   `json:"message_id"`
	Timestamp      string  `json:"timestamp"`
	Tail           string  `json:"tail,omitempty"`
	MsgType        string  `json:"msg_type"`
	Date           string  `json:"date,omitempty"`             // Report date (YYYY-MM-DD)
	Time           string  `json:"time,omitempty"`             // Report time (HH:MM:SS)
	OriginICAO     string  `json:"origin_icao,omitempty"`      // Origin airport
	OriginName     string  `json:"origin_name,omitempty"`      // Airport name from ICAO
	DestICAO       string  `json:"dest_icao,omitempty"`        // Destination airport
	DestName       string  `json:"dest_name,omitempty"`        // Airport name from ICAO
	Latitude       float64 `json:"latitude,omitempty"`         // Latitude from coordinates
	Longitude      float64 `json:"longitude,omitempty"`        // Longitude from coordinates
	GroundSpeed    int     `json:"ground_speed_kts,omitempty"` // Ground speed in knots
	FlightLevel    int     `json:"flight_level,omitempty"`     // Flight level (e.g., 360 for FL360)
	FuelOnBoard    int     `json:"fuel_on_board,omitempty"`    // Fuel on board
	Temperature    int     `json:"temperature_c,omitempty"`    // Temperature in Celsius
	WindDir        int     `json:"wind_dir,omitempty"`         // Wind direction in degrees
	WindSpeedKts   int     `json:"wind_speed_kts,omitempty"`   // Wind speed in knots
	WindSpeedKmh   int     `json:"wind_speed_kmh,omitempty"`   // Wind speed in km/h
	NextWaypoint   string  `json:"next_waypoint,omitempty"`    // Next waypoint identifier
	NextWptETA     string  `json:"next_wpt_eta,omitempty"`     // ETA to next waypoint (HH:MM)
	FollowWaypoint string  `json:"follow_waypoint,omitempty"`  // Following waypoint
	RawCoordinates string  `json:"raw_coordinates,omitempty"`  // Raw coordinate string
}

func (r *Result) Type() string     { return "position" }
func (r *Result) MessageID() int64 { return r.MsgID }

// Parser parses Label 33 position messages.
type Parser struct{}

func init() {
	registry.Register(&Parser{})
}

func (p *Parser) Name() string     { return "label33" }
func (p *Parser) Labels() []string { return []string{"33"} }
func (p *Parser) Priority() int    { return 100 }

func (p *Parser) QuickCheck(text string) bool {
	// Quick check: should contain date format and coordinates
	return strings.Contains(text, ",N") || strings.Contains(text, ",S")
}

func (p *Parser) Parse(msg *acars.Message) registry.Result {
	// Parse CSV format
	fields := strings.Split(msg.Text, ",")

	// Need at least 15 fields for a valid message
	if len(fields) < 15 {
		return nil
	}

	// Validate first field looks like a date (YYYY-MM-DD)
	if len(fields[0]) != 10 || !strings.Contains(fields[0], "-") {
		return nil
	}

	result := &Result{
		MsgID:     int64(msg.ID),
		Timestamp: msg.Timestamp,
		Tail:      msg.Tail,
		MsgType:   "position",
		Date:      fields[0],
	}

	// Parse time (field 1)
	if len(fields) > 1 {
		result.Time = fields[1]
	}

	// Parse origin (field 2)
	if len(fields) > 2 {
		result.OriginICAO = strings.TrimSpace(fields[2])
		if name := airports.GetName(result.OriginICAO); name != "" {
			result.OriginName = name
		}
	}

	// Parse destination (field 3)
	if len(fields) > 3 {
		result.DestICAO = strings.TrimSpace(fields[3])
		if name := airports.GetName(result.DestICAO); name != "" {
			result.DestName = name
		}
	}

	// Field 4 is unknown, skip it

	// Parse coordinates (field 5)
	if len(fields) > 5 {
		coord := strings.TrimSpace(fields[5])
		result.RawCoordinates = coord
		lat, lon := parseCoordinates(coord)
		if lat != 0 || lon != 0 {
			result.Latitude = lat
			result.Longitude = lon
		}
	}

	// Parse ground speed (field 6)
	if len(fields) > 6 {
		if gs, err := strconv.Atoi(strings.TrimSpace(fields[6])); err == nil {
			result.GroundSpeed = gs
		}
	}

	// Parse flight level (field 7)
	if len(fields) > 7 {
		fl := strings.TrimSpace(fields[7])
		if strings.HasPrefix(fl, "FL") {
			if level, err := strconv.Atoi(fl[2:]); err == nil {
				result.FlightLevel = level
			}
		}
	}

	// Parse fuel on board (field 8)
	if len(fields) > 8 {
		fob := strings.TrimSpace(fields[8])
		// Handle decimal values like "18.2" - convert to int
		if strings.Contains(fob, ".") {
			parts := strings.Split(fob, ".")
			if len(parts) > 0 {
				fob = parts[0]
			}
		}
		if fob != "" && fob != " " {
			if fuel, err := strconv.Atoi(fob); err == nil {
				result.FuelOnBoard = fuel
			}
		}
	}

	// Parse temperature (field 9)
	if len(fields) > 9 {
		if temp, err := strconv.Atoi(strings.TrimSpace(fields[9])); err == nil {
			result.Temperature = temp
		}
	}

	// Parse wind direction (field 10)
	if len(fields) > 10 {
		if wd, err := strconv.Atoi(strings.TrimSpace(fields[10])); err == nil {
			result.WindDir = wd
		}
	}

	// Parse wind speed (field 11)
	if len(fields) > 11 {
		ws := strings.TrimSpace(fields[11])
		if windSpeed, err := strconv.Atoi(ws); err == nil {
			result.WindSpeedKts = windSpeed
			// Convert knots to km/h (1 knot = 1.852 km/h)
			result.WindSpeedKmh = int(float64(windSpeed) * 1.852)
		}
	}

	// Parse next waypoint (field 12)
	if len(fields) > 12 {
		result.NextWaypoint = strings.TrimSpace(fields[12])
	}

	// Parse next waypoint ETA (field 13)
	if len(fields) > 13 {
		result.NextWptETA = strings.TrimSpace(fields[13])
	}

	// Parse following waypoint (field 14)
	if len(fields) > 14 {
		result.FollowWaypoint = strings.TrimSpace(fields[14])
	}

	// Additional fields may exist but are not yet understood

	return result
}

// parseCoordinates parses coordinates in format N43350E021400 or N42379E020382
// Returns latitude and longitude as decimal degrees
func parseCoordinates(coord string) (float64, float64) {
	coord = strings.TrimSpace(coord)
	if len(coord) < 10 {
		return 0, 0
	}

	// Find the position of E or W (longitude indicator)
	var lonIdx int
	for i, ch := range coord {
		if ch == 'E' || ch == 'W' {
			lonIdx = i
			break
		}
	}

	if lonIdx == 0 {
		return 0, 0
	}

	// Extract latitude part (e.g., N43350)
	latPart := coord[:lonIdx]
	if len(latPart) < 2 {
		return 0, 0
	}

	// Extract longitude part (e.g., E021400)
	lonPart := coord[lonIdx:]
	if len(lonPart) < 2 {
		return 0, 0
	}

	// Parse latitude
	latDir := latPart[0]
	latValue := latPart[1:]
	lat := parseCoordValue(latValue, 2) // 2 digits for degrees in latitude
	if latDir == 'S' {
		lat = -lat
	}

	// Parse longitude
	lonDir := lonPart[0]
	lonValue := lonPart[1:]
	lon := parseCoordValue(lonValue, 3) // 3 digits for degrees in longitude
	if lonDir == 'W' {
		lon = -lon
	}

	return lat, lon
}

// parseCoordValue converts coordinate string to decimal degrees
// degDigits specifies how many digits are for degrees (2 for lat, 3 for lon)
func parseCoordValue(value string, degDigits int) float64 {
	if len(value) < degDigits {
		return 0
	}

	// Extract degrees
	degStr := value[:degDigits]
	deg, err := strconv.ParseFloat(degStr, 64)
	if err != nil {
		return 0
	}

	// Extract minutes (remaining digits represent minutes)
	// The format varies based on number of digits:
	// For 3 digits (350): represents MM.M format, so 35.0 minutes (divide by 10)
	// For 4 digits (1400): represents MM.MM format as MMDD, so 14.00 minutes (first 2 digits are minutes, last 2 are decimals)
	if len(value) > degDigits {
		minStr := value[degDigits:]
		if minValue, err := strconv.ParseFloat(minStr, 64); err == nil {
			var minutes float64
			if len(minStr) == 3 {
				// 3 digits: 350 = 35.0 minutes (MM.M format)
				minutes = minValue / 10.0
			} else if len(minStr) == 4 {
				// 4 digits: 1400 = 14.00 minutes (MMDD format where MM=minutes, DD=decimal hundredths)
				// Extract first 2 digits as minutes, last 2 as decimal part
				minutesPart := float64(int(minValue) / 100)
				decimalPart := float64(int(minValue) % 100) / 100.0
				minutes = minutesPart + decimalPart
			} else {
				// Default: divide by 10
				minutes = minValue / 10.0
			}
			deg += minutes / 60.0
		}
	}

	return deg
}

// FormatPosition returns a formatted position string
func (r *Result) FormatPosition() string {
	if r.Latitude != 0 || r.Longitude != 0 {
		return fmt.Sprintf("%.4f, %.4f", r.Latitude, r.Longitude)
	}
	return ""
}
