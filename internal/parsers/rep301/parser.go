package rep301

import (
	"math"
	"strconv"
	"strings"

	"acars_parser/internal/acars"
	"acars_parser/internal/registry"
)

type Result struct {
	MsgID         int64   `json:"message_id"`
	Timestamp     string  `json:"timestamp"`
	MsgType       string  `json:"msg_type,omitempty"`
	Tail          string  `json:"tail,omitempty"`
	Route         string  `json:"route,omitempty"`
	Origin        string  `json:"origin,omitempty"`
	Destination   string  `json:"destination,omitempty"`
	Latitude      float64 `json:"latitude,omitempty"`
	Longitude     float64 `json:"longitude,omitempty"`
	ReportTime    string  `json:"report_time,omitempty"`
	FlightLevel   float64 `json:"flight_level,omitempty"`
	TemperatureC  int     `json:"temperature_c,omitempty"`
	WindDirection int     `json:"wind_direction,omitempty"`
	WindSpeedKts  int     `json:"wind_speed_kts,omitempty"`
	WindSpeedKmh  int     `json:"wind_speed_kmh,omitempty"`
	RawData       string  `json:"raw_data,omitempty"`
}

func (r *Result) Type() string     { return "rep301" }
func (r *Result) MessageID() int64 { return r.MsgID }

type Parser struct{}

func init() {
	registry.Register(&Parser{})
}

func (p *Parser) Name() string     { return "rep301" }
func (p *Parser) Labels() []string { return nil }
func (p *Parser) Priority() int    { return 420 }

func (p *Parser) QuickCheck(text string) bool {
	return strings.Contains(strings.ToUpper(text), "REP301")
}

func (p *Parser) Parse(msg *acars.Message) registry.Result {
	if msg == nil || strings.TrimSpace(msg.Text) == "" {
		return nil
	}

	tokens := strings.Fields(strings.ReplaceAll(msg.Text, "\r", " "))
	repIndex := findREP301Token(tokens)
	if repIndex < 0 || len(tokens) <= repIndex+2 {
		return nil
	}

	routeCode, ok := extractRouteCode(strings.ToUpper(tokens[repIndex+1]))
	if !ok {
		return nil
	}

	origin := routeCode[:4]
	destination := routeCode[4:]
	latitude, longitude, reportTime, flightLevel, temperatureC, windDirection, windSpeedKts, ok := parsePayload(strings.ToUpper(tokens[repIndex+2]))
	if !ok {
		return nil
	}

	tail := msg.Tail
	if tail == "" && msg.Airframe != nil {
		tail = msg.Airframe.Tail
	}

	return &Result{
		MsgID:         int64(msg.ID),
		Timestamp:     msg.Timestamp,
		MsgType:       "REP301",
		Tail:          tail,
		Route:         origin + "-" + destination,
		Origin:        origin,
		Destination:   destination,
		Latitude:      latitude,
		Longitude:     longitude,
		ReportTime:    reportTime,
		FlightLevel:   flightLevel,
		TemperatureC:  temperatureC,
		WindDirection: windDirection,
		WindSpeedKts:  windSpeedKts,
		WindSpeedKmh:  knotsToKmh(windSpeedKts),
		RawData:       msg.Text,
	}
}

func findREP301Token(tokens []string) int {
	for index, token := range tokens {
		if strings.Contains(strings.ToUpper(token), "REP301") {
			return index
		}
	}
	return -1
}

func extractRouteCode(raw string) (string, bool) {
	letters := make([]rune, 0, len(raw))
	for _, char := range raw {
		if char >= 'A' && char <= 'Z' {
			letters = append(letters, char)
		}
	}

	if len(letters) < 8 {
		return "", false
	}

	routeCode := string(letters[len(letters)-8:])
	if !isUpperAlpha(routeCode) {
		return "", false
	}

	return routeCode, true
}

func parsePayload(raw string) (float64, float64, string, float64, int, int, int, bool) {
	if len(raw) < 31 {
		return 0, 0, "", 0, 0, 0, 0, false
	}

	latitude, ok := parseThousandths(raw[1:6], 5)
	if !ok {
		return 0, 0, "", 0, 0, 0, 0, false
	}
	switch raw[0] {
	case 'N':
	case 'S':
		latitude = -latitude
	default:
		return 0, 0, "", 0, 0, 0, 0, false
	}

	longitude, ok := parseThousandths(raw[7:13], 6)
	if !ok {
		return 0, 0, "", 0, 0, 0, 0, false
	}
	switch raw[6] {
	case 'E':
	case 'W':
		longitude = -longitude
	default:
		return 0, 0, "", 0, 0, 0, 0, false
	}

	reportTime, ok := parseTimeHHMM(raw[13:17])
	if !ok {
		return 0, 0, "", 0, 0, 0, 0, false
	}

	flightLevelValue, err := strconv.Atoi(raw[17:21])
	if err != nil {
		return 0, 0, "", 0, 0, 0, 0, false
	}

	temperatureC, ok := parseTemperature(raw[21:25])
	if !ok {
		return 0, 0, "", 0, 0, 0, 0, false
	}

	windDirection, err := strconv.Atoi(raw[25:28])
	if err != nil || windDirection < 0 || windDirection > 360 {
		return 0, 0, "", 0, 0, 0, 0, false
	}

	windSpeedKts, err := strconv.Atoi(raw[28:31])
	if err != nil {
		return 0, 0, "", 0, 0, 0, 0, false
	}

	return latitude, longitude, reportTime, float64(flightLevelValue) / 10.0, temperatureC, windDirection, windSpeedKts, true
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

func parseTemperature(raw string) (int, bool) {
	if len(raw) != 4 {
		return 0, false
	}

	value, err := strconv.Atoi(raw[1:])
	if err != nil {
		return 0, false
	}

	switch raw[0] {
	case 'M', '-':
		return -value, true
	case 'P', '+':
		return value, true
	default:
		return 0, false
	}
}

func isUpperAlpha(raw string) bool {
	if len(raw) == 0 {
		return false
	}

	for _, char := range raw {
		if char < 'A' || char > 'Z' {
			return false
		}
	}

	return true
}

func knotsToKmh(knots int) int {
	return int(math.Round(float64(knots) * 1.852))
}
