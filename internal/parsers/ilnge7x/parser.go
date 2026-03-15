// Package ilnge7x parses ILNGE7X route summary messages.
package ilnge7x

import (
	"regexp"
	"strings"

	"acars_parser/internal/acars"
	"acars_parser/internal/registry"
)

var messageTokenRe = regexp.MustCompile(`^/ILNGE7X\.[A-Z0-9]+?\s*(?P<tail_flight>[A-Z0-9]{1,2}-[A-Z0-9]{3,4}[12]?[A-Z0-9]{3,8})\s+(?P<date>\d{6})(?P<time>\d{6})(?P<origin>[A-Z]{4})(?P<destination>[A-Z]{4})`)
var messageTARe = regexp.MustCompile(`^/ILNGE7X\.[^\s]+\s+\d\s+(?P<tail_flight>[A-Z0-9]{1,2}-[A-Z0-9]{3,4}[12]?[A-Z0-9]{3,8})\s+(?P<route>[A-Z]{8})\s+\d+TA(?P<date>\d{6})(?P<time>\d{6})`)
var messageERRe = regexp.MustCompile(`^/ILNGE7X\.[^\s]+\s+[A-Z]\s+\d\s+(?P<tail>[A-Z0-9]{1,2}-[A-Z0-9]{3,4})\s+(?P<flight>[A-Z0-9]{3,8})\s+(?P<route>[A-Z]{8})\s+\d+ER(?P<date>\d{2}/\d{2}/\d{2})(?P<time>\d{2}:\d{2}:\d{2})`)
var messageSCRRe = regexp.MustCompile(`^/ILNGE7X\.[A-Z0-9]+\.(?P<tail_flight>[A-Z0-9-]{12,20})\s+(?P<date>\d{6})(?P<time>\d{6})(?P<route>[A-Z]{8})`)

// Result represents the parsed ILNGE7X summary fields.
type Result struct {
	MsgID       int64  `json:"message_id"`
	Timestamp   string `json:"timestamp"`
	Tail        string `json:"tail,omitempty"`
	Flight      string `json:"flight,omitempty"`
	TakeOffDate string `json:"take_off_date,omitempty"`
	TakeOffTime string `json:"take_off_time,omitempty"`
	Origin      string `json:"origin,omitempty"`
	Destination string `json:"destination,omitempty"`
	Route       string `json:"route,omitempty"`
}

func (r *Result) Type() string     { return "ilnge7x" }
func (r *Result) MessageID() int64 { return r.MsgID }

// Parser parses ILNGE7X messages regardless of the ACARS label.
type Parser struct{}

func init() {
	registry.Register(&Parser{})
}

func (p *Parser) Name() string     { return "ilnge7x" }
func (p *Parser) Labels() []string { return nil }
func (p *Parser) Priority() int    { return 450 }

func (p *Parser) QuickCheck(text string) bool {
	return strings.HasPrefix(strings.TrimSpace(text), "/ILNGE7X.")
}

func (p *Parser) Parse(msg *acars.Message) registry.Result {
	text := strings.TrimSpace(msg.Text)
	if text == "" {
		return nil
	}

	matches := messageTokenRe.FindStringSubmatch(text)
	if matches != nil {
		captures := captureMap(messageTokenRe, matches)
		tail, flight, ok := parseTailAndFlight(captures["tail_flight"])
		if !ok {
			return nil
		}
		origin := captures["origin"]
		destination := captures["destination"]

		return &Result{
			MsgID:       int64(msg.ID),
			Timestamp:   msg.Timestamp,
			Tail:        tail,
			Flight:      flight,
			TakeOffDate: formatDateYYMMDD(captures["date"]),
			TakeOffTime: formatTimeHHMMSS(captures["time"]),
			Origin:      origin,
			Destination: destination,
			Route:       origin + "-" + destination,
		}
	}

	matches = messageSCRRe.FindStringSubmatch(text)
	if matches != nil {
		captures := captureMap(messageSCRRe, matches)
		tail, flight, ok := parseTailAndFlightWithSeparator(captures["tail_flight"], 3)
		if !ok {
			return nil
		}
		route := captures["route"]
		origin := route[0:4]
		destination := route[4:8]

		return &Result{
			MsgID:       int64(msg.ID),
			Timestamp:   msg.Timestamp,
			Tail:        tail,
			Flight:      flight,
			TakeOffDate: formatDateYYMMDD(captures["date"]),
			TakeOffTime: formatTimeHHMMSS(captures["time"]),
			Origin:      origin,
			Destination: destination,
			Route:       origin + "-" + destination,
		}
	}

	matches = messageTARe.FindStringSubmatch(text)
	if matches != nil {
		captures := captureMap(messageTARe, matches)
		tail, flight, ok := parseTailAndFlight(captures["tail_flight"])
		if !ok {
			return nil
		}
		route := captures["route"]
		origin := route[0:4]
		destination := route[4:8]

		return &Result{
			MsgID:       int64(msg.ID),
			Timestamp:   msg.Timestamp,
			Tail:        tail,
			Flight:      flight,
			TakeOffDate: formatDateDDMMYY(captures["date"]),
			TakeOffTime: formatTimeHHMMSS(captures["time"]),
			Origin:      origin,
			Destination: destination,
			Route:       origin + "-" + destination,
		}
	}

	matches = messageERRe.FindStringSubmatch(text)
	if matches == nil {
		return nil
	}

	captures := captureMap(messageERRe, matches)
	route := captures["route"]
	origin := route[0:4]
	destination := route[4:8]

	return &Result{
		MsgID:       int64(msg.ID),
		Timestamp:   msg.Timestamp,
		Tail:        captures["tail"],
		Flight:      captures["flight"],
		TakeOffDate: formatDateWithSlashes(captures["date"]),
		TakeOffTime: captures["time"],
		Origin:      origin,
		Destination: destination,
		Route:       origin + "-" + destination,
	}
}

