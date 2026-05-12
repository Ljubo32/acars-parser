// Package loadsheet parses aircraft loadsheet messages from ACARS.
// These contain weight and balance data for flight operations.
package loadsheet

import (
	"regexp"
	"strconv"
	"strings"

	"acars_parser/internal/acars"
	"acars_parser/internal/airlines"
	"acars_parser/internal/airports"
	"acars_parser/internal/registry"
)

// Result represents parsed loadsheet data.
type Result struct {
	MsgID        int64  `json:"message_id"`
	Timestamp    string `json:"timestamp"`
	MsgType      string `json:"msg_type,omitempty"`
	Tail         string `json:"tail,omitempty"`
	Flight       string `json:"flight,omitempty"`
	Origin       string `json:"origin,omitempty"`
	Destination  string `json:"destination,omitempty"`
	AircraftType string `json:"aircraft_type,omitempty"`
	ZFW          int    `json:"zfw,omitempty"`     // Zero Fuel Weight
	TOW          int    `json:"tow,omitempty"`     // Take Off Weight
	LAW          int    `json:"law,omitempty"`     // Landing Weight
	TOF          int    `json:"tof,omitempty"`     // Take Off Fuel
	PAX          int    `json:"pax,omitempty"`     // Passenger count
	Crew         string `json:"crew,omitempty"`    // Crew configuration
	Trim         string `json:"trim,omitempty"`    // Stabiliser trim
	MACZFW       string `json:"mac_zfw,omitempty"` // MAC at ZFW
	MACTOW       string `json:"mac_tow,omitempty"` // MAC at TOW
	Cargo        int    `json:"cargo,omitempty"`   // Cargo weight
	Edition      string `json:"edition,omitempty"` // Loadsheet edition
}

func (r *Result) Type() string     { return "loadsheet" }
func (r *Result) MessageID() int64 { return r.MsgID }

// Parser parses loadsheet messages.
type Parser struct{}

func init() {
	registry.Register(&Parser{})
}

func (p *Parser) Name() string     { return "loadsheet" }
func (p *Parser) Labels() []string { return nil }
func (p *Parser) Priority() int    { return 60 } // Higher priority than weather.

// QuickCheck looks for loadsheet keywords.
func (p *Parser) QuickCheck(text string) bool {
	normalised := normaliseLoadsheetText(text)
	return strings.Contains(normalised, "LOADSHEET")
}

