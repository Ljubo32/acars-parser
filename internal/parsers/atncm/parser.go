// Package atncm parses ATN CM logon route hints extracted from VDL2 logs.
package atncm

import (
	"fmt"
	"strings"

	"acars_parser/internal/acars"
	"acars_parser/internal/registry"
)

// Result represents an ATN CM logon request with route fields.
type Result struct {
	MsgID         int64  `json:"message_id"`
	Timestamp     string `json:"timestamp"`
	MessageType   string `json:"message_type"`
	Tail          string `json:"tail,omitempty"`
	FlightID      string `json:"flight_id,omitempty"`
	Origin        string `json:"origin,omitempty"`
	Destination   string `json:"destination,omitempty"`
	Route         string `json:"route,omitempty"`
	RawData       string `json:"raw_data,omitempty"`
	FormattedText string `json:"formatted_text,omitempty"`
}

func (r *Result) Type() string     { return "atn_cm" }
func (r *Result) MessageID() int64 { return r.MsgID }
func (r *Result) HumanReadableText() string {
	return strings.TrimSpace(r.FormattedText)
}

// Parser parses synthetic ATN CM logon messages.
type Parser struct{}

func init() {
	registry.Register(&Parser{})
}

func (p *Parser) Name() string     { return "atncm" }
func (p *Parser) Labels() []string { return []string{"ATNCM"} }
func (p *Parser) Priority() int    { return 40 }

func (p *Parser) QuickCheck(text string) bool {
	return strings.Contains(strings.ToUpper(text), "ATN CM LOGON")
}

func (p *Parser) Parse(msg *acars.Message) registry.Result {
	if msg == nil || msg.Flight == nil {
		return nil
	}

	flightID := strings.TrimSpace(msg.Flight.ID)
	origin := strings.ToUpper(strings.TrimSpace(msg.Flight.DepartingAirport))
	destination := strings.ToUpper(strings.TrimSpace(msg.Flight.DestinationAirport))
	if origin == "" || destination == "" {
		return nil
	}

	formatted := []string{"ATN CM LOGON"}
	if flightID != "" {
		formatted = append(formatted, flightID)
	}
	formatted = append(formatted, fmt.Sprintf("%s-%s", origin, destination))

	return &Result{
		MsgID:         int64(msg.ID),
		Timestamp:     msg.Timestamp,
		MessageType:   "atn_cm",
		Tail:          msg.Tail,
		FlightID:      flightID,
		Origin:        origin,
		Destination:   destination,
		Route:         origin + "-" + destination,
		RawData:       msg.Text,
		FormattedText: strings.Join(formatted, " "),
	}
}
