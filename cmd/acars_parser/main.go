// Command-line entry point for ACARS Parser (extract-focused).
//
// Note about input formats
// ------------------------
// The upstream parsers in this repo expect an "acars.Message" object with at least:
//   - label (e.g. "H1", "B6", "AA"...)
//   - text  (the ACARS/ARINC message text payload)
//
// In the real world, you may have any of these inputs:
//  1. NATS feed wrapper: {"message":{...}, "airframe":{...}, ...}
//  2. Flat message:      {"label":"H1","text":"...", ...}
//  3. Decoder logs:      dumpvdl2 / dumphfdl JSON where ACARS is nested deep.
//
// This CLI tries to autodetect all three. Use -all to keep messages even if no parser matched.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"acars_parser/internal/acars"
	"acars_parser/internal/airlines"
	_ "acars_parser/internal/parsers" // register all parsers via init()
	"acars_parser/internal/patterns"
	"acars_parser/internal/registry"
)

var (
	pdcFlightRe       = regexp.MustCompile(`\b([A-Z0-9]{3,8})\s+CLRD\s+TO\s+([A-Z]{4})\b`)
	pdcOriginHeaderRe = regexp.MustCompile(`(?s)/[A-Z]+\.[A-Z0-9]+/[A-Z]+\s+\d{4}\s+\d{6}\s+([A-Z]{4})\b`)
	pdcDestinationRe  = regexp.MustCompile(`\bCLRD\s+TO\s+([A-Z]{4})\b`)
	fsmHeaderRe       = regexp.MustCompile(`(?s)/[A-Z]+\.[A-Z0-9]+/FSM\s+\d{4}\s+\d{6}\s+([A-Z]{4})\s+([A-Z0-9]{3,8})\b`)
	iniMetadataRe     = regexp.MustCompile(`(?i)INI(\d{2})(\d{2})(\d{4})\s+([A-Z]{3}\d{1,4}[A-Z]?)\s*/\d{2}/([A-Z]{4})/([A-Z]{4})\b`)
	raFlightNumberRe  = regexp.MustCompile(`\bFLIGHT\s+NUMBER:\s*([A-Z0-9]{2,10}(?:/[A-Z0-9]{2,10})?)\b`)
	raSectorRe        = regexp.MustCompile(`\bSECTOR:\s*([A-Z]{4})-([A-Z]{4})\b`)
)

type ExtractOut struct {
	Message *OutputMessage `json:"message"`
	Results []any          `json:"results,omitempty"`
}

type OutputMessage struct {
	ID          acars.FlexInt64 `json:"id"`
	Source      string          `json:"source"`
	Timestamp   string          `json:"timestamp"`
	Tail        string          `json:"tail"`
	Flight      string          `json:"flight,omitempty"`
	FlightID    string          `json:"flight_id,omitempty"`
	Latitude    float64         `json:"latitude,omitempty"`
	Longitude   float64         `json:"longitude,omitempty"`
	Departing   string          `json:"departing_airport,omitempty"`
	Destination string          `json:"destination_airport,omitempty"`
	Text        string          `json:"text"`
	Label       string          `json:"label"`
	Frequency   float64         `json:"frequency"`
	Airframe    *acars.Airframe `json:"airframe,omitempty"`
	Station     *acars.Station  `json:"station,omitempty"`
}

type Stats struct {
	Lines          int
	ParsedJAERO    int
	ParsedNATS     int
	ParsedFlat     int
	ParsedNested   int
	SkippedNoLabel int
	Emitted        int
	Matched        int
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "acars_parser (extract) - commands:")
	fmt.Fprintln(w, "  extract  - parse JSONL or JAERO TXT file and output JSON or text")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  acars_parser extract -input messages.jsonl [-output out.json] [-pretty] [-all] [-stats] [-format json|text]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Notes:")
	fmt.Fprintln(w, "  - Input may be JSONL (one JSON object per line) or a JAERO TXT log.")
	fmt.Fprintln(w, "  - For dumpvdl2/dumphfdl logs, the tool will try to find label/text in nested paths.")
	fmt.Fprintln(w, "")
}

