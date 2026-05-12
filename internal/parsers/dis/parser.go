// Package dis parses RA DIS operational flight-plan info messages.
package dis

import (
	"regexp"
	"strconv"
	"strings"

	"acars_parser/internal/acars"
	"acars_parser/internal/airlines"
	"acars_parser/internal/registry"
)

var (
	disHeaderRe        = regexp.MustCompile(`(?i)DIS(\d{2})(\d{2})(\d{4})`)
	disOFPInfoRe       = regexp.MustCompile(`(?is)\bOFP\s+INFO\b\s+([A-Z0-9]{2,10})\s+([A-Z]{3})-([A-Z]{3})\s+([A-Z0-9-]{4,10}):\s*CURRENT\s+OFP\s+NUMBER\s+([0-9/]+)\b`)
	disLoadsheetAckRe  = regexp.MustCompile(`(?is)\bLDSHT\s+ACCEPT\s+ACK\b\s+FLIGHT\s+NUMBER:\s*([A-Z0-9]{2,10}(?:/[A-Z0-9]{2,10})?)\s+SECTOR:\s*([A-Z]{4})-([A-Z]{4})\s+FLIGHT\s+DATE:\s*(\d{1,2})\b`)
	disFlightSummAckRe = regexp.MustCompile(`(?is)\bFLT\s+SUMM\s+ACK\b.*?\bFLIGHT\s+SUMMARY\s+ACK\b.*?\bFLIGHT\s+NUMBER:\s*([A-Z0-9]{2,10}(?:/[A-Z0-9]{2,10})?)\s+SECTOR:\s*([A-Z]{4})-([A-Z]{4})\b`)
)

// Result represents a parsed DIS acknowledgement or OFP summary message.
type Result struct {
	MsgID       int64  `json:"message_id"`
	Timestamp   string `json:"timestamp"`
	Tail        string `json:"tail,omitempty"`
	MsgType     string `json:"msg_type,omitempty"`
	Format      string `json:"format,omitempty"`
	Flight      string `json:"flight,omitempty"`
	Origin      string `json:"origin,omitempty"`
	Destination string `json:"destination,omitempty"`
	Route       string `json:"route,omitempty"`
	Aircraft    string `json:"aircraft,omitempty"`
	OFPNumber   string `json:"ofp_number,omitempty"`
	DayOfMonth  int    `json:"day_of_month,omitempty"`
	ReportTime  string `json:"report_time,omitempty"`
	RawData     string `json:"raw_data,omitempty"`
}

func (r *Result) Type() string     { return "dis" }
func (r *Result) MessageID() int64 { return r.MsgID }

// Parser parses RA DIS OFP INFO messages.
type Parser struct{}

func init() {
	registry.Register(&Parser{})
}

func (p *Parser) Name() string     { return "dis" }
func (p *Parser) Labels() []string { return []string{"RA"} }
func (p *Parser) Priority() int    { return 45 }

func (p *Parser) QuickCheck(text string) bool {
	upper := strings.ToUpper(text)
	if !strings.Contains(upper, "DIS") {
		return false
	}
	return strings.Contains(upper, "OFP INFO") || strings.Contains(upper, "LDSHT ACCEPT ACK") || strings.Contains(upper, "FLT SUMM ACK") || strings.Contains(upper, "FLIGHT SUMMARY ACK")
}

func (p *Parser) Parse(msg *acars.Message) registry.Result {
	if msg == nil || strings.TrimSpace(msg.Text) == "" {
		return nil
	}

	upperText := strings.ToUpper(strings.TrimSpace(msg.Text))
	headerMatch := disHeaderRe.FindStringSubmatch(upperText)
	if len(headerMatch) != 4 {
		return nil
	}

	dayOfMonth, err := strconv.Atoi(headerMatch[2])
	if err != nil {
		return nil
	}

	reportTime, ok := parseHHMM(headerMatch[3])
	if !ok {
		return nil
	}

	if bodyMatch := disOFPInfoRe.FindStringSubmatch(upperText); len(bodyMatch) == 6 {
		origin := strings.TrimSpace(bodyMatch[2])
		destination := strings.TrimSpace(bodyMatch[3])
		if origin == "" || destination == "" {
			return nil
		}

		return &Result{
			MsgID:       int64(msg.ID),
			Timestamp:   msg.Timestamp,
			Tail:        msg.Tail,
			MsgType:     "DIS",
			Format:      "DIS" + headerMatch[1],
			Flight:      normaliseDISFlight(bodyMatch[1]),
			Origin:      origin,
			Destination: destination,
			Route:       origin + "-" + destination,
			Aircraft:    strings.TrimSpace(bodyMatch[4]),
			OFPNumber:   strings.TrimSpace(bodyMatch[5]),
			DayOfMonth:  dayOfMonth,
			ReportTime:  reportTime,
			RawData:     msg.Text,
		}
	}

	if bodyMatch := disLoadsheetAckRe.FindStringSubmatch(upperText); len(bodyMatch) == 5 {
		origin := strings.TrimSpace(bodyMatch[2])
		destination := strings.TrimSpace(bodyMatch[3])
		if origin == "" || destination == "" {
			return nil
		}

		return &Result{
			MsgID:       int64(msg.ID),
			Timestamp:   msg.Timestamp,
			Tail:        msg.Tail,
			MsgType:     "DIS",
			Format:      "DIS" + headerMatch[1],
			Flight:      normaliseDISFlight(preferredAckFlight(bodyMatch[1])),
			Origin:      origin,
			Destination: destination,
			Route:       origin + "-" + destination,
			DayOfMonth:  dayOfMonth,
			ReportTime:  reportTime,
			RawData:     msg.Text,
		}
	}

	if bodyMatch := disFlightSummAckRe.FindStringSubmatch(upperText); len(bodyMatch) == 4 {
		origin := strings.TrimSpace(bodyMatch[2])
		destination := strings.TrimSpace(bodyMatch[3])
		if origin == "" || destination == "" {
			return nil
		}

		return &Result{
			MsgID:       int64(msg.ID),
			Timestamp:   msg.Timestamp,
			Tail:        msg.Tail,
			MsgType:     "DIS",
			Format:      "DIS" + headerMatch[1],
			Flight:      normaliseDISFlight(preferredAckFlight(bodyMatch[1])),
			Origin:      origin,
			Destination: destination,
			Route:       origin + "-" + destination,
			DayOfMonth:  dayOfMonth,
			ReportTime:  reportTime,
			RawData:     msg.Text,
		}
	}

	return nil
}

func normaliseDISFlight(raw string) string {
	return airlines.TranslateFlight(strings.TrimSpace(raw))
}

func preferredAckFlight(raw string) string {
	flight := strings.TrimSpace(raw)
	if flight == "" {
		return ""
	}
	if !strings.Contains(flight, "/") {
		return flight
	}
	parts := strings.Split(flight, "/")
	for idx := len(parts) - 1; idx >= 0; idx-- {
		candidate := strings.TrimSpace(parts[idx])
		if candidate != "" {
			return candidate
		}
	}
	return flight
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
