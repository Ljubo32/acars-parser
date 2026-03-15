// Package hfdl parses synthetic HFDL support messages emitted by the extractor.
package hfdl

import (
	"strings"

	"acars_parser/internal/acars"
	"acars_parser/internal/registry"
)

// Result represents synthetic HFDL data with a flight identifier and coordinates.
type Result struct {
	MsgID      int64   `json:"message_id"`
	Timestamp  string  `json:"timestamp"`
	Tail       string  `json:"tail,omitempty"`
	FlightID   string  `json:"flight_id,omitempty"`
	Flight     string  `json:"flight,omitempty"`
	Latitude   float64 `json:"latitude,omitempty"`
	Longitude  float64 `json:"longitude,omitempty"`
	Source     string  `json:"source,omitempty"`
	HFNPDUType string  `json:"hfnpdu_type,omitempty"`
	ICAO       string  `json:"icao,omitempty"`
}

func (r *Result) Type() string     { return "hfdl_data" }
func (r *Result) MessageID() int64 { return r.MsgID }

// Parser handles synthetic HFDL data messages.
type Parser struct{}

func init() {
	registry.Register(&Parser{})
}

func (p *Parser) Name() string     { return "hfdl_data" }
func (p *Parser) Labels() []string { return []string{"HFDL"} }
func (p *Parser) Priority() int    { return 5 }

func (p *Parser) QuickCheck(text string) bool {
	return strings.Contains(strings.ToUpper(strings.TrimSpace(text)), " DATA")
}

func (p *Parser) Parse(msg *acars.Message) registry.Result {
	if msg == nil || msg.Flight == nil {
		return nil
	}

	if msg.Flight.Latitude < -90 || msg.Flight.Latitude > 90 || msg.Flight.Longitude < -180 || msg.Flight.Longitude > 180 {
		return nil
	}

	if strings.TrimSpace(msg.Flight.ID) == "" && strings.TrimSpace(msg.Flight.Flight) == "" {
		return nil
	}

	result := &Result{
		MsgID:      int64(msg.ID),
		Timestamp:  msg.Timestamp,
		Tail:       strings.TrimSpace(msg.Tail),
		FlightID:   strings.TrimSpace(msg.Flight.ID),
		Flight:     strings.TrimSpace(msg.Flight.Flight),
		Latitude:   msg.Flight.Latitude,
		Longitude:  msg.Flight.Longitude,
		Source:     strings.TrimSpace(msg.Source),
		HFNPDUType: strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(msg.Text), "HFDL")),
	}
	if idx := strings.Index(result.HFNPDUType, " GS "); idx >= 0 {
		result.HFNPDUType = strings.TrimSpace(result.HFNPDUType[:idx])
	}

	if msg.Airframe != nil {
		result.ICAO = strings.ToUpper(strings.TrimSpace(msg.Airframe.ICAO))
		if result.Tail == "" {
			result.Tail = strings.TrimSpace(msg.Airframe.Tail)
		}
	}

	return result
}