func main() {
	if len(os.Args) < 2 {
		usage(os.Stderr)
		os.Exit(2)
	}
	cmd := strings.ToLower(os.Args[1])
	switch cmd {
	case "extract":
		runExtract(os.Args[2:])
	case "-h", "--help", "help":
		usage(os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		usage(os.Stderr)
		os.Exit(2)
	}
}

func runExtract(args []string) {
	fs := flag.NewFlagSet("extract", flag.ExitOnError)
	inPath := fs.String("input", "", "Input JSONL file (default: stdin)")
	outPath := fs.String("output", "", "Output JSON file (default: stdout)")
	pretty := fs.Bool("pretty", false, "Pretty-print JSON output")
	outputFormat := fs.String("format", "json", "Output format: json or text")
	includeAll := fs.Bool("all", false, "Include messages even if no parser matched")
	showStats := fs.Bool("stats", false, "Print basic counters to stderr")
	_ = fs.Parse(args)

	if *outputFormat != "json" && *outputFormat != "text" {
		fmt.Fprintf(os.Stderr, "Unsupported output format: %s\n", *outputFormat)
		os.Exit(2)
	}

	// Ensure parsers priority ordering is stable.
	registry.Default().Sort()

	var r io.Reader = os.Stdin
	if *inPath != "" {
		f, err := os.Open(*inPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open input: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		r = f
	}

	scanner := bufio.NewScanner(r)
	// JSON lines can be long; bump buffer (20MB).
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 60*1024*1024)

	out := make([]ExtractOut, 0, 1024)
	st := &Stats{}

	firstLine := ""
	for scanner.Scan() {
		st.Lines++
		firstLine = strings.TrimSpace(scanner.Text())
		if firstLine != "" {
			break
		}
	}

	if firstLine != "" {
		if looksLikeJAEROHeader(firstLine) {
			out = processJAEROInput(scanner, firstLine, out, *includeAll, st)
		} else {
			out = processJSONLInput(scanner, firstLine, out, *includeAll, st)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Input read error: %v\n", err)
		os.Exit(1)
	}

	var wout io.Writer = os.Stdout
	if *outPath != "" {
		f, err := os.Create(*outPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create output: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		wout = f
	}

	if *outputFormat == "text" {
		_, _ = io.WriteString(wout, formatExtractText(out))
		if wout == os.Stdout {
			_, _ = wout.Write([]byte("\n"))
		}
	} else {
		enc, err := marshalJSON(out, *pretty)
		if err != nil {
			fmt.Fprintf(os.Stderr, "JSON encode error: %v\n", err)
			os.Exit(1)
		}
		_, _ = wout.Write(enc)
		if wout == os.Stdout {
			_, _ = wout.Write([]byte("\n"))
		}
	}

	if *showStats {
		fmt.Fprintf(os.Stderr,
			"stats: lines=%d parsed(jaero=%d nats=%d flat=%d nested=%d) skipped(no_label_text)=%d emitted=%d matched=%d\n",
			st.Lines, st.ParsedJAERO, st.ParsedNATS, st.ParsedFlat, st.ParsedNested, st.SkippedNoLabel, st.Emitted, st.Matched,
		)
	}
}

func processJSONLInput(scanner *bufio.Scanner, firstLine string, out []ExtractOut, includeAll bool, st *Stats) []ExtractOut {
	out = processJSONLLine(firstLine, out, includeAll, st)
	for scanner.Scan() {
		st.Lines++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		out = processJSONLLine(line, out, includeAll, st)
	}
	return out
}

func processJSONLLine(line string, out []ExtractOut, includeAll bool, st *Stats) []ExtractOut {
	b := []byte(line)

	msgs, kind := decodeToMessage(b)
	if len(msgs) == 0 {
		st.SkippedNoLabel++
		return out
	}

	switch kind {
	case "nats":
		st.ParsedNATS++
	case "flat":
		st.ParsedFlat++
	case "nested":
		st.ParsedNested++
	}

	for _, msg := range msgs {
		if msg == nil || (strings.TrimSpace(msg.Label) == "" && strings.TrimSpace(msg.Text) == "") {
			continue
		}
		var appended bool
		var matched bool
		out, appended, matched = appendOut(out, msg, includeAll)
		if appended {
			st.Emitted++
		}
		if matched {
			st.Matched++
		}
	}

	return out
}

func processJAEROInput(scanner *bufio.Scanner, firstHeader string, out []ExtractOut, includeAll bool, st *Stats) []ExtractOut {
	currentHeader := strings.TrimSpace(firstHeader)
	body := make([]string, 0, 8)

	emit := func() {
		if currentHeader == "" {
			return
		}
		st.ParsedJAERO++
		var appended bool
		var matched bool
		out, appended, matched = appendJAEROBlock(out, currentHeader, body, includeAll)
		if appended {
			st.Emitted++
		}
		if matched {
			st.Matched++
		}
		if !appended {
			st.SkippedNoLabel++
		}
		body = body[:0]
	}

	for scanner.Scan() {
		st.Lines++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if looksLikeJAEROHeader(trimmed) {
			emit()
			currentHeader = trimmed
			continue
		}
		if currentHeader == "" {
			continue
		}
		body = append(body, line)
	}

	emit()
	return out
}

func appendJAEROBlock(out []ExtractOut, header string, body []string, includeAll bool) ([]ExtractOut, bool, bool) {
	msg := parseJAEROBlock(header, body)
	if msg == nil {
		return out, false, false
	}
	return appendOut(out, msg, includeAll)
}

func parseJAEROBlock(header string, body []string) *acars.Message {
	timestamp, tail, label, airframe, ok := parseJAEROHeader(header)
	if !ok {
		return nil
	}

	text := extractJAEROPayload(body)
	if text == "" {
		return nil
	}

	return &acars.Message{
		Source:    "jaero",
		Timestamp: timestamp,
		Tail:      tail,
		Text:      text,
		Label:     label,
		Airframe:  airframe,
	}
}

func parseJAEROHeader(header string) (string, string, string, *acars.Airframe, bool) {
	fields := strings.Fields(strings.TrimSpace(header))
	if len(fields) < 8 || !looksLikeJAEROHeader(header) {
		return "", "", "", nil, false
	}

	parsedTime, err := time.Parse("15:04:05 02-01-06 MST", fields[0]+" "+fields[1]+" "+fields[2])
	if err != nil {
		return "", "", "", nil, false
	}

	bangIdx := strings.Index(header, " ! ")
	if bangIdx < 0 {
		return "", "", "", nil, false
	}

	leftFields := strings.Fields(strings.TrimSpace(header[:bangIdx]))
	if len(leftFields) == 0 {
		return "", "", "", nil, false
	}
	tail := strings.TrimSpace(leftFields[len(leftFields)-1])
	if tail == "" {
		return "", "", "", nil, false
	}

	aes := ""
	for _, field := range leftFields {
		if strings.HasPrefix(field, "AES:") {
			aes = strings.ToUpper(strings.TrimSpace(strings.TrimPrefix(field, "AES:")))
			break
		}
	}

	rightFields := strings.Fields(strings.TrimSpace(header[bangIdx+3:]))
	if len(rightFields) < 2 {
		return "", "", "", nil, false
	}
	label := strings.TrimSpace(rightFields[0])
	if label == "" {
		return "", "", "", nil, false
	}

	airframeDescription := ""
	if len(rightFields) > 2 {
		airframeDescription = strings.TrimSpace(strings.Join(rightFields[2:], " "))
	}

	var airframe *acars.Airframe
	if aes != "" || airframeDescription != "" || tail != "" {
		airframe = &acars.Airframe{
			Tail:              tail,
			ICAO:              aes,
			ManufacturerModel: airframeDescription,
		}
	}

	return parsedTime.UTC().Format(time.RFC3339), tail, label, airframe, true
}

func extractJAEROPayload(body []string) string {
	payloadLines := make([]string, 0, len(body))
	started := false

	for _, rawLine := range body {
		trimmed := strings.TrimSpace(rawLine)
		if !started {
			if trimmed == "" {
				continue
			}
			if trimmed == "-" {
				return ""
			}
			if looksLikeJAERODecoderCommentaryLine(trimmed) {
				continue
			}
			started = true
		}

		if trimmed == "" {
			break
		}
		if !looksLikeJAEROPayloadLine(trimmed) {
			break
		}

		payloadLines = append(payloadLines, trimmed)
	}

	if len(payloadLines) == 0 {
		return ""
	}

	joined := joinJAEROPayloadLines(payloadLines)
	return normaliseJAEROPayload(joined)
}

func looksLikeJAEROPayloadLine(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" || line == "-" {
		return false
	}
	if looksLikeJAEROHeader(line) || looksLikeJAERODecoderCommentaryLine(line) {
		return false
	}
	return true
}

func looksLikeJAERODecoderCommentaryLine(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}

	commentaryPrefixes := []string{
		"FANS-1/A ",
		"CPDLC ",
		"CPDLC Uplink Message:",
		"CPDLC Downlink Message:",
		"ADS-C message:",
		"Header:",
		"Message data:",
		"Msg ID:",
		"Timestamp:",
		"Facility designation:",
		"Flight level:",
		"Fix:",
		"Position:",
		"ATC CLEARANCE",
		"REQUEST POSITION REPORT",
	}
	for _, prefix := range commentaryPrefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}

	return false
}

func joinJAEROPayloadLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString(lines[0])
	for i := 1; i < len(lines); i++ {
		current := strings.TrimSpace(lines[i])
		previous := strings.TrimSpace(lines[i-1])
		if shouldJoinJAEROPayloadInline(previous, current) {
			builder.WriteString(current)
			continue
		}
		builder.WriteByte('\n')
		builder.WriteString(current)
	}

	return builder.String()
}

func shouldJoinJAEROPayloadInline(previous string, current string) bool {
	if current == "" {
		return false
	}
	if strings.HasPrefix(current, "/") || strings.HasPrefix(current, "#") || strings.HasPrefix(current, "- #") {
		return true
	}
	return !strings.ContainsAny(previous, " \t") && !strings.ContainsAny(current, " \t")
}

func normaliseJAEROPayload(payload string) string {
	payload = strings.TrimSpace(payload)
	payload = strings.ReplaceAll(payload, "- #MD", "")
	payload = strings.ReplaceAll(payload, "- #M1", "")
	payload = strings.TrimPrefix(payload, "- ")
	return strings.TrimSpace(payload)
}

func looksLikeJAEROHeader(line string) bool {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) < 8 {
		return false
	}
	if fields[2] != "UTC" {
		return false
	}
	if !strings.HasPrefix(fields[3], "AES:") || !strings.HasPrefix(fields[4], "GES:") {
		return false
	}
	if _, err := time.Parse("15:04:05 02-01-06 MST", fields[0]+" "+fields[1]+" "+fields[2]); err != nil {
		return false
	}
	return strings.Contains(line, " ! ")
}