// Pattern matchers.
var (
	// ZFW patterns.
	zfwRe = regexp.MustCompile(`\bZFW\s+(\d+)`)
	zfwSlashRe = regexp.MustCompile(`\bZFW/(\d+)`)
	zfwActualRe = regexp.MustCompile(`(?im)\bZERO\s+FUEL\s+WEIGHT\s+ACTUAL\s+(\d+)`)

	// TOW patterns.
	towRe = regexp.MustCompile(`\bTOW\s+(\d+)`)
	towSlashRe = regexp.MustCompile(`\bTOW/(\d+)`)
	towActualRe = regexp.MustCompile(`(?im)\bTAKE\s+OFF\s+WEIGHT\s+ACTUAL\s+(\d+)`)

	// LAW/LDW patterns.
	lawRe = regexp.MustCompile(`\b(?:LAW|LDW)\s+(\d+)`)

	// TOF patterns.
	tofRe = regexp.MustCompile(`\bTOF\s+(\d+)`)
	tofSlashRe = regexp.MustCompile(`\b(?:TOF|FWT)/(\d+)`)
	takeOffFuelRe = regexp.MustCompile(`(?im)\bTAKE\s+OFF\s+FUEL\s+(\d+)`)

	// PAX patterns - various formats.
	paxRe = regexp.MustCompile(`\bPAX[/\s]+(\d+)[/\s]*(\d+)?[/\s]*(\d+)?`)
	paxInlineTTLRe = regexp.MustCompile(`\bPAX(?:[\s/]+\d+){1,4}(?:\s+PAX)?\s+TTL[:\s]+(\d+)\.?`)
	paxTTLRe       = regexp.MustCompile(`\bPAX\s+TTL[:\s]+(\d+)`)
	paxPlusRe      = regexp.MustCompile(`\bPAX[:\s]+(\d+)\s+PLUS\s+(\d+)`)
	paxWithTTLRe   = regexp.MustCompile(`\bPAX[:\s]+(\d+)\s+TTL[:\s]+(\d+)`)
	passengerTTLRe = regexp.MustCompile(`(?im)^.*PASSENGER.*\bTTL[:\s]+(\d+)\b`)

	// TTL (total) pattern.
	ttlRe = regexp.MustCompile(`\bTTL[:\s]+(\d+)`)

	// Crew pattern.
	crewRe = regexp.MustCompile(`\bCREW[:\s]+(\d+/\d+(?:/\d+)?)`)
	crewSlashRe = regexp.MustCompile(`\bCR[EW]/(\d+/\d+(?:/\d+)?)`)
	routeLineCrewRe = regexp.MustCompile(`(?im)^\s*[A-Z]{3}(?:/|\s+)[A-Z]{3}(?:\s+[A-Z0-9-]+){1,3}\s+(\d+/\d+(?:/\d+)?)\s*$`)

	// Trim/stabiliser pattern.
	trimRe = regexp.MustCompile(`\bSTAB[:\s]+(?:FLAPS\s+\d+/\d+\s+\d+K)?\s*([\d.]+\s*(?:UP|DN|DOWN))`)

	// MAC patterns.
	maczfwRe = regexp.MustCompile(`\bMACZFW[:\s]+([\d.]+)`)
	mactowRe = regexp.MustCompile(`\bMACTOW[:\s]+([\d.]+)`)

	// Edition pattern.
	ednoRe = regexp.MustCompile(`\bEDNO?[-:\s]*(\d+)`)
	loadsheetFinalEditionRe = regexp.MustCompile(`(?im)^LOADSHEET\s+FINAL/\d{3,4}/(\d+)\b`)
	finalEditionRe          = regexp.MustCompile(`(?im)^FINAL0*(\d+)\b`)
	multilineEditionRe      = regexp.MustCompile(`(?im)^.*\bEDNO\b.*\r?\n[^\r\n]*?\b(\d{1,2})\s*$`)

	// Flight/route patterns.
	flightRouteRe = regexp.MustCompile(`\b([A-Z0-9]{2,3}\d{3,4}[A-Z]?)/\d+\s+\d+[A-Z]{3}\d+\s+([A-Z]{3})\s*([A-Z]{3})`)
	flightRouteDateCompactRe = regexp.MustCompile(`(?im)\bFLIGHT:\s*([A-Z0-9]{2,3}\d{3,4}[A-Z]?)/\d{2}[A-Z]{3}\d{2}\s+([A-Z]{3})([A-Z]{3})\b`)
	flightRouteInlineCompactRe = regexp.MustCompile(`(?im)\b([A-Z0-9]{2,3}\d{3,4}[A-Z]?)/\d+\s+([A-Z]{3})([A-Z]{3})\b`)
	flightRouteCompactRe    = regexp.MustCompile(`\b([A-Z0-9]{2,3}\d{3,4}[A-Z]?)/\d+/\d{2}[A-Z]{3}\d{2}([A-Z]{3})([A-Z]{3})(?:[A-Z0-9-]|\s|$)`)
	flightRouteFinalLineRe  = regexp.MustCompile(`\b([A-Z0-9]{2,3}\d{3,4}[A-Z]?)/\d+\s+([A-Z]{3})([A-Z]{3})\s+[A-Z0-9-]+\s+\d{2}[A-Z]{3}\d{2}\b`)
	flightHeaderLineRe     = regexp.MustCompile(`^\s*([A-Z0-9]{2,3}\d{3,4}[A-Z]?)/\d+\s+\d{2}[A-Z]{3}\d{2}\b`)
	flightHeaderDateLineRe = regexp.MustCompile(`^\s*([A-Z0-9]{2,3}\d{3,4}[A-Z]?)/\d{2}[A-Z]{3}\d{2}\s+\d{2}[A-Z]{3}\d{2}\b`)
	routeLineRe            = regexp.MustCompile(`^\s*([A-Z]{3})(?:/|\s+)([A-Z]{3})\b`)
	secRouteRe             = regexp.MustCompile(`(?im)^\s*-?\s*SEC/([A-Z]{3})-([A-Z]{3})\b`)
	flightReferenceRe      = regexp.MustCompile(`(?m)/([A-Z0-9]{2,3}\d{3,4}[A-Z]?)\s*$`)
	tableCombinedFlightRouteRe = regexp.MustCompile(`(?im)^\s*([A-Z]{3})\s+([A-Z]{3})\s+([A-Z0-9]{2,3}\d{3,4}[A-Z]?)\b`)
	tableFlightRouteRe     = regexp.MustCompile(`(?im)^\s*([A-Z]{3})\s+([A-Z]{3})\s+([A-Z0-9]{2,3})\s+(\d{1,4})\b`)

	// Aircraft type pattern.
	acTypeRe = regexp.MustCompile(`\bAIRCRAFT\s+TYPE\s*:\s*([A-Z0-9-]+)`)
)

