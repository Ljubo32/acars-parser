// Package miam parses MIAM (Message In A Message) ACARS messages with label
// "MA".  JAERO/libacars decodes the MIAM envelope and writes a human-readable
// block below the raw (deflate-compressed) payload.  The Go parser
// (cmd/acars_parser) substitutes that decoded block as the message text; this
// parser then reads the structured fields from it.
//
// MIAM has two primary PDU types:
//   - MIAM CORE Data  – carries an inner ACARS message, possibly compressed.
//   - MIAM CORE Ack   – acknowledges receipt of a Data frame.
//
// Example block (Ack):
//
//	MIAM:
//	 Single Transfer:
//	  MIAM CORE Ack, version 1:
//	   PDU Length: 20
//	   Aircraft ID: .A7-AMF
//	   Msg ACK num: 12
//	   Transfer result: ack
//
// Example block (Data):
//
//	MIAM:
//	 Single Transfer:
//	  MIAM CORE Data, version 1:
//	   PDU Length: 616
//	   Aircraft ID: .D-AIVA
//	   Msg num: 52
//	   ACK: not required
//	   Compression: deflate
//	   Encoding: ISO #5
//	   ACARS:
//	    Label: 3L
//	    Message:
//	     <inner content — may be garbled if deflate-compressed>
package miam

import (
	"strconv"
	"strings"

	"acars_parser/internal/acars"
	"acars_parser/internal/registry"
)

// Result holds the parsed fields from a MIAM message block.
type Result struct {
	MsgID          int64  `json:"message_id"`
	Timestamp      string `json:"timestamp,omitempty"`
	MessageType    string `json:"message_type"` // "miam_ack" or "miam_data"
	Version        int    `json:"version,omitempty"`
	TransferType   string `json:"transfer_type,omitempty"` // e.g. "Single Transfer"
	PDULength      int    `json:"pdu_length,omitempty"`
	AircraftID     string `json:"aircraft_id,omitempty"`
	MsgNum         int    `json:"msg_num,omitempty"`      // Data only
	MsgACKNum      int    `json:"msg_ack_num,omitempty"`  // Ack only
	ACKRequired    bool   `json:"ack_required,omitempty"` // Data only
	Compression    string `json:"compression,omitempty"`  // Data only
	Encoding       string `json:"encoding,omitempty"`     // Data only
	TransferResult string `json:"transfer_result,omitempty"` // Ack only
	InnerLabel     string `json:"inner_label,omitempty"`     // Data only
	InnerSublabel  string `json:"inner_sublabel,omitempty"`  // Data only
	InnerMessage   string `json:"inner_message,omitempty"`   // Data only (may be garbled)
	// FormattedText holds the full decoded MIAM block as printed by JAERO/libacars.
	// The viewer uses this to expand the raw text display, keeping the original
	// compressed payload in message.text for the default table view.
	FormattedText string `json:"formatted_text,omitempty"`
	// AssembledPayload and SegmentCount are populated for message_type
	// "miam_assembled": transfers whose compressed segments were concatenated
	// by the reassembly logic but whose MIAM block was not decoded by JAERO.
	// AssembledPayload is the raw MIAM 6-bit ACARS encoding of all segments
	// concatenated in chronological order.
	AssembledPayload string `json:"assembled_payload,omitempty"`
	SegmentCount     int    `json:"segment_count,omitempty"`

	// OriginICAO and DestICAO are extracted from the /H02 segment of REP inner
	// messages (e.g. "/H02,ZGSZ FAOR,CCA867 ,...").  These use the same JSON
	// keys as other route-bearing parsers so the state extractor picks them up
	// automatically.
	OriginICAO string `json:"origin_icao,omitempty"`
	DestICAO   string `json:"dest_icao,omitempty"`
	// FlightNum holds the flight identifier from the /H02 segment of a REP
	// inner message.  It is only populated when the outer MIAM message does
	// not already carry a flight number.
	FlightNum string `json:"flight_num,omitempty"`

	// Latitude, Longitude, FlightLevel, Temperature, WindDir and WindSpeed are
	// extracted from the /NX data segment of a REP inner message.  The format is:
	//   /NX,<ORIG DEST>/<f0>,<f1>,<lat×10⁴>,<lon×10⁴>,<f4>,<FL×10>,<temp×10>,<winddir>,<windspeed>,...
	// Latitude and Longitude use the same JSON keys as other position parsers so
	// the state extractor picks them up automatically.  FlightLevel uses the
	// "flight_level" key, which the extractor converts to feet (× 100).
	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
	FlightLevel int     `json:"flight_level,omitempty"`  // e.g. 400 for FL400
	Temperature float64 `json:"temperature_c,omitempty"` // °C (e.g. -54.4)
	WindDir     int     `json:"wind_dir_deg,omitempty"`  // degrees true
	WindSpeed   int     `json:"wind_speed_kts,omitempty"` // knots
}

