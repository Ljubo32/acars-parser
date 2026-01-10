// Package fst parses FST (Label 15) flight status messages.
package fst

import (
	"regexp"
	"strconv"
	"strings"
	"sync"

	"acars_parser/internal/acars"
	"acars_parser/internal/patterns"
	"acars_parser/internal/registry"
)

// Result represents a parsed Label 15 FST (Flight Status) report.
type Result struct {
	MsgID         int64   `json:"message_id"`
	Timestamp     string  `json:"timestamp"`
	Tail          string  `json:"tail,omitempty"`
	Sequence      string  `json:"sequence,omitempty"`    // Usually "01"
	Origin        string  `json:"origin,omitempty"`      // ICAO code
	Destination   string  `json:"destination,omitempty"` // ICAO code
	Latitude      float64 `json:"latitude,omitempty"`
	Longitude     float64 `json:"longitude,omitempty"`
	FlightLevel   int     `json:"flight_level,omitempty"`
	GroundSpeed   int     `json:"ground_speed,omitempty"`   // Ground speed value
	SpeedUnit     string  `json:"speed_unit,omitempty"`     // "knots" or "kmh"
	IAS           int     `json:"ias,omitempty"`            // Indicated Airspeed in knots
	TAS           int     `json:"tas,omitempty"`            // True Airspeed in knots
	SpeedType     string  `json:"speed_type,omitempty"`     // "GS", "IAS", "TAS", "IAS+GS", "IAS+KMH"
	Temperature   int     `json:"temperature,omitempty"`    // Celsius, can be negative
	WindSpeed     int     `json:"wind_speed,omitempty"`     // Knots
	WindDirection int     `json:"wind_direction,omitempty"` // Degrees
	RawData       string  `json:"raw_data,omitempty"`       // Remaining unparsed data
}

func (r *Result) Type() string     { return "fst" }
func (r *Result) MessageID() int64 { return r.MsgID }

// Parser parses Label 15 FST messages.
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

func (p *Parser) Name() string     { return "fst" }
func (p *Parser) Labels() []string { return []string{"15"} }
func (p *Parser) Priority() int    { return 100 }

func (p *Parser) QuickCheck(text string) bool {
	return strings.HasPrefix(strings.TrimSpace(text), "FST") ||
		strings.Contains(text, "FST01") ||
		strings.Contains(text, "FST02")
}

func (p *Parser) Parse(msg *acars.Message) registry.Result {
	if msg.Text == "" {
		return nil
	}

	text := strings.TrimSpace(msg.Text)

	// Strip any prefix before FST (like M51ABA0012).
	if idx := strings.Index(text, "FST"); idx > 0 {
		text = text[idx:]
	}

	// Try grok-based parsing.
	compiler, err := getCompiler()
	if err != nil {
		return nil
	}

	match := compiler.Parse(text)
	if match == nil {
		return nil
	}

	result := &Result{
		MsgID:       int64(msg.ID),
		Timestamp:   msg.Timestamp,
		Tail:        msg.Tail,
		Sequence:    match.Captures["seq"],
		Origin:      match.Captures["origin"],
		Destination: match.Captures["dest"],
	}

	// Parse latitude (DDMMD or DDMMDD format - degrees, minutes, tenths).
	lat := parseCoord(match.Captures["lat"])
	if match.Captures["lat_dir"] == "S" {
		lat = -lat
	}
	result.Latitude = lat

	// Parse longitude.
	lon := parseCoord(match.Captures["lon"])
	if match.Captures["lon_dir"] == "W" {
		lon = -lon
	}
	result.Longitude = lon

	// Parse the rest of the fields.
	rest := match.Captures["rest"]
	if len(rest) > 0 {
		result.RawData = rest
		parseFields(rest, result)
	}

	return result
}