func appendOut(out []ExtractOut, msg *acars.Message, includeAll bool) ([]ExtractOut, bool, bool) {
	enrichMessageFromText(msg)
	results := registry.Default().Dispatch(msg)
	if !includeAll && len(results) == 0 {
		return out, false, false
	}
	rany := make([]any, 0, len(results))
	for _, r := range results {
		rany = append(rany, r) // keep concrete types for JSON marshal
	}
	out = append(out, ExtractOut{Message: newOutputMessage(msg), Results: rany})
	return out, true, len(results) > 0
}

func enrichMessageFromText(msg *acars.Message) {
	if msg == nil {
		return
	}

	flightNumber, cleanTail, destinationAirport := parseAFNMetadataFromText(msg.Text)
	departingAirport := ""
	if flightNumber == "" {
		flightNumber = parseFPNFlightFromText(msg.Text)
	}
	if flightNumber == "" || destinationAirport == "" || departingAirport == "" {
		iniFlightNumber, iniDepartingAirport, iniDestinationAirport := parseINIMetadataFromText(msg.Text)
		if flightNumber == "" {
			flightNumber = iniFlightNumber
		}
		if departingAirport == "" {
			departingAirport = iniDepartingAirport
		}
		if destinationAirport == "" {
			destinationAirport = iniDestinationAirport
		}
	}
	if strings.EqualFold(strings.TrimSpace(msg.Label), "RA") && (flightNumber == "" || destinationAirport == "" || departingAirport == "") {
		raFlightNumber, raDepartingAirport, raDestinationAirport := parseRAFlightMetadataFromText(msg.Text)
		if flightNumber == "" {
			flightNumber = raFlightNumber
		}
		if departingAirport == "" {
			departingAirport = raDepartingAirport
		}
		if destinationAirport == "" {
			destinationAirport = raDestinationAirport
		}
	}
	if flightNumber == "" || destinationAirport == "" || departingAirport == "" {
		pdcFlightNumber, pdcDepartingAirport, pdcDestinationAirport := parsePDCMetadataFromText(msg.Text)
		if flightNumber == "" {
			flightNumber = pdcFlightNumber
		}
		if departingAirport == "" {
			departingAirport = pdcDepartingAirport
		}
		if destinationAirport == "" {
			destinationAirport = pdcDestinationAirport
		}
	}
	if flightNumber == "" || departingAirport == "" {
		fsmFlightNumber, fsmDepartingAirport := parseFSMMetadataFromText(msg.Text)
		if flightNumber == "" {
			flightNumber = fsmFlightNumber
		}
		if departingAirport == "" {
			departingAirport = fsmDepartingAirport
		}
	}
	if flightNumber == "" && cleanTail == "" && destinationAirport == "" && departingAirport == "" {
		return
	}

	if cleanTail != "" {
		if msg.Airframe == nil {
			msg.Airframe = &acars.Airframe{}
		}
		if strings.TrimSpace(msg.Airframe.Tail) == "" || strings.HasPrefix(strings.TrimSpace(msg.Airframe.Tail), ".") {
			msg.Airframe.Tail = cleanTail
		}
		if strings.TrimSpace(msg.Tail) == "" {
			msg.Tail = cleanTail
		}
	}

	if flightNumber == "" && destinationAirport == "" && departingAirport == "" {
		return
	}

	if msg.Flight == nil {
		msg.Flight = &acars.Flight{}
	}
	if strings.TrimSpace(msg.Flight.Flight) == "" && flightNumber != "" {
		msg.Flight.Flight = airlines.TranslateFlight(strings.TrimSpace(flightNumber))
	}
	if strings.TrimSpace(msg.Flight.DepartingAirport) == "" && departingAirport != "" {
		msg.Flight.DepartingAirport = strings.TrimSpace(departingAirport)
	}
	if strings.TrimSpace(msg.Flight.DestinationAirport) == "" && destinationAirport != "" {
		msg.Flight.DestinationAirport = strings.TrimSpace(destinationAirport)
	}
	normaliseMessageFlight(msg)
}

func parseAFNMetadataFromText(text string) (flightNumber string, cleanTail string, destinationAirport string) {
	text = strings.TrimSpace(text)
	if !strings.Contains(text, ".AFN/") {
		return "", "", ""
	}

	fmhIdx := strings.Index(text, "/FMH")
	if fmhIdx >= 0 {
		rest := text[fmhIdx+4:]
		commaIdx := strings.IndexByte(rest, ',')
		if commaIdx > 0 {
			flightNumber = strings.TrimSpace(rest[:commaIdx])
			tailAndRest := rest[commaIdx+1:]
			fields := strings.Split(tailAndRest, ",")
			if len(fields) > 0 {
				cleanTail = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(fields[0]), "."))
			}
		}
	}

	fakIdx := strings.Index(text, "/FAK0,")
	if fakIdx >= 0 {
		rest := text[fakIdx+6:]
		endIdx := strings.IndexByte(rest, '/')
		if endIdx >= 0 {
			destinationAirport = strings.TrimSpace(rest[:endIdx])
		} else {
			destinationAirport = strings.TrimSpace(rest)
		}
	}

	if len(destinationAirport) != 4 {
		destinationAirport = ""
	}

	return strings.TrimSpace(flightNumber), strings.TrimSpace(cleanTail), destinationAirport
}