func (r *Result) Type() string     { return r.MessageType }
func (r *Result) MessageID() int64 { return r.MsgID }
func (r *Result) HumanReadableText() string {
	var sb strings.Builder
	if r.MessageType == "miam_ack" {
		sb.WriteString("MIAM ACK")
		if r.TransferResult != "" {
			sb.WriteString(": ")
			sb.WriteString(r.TransferResult)
		}
		if r.MsgACKNum != 0 {
			sb.WriteString("  (ACK num: ")
			sb.WriteString(strconv.Itoa(r.MsgACKNum))
			sb.WriteByte(')')
		}
	} else {
		sb.WriteString("MIAM DATA")
		if r.InnerLabel != "" {
			sb.WriteString("  inner label: ")
			sb.WriteString(r.InnerLabel)
		}
		if r.Compression != "" {
			sb.WriteString("  compression: ")
			sb.WriteString(r.Compression)
		}
	}
	return sb.String()
}

// Parser handles MIAM messages (label "MA").
type Parser struct{}

func init() {
	registry.Register(&Parser{})
}

func (p *Parser) Name() string     { return "miam" }
func (p *Parser) Labels() []string { return []string{"MA"} }
func (p *Parser) Priority() int    { return 10 }

// QuickCheck returns true when the text looks like a JAERO-decoded MIAM block.
func (p *Parser) QuickCheck(text string) bool {
	return strings.HasPrefix(strings.TrimSpace(text), "MIAM:")
}

// Parse extracts structured fields from a JAERO-decoded MIAM block.
func (p *Parser) Parse(msg *acars.Message) registry.Result {
	if msg == nil {
		return nil
	}
	return parseMIAMBlock(msg)
}

