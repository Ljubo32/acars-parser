// Package abs parses ABS0 route hints embedded in H1 label messages.
package abs

import (
	"strconv"
	"regexp"
	"strings"

	"acars_parser/internal/acars"
	"acars_parser/internal/registry"
)

var (
	absIDRe = regexp.MustCompile(`\b(ABS0[0-9A-Z_\-]*)`)
	absCoordLineRe = regexp.MustCompile(`^\s*(\d{5})\s+(\S+)`)
	absTempRe     = regexp.MustCompile(`[+-]\d{2}`)
)

// Position represents one ABS0 position row.
type Position struct {
	Latitude     float64 `json:"latitude,omitempty"`
	Longitude    float64 `json:"longitude,omitempty"`
	AltitudeFt   int     `json:"altitude_ft,omitempty"`
	TemperatureC int     `json:"temperature_c,omitempty"`
	RawLine      string  `json:"raw_line,omitempty"`
}

// Result represents a parsed ABS0 route hint.
type Result struct {
	MsgID       int64  `json:"message_id"`
	Timestamp   string `json:"timestamp"`
	MsgType     string `json:"msg_type,omitempty"`
	Tail        string `json:"tail,omitempty"`
	BlockID     string `json:"block_id,omitempty"`
	Origin      string `json:"origin,omitempty"`
	Destination string `json:"destination,omitempty"`
	Route       string `json:"route,omitempty"`
	Level       string `json:"level,omitempty"`
	Latitude    float64    `json:"latitude,omitempty"`
	Longitude   float64    `json:"longitude,omitempty"`
	AltitudeFt  int        `json:"altitude_ft,omitempty"`
	Temperature int        `json:"temperature_c,omitempty"`
	Positions   []Position `json:"positions,omitempty"`
	RawData     string `json:"raw_data,omitempty"`
}

func (r *Result) Type() string     { return "abs" }
func (r *Result) MessageID() int64 { return r.MsgID }

// Parser extracts route hints from ABS0 blocks in H1 messages.
type Parser struct{}

func init() {
	registry.Register(&Parser{})
}

func (p *Parser) Name() string     { return "abs" }
func (p *Parser) Labels() []string { return []string{"H1"} }
func (p *Parser) Priority() int    { return 40 }

func (p *Parser) QuickCheck(text string) bool {
	return strings.Contains(strings.ToUpper(text), "ABS0")
}

func (p *Parser) Parse(msg *acars.Message) registry.Result {
	if msg.Text == "" {
		return nil
	}

	text := strings.ReplaceAll(strings.ReplaceAll(msg.Text, "\r\n", "\n"), "\r", "\n")
	lines := strings.Split(text, "\n")

	absIndex := -1
	var absMatch []int
	for index, line := range lines {
		absMatch = absIDRe.FindStringIndex(strings.ToUpper(line))
		if absMatch != nil {
			absIndex = index
			break
		}
	}
	if absIndex < 0 || absMatch == nil {
		return nil
	}

	lineUpper := strings.ToUpper(lines[absIndex])
	blockID := strings.TrimSpace(lineUpper[absMatch[0]:absMatch[1]])
	afterLine := strings.TrimSpace(lineUpper[absMatch[1]:])
	nextLine := ""
	if absIndex+1 < len(lines) {
		nextLine = strings.TrimSpace(strings.ToUpper(lines[absIndex+1]))
	}

	result := &Result{
		MsgID:     int64(msg.ID),
		Timestamp: msg.Timestamp,
		MsgType:   "ABS",
		Tail:      msg.Tail,
		BlockID:   blockID,
		RawData:   msg.Text,
	}

	for _, line := range lines[absIndex+1:] {
		position, ok := parsePositionLine(line)
		if !ok {
			continue
		}
		result.Positions = append(result.Positions, position)
	}
	if len(result.Positions) > 0 {
		result.Latitude = result.Positions[0].Latitude
		result.Longitude = result.Positions[0].Longitude
		result.AltitudeFt = result.Positions[0].AltitudeFt
		result.Temperature = result.Positions[0].TemperatureC
	}

	candidates := uniqueCandidates(afterLine, nextLine)
	for _, candidate := range candidates {
		origin, destination, level, ok := parseRouteCandidate(candidate)
		if !ok {
			continue
		}

		result.Origin = origin
		result.Destination = destination
		result.Route = origin + "-" + destination
		result.Level = level
		return result
	}

	if len(result.Positions) == 0 {
		return nil
	}

	return result
}