// parseCoord parses FST coordinate format.
// 5-digit: DDMMD format (deg, min, tenths) - 51420 = 51 deg 42.0' = 51.7 deg
// 6-digit: DDMMTT format (deg, min, hundredths) - 452140 = 45 deg 21.40' = 45.3567 deg
//
//	or DDDMMD for longitudes > 99 deg - 1043245 = 104 deg 32.45'
//
// 7-digit: DDDMMTT format (3-digit deg, min, hundredths) - 0249275 = 024 deg 92.75'
func parseCoord(s string) float64 {
	if len(s) < 5 {
		return 0
	}

	var deg int
	var min float64
	var err error

	if len(s) == 5 {
		// DDMMD format: 2 digits degrees, 2 digits minutes, 1 digit tenths.
		deg, err = strconv.Atoi(s[0:2])
		if err != nil {
			return 0
		}
		minWhole, err := strconv.Atoi(s[2:4])
		if err != nil {
			return 0
		}
		minTenths, err := strconv.Atoi(s[4:5])
		if err != nil {
			return 0
		}
		min = float64(minWhole) + float64(minTenths)/10.0
	} else if len(s) == 6 {
		// Try DDMMTT format first (works for coordinates with deg < 100)
		deg2, err2 := strconv.Atoi(s[0:2])
		deg3, err3 := strconv.Atoi(s[0:3])
		minWhole2, errMin2 := strconv.Atoi(s[2:4])
		minHundredths2, errMinH2 := strconv.Atoi(s[4:6])
		minWhole3, errMin3 := strconv.Atoi(s[3:5])
		minTenths3, errMinT3 := strconv.Atoi(s[5:6])

		// Try DDMMTT format (2-digit degree, 2-digit minute whole, 2-digit minute decimal)
		if err2 == nil && errMin2 == nil && errMinH2 == nil {
			minTest := float64(minWhole2) + float64(minHundredths2)/100.0
			if deg2 <= 180 && minTest < 60 {
				deg = deg2
				min = minTest
			} else if deg2 <= 90 && minTest >= 60 {
				// Minutes >= 60 means this is actually decimal format: DD.DDDD
				// Example: 467315 = 46.7315° (not 46° 73.15')
				decimal, errDec := strconv.Atoi(s[2:6])
				if errDec == nil {
					return float64(deg2) + float64(decimal)/10000.0
				}
			}
		}

		// If DDMMTT didn't work or gave invalid result, try DDDMMD (3-digit degree for longitude > 99)
		if deg == 0 && err3 == nil && deg3 > 90 && deg3 <= 180 && errMin3 == nil && errMinT3 == nil {
			minTest := float64(minWhole3) + float64(minTenths3)/10.0
			if minTest < 60 {
				deg = deg3
				min = minTest
			}
		}

		if deg == 0 {
			return 0
		}
	} else if len(s) == 7 {
		// DDDDDDD format: 3 digits degrees + 4 digits decimal fraction.
		// 0249275 = 024.9275 degrees (not minutes!)
		deg, err = strconv.Atoi(s[0:3])
		if err != nil || deg > 180 {
			return 0
		}
		decimal, err := strconv.Atoi(s[3:7])
		if err != nil {
			return 0
		}
		return float64(deg) + float64(decimal)/10000.0
	} else {
		return 0
	}

	if min >= 60 {
		return 0
	}

	return float64(deg) + min/60.0
}

