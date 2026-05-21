// Package label83 parses Label 83 position report messages.
package label83

import (
	"strconv"
	"strings"
	"sync"

	"acars_parser/internal/acars"
	"acars_parser/internal/airports"
	"acars_parser/internal/patterns"
	"acars_parser/internal/registry"
)

// Result represents a parsed Label 83 position report.
type Result struct {
	MsgID           int64   `json:"message_id"`
	Timestamp       string  `json:"timestamp"`
	Tail            string  `json:"tail,omitempty"`
	MessageType     string  `json:"message_type"` // PR, ZSPD, or POSRPT
	DayOfMonth      int     `json:"day_of_month,omitempty"`
	ReportTime      string  `json:"report_time,omitempty"`
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
	FlightLevel     int     `json:"flight_level,omitempty"`
	Heading         int     `json:"heading,omitempty"`
	GroundSpeed     float64 `json:"ground_speed,omitempty"`
	Origin          string  `json:"origin,omitempty"`
	Destination     string  `json:"destination,omitempty"`
	OriginName      string  `json:"origin_name,omitempty"`
	DestinationName string  `json:"destination_name,omitempty"`
	// POSRPT-only meteorological fields (SAT, SWND, DWND keys).
	// Note: omitempty means a true 0 °C temperature will be absent from the JSON.
	TemperatureC  int `json:"temperature_c,omitempty"`  // SAT (Static Air Temperature) in Celsius
	WindSpeedKts  int `json:"wind_speed_kts,omitempty"` // SWND (wind speed) in knots
	WindSpeedKmh  int `json:"wind_speed_kmh,omitempty"` // Computed from SWND
	WindDirection int `json:"wind_direction,omitempty"` // DWND (wind direction) in degrees
}

func (r *Result) Type() string     { return "label83_position" }
func (r *Result) MessageID() int64 { return r.MsgID }

// Parser parses Label 83 position report messages.
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

func (p *Parser) Name() string     { return "label83" }
func (p *Parser) Labels() []string { return []string{"83"} }
func (p *Parser) Priority() int    { return 100 }

func (p *Parser) QuickCheck(text string) bool {
	t := strings.TrimSpace(text)
	if strings.Contains(t, "PR") || strings.Contains(t, "ZSPD") || strings.Contains(t, "POSRPT") {
		return true
	}
	// CSV position format: starts directly with two 4-character ICAO codes
	// separated by a comma (e.g. "KORD,EGLL,130317,...").
	return len(t) >= 9 && t[4] == ',' && isUpperAlpha(t[0:4]) && isUpperAlpha(t[5:9])
}

// isUpperAlpha reports whether s consists entirely of uppercase ASCII letters.
func isUpperAlpha(s string) bool {
	for _, c := range s {
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	return len(s) > 0
}

// normaliseSignedFloat strips any internal whitespace from a captured signed
// decimal value so that strconv.ParseFloat succeeds (e.g. "- 29.34" → "-29.34").
func normaliseSignedFloat(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), " ", "")
}

func (p *Parser) Parse(msg *acars.Message) registry.Result {
	if msg.Text == "" {
		return nil
	}

	text := strings.TrimSpace(msg.Text)

	// POSRPT messages use a slash-delimited key/value structure that doesn't
	// fit the grok pattern model; parse them directly.
	if strings.Contains(text, "POSRPT") {
		return parsePOSRPT(msg, text)
	}

	// Try grok-based parsing.
	compiler, err := getCompiler()
	if err != nil {
		return nil
	}

	match := compiler.Parse(text)
	if match == nil {
		return nil
	}

	result := &Result{
		MsgID:     int64(msg.ID),
		Timestamp: msg.Timestamp,
		Tail:      msg.Tail,
	}

	switch match.FormatName {
	case "pr_position":
		result.MessageType = "PR"
		result.ReportTime = match.Captures["time"]

		if day, err := strconv.Atoi(match.Captures["day"]); err == nil {
			result.DayOfMonth = day
		}

		// Parse latitude (format: DDMM.D - 2 degree digits, decimal minutes).
		result.Latitude = patterns.ParseLatitude(match.Captures["lat"], match.Captures["lat_dir"])

		// Parse longitude (format: DDDMM.D - 3 degree digits, decimal minutes).
		result.Longitude = patterns.ParseLongitude(match.Captures["lon"], match.Captures["lon_dir"])

		// Parse flight level from altitude (show only first 3 digits as int, e.g. 370465 -> 370)
		if alt, err := strconv.Atoi(match.Captures["altitude"]); err == nil {
			result.FlightLevel = alt / 1000
		}

	case "zspd_position":
		result.MessageType = "ZSPD"
		result.Origin = match.Captures["origin"]
		result.Destination = match.Captures["dest"]
		result.OriginName = airports.GetName(result.Origin)
		result.DestinationName = airports.GetName(result.Destination)
		result.ReportTime = match.Captures["time"]

		// Signed decimal coordinates may carry an internal space after the minus
		// sign (e.g. "- 29.34"); normalise before parsing.
		if lat, err := strconv.ParseFloat(normaliseSignedFloat(match.Captures["lat"]), 64); err == nil {
			result.Latitude = lat
		}
		if lon, err := strconv.ParseFloat(normaliseSignedFloat(match.Captures["lon"]), 64); err == nil {
			result.Longitude = lon
		}

		// Altitude is in feet; derive the flight level by dividing by 100.
		if alt, err := strconv.Atoi(match.Captures["altitude"]); err == nil {
			result.FlightLevel = alt / 100
		}

		if hdg, err := strconv.Atoi(match.Captures["heading"]); err == nil {
			result.Heading = hdg
		}

		// Ground speed may also carry a sign with an internal space.
		if gs, err := strconv.ParseFloat(normaliseSignedFloat(match.Captures["ground_speed"]), 64); err == nil {
			result.GroundSpeed = gs
		}
	}

	return result
}

