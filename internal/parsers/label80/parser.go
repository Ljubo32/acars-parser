// Package label80 parses Label 80 position messages.
package label80

import (
	"regexp"
	"strconv"
	"strings"
	"sync"

	"acars_parser/internal/acars"
	"acars_parser/internal/airports"
	"acars_parser/internal/patterns"
	"acars_parser/internal/registry"
)

// Result represents position data from label 80 messages.
type Result struct {
	MsgID        int64   `json:"message_id"`
	Timestamp    string  `json:"timestamp"`
	Tail         string  `json:"tail,omitempty"`
	MsgType      string  `json:"msg_type"`
	FlightNum    string  `json:"flight_num,omitempty"`
	OriginICAO   string  `json:"origin_icao,omitempty"`
	OriginName   string  `json:"origin_name,omitempty"`
	DestICAO     string  `json:"dest_icao,omitempty"`
	DestName     string  `json:"dest_name,omitempty"`
	Latitude     float64 `json:"latitude,omitempty"`
	Longitude    float64 `json:"longitude,omitempty"`
	Altitude     int     `json:"altitude,omitempty"`
	Mach         string  `json:"mach,omitempty"`
	TAS          int     `json:"tas,omitempty"`
	FuelOnBoard  int     `json:"fuel_on_board,omitempty"`
	ETA          string  `json:"eta,omitempty"`
	ReportTime   string  `json:"report_time,omitempty"`
	WindDir      int     `json:"wind_dir,omitempty"`
	WindSpeedKts int     `json:"wind_speed_kts,omitempty"`
	WindSpeedKmh int     `json:"wind_speed_kmh,omitempty"`
	OATC         int     `json:"oat_c,omitempty"`
	OutTime      string  `json:"out_time,omitempty"`
	OffTime      string  `json:"off_time,omitempty"`
	OnTime       string  `json:"on_time,omitempty"`
	InTime       string  `json:"in_time,omitempty"`
}

func (r *Result) Type() string     { return "position" }
func (r *Result) MessageID() int64 { return r.MsgID }

// Parser parses Label 80 position messages.
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

func (p *Parser) Name() string     { return "label80" }
func (p *Parser) Labels() []string { return []string{"80"} }
func (p *Parser) Priority() int    { return 100 }

func (p *Parser) QuickCheck(text string) bool {
	return true // Label check is sufficient for 80.
}

