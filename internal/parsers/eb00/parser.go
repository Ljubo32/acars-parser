// Package eb00 parses EB00 compact position/status messages carried on label H1.
package eb00

import (
	"strconv"
	"strings"

	"acars_parser/internal/acars"
	"acars_parser/internal/registry"
)

// Result represents a parsed EB00 report.
type Result struct {
	MsgID         int64   `json:"message_id"`
	Timestamp     string  `json:"timestamp"`
	Tail          string  `json:"tail,omitempty"`
	MsgType       string  `json:"msg_type,omitempty"`
	Registration  string  `json:"registration,omitempty"`
	Sequence      string  `json:"sequence,omitempty"`
	MsgNo         int     `json:"msg_no,omitempty"`
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

func (r *Result) Type() string     { return "eb00" }
func (r *Result) MessageID() int64 { return r.MsgID }

// Parser parses EB00 H1 messages.
type Parser struct{}

func init() {
	registry.Register(&Parser{})
}

func (p *Parser) Name() string     { return "eb00" }
func (p *Parser) Labels() []string { return []string{"H1"} }
func (p *Parser) Priority() int    { return 15 }

func (p *Parser) QuickCheck(text string) bool {
	return strings.HasPrefix(strings.TrimSpace(text), "EB00")
}

func (p *Parser) Parse(msg *acars.Message) registry.Result {
	if msg == nil || strings.TrimSpace(msg.Text) == "" {
		return nil
	}

	fields := strings.Fields(strings.ReplaceAll(msg.Text, "\r", " "))
	header, registration, bodyStart, ok := parseEB00Header(fields)
	if !ok {
		return nil
	}

	routeCode, msgNo, latIndex, ok := parseRouteAndMessageNumber(fields, bodyStart)
	if !ok {
		return nil
	}

	if len(fields) <= latIndex+2 {
		return nil
	}

	lat, ok := parseThousandths(fields[latIndex], 5)
	if !ok {
		return nil
	}

	lonTime := strings.TrimSpace(fields[latIndex+1])
	if len(lonTime) != 10 {
		return nil
	}

	lon, ok := parseThousandths(lonTime[:6], 6)
	if !ok {
		return nil
	}

	reportTime, ok := parseTimeHHMM(lonTime[6:10])
	if !ok {
		return nil
	}

	payload := strings.TrimSpace(fields[latIndex+2])
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

	return &Result{
		MsgID:         int64(msg.ID),
		Timestamp:     msg.Timestamp,
		Tail:          registration,
		MsgType:       "EBSB",
		Registration:  registration,
		Sequence:      header[4:6],
		MsgNo:         msgNo,
		Route:         routeCode[:4] + "-" + routeCode[4:],
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
}

func parseEB00Header(fields []string) (header, registration string, bodyStart int, ok bool) {
	if len(fields) < 5 {
		return "", "", 0, false
	}

	header = strings.TrimSpace(fields[0])
	parts := strings.SplitN(header, "_", 2)
	if len(parts) != 2 || len(parts[0]) != 8 || !strings.HasPrefix(parts[0], "EB00") {
		return "", "", 0, false
	}

	registration = strings.TrimSpace(parts[1])
	bodyStart = 1
	if registration == "" {
		if len(fields) < 6 {
			return "", "", 0, false
		}
		registration = strings.TrimSpace(fields[1])
		bodyStart = 2
	}

	if registration == "" {
		return "", "", 0, false
	}

	return parts[0], registration, bodyStart, true
}

func parseRouteAndMessageNumber(fields []string, bodyStart int) (routeCode string, msgNo int, latIndex int, ok bool) {
	if len(fields) <= bodyStart+2 {
		return "", 0, 0, false
	}

	routeField := strings.TrimSpace(fields[bodyStart])
	if len(routeField) >= 9 && isRouteCode(routeField[:8]) {
		msgNo, err := strconv.Atoi(routeField[8:])
		if err == nil {
			return routeField[:8], msgNo, bodyStart + 1, true
		}
	}

	if !isRouteCode(routeField) || len(fields) <= bodyStart+3 {
		return "", 0, 0, false
	}

	msgNo, err := strconv.Atoi(strings.TrimSpace(fields[bodyStart+1]))
	if err != nil {
		return "", 0, 0, false
	}

	return routeField, msgNo, bodyStart + 2, true
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

func parseThousandths(raw string, width int) (float64, bool) {
	if len(raw) != width {
		return 0, false
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return float64(value) / 1000.0, true
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