func parseFPNFlightFromText(text string) string {
	text = strings.TrimSpace(strings.ToUpper(text))
	if !strings.HasPrefix(text, "FPN/") {
		return ""
	}

	match := patterns.FPNFlightPattern.FindStringSubmatch(text)
	if len(match) < 2 {
		return ""
	}

	return strings.TrimSpace(match[1])
}

func parsePDCMetadataFromText(text string) (flightNumber string, departingAirport string, destinationAirport string) {
	text = strings.TrimSpace(strings.ToUpper(text))
	if text == "" {
		return "", "", ""
	}
	if !strings.Contains(text, "PDC") && !strings.Contains(text, "CLRD TO") {
		return "", "", ""
	}

	if match := pdcFlightRe.FindStringSubmatch(text); len(match) == 3 {
		flightNumber = strings.TrimSpace(match[1])
		destinationAirport = strings.TrimSpace(match[2])
	}
	if match := pdcOriginHeaderRe.FindStringSubmatch(text); len(match) == 2 {
		departingAirport = strings.TrimSpace(match[1])
	}
	if destinationAirport == "" {
		if match := pdcDestinationRe.FindStringSubmatch(text); len(match) == 2 {
			destinationAirport = strings.TrimSpace(match[1])
		}
	}
	if flightNumber == "" {
		tokens := strings.Fields(text)
		flightNumber = strings.TrimSpace(patterns.ExtractFlightNumber(text, tokens))
	}
	return flightNumber, strings.TrimSpace(departingAirport), strings.TrimSpace(destinationAirport)
}

func parseFSMMetadataFromText(text string) (flightNumber string, departingAirport string) {
	text = strings.TrimSpace(strings.ToUpper(text))
	if text == "" || !strings.Contains(text, "FS1/FSM") {
		return "", ""
	}

	match := fsmHeaderRe.FindStringSubmatch(text)
	if len(match) != 3 {
		return "", ""
	}

	departingAirport = strings.TrimSpace(match[1])
	flightNumber = strings.TrimSpace(match[2])
	if len(departingAirport) != 4 || flightNumber == "" {
		return "", ""
	}

	return flightNumber, departingAirport
}

func parseINIMetadataFromText(text string) (flightNumber string, departingAirport string, destinationAirport string) {
	text = strings.TrimSpace(strings.ToUpper(text))
	if text == "" || !strings.Contains(text, "INI01") {
		return "", "", ""
	}

	match := iniMetadataRe.FindStringSubmatch(text)
	if len(match) != 7 {
		return "", "", ""
	}

	flightNumber = strings.TrimSpace(match[4])
	departingAirport = strings.TrimSpace(match[5])
	destinationAirport = strings.TrimSpace(match[6])
	if len(departingAirport) != 4 || len(destinationAirport) != 4 || flightNumber == "" {
		return "", "", ""
	}

	return flightNumber, departingAirport, destinationAirport
}

func parseRAFlightMetadataFromText(text string) (flightNumber string, departingAirport string, destinationAirport string) {
	text = strings.TrimSpace(strings.ToUpper(text))
	if text == "" || !strings.Contains(text, "FLIGHT NUMBER:") {
		return "", "", ""
	}

	if match := raFlightNumberRe.FindStringSubmatch(text); len(match) == 2 {
		flightNumber = strings.TrimSpace(match[1])
		if strings.Contains(flightNumber, "/") {
			parts := strings.SplitN(flightNumber, "/", 2)
			if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
				flightNumber = strings.TrimSpace(parts[1])
			} else {
				flightNumber = strings.TrimSpace(parts[0])
			}
		}
	}

	if match := raSectorRe.FindStringSubmatch(text); len(match) == 3 {
		departingAirport = strings.TrimSpace(match[1])
		destinationAirport = strings.TrimSpace(match[2])
	}

	if flightNumber == "" {
		return "", "", ""
	}

	return flightNumber, departingAirport, destinationAirport
}