func (p *Parser) Parse(msg *acars.Message) registry.Result {
	if msg.Text == "" {
		return nil
	}

	result := &Result{
		MsgID:     int64(msg.ID),
		Timestamp: msg.Timestamp,
		MsgType:   "LOADSHEET",
		Tail:      msg.Tail,
	}

	text := msg.Text

	// Extract weights.
	if m := zfwRe.FindStringSubmatch(text); len(m) > 1 {
		result.ZFW, _ = strconv.Atoi(m[1])
	} else if m := zfwSlashRe.FindStringSubmatch(text); len(m) > 1 {
		result.ZFW, _ = strconv.Atoi(m[1])
	} else if m := zfwActualRe.FindStringSubmatch(text); len(m) > 1 {
		result.ZFW, _ = strconv.Atoi(m[1])
	}
	if m := towRe.FindStringSubmatch(text); len(m) > 1 {
		result.TOW, _ = strconv.Atoi(m[1])
	} else if m := towSlashRe.FindStringSubmatch(text); len(m) > 1 {
		result.TOW, _ = strconv.Atoi(m[1])
	} else if m := towActualRe.FindStringSubmatch(text); len(m) > 1 {
		result.TOW, _ = strconv.Atoi(m[1])
	}
	if m := lawRe.FindStringSubmatch(text); len(m) > 1 {
		result.LAW, _ = strconv.Atoi(m[1])
	}
	if m := tofRe.FindStringSubmatch(text); len(m) > 1 {
		result.TOF, _ = strconv.Atoi(m[1])
	} else if m := tofSlashRe.FindStringSubmatch(text); len(m) > 1 {
		result.TOF, _ = strconv.Atoi(m[1])
	} else if m := takeOffFuelRe.FindStringSubmatch(text); len(m) > 1 {
		result.TOF, _ = strconv.Atoi(m[1])
	}

	// Extract passenger count.
	if pax, ok := extractPassengerCount(text); ok {
		result.PAX = pax
	}

	// Extract crew.
	if m := crewRe.FindStringSubmatch(text); len(m) > 1 {
		result.Crew = m[1]
	} else if m := crewSlashRe.FindStringSubmatch(text); len(m) > 1 {
		result.Crew = m[1]
	} else if m := routeLineCrewRe.FindStringSubmatch(text); len(m) > 1 {
		result.Crew = m[1]
	}

	// Extract trim.
	if m := trimRe.FindStringSubmatch(text); len(m) > 1 {
		result.Trim = strings.TrimSpace(m[1])
	}

	// Extract MAC values.
	if m := maczfwRe.FindStringSubmatch(text); len(m) > 1 {
		result.MACZFW = m[1]
	}
	if m := mactowRe.FindStringSubmatch(text); len(m) > 1 {
		result.MACTOW = m[1]
	}

	// Extract edition.
	if m := ednoRe.FindStringSubmatch(text); len(m) > 1 {
		result.Edition = m[1]
	} else if m := loadsheetFinalEditionRe.FindStringSubmatch(text); len(m) > 1 {
		result.Edition = m[1]
	} else if m := finalEditionRe.FindStringSubmatch(text); len(m) > 1 {
		result.Edition = m[1]
	} else if m := multilineEditionRe.FindStringSubmatch(text); len(m) > 1 {
		result.Edition = m[1]
	}

	// Extract flight/route.
	if flight, origin, destination, ok := extractFlightRoute(text); ok {
		result.Flight = airlines.TranslateFlight(flight)
		result.Origin = airports.NormaliseCode(origin)
		result.Destination = airports.NormaliseCode(destination)
	}

	// Extract aircraft type.
	if m := acTypeRe.FindStringSubmatch(text); len(m) > 1 {
		result.AircraftType = strings.TrimSpace(m[1])
	}

	// Only return if we got useful weight data.
	if result.ZFW == 0 && result.TOW == 0 && result.PAX == 0 {
		return nil
	}

	return result
}