func captureMap(re *regexp.Regexp, matches []string) map[string]string {
	result := make(map[string]string, len(matches))
	for index, name := range re.SubexpNames() {
		if index == 0 || name == "" {
			continue
		}
		result[name] = matches[index]
	}
	return result
}

func formatDateYYMMDD(value string) string {
	if len(value) != 6 {
		return value
	}

	return value[0:2] + "-" + value[2:4] + "-" + value[4:6]
}

func formatDateDDMMYY(value string) string {
	if len(value) != 6 {
		return value
	}

	return value[0:2] + "-" + value[2:4] + "-" + value[4:6]
}

func formatDateWithSlashes(value string) string {
	if len(value) != 8 {
		return value
	}

	return strings.ReplaceAll(value, "/", "-")
}

func formatTimeHHMMSS(value string) string {
	if len(value) != 6 {
		return value
	}

	return value[0:2] + ":" + value[2:4] + ":" + value[4:6]
}

func parseTailAndFlight(token string) (string, string, bool) {
	for _, tailLength := range []int{6, 7} {
		if len(token) <= tailLength {
			continue
		}

		tail := token[:tailLength]
		if !isValidTail(tail) {
			continue
		}

		flight := token[tailLength:]
		if len(flight) > 0 && (flight[0] == '1' || flight[0] == '2') {
			flight = flight[1:]
		}

		if !isValidFlight(flight) {
			continue
		}

		return tail, flight, true
	}

	return "", "", false
}

func parseTailAndFlightWithSeparator(token string, separatorLength int) (string, string, bool) {
	for _, tailLength := range []int{6, 7} {
		if len(token) <= tailLength+separatorLength {
			continue
		}

		tail := token[:tailLength]
		if !isValidTail(tail) {
			continue
		}

		flight := token[tailLength+separatorLength:]
		if !isValidFlight(flight) {
			continue
		}

		return tail, flight, true
	}

	return "", "", false
}

func isValidTail(value string) bool {
	if len(value) != 6 && len(value) != 7 {
		return false
	}

	if !strings.Contains(value, "-") {
		for _, char := range value {
			if (char < 'A' || char > 'Z') && (char < '0' || char > '9') {
				return false
			}
		}
		return true
	}

	parts := strings.Split(value, "-")
	if len(parts) != 2 {
		return false
	}
	if len(parts[0]) < 1 || len(parts[0]) > 2 {
		return false
	}
	if len(parts[1]) < 3 || len(parts[1]) > 4 {
		return false
	}

	for _, part := range parts {
		for _, char := range part {
			if (char < 'A' || char > 'Z') && (char < '0' || char > '9') {
				return false
			}
		}
	}

	return true
}

func isValidFlight(value string) bool {
	if len(value) < 3 || len(value) > 8 {
		return false
	}

	letterCount := 0
	for index, char := range value {
		if char >= 'A' && char <= 'Z' {
			if index < 3 {
				letterCount++
			}
			continue
		}
		if char < '0' || char > '9' {
			return false
		}
	}

	return letterCount >= 2
}