// parsePOSRPT parses a POSRPT (position report) message from its
// slash-delimited key/value structure.
//
// Example:
//
//	3N01 POSRPT 0182/SKBO/LEMD .N783AV/03A03:40/WPY N34W0/NWYP MANOX
//	/HDG  64.80/MCH    .86/POS N29304 W029014/FL 40000/TAS 495/...
func parsePOSRPT(msg *acars.Message, text string) *Result {
	slashFields := strings.Split(text, "/")
	if len(slashFields) < 3 {
		return nil
	}

	result := &Result{
		MsgID:       int64(msg.ID),
		Timestamp:   msg.Timestamp,
		Tail:        msg.Tail,
		MessageType: "POSRPT",
	}

	// Field 1: origin ICAO code (e.g. "SKBO").
	if origin := strings.TrimSpace(slashFields[1]); isICAOCode(origin) {
		result.Origin = origin
		result.OriginName = airports.GetName(origin)
	}

	// Field 2: destination ICAO code, optionally followed by the aircraft
	// registration separated by a dot (e.g. "LEMD .N783AV").
	if len(slashFields) > 2 {
		destField := strings.TrimSpace(slashFields[2])
		if len(destField) >= 4 && isICAOCode(destField[:4]) {
			result.Destination = destField[:4]
			result.DestinationName = airports.GetName(result.Destination)
		}
		if dotIdx := strings.LastIndex(destField, "."); dotIdx >= 0 {
			if reg := strings.TrimSpace(destField[dotIdx+1:]); reg != "" && result.Tail == "" {
				result.Tail = reg
			}
		}
	}

	// Field 3: time block (e.g. "03A03:40"); extract the HH:MM portion.
	if len(slashFields) > 3 {
		tf := strings.TrimSpace(slashFields[3])
		if idx := strings.Index(tf, ":"); idx >= 2 && idx+3 <= len(tf) {
			result.ReportTime = tf[idx-2 : idx+3]
		}
	}

	// Fields 4+: space-separated KEY VALUE pairs.
	for _, f := range slashFields[4:] {
		parsePOSRPTField(strings.TrimSpace(f), result)
	}

	if result.Origin == "" && result.Destination == "" {
		return nil
	}
	return result
}

// parsePOSRPTField parses one slash-delimited field from a POSRPT message
// (e.g. "HDG  64.80", "POS N29304 W029014", "FL 40000") and writes recognised
// values into result.
func parsePOSRPTField(f string, result *Result) {
	spIdx := strings.IndexByte(f, ' ')
	if spIdx < 0 {
		return
	}
	key := strings.TrimSpace(f[:spIdx])
	val := strings.TrimSpace(f[spIdx+1:])

	switch key {
	case "POS":
		// Format: "N29304 W029014" — latitude in DDMMD, longitude in DDDMMD.
		parts := strings.Fields(val)
		if len(parts) < 2 {
			return
		}
		latPart, lonPart := parts[0], parts[1]
		if len(latPart) >= 2 {
			result.Latitude = patterns.ParseDMSCoord(latPart[1:], 2, string(latPart[0]))
		}
		if len(lonPart) >= 2 {
			result.Longitude = patterns.ParseDMSCoord(lonPart[1:], 3, string(lonPart[0]))
		}
	case "FL":
		// The value is the altitude in feet (e.g. "40000" = FL400).
		if alt, err := strconv.Atoi(val); err == nil && alt > 0 {
			result.FlightLevel = alt / 100
		}
	case "HDG":
		// Heading in degrees, may be fractional; round to nearest integer.
		if hdg, err := strconv.ParseFloat(val, 64); err == nil {
			result.Heading = int(hdg + 0.5)
		}
	case "TAS":
		// True airspeed in knots; stored as ground speed (the best speed
		// indicator available in a POSRPT).
		if tas, err := strconv.Atoi(val); err == nil {
			result.GroundSpeed = float64(tas)
		}
	case "SAT":
		// Static Air Temperature in Celsius.  The value may carry a space
		// between the sign and digits (e.g. "- 58"); strip internal whitespace
		// before parsing.
		if temp, err := strconv.Atoi(strings.ReplaceAll(strings.TrimSpace(val), " ", "")); err == nil {
			result.TemperatureC = temp
		}
	case "SWND":
		// Wind speed in knots.
		if ws, err := strconv.Atoi(strings.TrimSpace(val)); err == nil && ws >= 0 {
			result.WindSpeedKts = ws
			result.WindSpeedKmh = int(float64(ws)*1.852 + 0.5)
		}
	case "DWND":
		// Wind direction in degrees.
		if wd, err := strconv.Atoi(strings.TrimSpace(val)); err == nil && wd >= 0 && wd <= 360 {
			result.WindDirection = wd
		}
	}
}

// isICAOCode reports whether s is a 4-character uppercase ASCII string
// suitable as an ICAO airport code.
func isICAOCode(s string) bool {
	if len(s) != 4 {
		return false
	}
	for _, c := range s {
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	return true
}
