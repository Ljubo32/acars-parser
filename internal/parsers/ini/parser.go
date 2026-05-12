// Package ini parses RA INI01 initialisation messages.
package ini

import (
	"regexp"
	"strconv"
	"strings"

	"acars_parser/internal/acars"
	"acars_parser/internal/registry"
)

var iniRe = regexp.MustCompile(`(?i)INI(\d{2})(\d{2})(\d{4})\s+([A-Z]{3}\d{1,4}[A-Z]?)\s*/\d{2}/([A-Z]{4})/([A-Z]{4})\b`)
var iniIDRouteRe = regexp.MustCompile(`(?i)^INI/ID[0-9A-Z]+,([^,]+),[^/]*/MR\d+,[^/]*/(AF)?([A-Z]{4}),([A-Z]{4})/TD(\d{2})(\d{4}),`)

// Result represents a parsed INI message.
type Result struct {
	MsgID       int64  `json:"message_id"`
	Timestamp   string `json:"timestamp"`
	Tail        string `json:"tail,omitempty"`
	MsgType     string `json:"msg_type,omitempty"`
	Format      string `json:"format,omitempty"`
	Flight      string `json:"flight,omitempty"`
	DayOfMonth  int    `json:"day_of_month,omitempty"`
	ReportTime  string `json:"report_time,omitempty"`
	Origin      string `json:"origin,omitempty"`
	Destination string `json:"destination,omitempty"`
	RawData     string `json:"raw_data,omitempty"`
}

func (r *Result) Type() string     { return "ini" }
func (r *Result) MessageID() int64 { return r.MsgID }

// Parser parses RA INI01 messages.
type Parser struct{}

func init() {
	registry.Register(&Parser{})
}

func (p *Parser) Name() string     { return "ini" }
func (p *Parser) Labels() []string { return nil }
func (p *Parser) Priority() int    { return 40 }

func (p *Parser) QuickCheck(text string) bool {
	upper := strings.ToUpper(text)
	return strings.Contains(upper, "INI01") || strings.Contains(upper, "INI/ID")
}

func (p *Parser) Parse(msg *acars.Message) registry.Result {
	if msg == nil || strings.TrimSpace(msg.Text) == "" {
		return nil
	}

	upperText := strings.ToUpper(strings.TrimSpace(msg.Text))
	if result := parseINIIDMessage(msg, upperText); result != nil {
		return result
	}

	match := iniRe.FindStringSubmatch(upperText)
	if len(match) != 7 {
		return nil
	}

	dayOfMonth, err := strconv.Atoi(match[2])
	if err != nil {
		return nil
	}

	reportTime, ok := parseHHMM(match[3])
	if !ok {
		return nil
	}

	return &Result{
		MsgID:       int64(msg.ID),
		Timestamp:   msg.Timestamp,
		Tail:        msg.Tail,
		MsgType:     "INI",
		Format:      "INI" + match[1],
		Flight:      match[4],
		DayOfMonth:  dayOfMonth,
		ReportTime:  reportTime,
		Origin:      match[5],
		Destination: match[6],
		RawData:     msg.Text,
	}
}

func parseINIIDMessage(msg *acars.Message, upperText string) registry.Result {
	match := iniIDRouteRe.FindStringSubmatch(upperText)
	if len(match) != 7 {
		return nil
	}

	dayOfMonth, err := strconv.Atoi(match[5])
	if err != nil {
		return nil
	}

	reportTime, ok := parseHHMM(match[6])
	if !ok {
		return nil
	}

	origin := strings.TrimSpace(match[3])
	destination := strings.TrimSpace(match[4])
	if origin == "" || destination == "" {
		return nil
	}

	return &Result{
		MsgID:       int64(msg.ID),
		Timestamp:   msg.Timestamp,
		Tail:        msg.Tail,
		MsgType:     "INI",
		Format:      "INI/ID",
		Flight:      strings.TrimSpace(match[1]),
		DayOfMonth:  dayOfMonth,
		ReportTime:  reportTime,
		Origin:      origin,
		Destination: destination,
		RawData:     msg.Text,
	}
}

func parseHHMM(raw string) (string, bool) {
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