func normaliseLoadsheetText(text string) string {
	upper := strings.ToUpper(text)
	return strings.Join(strings.Fields(upper), "")
}

func extractFlightRoute(text string) (flight, origin, destination string, ok bool) {
	if m := flightRouteRe.FindStringSubmatch(text); len(m) > 3 {
		return m[1], m[2], m[3], true
	}
	if m := flightRouteDateCompactRe.FindStringSubmatch(strings.ToUpper(text)); len(m) > 3 {
		return m[1], m[2], m[3], true
	}
	if m := flightRouteInlineCompactRe.FindStringSubmatch(strings.ToUpper(text)); len(m) > 3 {
		return m[1], m[2], m[3], true
	}
	if m := flightRouteFinalLineRe.FindStringSubmatch(strings.ToUpper(text)); len(m) > 3 {
		return m[1], m[2], m[3], true
	}
	if m := flightRouteCompactRe.FindStringSubmatch(strings.ToUpper(text)); len(m) > 3 {
		return m[1], m[2], m[3], true
	}
	if route := secRouteRe.FindStringSubmatch(strings.ToUpper(text)); len(route) > 2 {
		if flight := flightReferenceRe.FindStringSubmatch(strings.ToUpper(text)); len(flight) > 1 {
			return flight[1], route[1], route[2], true
		}
	}

	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	for index, line := range lines {
		trimmedLine := strings.ToUpper(strings.TrimSpace(line))
		m := flightHeaderLineRe.FindStringSubmatch(trimmedLine)
		if len(m) < 2 {
			m = flightHeaderDateLineRe.FindStringSubmatch(trimmedLine)
		}
		if len(m) < 2 {
			continue
		}

		flight = m[1]
		if route := routeLineRe.FindStringSubmatch(strings.ToUpper(strings.TrimSpace(line))); len(route) > 2 {
			return flight, route[1], route[2], true
		}

		for next := index + 1; next < len(lines) && next <= index+2; next++ {
			candidate := strings.ToUpper(strings.TrimSpace(lines[next]))
			if candidate == "" {
				continue
			}
			if route := routeLineRe.FindStringSubmatch(candidate); len(route) > 2 {
				return flight, route[1], route[2], true
			}
			break
		}
	}

	if m := tableCombinedFlightRouteRe.FindStringSubmatch(strings.ToUpper(text)); len(m) > 3 {
		return m[3], m[1], m[2], true
	}
	if m := tableFlightRouteRe.FindStringSubmatch(strings.ToUpper(text)); len(m) > 4 {
		return m[3] + m[4], m[1], m[2], true
	}

	return "", "", "", false
}

func extractPassengerCount(text string) (int, bool) {
	if m := paxPlusRe.FindStringSubmatch(text); len(m) > 2 {
		base, _ := strconv.Atoi(m[1])
		extra, _ := strconv.Atoi(m[2])
		return base + extra, true
	}
	if m := passengerTTLRe.FindStringSubmatch(text); len(m) > 1 {
		count, _ := strconv.Atoi(m[1])
		return count, true
	}
	if m := paxInlineTTLRe.FindStringSubmatch(text); len(m) > 1 {
		count, _ := strconv.Atoi(m[1])
		return count, true
	}
	if m := paxTTLRe.FindStringSubmatch(text); len(m) > 1 {
		count, _ := strconv.Atoi(m[1])
		return count, true
	}
	if m := paxWithTTLRe.FindStringSubmatch(text); len(m) > 2 {
		count, _ := strconv.Atoi(m[2])
		return count, true
	}
	if m := paxRe.FindStringSubmatch(text); len(m) > 1 {
		total := 0
		for i := 1; i < len(m) && m[i] != ""; i++ {
			val, _ := strconv.Atoi(m[i])
			total += val
		}
		return total, true
	}
	if m := ttlRe.FindStringSubmatch(text); len(m) > 1 {
		count, _ := strconv.Atoi(m[1])
		return count, true
	}
	return 0, false
}
