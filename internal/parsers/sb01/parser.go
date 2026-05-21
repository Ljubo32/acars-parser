// Package sb01 parses SB01 compact position/status messages carried on label H1.
package sb01

import (
	"strconv"
	"strings"

	"acars_parser/internal/acars"
	"acars_parser/internal/registry"
)

// Result represents a parsed SB01 report.
type Result struct {
	MsgID         int64   `json:"message_id"`
	Timestamp     string  `json:"timestamp"`
	Tail          string  `json:"tail,omitempty"`
	MsgType       string  `json:"msg_type,omitempty"`
	Registration  string  `json:"registration,omitempty"`
	Sequence      string  `json:"sequence,omitempty"`
	Route         string  `json:"route,omitempty"`
	Latitude      float64 `json:"latitude,omitempty"`
	Longitude     float64 `json:"longitude,omitempty"`
	ReportTime    string  `json:"report_time,omitempty"`
	AltitudeFt    int     `json:"altitude_ft,omitempty"`
	AltitudeM     int     `json:"altitude_m,omitempty"`
	TemperatureC  float64 `json:"temperature_c,omitempty"`
	WindDirection int     `json:"wind_direction,omitempty"`
	WindSpeedKts  int     `json:"wind_speed_kts,omitempty"`
	WindSpeedKmh  int     `json:"wind_speed_kmh,omitempty"`
	RawData       string  `json:"raw_data,omitempty"`
}

func (r *Result) Type() string     { return "sb01" }
func (r *Result) MessageID() int64 { return r.MsgID }

// Parser parses SB01 H1 messages.
type Parser struct{}

func init() {
	registry.Register(&Parser{})
}

func (p *Parser) Name() string     { return "sb01" }
func (p *Parser) Labels() []string { return []string{"H1"} }
func (p *Parser) Priority() int    { return 15 }

func (p *Parser) QuickCheck(text string) bool {
	return strings.HasPrefix(strings.TrimSpace(text), "SB01")
}

func (p *Parser) Parse(msg *acars.Message) registry.Result {
	if msg == nil || strings.TrimSpace(msg.Text) == "" {
		return nil
	}

	fields := strings.Fields(strings.ReplaceAll(msg.Text, "\r", " "))
	if len(fields) < 4 {
		return nil
	}

	header := fields[0]
	parts := strings.SplitN(header, "_", 2)
	if len(parts) != 2 || len(parts[0]) != 8 || !strings.HasPrefix(parts[0], "SB01") {
		return nil
	}

	registration := strings.TrimSpace(parts[1])
	sequence := parts[0][4:6]

	routeCode, latIndex, lonTimeIndex, payloadIndex, ok := locateSB01BodyFields(fields)
	if !ok {
		return nil
	}
	route := routeCode[:4] + "-" + routeCode[4:]

	// Latitude, longitude, and report time may be spread across two separate
	// fields (positive longitude) or merged into one field (negative longitude,
	// whose sign character merges the two whitespace-delimited tokens).
	var latLonTimeStr string
	if latIndex == lonTimeIndex {
		latLonTimeStr = strings.TrimSpace(fields[latIndex])
	} else {
		latLonTimeStr = strings.TrimSpace(fields[latIndex]) + strings.TrimSpace(fields[lonTimeIndex])
	}
	lat, lon, reportTime, ok := parseLatLonTime(latLonTimeStr)
	if !ok {
		return nil
	}

	payload := strings.TrimSpace(fields[payloadIndex])
	if len(payload) < 15 {
		return nil
	}

	altitudeFt, err := strconv.Atoi(payload[:5])
	if err != nil {
		return nil
	}

	temperatureC, ok := parseSignedTenths(payload[5:9])
	if !ok {
		return nil
	}

	windDirection, err := strconv.Atoi(payload[9:12])
	if err != nil {
		return nil
	}
	windSpeedKts, err := strconv.Atoi(payload[12:15])
	if err != nil {
		return nil
	}

	result := &Result{
		MsgID:         int64(msg.ID),
		Timestamp:     msg.Timestamp,
		Tail:          msg.Tail,
		MsgType:       "EBSB",
		Registration:  registration,
		Sequence:      sequence,
		Route:         route,
		Latitude:      lat,
		Longitude:     lon,
		ReportTime:    reportTime,
		AltitudeFt:    altitudeFt,
		AltitudeM:     feetToMetres(altitudeFt),
		TemperatureC:  temperatureC,
		WindDirection: windDirection,
		WindSpeedKts:  windSpeedKts,
		WindSpeedKmh:  knotsToKmh(windSpeedKts),
		RawData:       msg.Text,
	}

	return result
}

