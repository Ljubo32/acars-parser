package airlines

import (
	"embed"
	"encoding/csv"
	"io"
	"strconv"
	"strings"
)

//go:embed iataicao.csv
var codesFS embed.FS

var defaultTranslator = mustLoadTranslator()

type translator struct {
	iataToIcao map[string]string
}

func mustLoadTranslator() *translator {
	data, err := codesFS.ReadFile("iataicao.csv")
	if err != nil {
		return &translator{iataToIcao: map[string]string{}}
	}

	lookup, err := parseCSV(string(data))
	if err != nil {
		return &translator{iataToIcao: map[string]string{}}
	}

	return &translator{iataToIcao: lookup}
}

func parseCSV(data string) (map[string]string, error) {
	reader := csv.NewReader(strings.NewReader(data))
	records, err := reader.ReadAll()
	if err != nil {
		if err == io.EOF {
			return map[string]string{}, nil
		}
		return nil, err
	}

	lookup := make(map[string]string, len(records))
	for index, record := range records {
		if len(record) < 2 {
			continue
		}

		iata := strings.ToUpper(strings.TrimSpace(record[0]))
		icao := strings.ToUpper(strings.TrimSpace(record[1]))

		if index == 0 && iata == "IATA" && icao == "ICAO" {
			continue
		}
		if len(iata) != 2 || len(icao) != 3 || !isAlphaNumeric(iata) {
			continue
		}

		lookup[iata] = icao
	}

	return lookup, nil
}

func isAlpha(value string) bool {
	for _, char := range value {
		if char < 'A' || char > 'Z' {
			return false
		}
	}
	return value != ""
}

func isAlphaNumeric(value string) bool {
	for _, char := range value {
		isUpperAlpha := char >= 'A' && char <= 'Z'
		isDigit := char >= '0' && char <= '9'
		if !isUpperAlpha && !isDigit {
			return false
		}
	}
	return value != ""
}

// TranslateFlight converts a leading two-character IATA airline designator to
// the corresponding three-letter ICAO code. If no mapping exists, the input is
// returned unchanged apart from trimming surrounding whitespace.
func TranslateFlight(flight string) string {
	trimmed := strings.TrimSpace(flight)
	if len(trimmed) < 3 {
		return trimmed
	}

	normalised := trimmed
	prefix := strings.ToUpper(trimmed[:2])
	if !isAlphaNumeric(prefix) {
		return normaliseFlightNumber(normalised)
	}
	if trimmed[2] < '0' || trimmed[2] > '9' {
		return normaliseFlightNumber(normalised)
	}

	icao, ok := defaultTranslator.iataToIcao[prefix]
	if ok {
		normalised = icao + trimmed[2:]
	}

	return normaliseFlightNumber(normalised)
}

func normaliseFlightNumber(flight string) string {
	trimmed := strings.TrimSpace(flight)
	if trimmed == "" {
		return ""
	}

	parts := splitFlightNumber(trimmed)
	if parts == nil {
		return trimmed
	}

	numberValue, err := strconv.Atoi(parts.number)
	if err != nil {
		return trimmed
	}

	return parts.prefix + strconv.Itoa(numberValue) + parts.suffix
}

type flightNumberParts struct {
	prefix string
	number string
	suffix string
}

func splitFlightNumber(flight string) *flightNumberParts {
	trimmed := strings.TrimSpace(flight)
	if trimmed == "" {
		return nil
	}

	prefixEnd := 0
	for prefixEnd < len(trimmed) {
		char := trimmed[prefixEnd]
		if (char < 'A' || char > 'Z') && char != '-' {
			break
		}
		prefixEnd++
	}
	if prefixEnd == 0 || prefixEnd >= len(trimmed) {
		return nil
	}

	numberEnd := prefixEnd
	for numberEnd < len(trimmed) {
		char := trimmed[numberEnd]
		if char < '0' || char > '9' {
			break
		}
		numberEnd++
	}
	if numberEnd == prefixEnd {
		return nil
	}

	suffix := trimmed[numberEnd:]
	for i := 0; i < len(suffix); i++ {
		char := suffix[i]
		isUpperAlpha := char >= 'A' && char <= 'Z'
		isDigit := char >= '0' && char <= '9'
		if !isUpperAlpha && !isDigit {
			return nil
		}
	}

	return &flightNumberParts{
		prefix: trimmed[:prefixEnd],
		number: trimmed[prefixEnd:numberEnd],
		suffix: suffix,
	}
}