// parseMIAMBlock reads the line-based MIAM block format that JAERO/libacars
// writes for each "MA" label message and returns a populated Result.
//
// Both the original JAERO format (title-case keys, e.g. "PDU Length:") and the
// C-Band decoder format (ALL CAPS keys, e.g. "PDU LENGTH:") are handled by
// normalising all keys to uppercase before matching.
func parseMIAMBlock(msg *acars.Message) *Result {
	lines := strings.Split(msg.Text, "\n")

	result := &Result{
		MsgID:     int64(msg.ID),
		Timestamp: msg.Timestamp,
	}

	// State flags for parsing the nested ACARS sub-section.
	inACARSSection := false
	inMessageSection := false
	var innerMessageLines []string

	for _, rawLine := range lines {
		trimmed := strings.TrimSpace(rawLine)
		upper := strings.ToUpper(trimmed)

		if trimmed == "" {
			if inMessageSection {
				innerMessageLines = append(innerMessageLines, "")
			}
			continue
		}

		// Detect the MIAM PDU type and version.  Match case-insensitively to
		// handle both "MIAM CORE Ack, version 1:" and "MIAM CORE ACK, VERSION 1:".
		if strings.Contains(upper, "MIAM CORE ACK") {
			result.MessageType = "miam_ack"
			result.Version = extractVersionSuffix(trimmed)
			inACARSSection = false
			inMessageSection = false
			continue
		}
		if strings.Contains(upper, "MIAM CORE DATA") {
			result.MessageType = "miam_data"
			result.Version = extractVersionSuffix(trimmed)
			inACARSSection = false
			inMessageSection = false
			continue
		}

		// Detect the transfer type from lines like "Single Transfer:" or
		// "SINGLE TRANSFER:" (C-Band format).
		if strings.HasSuffix(upper, "TRANSFER:") && result.TransferType == "" {
			result.TransferType = strings.TrimSuffix(trimmed, ":")
			continue
		}

		// Entering the nested ACARS sub-section.
		if upper == "ACARS:" {
			inACARSSection = true
			inMessageSection = false
			continue
		}
		// Entering the inner message content within the ACARS sub-section.
		if inACARSSection && upper == "MESSAGE:" {
			inMessageSection = true
			continue
		}

		// Collect lines of the inner ACARS message (may be garbled / compressed).
		if inMessageSection {
			// Skip JAERO/libacars decompression and CRC error annotations which
			// start with "--" (e.g. "-- DECOMPRESSION FAILED").
			if strings.HasPrefix(trimmed, "--") {
				continue
			}
			innerMessageLines = append(innerMessageLines, trimmed)
			continue
		}

		// Parse key: value pairs.  Normalise the key to uppercase so that both
		// title-case and ALL CAPS variants map to the same case arm.
		key, value, found := strings.Cut(trimmed, ":")
		if !found {
			continue
		}
		key = strings.ToUpper(strings.TrimSpace(key))
		value = strings.TrimSpace(value)

		if inACARSSection {
			switch key {
			case "LABEL":
				// The C-Band format combines label and sublabel on one line:
				// "LABEL: H1 SUBLABEL: DF"
				const sublabelMarker = " SUBLABEL:"
				if idx := strings.Index(strings.ToUpper(value), sublabelMarker); idx >= 0 {
					result.InnerLabel = strings.TrimSpace(value[:idx])
					sublabelValue := strings.TrimSpace(value[idx+len(sublabelMarker):])
					// The sublabel may be followed by further content; take the first token.
					if parts := strings.Fields(sublabelValue); len(parts) > 0 {
						result.InnerSublabel = parts[0]
					}
				} else {
					result.InnerLabel = value
				}
			case "SUBLABEL":
				result.InnerSublabel = value
			}
			continue
		}

		switch key {
		case "PDU LENGTH":
			if n, err := strconv.Atoi(value); err == nil {
				result.PDULength = n
			}
		case "AIRCRAFT ID":
			result.AircraftID = value
		case "MSG NUM":
			if n, err := strconv.Atoi(value); err == nil {
				result.MsgNum = n
			}
		case "MSG ACK NUM":
			if n, err := strconv.Atoi(value); err == nil {
				result.MsgACKNum = n
			}
		case "ACK":
			result.ACKRequired = !strings.EqualFold(value, "not required")
		case "COMPRESSION":
			result.Compression = value
		case "ENCODING":
			result.Encoding = value
		case "TRANSFER RESULT":
			result.TransferResult = value
		}
	}

	// Only emit a result when a recognisable PDU type was found.
	if result.MessageType == "" {
		return nil
	}

	if len(innerMessageLines) > 0 {
		result.InnerMessage = strings.TrimSpace(strings.Join(innerMessageLines, "\n"))
	}

	// Attempt to extract route and flight data from the inner message.
	// REP messages (/REP marker) are tried first; RTR (XML <RTR> root element)
	// is the fallback for route-report messages that use the XML-based format.
	if result.InnerMessage != "" {
		origin, dest, flight := parseREPRoute(result.InnerMessage)
		if origin == "" {
			origin, dest, flight = parseRTRRoute(result.InnerMessage)
		}
		result.OriginICAO = origin
		result.DestICAO = dest
		// Only use the extracted flight number when the outer ACARS message does
		// not already carry one; the outer Flight field is considered authoritative.
		if msg.Flight == nil || strings.TrimSpace(msg.Flight.Flight) == "" {
			result.FlightNum = flight
		}

		// Attempt to extract position and meteorological data from the /NX segment
		// (REP-specific; harmlessly skipped for other inner message types).
		if lat, lon, fl, temp, windDir, windSpeed, ok := parseREPNXData(result.InnerMessage); ok {
			result.Latitude = lat
			result.Longitude = lon
			result.FlightLevel = fl
			result.Temperature = temp
			result.WindDir = windDir
			result.WindSpeed = windSpeed
		}
	}

	// Store the full decoded block as-is for the viewer's raw text expansion.
	result.FormattedText = strings.TrimSpace(msg.Text)

	return result
}