func newOutputMessage(msg *acars.Message) *OutputMessage {
	if msg == nil {
		return nil
	}

	normaliseMessageFlight(msg)

	out := &OutputMessage{
		ID:        msg.ID,
		Source:    msg.Source,
		Timestamp: msg.Timestamp,
		Tail:      msg.Tail,
		Text:      msg.Text,
		Label:     msg.Label,
		Frequency: msg.Frequency,
		Airframe:  msg.Airframe,
		Station:   msg.Station,
	}

	if msg.Flight != nil {
		out.Flight = airlines.TranslateFlight(strings.TrimSpace(msg.Flight.Flight))
		out.FlightID = strings.TrimSpace(msg.Flight.ID)
		out.Latitude = msg.Flight.Latitude
		out.Longitude = msg.Flight.Longitude
		out.Departing = strings.TrimSpace(msg.Flight.DepartingAirport)
		out.Destination = strings.TrimSpace(msg.Flight.DestinationAirport)
	}

	return out
}

func marshalJSON(v any, pretty bool) ([]byte, error) {
	if pretty {
		return json.MarshalIndent(v, "", "  ")
	}
	return json.Marshal(v)
}

func formatExtractText(out []ExtractOut) string {
	if len(out) == 0 {
		return ""
	}

	blocks := make([]string, 0, len(out))
	for _, item := range out {
		block := formatExtractTextBlock(item)
		if strings.TrimSpace(block) == "" {
			continue
		}
		blocks = append(blocks, block)
	}

	return strings.Join(blocks, "\n\n")
}

func formatExtractTextBlock(out ExtractOut) string {
	rawText := strings.TrimRightFunc(strings.TrimSpace(extractRawText(out)), func(r rune) bool {
		return r == '\r' || r == '\n'
	})
	formattedResults := collectHumanReadableResults(out.Results)

	switch {
	case rawText != "" && len(formattedResults) > 0:
		return rawText + "\n" + strings.Join(formattedResults, "\n")
	case rawText != "":
		return rawText
	case len(formattedResults) > 0:
		return strings.Join(formattedResults, "\n")
	default:
		return ""
	}
}

func extractRawText(out ExtractOut) string {
	if out.Message == nil {
		return ""
	}
	return strings.TrimRight(out.Message.Text, "\r\n")
}

func collectHumanReadableResults(results []any) []string {
	if len(results) == 0 {
		return nil
	}

	formatted := make([]string, 0, len(results))
	seen := make(map[string]struct{}, len(results))
	for _, result := range results {
		humanReadable, ok := result.(registry.HumanReadableResult)
		if !ok {
			continue
		}
		text := strings.TrimRight(humanReadable.HumanReadableText(), "\r\n")
		normalised := strings.TrimSpace(text)
		if normalised == "" {
			continue
		}
		if _, exists := seen[normalised]; exists {
			continue
		}
		seen[normalised] = struct{}{}
		formatted = append(formatted, text)
	}

	return formatted
}

func decodeToMessage(b []byte) ([]*acars.Message, string) {
	// 1) NATS wrapper
	var w acars.NATSWrapper
	if err := json.Unmarshal(b, &w); err == nil && w.Message != nil {
		if msg := w.ToMessage(); msg != nil && (msg.Label != "" || msg.Text != "") {
			normaliseMessageFlight(msg)
			return []*acars.Message{msg}, "nats"
		}
	}

	// 2) Flat message (only accept if it actually contains label/text)
	var m acars.Message
	if err := json.Unmarshal(b, &m); err == nil {
		var flatRoot map[string]any
		if err := json.Unmarshal(b, &flatRoot); err == nil {
			enrichMessageFlight(&m, flatRoot)
		}
		if strings.TrimSpace(m.Label) != "" || strings.TrimSpace(m.Text) != "" {
			normaliseMessageFlight(&m)
			return []*acars.Message{&m}, "flat"
		}
	}

	// 3) Nested formats (dumpvdl2/dumphfdl, etc.)
	var anyObj any
	if err := json.Unmarshal(b, &anyObj); err != nil {
		return nil, ""
	}
	msgs := buildMessagesFromNested(anyObj)
	if len(msgs) > 0 {
		return msgs, "nested"
	}
	return nil, ""
}