func (p *Parser) Parse(msg *acars.Message) registry.Result {
	if msg.Text == "" {
		return nil
	}

	text := strings.TrimSpace(msg.Text)

	// Try grok-based parsing.
	compiler, err := getCompiler()
	if err != nil {
		return nil
	}

	result := &Result{
		MsgID:     int64(msg.ID),
		Timestamp: msg.Timestamp,
		Tail:      msg.Tail,
	}

	// Parse all formats to extract different fields.
	matches := compiler.ParseAll(text)

	// Track if we found a header.
	foundHeader := false

	for _, match := range matches {
		switch match.FormatName {
		case "header_format":
			result.MsgType = match.Captures["msg_type"]
			result.OriginICAO = match.Captures["origin"]
			result.DestICAO = match.Captures["dest"]
			result.Tail = strings.TrimPrefix(match.Captures["tail"], ".")

			// Resolve airport names
			if result.OriginICAO != "" {
				result.OriginName = airports.GetName(result.OriginICAO)
			}
			if result.DestICAO != "" {
				result.DestName = airports.GetName(result.DestICAO)
			}

			// hdr1 often contains flight number (e.g. UAE81) but sometimes is something else (e.g. 06WO/25).
			if tok := strings.TrimSpace(match.Captures["hdr1"]); isLikelyFlight(tok) {
				result.FlightNum = tok
			}
			foundHeader = true

		case "alt_format":
			if !foundHeader {
				result.FlightNum = match.Captures["flight"]
				result.OriginICAO = match.Captures["origin"]
				result.DestICAO = match.Captures["dest"]
				result.MsgType = "FLT"

				// Resolve airport names
				if result.OriginICAO != "" {
					result.OriginName = airports.GetName(result.OriginICAO)
				}
				if result.DestICAO != "" {
					result.DestName = airports.GetName(result.DestICAO)
				}

				foundHeader = true
			}

		case "position":
			result.Latitude = parseLabel80Coord(match.Captures["lat"], match.Captures["lat_dir"])
			result.Longitude = parseLabel80Coord(match.Captures["lon"], match.Captures["lon_dir"])

		case "altitude":
			altStr := strings.TrimSpace(match.Captures["altitude"])
			if alt, err := strconv.Atoi(altStr); err == nil {
				// Many implementations use ALT 400 to mean FL400 -> 40000 ft.
				if len(altStr) <= 3 && alt > 0 && alt <= 500 {
					alt *= 100
				}
				result.Altitude = alt
			}

		case "mach":
			result.Mach = match.Captures["mach"]

		case "tas":
			if tas, err := strconv.Atoi(match.Captures["tas"]); err == nil {
				result.TAS = tas
			}

		case "fob":
			if fob, err := strconv.Atoi(match.Captures["fob"]); err == nil {
				result.FuelOnBoard = fob
			}

		case "eta":
			result.ETA = match.Captures["eta"]

		case "tme":
			if t := formatHHMM(match.Captures["tme"]); t != "" {
				result.ReportTime = t
			}

		case "wind":
			if d, err := strconv.Atoi(match.Captures["wdir"]); err == nil {
				d %= 360
				if d < 0 {
					d += 360
				}
				result.WindDir = d
			}
			if spd, err := strconv.Atoi(match.Captures["wspd"]); err == nil {
				result.WindSpeedKts = spd
				// knots -> km/h (rounded)
				result.WindSpeedKmh = (spd*1852 + 500) / 1000
			}

		case "oat":
			if oat, err := strconv.Atoi(match.Captures["oat"]); err == nil {
				result.OATC = oat
			}

		case "out_time":
			result.OutTime = match.Captures["out"]

		case "off_time":
			result.OffTime = match.Captures["off"]

		case "on_time":
			result.OnTime = match.Captures["on"]

		case "in_time":
			result.InTime = match.Captures["in"]
		}
	}

	// Return nil if we couldn't parse the header.
	if !foundHeader {
		return nil
	}

	return result
}

// parseLabel80Coord parses /POS coordinates that may be encoded as:
// - decimal degrees with a dot: "44.038"
// - compact decimal degrees without a dot: "44038" (=> 44.038), "019408" (=> 19.408)
// For compact form we insert a dot after degree digits (lat:2, lon:2 or 3 depending on length).
func parseLabel80Coord(s string, dir string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if strings.Contains(s, ".") {
		return patterns.ParseDecimalCoord(s, dir)
	}
	// compact digits-only form
	if dir == "E" || dir == "W" {
		degDigits := 3
		if len(s) <= 5 { // e.g. "19408" => 19.408
			degDigits = 2
		}
		if len(s) > degDigits {
			s = s[:degDigits] + "." + s[degDigits:]
		}
	} else {
		degDigits := 2
		if len(s) > degDigits {
			s = s[:degDigits] + "." + s[degDigits:]
		}
	}
	return patterns.ParseDecimalCoord(s, dir)
}

// isLikelyFlight returns true if s looks like a flight number token (e.g. "UAE81" or "1234"),
// and false for non-flight tokens like "06WO/25".
func isLikelyFlight(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if strings.Contains(s, "/") {
		return false
	}
	// Typical: 2-3 letters + 1-4 digits (optional trailing letter)
	if regexp.MustCompile(`^[A-Z]{2,3}\d{1,4}[A-Z]?$`).MatchString(s) {
		return true
	}
	// Sometimes only digits are used for flight number.
	return regexp.MustCompile(`^\d{1,4}$`).MatchString(s)
}

// formatHHMM converts "1037" -> "10:37" and "937" -> "09:37".
func formatHHMM(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if len(s) == 3 {
		s = "0" + s
	}
	if len(s) != 4 {
		return ""
	}
	hh := s[:2]
	mm := s[2:]
	h, err1 := strconv.Atoi(hh)
	m, err2 := strconv.Atoi(mm)
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return ""
	}
	return hh + ":" + mm
}