// parseREPRoute extracts the origin, destination, and flight number from a
// REP inner ACARS message.  REP messages are identified by the /REP token.
// Two segment formats are recognised:
//
//   - /H02 format (carries flight number):
//     /H02,<ORIG DEST>,<FLIGHT>,<optional further fields>
//     Example: /H02,ZGSZ FAOR,CCA867 ,S0385,...
//
//   - /NX format (flight number not embedded in the route field):
//     /NX,<ORIG DEST>/<data block>
//     Example: /NX,VHHH FAOR/0,7,-068033,...
//
// Returns empty strings when the message is not a REP or no known route
// segment is found.
func parseREPRoute(inner string) (origin, dest, flight string) {
	// Confirm this is a REP message before doing any further work.
	if !strings.Contains(inner, "/REP") {
		return
	}

	// /H02,<ORIG DEST>,<FLIGHT>,... format.
	const h02Marker = "/H02,"
	if h02Idx := strings.Index(inner, h02Marker); h02Idx >= 0 {
		// Grab content up to the next segment marker (next slash).
		rest := inner[h02Idx+len(h02Marker):]
		if nextSeg := strings.IndexByte(rest, '/'); nextSeg >= 0 {
			rest = rest[:nextSeg]
		}
		// The segment is comma-delimited: "ORIG DEST", "FLIGHT", ...
		parts := strings.SplitN(rest, ",", 3)
		if len(parts) >= 2 {
			routeParts := strings.Fields(parts[0])
			if len(routeParts) == 2 && isREPICAOCode(routeParts[0]) && isREPICAOCode(routeParts[1]) {
				origin = routeParts[0]
				dest = routeParts[1]
			}
			flight = strings.TrimSpace(parts[1])
		}
		if origin != "" {
			return
		}
	}

	// /NX,<ORIG DEST>/<data block> format.  The flight number is not embedded
	// in the route field for this variant; it comes from the outer message.
	const nxMarker = "/NX,"
	if nxIdx := strings.Index(inner, nxMarker); nxIdx >= 0 {
		// Route ends at the first slash after the marker.
		rest := inner[nxIdx+len(nxMarker):]
		if slashIdx := strings.IndexByte(rest, '/'); slashIdx >= 0 {
			rest = rest[:slashIdx]
		}
		routeParts := strings.Fields(rest)
		if len(routeParts) == 2 && isREPICAOCode(routeParts[0]) && isREPICAOCode(routeParts[1]) {
			origin = routeParts[0]
			dest = routeParts[1]
		}
	}

	return
}

// parseREPNXData extracts position and meteorological data from the /NX data
// segment of a REP inner ACARS message.  The data block immediately follows
// the ORIG/DEST route pair and has the fixed comma-separated layout:
//
//	/NX,<ORIG DEST>/<f0>,<f1>,<lat×10⁴>,<lon×10⁴>,<f4>,<FL×10>,<temp×10>,<winddir>,<windspeed>,...
//
// Field indices within the data block (0-based):
//
//	[2] Latitude  — signed integer × 10⁻⁴ degrees (e.g. -068033 → -6.8033°S)
//	[3] Longitude — signed integer × 10⁻⁴ degrees (e.g. +170352 → +17.0352°E)
//	[5] Flight level × 10                          (e.g.   4000  → FL400)
//	[6] Outside air temperature × 10, °C          (e.g.   -544  → -54.4°C)
//	[7] Wind direction, degrees true               (e.g.    257  → 257°)
//	[8] Wind speed, knots                          (e.g.    039  → 39 kt)
//
// Returns ok=false when the /NX segment is absent or the data block is
// too short to contain all required fields.
func parseREPNXData(inner string) (lat, lon float64, fl int, tempC float64, windDir, windSpeed int, ok bool) {
	const nxMarker = "/NX,"
	nxIdx := strings.Index(inner, nxMarker)
	if nxIdx < 0 {
		return
	}

	// Skip the route field (text up to and including the first slash).
	rest := inner[nxIdx+len(nxMarker):]
	slashIdx := strings.IndexByte(rest, '/')
	if slashIdx < 0 {
		return
	}
	rest = rest[slashIdx+1:]

	// Trim at the end of the data block (the next slash terminates it).
	if endIdx := strings.IndexByte(rest, '/'); endIdx >= 0 {
		rest = rest[:endIdx]
	}

	parts := strings.Split(rest, ",")
	if len(parts) < 9 {
		return
	}

	// Field [2]: latitude (signed integer, × 10⁻⁴ degrees).
	latRaw, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return
	}
	lat = float64(latRaw) / 10000.0

	// Field [3]: longitude (signed integer, × 10⁻⁴ degrees).
	lonRaw, err := strconv.ParseInt(parts[3], 10, 64)
	if err != nil {
		return
	}
	lon = float64(lonRaw) / 10000.0

	// Field [5]: flight level encoded as FL × 10 (e.g. 4000 → FL400).
	flRaw, err := strconv.Atoi(parts[5])
	if err != nil {
		return
	}
	fl = flRaw / 10

	// Field [6]: outside air temperature encoded as °C × 10 (e.g. -544 → -54.4°C).
	tempRaw, err := strconv.ParseInt(parts[6], 10, 64)
	if err != nil {
		return
	}
	tempC = float64(tempRaw) / 10.0

	// Field [7]: wind direction in degrees true.
	windDir, err = strconv.Atoi(parts[7])
	if err != nil {
		return
	}

	// Field [8]: wind speed in knots.
	windSpeed, err = strconv.Atoi(parts[8])
	if err != nil {
		return
	}

	ok = true
	return
}

