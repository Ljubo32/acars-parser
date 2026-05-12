package airports

import (
	"bytes"
	"embed"
	"encoding/json"
	"strings"
)

//go:embed airports.json
var codesFS embed.FS

var defaultTranslator = mustLoadTranslator()

type translator struct {
	iataToICAO map[string]string
}

type airportRecord struct {
	IATA flexibleString `json:"iata"`
	ICAO flexibleString `json:"icao"`
}

type flexibleString string

func (value *flexibleString) UnmarshalJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return err
	}

	switch typed := decoded.(type) {
	case nil:
		*value = ""
	case string:
		*value = flexibleString(typed)
	case json.Number:
		*value = flexibleString(typed.String())
	default:
		*value = ""
	}

	return nil
}

func mustLoadTranslator() *translator {
	data, err := codesFS.ReadFile("airports.json")
	if err != nil {
		return &translator{iataToICAO: map[string]string{}}
	}

	lookup, err := parseAirportJSON(data)
	if err != nil {
		return &translator{iataToICAO: map[string]string{}}
	}

	return &translator{iataToICAO: lookup}
}

func parseAirportJSON(data []byte) (map[string]string, error) {
	var records []airportRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, err
	}

	lookup := make(map[string]string, len(records))
	for _, record := range records {
		iata := strings.ToUpper(strings.TrimSpace(string(record.IATA)))
		icao := strings.ToUpper(strings.TrimSpace(string(record.ICAO)))
		if len(iata) != 3 || len(icao) != 4 || !isUpperAlpha(iata) || !isUpperAlpha(icao) {
			continue
		}
		lookup[iata] = icao
	}

	return lookup, nil
}

// NormaliseCode returns an ICAO airport code when the input is a mapped IATA
// code. Existing ICAO codes are returned unchanged apart from trimming and
// upper-casing.
func NormaliseCode(code string) string {
	trimmed := strings.ToUpper(strings.TrimSpace(code))
	if trimmed == "" {
		return ""
	}
	if len(trimmed) == 4 && isUpperAlpha(trimmed) {
		return trimmed
	}
	if icao, ok := defaultTranslator.iataToICAO[trimmed]; ok {
		return icao
	}
	return trimmed
}

func isUpperAlpha(value string) bool {
	for _, char := range value {
		if char < 'A' || char > 'Z' {
			return false
		}
	}
	return value != ""
}