// buildMessagesFromNested tries common paths used by dumpvdl2 / dumphfdl logs.
// It returns multiple messages if MIAM decoded content is present (both outer and inner).
func buildMessagesFromNested(obj any) []*acars.Message {
	root, ok := obj.(map[string]any)
	if !ok {
		return nil
	}

	var msgs []*acars.Message

	// First, try to extract outer ACARS message (e.g., MA with compressed text)
	outerLabel := firstString(root,
		"label",
		"message.label",
		"acars.label",
		"vdl2.avlc.acars.label",
		"vdl2.avlc.acars.lbl",
		"hfdl.lpdu.hfnpdu.acars.label",
		"hfdl.lpdu.hfnpdu.acars.acars_label",
	)

	outerText := firstString(root,
		"text",
		"message.text",
		"msg_text",
		"message.msg_text",
		"acars.text",
		"acars.message.text",
		"vdl2.avlc.acars.text",
		"vdl2.avlc.acars.msg_text",
		"vdl2.avlc.acars.message.text",
		"hfdl.lpdu.hfnpdu.acars.text",
		"hfdl.lpdu.hfnpdu.acars.msg_text",
	)

	// Check if MIAM decoded content exists
	miamLabel := firstString(root,
		"vdl2.avlc.acars.miam.single_transfer.miam_core.data.acars.label",
		"hfdl.lpdu.hfnpdu.acars.miam.single_transfer.miam_core.data.acars.label",
	)

	miamText := firstString(root,
		"vdl2.avlc.acars.miam.single_transfer.miam_core.data.acars.message.text",
		"hfdl.lpdu.hfnpdu.acars.miam.single_transfer.miam_core.data.acars.message.text",
	)

	// If we have both outer and MIAM content, create both messages
	if strings.TrimSpace(outerLabel) != "" || strings.TrimSpace(outerText) != "" {
		meta := extractNestedMessageMetadata(root)

		// Create outer message (e.g., MA with compressed text)
		outerMsg := &acars.Message{
			Label:     outerLabel,
			Text:      outerText,
			Tail:      meta.tail,
			Timestamp: meta.timestamp,
			Frequency: meta.frequency,
			Source:    meta.source,
			Airframe:  meta.airframe,
			Flight:    extractFlight(root),
		}
		normaliseMessageFlight(outerMsg)
		msgs = append(msgs, outerMsg)

		// If MIAM decoded content exists, create second message with decoded content
		// Use label "MB" for MIAM decoded messages to distinguish from outer "MA" message
		if strings.TrimSpace(miamLabel) != "" || strings.TrimSpace(miamText) != "" {
			miamMsg := &acars.Message{
				Label:     "MB",
				Text:      miamText,
				Tail:      meta.tail,
				Timestamp: meta.timestamp,
				Frequency: meta.frequency,
				Source:    meta.source,
				Airframe:  meta.airframe,
				Flight:    extractFlight(root),
			}
			normaliseMessageFlight(miamMsg)
			msgs = append(msgs, miamMsg)
		}
	}

	if len(msgs) == 0 {
		if synthetic := buildSyntheticHFDLDataMessage(root); synthetic != nil {
			msgs = append(msgs, synthetic)
		}
	}

	return msgs
}

type nestedMessageMeta struct {
	tail      string
	timestamp string
	frequency float64
	source    string
	airframe  *acars.Airframe
}

func extractNestedMessageMetadata(root map[string]any) nestedMessageMeta {
	tail := firstString(root,
		"tail",
		"airframe.tail",
		"vdl2.avlc.acars.reg",
		"vdl2.avlc.acars.tail",
		"hfdl.lpdu.hfnpdu.acars.reg",
	)

	ts := firstString(root,
		"timestamp",
		"message.timestamp",
	)
	if ts == "" {
		sec := firstInt64(root,
			"vdl2.t.sec",
			"hfdl.t.sec",
			"t.sec",
		)
		usec := firstInt64(root,
			"vdl2.t.usec",
			"hfdl.t.usec",
			"t.usec",
		)
		if sec > 0 {
			t := time.Unix(sec, usec*1000).UTC()
			ts = t.Format(time.RFC3339Nano)
		}
	}

	freq := firstFloat64(root,
		"frequency",
		"message.frequency",
		"vdl2.freq",
		"hfdl.freq",
	)
	if freq > 1_000_000 {
		freq = freq / 1_000_000.0
	}

	src := firstString(root,
		"source",
		"vdl2.app.name",
		"hfdl.app.name",
		"app.name",
	)

	icao := firstString(root,
		"airframe.icao",
		"hfdl.lpdu.ac_info.icao",
	)

	var airframe *acars.Airframe
	if strings.TrimSpace(tail) != "" || strings.TrimSpace(icao) != "" {
		airframe = &acars.Airframe{
			Tail: strings.TrimSpace(tail),
			ICAO: strings.ToUpper(strings.TrimSpace(icao)),
		}
	}

	return nestedMessageMeta{
		tail:      tail,
		timestamp: ts,
		frequency: freq,
		source:    src,
		airframe:  airframe,
	}
}

func buildSyntheticHFDLDataMessage(root map[string]any) *acars.Message {
	hfnpduType := strings.TrimSpace(firstString(root, "hfdl.lpdu.hfnpdu.type.name"))
	if hfnpduType == "" || !strings.HasSuffix(strings.ToLower(hfnpduType), "data") {
		return nil
	}

	flight := extractFlight(root)
	if flight == nil || strings.TrimSpace(flight.ID) == "" {
		return nil
	}
	if flight.Latitude == 0 && flight.Longitude == 0 {
		return nil
	}

	meta := extractNestedMessageMetadata(root)
	text := fmt.Sprintf("HFDL %s", hfnpduType)
	if gsID := firstString(root, "hfdl.lpdu.dst.id"); strings.TrimSpace(gsID) != "" {
		text = fmt.Sprintf("HFDL %s GS %s", hfnpduType, strings.TrimSpace(gsID))
	}

	msg := &acars.Message{
		Label:     "HFDL",
		Text:      text,
		Tail:      meta.tail,
		Timestamp: meta.timestamp,
		Frequency: meta.frequency,
		Source:    meta.source,
		Airframe:  meta.airframe,
		Flight:    flight,
	}
	normaliseMessageFlight(msg)
	return msg
}