// parseRTRRoute extracts the origin, destination, and flight number from an
// RTR (Route Report) inner ACARS message.  RTR messages use an XML-based
// format and are identified by the <RTR> root element.  The route and flight
// data reside in the <DCMSAD> section:
//
//	<FROM>WSSS</FROM>
//	<TO>EGKK</TO>
//	<FNBR>SIA312</FNBR>
//
// Returns empty strings when the message is not an RTR or the required tags
// cannot be found or do not contain valid 4-letter ICAO airport codes.
func parseRTRRoute(inner string) (origin, dest, flight string) {
	upper := strings.ToUpper(inner)
	// The RTR root element appears as either "<RTR>" or "<RTR " (with attributes).
	if !strings.Contains(upper, "<RTR>") && !strings.Contains(upper, "<RTR ") {
		return
	}

	origin = extractXMLTagValue(inner, "FROM")
	dest = extractXMLTagValue(inner, "TO")
	flight = strings.TrimSpace(extractXMLTagValue(inner, "FNBR"))

	// Discard both airport codes if either fails the ICAO syntax check.
	if !isREPICAOCode(origin) || !isREPICAOCode(dest) {
		origin, dest = "", ""
	}
	return
}

// extractXMLTagValue finds and returns the text content of the first occurrence
// of <TAG>...</TAG> in s.  The tag name comparison is case-insensitive.
//
// Two strategies are attempted in order to handle the bit-corrupted XML that
// commonly appears in ACARS transmissions where either the open or the close
// tag may have extra or substituted characters:
//
//   - Close-tag-first: locate </TAG>, then scan backward to the nearest
//     preceding '>'.  The text between that '>' and '</TAG>' is the content.
//     This correctly extracts content even when the open tag is garbled (e.g.
//     <IFROM>WSSS</FROM> → "WSSS").
//
//   - Open-tag-first (fallback): locate <TAG>, then take everything up to the
//     next '<' as the content.  This correctly extracts content even when the
//     close tag is garbled (e.g. <TO>EGKK</HO → "EGKK").
//
// Returns an empty string when neither strategy produces a non-empty result.
func extractXMLTagValue(s, tag string) string {
	upperS := strings.ToUpper(s)
	openTag := "<" + strings.ToUpper(tag) + ">"
	closeTag := "</" + strings.ToUpper(tag) + ">"

	// Strategy 1: close-tag-first.
	// Locate </TAG> then walk backward to the end of the preceding open tag.
	if closeIdx := strings.Index(upperS, closeTag); closeIdx >= 0 {
		// Find the last '>' before the close tag — the end of the (possibly
		// garbled) open tag.
		if lastGT := strings.LastIndex(upperS[:closeIdx], ">"); lastGT >= 0 {
			if content := strings.TrimSpace(s[lastGT+1 : closeIdx]); content != "" {
				return content
			}
		}
	}

	// Strategy 2: open-tag-first fallback.
	// Locate <TAG> then take content up to the next '<' (the start of the
	// following tag, which may itself be garbled).
	if openIdx := strings.Index(upperS, openTag); openIdx >= 0 {
		contentStart := openIdx + len(openTag)
		rest := s[contentStart:]
		upperRest := upperS[contentStart:]
		end := strings.Index(upperRest, "<")
		if end < 0 {
			end = len(rest)
		}
		if content := strings.TrimSpace(rest[:end]); content != "" {
			return content
		}
	}

	return ""
}

// isREPICAOCode reports whether s is a syntactically valid 4-letter ICAO
// airport code (A-Z only).  3-letter IATA codes are intentionally rejected
// here because the REP /H02 route field always uses ICAO codes.
func isREPICAOCode(s string) bool {
	if len(s) != 4 {
		return false
	}
	for _, c := range s {
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	return true
}

// extractVersionSuffix parses the version number from a line such as
// "MIAM CORE Ack, version 1:" and returns it as an integer.
func extractVersionSuffix(line string) int {
	const prefix = "version "
	idx := strings.Index(strings.ToLower(line), prefix)
	if idx < 0 {
		return 0
	}
	rest := strings.TrimSuffix(strings.TrimSpace(line[idx+len(prefix):]), ":")
	n, err := strconv.Atoi(rest)
	if err != nil {
		return 0
	}
	return n
}