func locateSB01BodyFields(fields []string) (routeCode string, latIndex, lonTimeIndex, payloadIndex int, ok bool) {
	if len(fields) < 4 {
		return "", 0, 0, 0, false
	}

	routeToken := strings.TrimSpace(fields[1])

	// Case A: Combined route+sequence in fields[1] — at least 11 chars where
	// the first 8 are an alphabetic route code.
	if len(routeToken) >= 11 && isRouteCode(routeToken[:8]) {
		if len(fields) >= 5 {
			// Normal: separate lat (fields[2]) and lonTime (fields[3]).
			return routeToken[:8], 2, 3, 4, true
		}
		// Merged: a negative longitude sign attached the lonTime to the lat
		// field (fields[2] contains lat+lonTime combined).
		if len(fields) == 4 {
			return routeToken[:8], 2, 2, 3, true
		}
		return "", 0, 0, 0, false
	}

	// Case B/C: Separate 8-char alphabetic route code in fields[1];
	// sequence number in fields[2].
	if isRouteCode(routeToken) {
		if len(fields) >= 6 {
			// Normal: separate lat (fields[3]) and lonTime (fields[4]).
			return routeToken, 3, 4, 5, true
		}
		// Merged: a negative longitude sign attached the lonTime to the lat
		// field (fields[3] contains lat+lonTime combined).
		if len(fields) == 5 {
			return routeToken, 3, 3, 4, true
		}
		return "", 0, 0, 0, false
	}

	return "", 0, 0, 0, false
}

func isRouteCode(raw string) bool {
	if len(raw) != 8 {
		return false
	}
	for _, ch := range raw {
		if ch < 'A' || ch > 'Z' {
			return false
		}
	}
	return true
}

// parseLatLonTime parses a latitude, longitude, and report time from a
// concatenated string. The latitude is 5 unsigned or 6 signed digit-characters;
// the longitude is 6 unsigned or 7 signed digit-characters; the time is always
// 4 digit-characters. The fields are concatenated without a separator — a
// leading sign on the longitude (or latitude) serves as the only field-boundary
// marker when the two tokens have been merged by strings.Fields.
func parseLatLonTime(s string) (lat, lon float64, reportTime string, ok bool) {
	// Latitude: optional sign prefix + 5 digits.
	latEnd := 5
	if len(s) > 0 && (s[0] == '-' || s[0] == '+') {
		latEnd = 6
	}
	if len(s) < latEnd {
		return 0, 0, "", false
	}
	latVal, err := strconv.Atoi(s[:latEnd])
	if err != nil {
		return 0, 0, "", false
	}
	lat = float64(latVal) / 1000.0

	// Longitude: optional sign prefix + 6 digits.
	lonStart := latEnd
	lonEnd := lonStart + 6
	if lonStart < len(s) && (s[lonStart] == '-' || s[lonStart] == '+') {
		lonEnd = lonStart + 7
	}
	if len(s) < lonEnd+4 {
		return 0, 0, "", false
	}
	lonVal, err := strconv.Atoi(s[lonStart:lonEnd])
	if err != nil {
		return 0, 0, "", false
	}
	lon = float64(lonVal) / 1000.0

	// Report time: always 4 digit-characters.
	reportTime, ok = parseTimeHHMM(s[lonEnd : lonEnd+4])
	return lat, lon, reportTime, ok
}

func parseTimeHHMM(raw string) (string, bool) {
	if len(raw) != 4 {
		return "", false
	}
	hour, err := strconv.Atoi(raw[:2])
	if err != nil || hour < 0 || hour > 23 {
		return "", false
	}
	minute, err := strconv.Atoi(raw[2:])
	if err != nil || minute < 0 || minute > 59 {
		return "", false
	}
	return raw[:2] + ":" + raw[2:], true
}

func parseSignedTenths(raw string) (float64, bool) {
	if len(raw) != 4 {
		return 0, false
	}
	sign := 1.0
	switch raw[0] {
	case '-':
		sign = -1.0
	case '+':
	default:
		return 0, false
	}
	value, err := strconv.Atoi(raw[1:])
	if err != nil {
		return 0, false
	}
	return sign * (float64(value) / 10.0), true
}

func feetToMetres(feet int) int {
	return int(float64(feet) * 0.3048)
}

func knotsToKmh(knots int) int {
	return int(float64(knots) * 1.852)
}