func enrichMessageFlight(msg *acars.Message, root map[string]any) {
	if msg == nil || root == nil {
		return
	}

	if flight := extractFlight(root); flight != nil {
		msg.Flight = flight
	}
}

func extractFlight(root map[string]any) *acars.Flight {
	flightNumber := firstString(root,
		"flight",
		"message.flight",
		"acars.flight",
		"vdl2.avlc.acars.flight",
		"hfdl.lpdu.hfnpdu.acars.flight",
	)

	flightID := firstString(root,
		"flight_id",
		"message.flight_id",
		"hfdl.lpdu.hfnpdu.flight_id",
		"vdl2.avlc.x25.clnp.cotp.x225_spdu.x227_apdu.context_mgmt.cm_aircraft_message.data.atn_context_mgmt_logon_request.flight_id",
	)

	departureAirport := firstString(root,
		"departure_airport",
		"message.departure_airport",
		"vdl2.avlc.x25.clnp.cotp.x225_spdu.x227_apdu.context_mgmt.cm_aircraft_message.data.atn_context_mgmt_logon_request.departure_airport",
	)

	destinationAirport := firstString(root,
		"destination_airport",
		"message.destination_airport",
		"vdl2.avlc.x25.clnp.cotp.x225_spdu.x227_apdu.context_mgmt.cm_aircraft_message.data.atn_context_mgmt_logon_request.destination_airport",
	)

	latitude := firstFloat64(root,
		"latitude",
		"message.latitude",
		"hfdl.lpdu.hfnpdu.pos.lat",
	)
	longitude := firstFloat64(root,
		"longitude",
		"message.longitude",
		"hfdl.lpdu.hfnpdu.pos.lon",
	)

	if strings.TrimSpace(flightNumber) == "" && strings.TrimSpace(flightID) == "" &&
		strings.TrimSpace(departureAirport) == "" && strings.TrimSpace(destinationAirport) == "" &&
		latitude == 0 && longitude == 0 {
		return nil
	}

	return &acars.Flight{
		ID:                 strings.TrimSpace(flightID),
		Flight:             airlines.TranslateFlight(strings.TrimSpace(flightNumber)),
		DepartingAirport:   strings.TrimSpace(departureAirport),
		DestinationAirport: strings.TrimSpace(destinationAirport),
		Latitude:           latitude,
		Longitude:          longitude,
	}
}

func normaliseMessageFlight(msg *acars.Message) {
	if msg == nil || msg.Flight == nil {
		return
	}

	msg.Flight = &acars.Flight{
		ID:                 strings.TrimSpace(msg.Flight.ID),
		Flight:             airlines.TranslateFlight(strings.TrimSpace(msg.Flight.Flight)),
		Status:             msg.Flight.Status,
		DepartingAirport:   strings.TrimSpace(msg.Flight.DepartingAirport),
		DestinationAirport: strings.TrimSpace(msg.Flight.DestinationAirport),
		Latitude:           msg.Flight.Latitude,
		Longitude:          msg.Flight.Longitude,
		Altitude:           msg.Flight.Altitude,
	}
}

func firstString(root map[string]any, paths ...string) string {
	for _, p := range paths {
		if v, ok := deepGet(root, p); ok {
			switch t := v.(type) {
			case string:
				if strings.TrimSpace(t) != "" {
					return t
				}
			case float64:
				// Sometimes labels are numeric; preserve as int string where possible.
				if t == float64(int64(t)) {
					return strconv.FormatInt(int64(t), 10)
				}
				return strconv.FormatFloat(t, 'f', -1, 64)
			case bool:
				if t {
					return "true"
				}
				return "false"
			}
		}
	}
	return ""
}

func firstInt64(root map[string]any, paths ...string) int64 {
	for _, p := range paths {
		if v, ok := deepGet(root, p); ok {
			switch t := v.(type) {
			case float64:
				return int64(t)
			case string:
				if i, err := strconv.ParseInt(strings.TrimSpace(t), 10, 64); err == nil {
					return i
				}
			}
		}
	}
	return 0
}

func firstFloat64(root map[string]any, paths ...string) float64 {
	for _, p := range paths {
		if v, ok := deepGet(root, p); ok {
			switch t := v.(type) {
			case float64:
				return t
			case string:
				if f, err := strconv.ParseFloat(strings.TrimSpace(t), 64); err == nil {
					return f
				}
			}
		}
	}
	return 0
}

// deepGet walks a map[string]any using a dotted path: "a.b.c".
func deepGet(root map[string]any, dotted string) (any, bool) {
	parts := strings.Split(dotted, ".")
	var cur any = root
	for _, part := range parts {
		switch node := cur.(type) {
		case map[string]any:
			v, ok := node[part]
			if !ok {
				return nil, false
			}
			cur = v
		default:
			return nil, false
		}
	}
	return cur, true
}