// parseFields tries to extract additional fields from the FST data.
// Expected format after coordinates: FL(3) GS(3-4) UNKNOWN(3) TEMP(M/P+2-3+C) WIND(3-4 digits)
// Example: "330 854 242 M54C 6235410711950911600009590004"
//
//	FL=330, GS=854, skip=242, temp=-54C, wind=623541 (wind speed=62, direction=354)
func parseFields(data string, result *Result) {
	// Split on spaces and process structured fields first.
	parts := strings.Fields(data)

	if len(parts) >= 1 {
		// First field: Flight Level (3 digits)
		if fl, err := strconv.Atoi(parts[0]); err == nil && fl >= 0 && fl <= 600 {
			result.FlightLevel = fl
		}
	}

	if len(parts) >= 2 {
		// Second field: Speed - can be in different formats:
		// Format 1 (newer aircraft): IAS+KMH concatenated (7 digits) - e.g., "2001084" = IAS 200 + KMH 1084
		// Format 2 (older aircraft): IAS only (3 digits) - e.g., "192", followed by GS in next field
		// Format 3: Single speed value (GS or KMH)
		speedStr := parts[1]
		if speed, err := strconv.Atoi(speedStr); err == nil && speed > 0 {
			// Check if this is concatenated IAS+KMH (7 digits, value > 1000000)
			if len(speedStr) == 7 && speed > 1000000 {
				// Format: IIIKKKKK (IAS 3 digits + KMH 4 digits)
				// Example: 2001084 = IAS 200 + KMH 1084
				if ias, err := strconv.Atoi(speedStr[0:3]); err == nil {
					result.IAS = ias
					if kmh, err := strconv.Atoi(speedStr[3:7]); err == nil {
						// Keep KMH value as-is, don't convert
						result.GroundSpeed = kmh
						result.SpeedUnit = "kmh"
						result.SpeedType = "IAS+KMH"
					}
				}
			} else if speed > 1000 && len(speedStr) > 4 {
				// Old logic: if > 1000 but not 7 digits, split first 3 as IAS
				if s, err := strconv.Atoi(speedStr[0:3]); err == nil {
					result.IAS = s
					result.SpeedType = "IAS"
				}
			} else if speed >= 700 && speed <= 1000 {
				// High value (~800-850): likely KM/H, keep as-is
				result.GroundSpeed = speed
				result.SpeedUnit = "kmh"
				result.SpeedType = "KMH"
			} else if speed >= 350 && speed < 700 {
				// Medium value (~350-699): Ground Speed in knots
				result.GroundSpeed = speed
				result.SpeedUnit = "knots"
				result.SpeedType = "GS"
			} else if speed >= 100 && speed < 350 {
				// Low value (~100-349): Indicated Airspeed in knots
				result.IAS = speed
				result.SpeedType = "IAS"

				// Check if next field is GS (for older aircraft format: "192 312")
				if len(parts) >= 3 {
					if nextSpeed, err := strconv.Atoi(parts[2]); err == nil && nextSpeed >= 100 && nextSpeed < 1000 {
						// Next field looks like a speed value
						if nextSpeed >= 700 {
							// It's KMH, keep as-is
							result.GroundSpeed = nextSpeed
							result.SpeedUnit = "kmh"
							result.SpeedType = "IAS+KMH"
						} else if nextSpeed >= 100 && nextSpeed < 700 {
							// It's Ground Speed in knots (can be 200-600 range)
							result.GroundSpeed = nextSpeed
							result.SpeedUnit = "knots"
							result.SpeedType = "IAS+GS"
						}
					}
				}
			} else {
				// Fallback: treat as ground speed in knots
				result.GroundSpeed = speed
				result.SpeedUnit = "knots"
				result.SpeedType = "GS"
			}
		}
	}

	// Temperature: Search for M/P pattern in remaining fields (could be at index 2, 3, or later)
	// Examples: M54C, P15C, M020
	tempPattern := regexp.MustCompile(`^([MP])(\d{2,3})C?$`)
	for i := 2; i < len(parts); i++ {
		if m := tempPattern.FindStringSubmatch(parts[i]); m != nil {
			if temp, err := strconv.Atoi(m[2]); err == nil {
				if m[1] == "M" {
					result.Temperature = -temp
				} else {
					result.Temperature = temp
				}

				// Wind data might be in the next field after temperature
				if i+1 < len(parts) {
					windData := parts[i+1]
					if len(windData) >= 5 {
						// Try parsing as SSDDD (2 digit speed, 3 digit direction)
						if ws, err := strconv.Atoi(windData[0:2]); err == nil && ws >= 0 && ws <= 99 {
							if wd, err := strconv.Atoi(windData[2:5]); err == nil && wd >= 0 && wd <= 360 {
								result.WindSpeed = ws
								result.WindDirection = wd
							}
						}
						// If that didn't work, try SSSDDD (3 digit speed, 3 digit direction)
						if result.WindSpeed == 0 && len(windData) >= 6 {
							if ws, err := strconv.Atoi(windData[0:3]); err == nil && ws >= 100 && ws <= 999 {
								if wd, err := strconv.Atoi(windData[3:6]); err == nil && wd >= 0 && wd <= 360 {
									result.WindSpeed = ws
									result.WindDirection = wd
								}
							}
						}
					}
				}
				break
			}
		}
	}
}