func uniqueCandidates(afterLine, nextLine string) []string {
	var candidates []string
	seen := map[string]struct{}{}
	for _, candidate := range []string{afterLine, nextLine, strings.TrimSpace(afterLine + " " + nextLine)} {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		candidates = append(candidates, candidate)
	}
	return candidates
}

func parsePositionLine(line string) (Position, bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return Position{}, false
	}

	matches := absCoordLineRe.FindStringSubmatch(trimmed)
	if len(matches) < 3 {
		return Position{}, false
	}

	lat, ok := parseDegMMM(matches[1], true)
	if !ok {
		return Position{}, false
	}
	lon, ok := parseDegMMM(matches[2], false)
	if !ok {
		return Position{}, false
	}

	position := Position{
		Latitude:  lat,
		Longitude: lon,
		RawLine:   trimmed,
	}

	remaining := strings.TrimSpace(matches[2])
	tempIdx := absTempRe.FindStringIndex(remaining)
	if tempIdx != nil {
		digitsBeforeTemp := digitsOnly(remaining[:tempIdx[0]])
		if len(digitsBeforeTemp) >= 5 {
			altitude, err := strconv.Atoi(digitsBeforeTemp[len(digitsBeforeTemp)-5:])
			if err == nil {
				position.AltitudeFt = altitude
			}
		}
		temperature, err := strconv.Atoi(remaining[tempIdx[0]:tempIdx[1]])
		if err == nil {
			position.TemperatureC = temperature
		}
	}

	return position, true
}

func parseDegMMM(digits string, isLat bool) (float64, bool) {
	digits = digitsOnly(digits)
	if len(digits) < 5 {
		return 0, false
	}

	raw := digits[:5]
	degreeDigits := 2
	degrees, err := strconv.Atoi(raw[:degreeDigits])
	if err != nil {
		return 0, false
	}
	thousandths, err := strconv.Atoi(raw[degreeDigits:])
	if err != nil {
		return 0, false
	}

	value := float64(degrees) + (float64(thousandths) / 1000.0)
	if isLat {
		if value < 0 || value > 90 {
			return 0, false
		}
		return value, true
	}
	if value < 0 || value > 180 {
		return 0, false
	}
	return value, true
}

func digitsOnly(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			b.WriteRune(ch)
		}
	}
	return b.String()
}

func parseRouteCandidate(candidate string) (origin, destination, level string, ok bool) {
	fields := strings.Fields(strings.ToUpper(candidate))
	for index, token := range fields {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}

		start := 0
		for start < len(token) && start < 5 && token[start] >= '0' && token[start] <= '9' {
			start++
		}
		core := token[start:]
		if len(core) == 8 {
			if index+1 >= len(fields) {
				continue
			}
			next := strings.TrimSpace(fields[index+1])
			if !isDigits(next) || len(next) > 3 {
				continue
			}

			origin = core[:4]
			destination = core[4:8]
			if !isUpperAlpha(origin) || !isUpperAlpha(destination) {
				continue
			}
			level = next
			return origin, destination, level, true
		}

		if len(core) < 9 || len(core) > 16 {
			continue
		}

		origin = core[:4]
		destination = core[4:8]
		if !isUpperAlpha(origin) || !isUpperAlpha(destination) {
			continue
		}

		tail := core[8:]
		if tail == "" || !isDigits(tail) {
			continue
		}
		if len(tail) == 3 {
			level = tail
		} else {
			level = ""
		}
		return origin, destination, level, true
	}

	return "", "", "", false
}

func isUpperAlpha(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, ch := range s {
		if ch < 'A' || ch > 'Z' {
			return false
		}
	}
	return true
}

func isDigits(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}