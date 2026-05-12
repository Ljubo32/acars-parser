package loadsheet

import (
	"testing"

	"acars_parser/internal/acars"
)

func TestLoadsheetParseEmitsMsgType(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		ID:        acars.FlexInt64(42),
		Timestamp: "2026-03-20T00:00:00Z",
		Tail:      "9H-VJG",
		Label:     "C1",
		Text:      "LOADSHEET U21234/001 123ABC456 LTN DUB AIRCRAFT TYPE: A320 ZFW 62000 TOW 70100 LAW 64500 TOF 8100 TTL: 176 CREW: 2/4 EDNO 7",
	}

	if !parser.QuickCheck(msg.Text) {
		t.Fatal("QuickCheck() = false, want true")
	}

	parsed := parser.Parse(msg)
	if parsed == nil {
		t.Fatal("Parse() returned nil")
	}

	result, ok := parsed.(*Result)
	if !ok {
		t.Fatalf("Parse() returned %T, want *Result", parsed)
	}

	assertStringEqual(t, "msg_type", result.MsgType, "LOADSHEET")
	assertStringEqual(t, "tail", result.Tail, "9H-VJG")
	assertIntEqual(t, "zfw", result.ZFW, 62000)
	assertIntEqual(t, "tow", result.TOW, 70100)
	assertIntEqual(t, "law", result.LAW, 64500)
	assertIntEqual(t, "tof", result.TOF, 8100)
	assertIntEqual(t, "pax", result.PAX, 176)
	assertStringEqual(t, "crew", result.Crew, "2/4")
	assertStringEqual(t, "aircraft_type", result.AircraftType, "A320")
	assertStringEqual(t, "edition", result.Edition, "7")
	assertStringEqual(t, "flight", result.Flight, "EZY1234")
	assertStringEqual(t, "origin", result.Origin, "EGGW")
	assertStringEqual(t, "destination", result.Destination, "EIDW")
	if result.MessageID() != 42 {
		t.Fatalf("message_id = %d, want 42", result.MessageID())
	}
}

func TestLoadsheetNormalisesIATAFlightAndAirports(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		Label: "C1",
		Text:  "LOADSHEET U21234/001 123ABC456 LTN DUB ZFW 62000 TOW 70100 TTL: 176",
	}

	parsed := parser.Parse(msg)
	if parsed == nil {
		t.Fatal("Parse() returned nil")
	}

	result, ok := parsed.(*Result)
	if !ok {
		t.Fatalf("Parse() returned %T, want *Result", parsed)
	}

	assertStringEqual(t, "flight", result.Flight, "EZY1234")
	assertStringEqual(t, "origin", result.Origin, "EGGW")
	assertStringEqual(t, "destination", result.Destination, "EIDW")
}

func TestLoadsheetParsesMultilineFlightAndRouteHeader(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		Label: "C1",
		Text: "/LOADSHEET FINAL   EDN01\r\n" +
			"WFL2093/26  26APR26\r\n" +
			"MAD CLO  ECNOI   3/10\r\n" +
			"DOW136570\r\n" +
			"TTL 24217\r\n" +
			"ZFW 160787 MAX 194000 L\r\n" +
			"TOF 73500\r\n" +
			"TOW 234287 MAX 272000\r\n" +
			"TIF 61514\r\n" +
			"LAW 172773 MAX 207000\r\n" +
			"UNDLD 33213\r\n" +
			"PAX 214       TTL 214\r\n" +
			"91/113/10/0\r\n",
	}

	parsed := parser.Parse(msg)
	if parsed == nil {
		t.Fatal("Parse() returned nil")
	}

	result, ok := parsed.(*Result)
	if !ok {
		t.Fatalf("Parse() returned %T, want *Result", parsed)
	}

	assertStringEqual(t, "edition", result.Edition, "01")
	assertStringEqual(t, "flight", result.Flight, "WFL2093")
	assertStringEqual(t, "origin", result.Origin, "LEMD")
	assertStringEqual(t, "destination", result.Destination, "SKCL")
	assertIntEqual(t, "pax", result.PAX, 214)
	assertIntEqual(t, "zfw", result.ZFW, 160787)
	assertIntEqual(t, "tow", result.TOW, 234287)
	assertIntEqual(t, "law", result.LAW, 172773)
	assertIntEqual(t, "tof", result.TOF, 73500)
}

func TestLoadsheetParsesSlashRouteAndPAXTTLTotal(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		Label: "C1",
		Text: "QUMCTASWY~1DIS01261844LOADSHEET\r\n" +
			"OMANAIR LOADSHEET FINAL\r\n" +
			"EDNO 1\r\n" +
			"WY172/26 26APR26 2044\r\n" +
			"AMS/MCT A4OSH CREW 2/8\r\n" +
			"ZFW  156193 MAX 181436 L  ADJ\r\n" +
			"TOF   45200\r\n" +
			"TOW  201393 MAX 243200    ADJ\r\n" +
			"LAW  166593 MAX 192776    ADJ\r\n" +
			"PAX/23/159 PAX TTL 187\r\n",
	}

	parsed := parser.Parse(msg)
	if parsed == nil {
		t.Fatal("Parse() returned nil")
	}

	result, ok := parsed.(*Result)
	if !ok {
		t.Fatalf("Parse() returned %T, want *Result", parsed)
	}

	assertStringEqual(t, "edition", result.Edition, "1")
	assertStringEqual(t, "flight", result.Flight, "OMA172")
	assertStringEqual(t, "origin", result.Origin, "EHAM")
	assertStringEqual(t, "destination", result.Destination, "OOMS")
	assertStringEqual(t, "crew", result.Crew, "2/8")
	assertIntEqual(t, "pax", result.PAX, 187)
	assertIntEqual(t, "zfw", result.ZFW, 156193)
	assertIntEqual(t, "tow", result.TOW, 201393)
	assertIntEqual(t, "law", result.LAW, 166593)
	assertIntEqual(t, "tof", result.TOF, 45200)
}

func TestLoadsheetParsesTrailingAircraftTypeHeaderAndPAXPlus(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		Label: "C1",
		Text: "QUAUHASEY~1WAB01261858\r\n" +
			"LOADSHEET FINAL   001 2158\r\n" +
			"EY844/26 26APR26 B789\r\n" +
			"SVO AUH A6BNA    2/9\r\n" +
			"ZFW 158151 MAX 181436\r\n" +
			"TOF 73100\r\n" +
			"TOW 231251 MAX 240000\r\n" +
			"LAW 192277 MAX 192776  L\r\n" +
			"PAX/24/253 TTL 281\r\n" +
			"PAX 277 PLUS 4\r\n",
	}

	parsed := parser.Parse(msg)
	if parsed == nil {
		t.Fatal("Parse() returned nil")
	}

	result, ok := parsed.(*Result)
	if !ok {
		t.Fatalf("Parse() returned %T, want *Result", parsed)
	}

	assertStringEqual(t, "flight", result.Flight, "ETD844")
	assertStringEqual(t, "origin", result.Origin, "UUEE")
	assertStringEqual(t, "destination", result.Destination, "OMAA")
	assertIntEqual(t, "pax", result.PAX, 281)
	assertIntEqual(t, "zfw", result.ZFW, 158151)
	assertIntEqual(t, "tow", result.TOW, 231251)
	assertIntEqual(t, "law", result.LAW, 192277)
	assertIntEqual(t, "tof", result.TOF, 73100)
	assertStringEqual(t, "aircraft_type", result.AircraftType, "")
}

func TestLoadsheetParsesCompactAirFranceHeaderAndPAXTTL(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		Label: "C1",
		Text: "QUMUCEPAF~1ECD01\r\n" +
			"LOADSHEET FINAL/2041/01\r\n" +
			"AF0274/26/26APR26CDGHNDF-GSQK4/13\r\n" +
			"CFG 4/60/44/204.........\r\n" +
			"PAX/4/60/33/200 TTL 300.\r\n" +
			"ZFW 214460\r\n" +
			"TOF 344548\r\n" +
			"TOW 251290\r\n",
	}

	parsed := parser.Parse(msg)
	if parsed == nil {
		t.Fatal("Parse() returned nil")
	}

	result, ok := parsed.(*Result)
	if !ok {
		t.Fatalf("Parse() returned %T, want *Result", parsed)
	}

	assertStringEqual(t, "edition", result.Edition, "01")
	assertStringEqual(t, "flight", result.Flight, "AFR274")
	assertStringEqual(t, "origin", result.Origin, "LFPG")
	assertStringEqual(t, "destination", result.Destination, "RJTT")
	assertIntEqual(t, "pax", result.PAX, 300)
	assertIntEqual(t, "zfw", result.ZFW, 214460)
	assertIntEqual(t, "tow", result.TOW, 251290)
}

func TestLoadsheetParsesEmiratesFinalHeaderAndPAXTTL(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		Label: "C1",
		Text: "QUDXBEGEK~1DIS01261628LOADSHEET\r\n" +
			"FINAL01 EK784/26    LOSDXB A6ENN 26APR26\r\n" +
			"CREW   2/15  PAX  7/26/238       TTL 273\r\n" +
			"ZFW  207070  MAX   237682L\r\n" +
			"TOF   58900\r\n" +
			"TOW  265970  MAX   340194\r\n" +
			"LAW  214570  MAX   251290\r\n",
	}

	parsed := parser.Parse(msg)
	if parsed == nil {
		t.Fatal("Parse() returned nil")
	}

	result, ok := parsed.(*Result)
	if !ok {
		t.Fatalf("Parse() returned %T, want *Result", parsed)
	}

	assertStringEqual(t, "edition", result.Edition, "1")
	assertStringEqual(t, "flight", result.Flight, "UAE784")
	assertStringEqual(t, "origin", result.Origin, "DNMM")
	assertStringEqual(t, "destination", result.Destination, "OMDB")
	assertStringEqual(t, "crew", result.Crew, "2/15")
	assertIntEqual(t, "pax", result.PAX, 273)
	assertIntEqual(t, "zfw", result.ZFW, 207070)
	assertIntEqual(t, "tow", result.TOW, 265970)
	assertIntEqual(t, "law", result.LAW, 214570)
	assertIntEqual(t, "tof", result.TOF, 58900)
}

func TestLoadsheetParsesBARouteFromSECField(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		Label: "C1",
		Text: ":PAD12.LHRLDBA.ZFWAZF/GYMMU/BA0085\r\n" +
			"- PAX/274 INCLUDING 002 INF\r\n" +
			"- CGO/003925\r\n" +
			"- ZFW/181123\r\n" +
			"- CRW/03/10\r\n" +
			"- FWT/068024\r\n" +
			"- OWT/215243\r\n" +
			"- TTL/033082\r\n" +
			"- TOW/248325\r\n" +
			"- DEP/1620\r\n" +
			"- SEC/LHR-YVR\r\n" +
			"FLT STATUS:CLOSED\r\n",
	}

	parsed := parser.Parse(msg)
	if parsed == nil {
		t.Fatal("Parse() returned nil")
	}

	result, ok := parsed.(*Result)
	if !ok {
		t.Fatalf("Parse() returned %T, want *Result", parsed)
	}

	assertStringEqual(t, "flight", result.Flight, "BAW85")
	assertStringEqual(t, "origin", result.Origin, "EGLL")
	assertStringEqual(t, "destination", result.Destination, "CYVR")
	assertStringEqual(t, "crew", result.Crew, "03/10")
	assertIntEqual(t, "pax", result.PAX, 274)
	assertIntEqual(t, "zfw", result.ZFW, 181123)
	assertIntEqual(t, "tow", result.TOW, 248325)
	assertIntEqual(t, "tof", result.TOF, 68024)
}

func TestLoadsheetParsesIberiaStyleTableHeader(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		Label: "C1",
		Text: "4WAB11/.\r\n" +
			"L O A D S H E E T           CHECKED      APPROVED         EDNO\r\n" +
			"ALL WEIGHTS IN KG      LIC   3665//M.D.R                   01\r\n" +
			".\r\n" +
			"FROM/TO FLIGHT       A/C REG VERSION      CREW    DATE    TIME\r\n" +
			"BCN SCL LL 2605      ECNNH   C42Y269      3/08    27APR26 2356\r\n" +
			"WEIGHT           DISTRIBUTION\r\n" +
			"PASSENGER/CABIN BAG      18220 110/119/6/1     TTL 236 CAB 0\r\n" +
			"PAX 39/196      SOC\r\n" +
			"ZERO FUEL WEIGHT ACTUAL 145070 MAX 166000      ADJ\r\n" +
			"TAKE OFF FUEL            86700\r\n" +
			"TAKE OFF WEIGHT  ACTUAL 231770 MAX 242000  L   ADJ\r\n" +
			"TRIP FUEL                79871\r\n",
	}

	parsed := parser.Parse(msg)
	if parsed == nil {
		t.Fatal("Parse() returned nil")
	}

	result, ok := parsed.(*Result)
	if !ok {
		t.Fatalf("Parse() returned %T, want *Result", parsed)
	}

	assertStringEqual(t, "edition", result.Edition, "01")
	assertStringEqual(t, "flight", result.Flight, "LVL2605")
	assertStringEqual(t, "origin", result.Origin, "LEBL")
	assertStringEqual(t, "destination", result.Destination, "SCEL")
	assertIntEqual(t, "pax", result.PAX, 236)
	assertIntEqual(t, "zfw", result.ZFW, 145070)
	assertIntEqual(t, "tow", result.TOW, 231770)
	assertIntEqual(t, "tof", result.TOF, 86700)
	assertStringEqual(t, "crew", result.Crew, "")
}

func TestLoadsheetParsesRunMruRouteFromMultilineHeader(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		Label: "C1",
		Text: ".ORYCASS 290726\r\n" +
			"AGM\r\n" +
			"AN F-HHUG/MA 279I\r\n" +
			"-  LOADSHEET FINAL 1123 EDNO1\r\n" +
			"SS632/28      29APR26\r\n" +
			"RUN MRU F-HHUG   2/8\r\n" +
			"ZFW 147528 MAX 181000\r\n" +
			"TOF 14800\r\n" +
			"TOW 162328 MAX 240000\r\n" +
			"TIF 2600\r\n" +
			"LAW 159728 MAX 191000  L\r\n" +
			"UNDLD 31272\r\n" +
			"PAX/13/6/101 TTL 123\r\n" +
			"PAX 120 PLUS 3\r\n",
	}

	parsed := parser.Parse(msg)
	if parsed == nil {
		t.Fatal("Parse() returned nil")
	}

	result, ok := parsed.(*Result)
	if !ok {
		t.Fatalf("Parse() returned %T, want *Result", parsed)
	}

	assertStringEqual(t, "edition", result.Edition, "1")
	assertStringEqual(t, "flight", result.Flight, "CRL632")
	assertStringEqual(t, "origin", result.Origin, "FMEE")
	assertStringEqual(t, "destination", result.Destination, "FIMP")
	assertStringEqual(t, "crew", result.Crew, "2/8")
	assertIntEqual(t, "pax", result.PAX, 123)
	assertIntEqual(t, "zfw", result.ZFW, 147528)
	assertIntEqual(t, "tow", result.TOW, 162328)
	assertIntEqual(t, "law", result.LAW, 159728)
	assertIntEqual(t, "tof", result.TOF, 14800)
}

func TestLoadsheetParsesChinaEasternFlightRouteHeader(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		Label: "C1",
		Text: "CHINA EASTERN AIRLINES\r\n" +
			"LOADSHEET           EDNO 02\r\n" +
			"ALL WEIGHTS IN KG\r\n" +
			"DATE:04APR26 TIME:2216\r\n" +
			"ISSUE:AIRPORT_FREEMAN579\r\n" +
			"FLIGHT:MU2076/04APR26   SVOPKX B6083\r\n" +
			"VERSION:J38Y262    CREW:4/14/0   CAB:0\r\n" +
			"WEIGHT    DISTRIBUTION\r\n" +
			"MAX TRAFFIC PAYLOAD      48067\r\n" +
			"DOW       126933  DOI:      82.10\r\n" +
			"PAYLOAD    25991 BLKD 2/15\r\n" +
			"ZFW       152924  MACZFW: 28.06\r\n" +
			"MZFW   175000  L\r\n" +
			"TOF        48842\r\n" +
			"TOW       201766  MACTOW: 27.83\r\n" +
			"MTOW   233000\r\n" +
			"TRIP FUEL  37873\r\n" +
			"LDW       163893  MACLAW: 26.64\r\n" +
			"MLDW   187000\r\n" +
			"STAB TO    3.6\r\n" +
			"PASSENGER  21058 272/3/0\r\n" +
			"SEATING   35/240             TTL:275\r\n" +
			"0A/35 0B/127 0C/113\r\n",
	}

	parsed := parser.Parse(msg)
	if parsed == nil {
		t.Fatal("Parse() returned nil")
	}

	result, ok := parsed.(*Result)
	if !ok {
		t.Fatalf("Parse() returned %T, want *Result", parsed)
	}

	assertStringEqual(t, "edition", result.Edition, "02")
	assertStringEqual(t, "flight", result.Flight, "CES2076")
	assertStringEqual(t, "origin", result.Origin, "UUEE")
	assertStringEqual(t, "destination", result.Destination, "ZBAD")
	assertStringEqual(t, "crew", result.Crew, "4/14/0")
	assertIntEqual(t, "pax", result.PAX, 275)
	assertIntEqual(t, "zfw", result.ZFW, 152924)
	assertIntEqual(t, "tow", result.TOW, 201766)
	assertIntEqual(t, "law", result.LAW, 163893)
	assertIntEqual(t, "tof", result.TOF, 48842)
}

func TestLoadsheetParsesInlineCompactRouteHeader(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		Label: "C1",
		Text: "QUASRV1UU~1WAB01041847\r\n" +
			"LOADSHEET FINAL 1847 EDN02\r\n" +
			"UU969/04 RUNCDG F-OLRD 3/12\r\n" +
			"TLD  47370 UNDLD 21036\r\n" +
			"DOW 171544 DOI 47.5\r\n" +
			"ZFW 218914 M239950  L\r\n" +
			"TOF 103200\r\n" +
			"TOW 322114 M344730\r\n" +
			"TIF  94900\r\n" +
			"LAW 227214 M251290\r\n" +
			"PAX 424 195/212/8/9\r\n" +
			"9/29/377 A30.B128.C140.D117\r\n" +
			"CGO 1/1735 2/4246 3/6911 4/2018\r\n" +
			"MACTOW  28.8   MACZFW  29.5\r\n",
	}

	parsed := parser.Parse(msg)
	if parsed == nil {
		t.Fatal("Parse() returned nil")
	}

	result, ok := parsed.(*Result)
	if !ok {
		t.Fatalf("Parse() returned %T, want *Result", parsed)
	}

	assertStringEqual(t, "edition", result.Edition, "02")
	assertStringEqual(t, "flight", result.Flight, "REU969")
	assertStringEqual(t, "origin", result.Origin, "FMEE")
	assertStringEqual(t, "destination", result.Destination, "LFPG")
	assertIntEqual(t, "zfw", result.ZFW, 218914)
	assertIntEqual(t, "tow", result.TOW, 322114)
	assertIntEqual(t, "law", result.LAW, 227214)
	assertIntEqual(t, "tof", result.TOF, 103200)
	assertStringEqual(t, "crew", result.Crew, "")
}

func TestLoadsheetParsesDateHeaderRouteOnNextLine(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		Label: "C1",
		Text: ".ADDAMET 010727\r\n" +
			"AGM\r\n" +
			"AN ET-BBG/MA 105I\r\n" +
			"-  QUADDAMET~1CGM01010727\r\n" +
			"LOADSHEET FINAL 1024\r\n" +
			"CHECKED      APPROVED\r\n" +
			"34641          ..........\r\n" +
			"ET863/01APR26 01APR26 EDNO-2\r\n" +
			"ADD LUN ETANQ C28Y287    3/10\r\n" +
			"TRAFFIC LOAD  36501\r\n" +
			"DRY OPERATING WEIGHT  156598\r\n" +
			"ZFW  193099 MAX 209106\r\n" +
			"TOF   41605\r\n" +
			"TOW  234704 MAX 235605 L\r\n" +
			"TIF   24245\r\n" +
			"LAW  210459 MAX 223167\r\n" +
			"UNDLD   901\r\n" +
			"MACZFW   26.34\r\n" +
			"MACTOW   27.78\r\n" +
			"DOI  36.62\r\n" +
			"PAX/17/229 TTL 246\r\n",
	}

	parsed := parser.Parse(msg)
	if parsed == nil {
		t.Fatal("Parse() returned nil")
	}

	result, ok := parsed.(*Result)
	if !ok {
		t.Fatalf("Parse() returned %T, want *Result", parsed)
	}

	assertStringEqual(t, "edition", result.Edition, "2")
	assertStringEqual(t, "flight", result.Flight, "ETH863")
	assertStringEqual(t, "origin", result.Origin, "HAAB")
	assertStringEqual(t, "destination", result.Destination, "FLLS")
	assertStringEqual(t, "crew", result.Crew, "3/10")
	assertIntEqual(t, "pax", result.PAX, 246)
	assertIntEqual(t, "zfw", result.ZFW, 193099)
	assertIntEqual(t, "tow", result.TOW, 234704)
	assertIntEqual(t, "law", result.LAW, 210459)
	assertIntEqual(t, "tof", result.TOF, 41605)
}

func TestLoadsheetParsesTableCombinedFlightToken(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		Label: "C1",
		Text: "DATE    TIME\r\n" +
			"MAD CLO WFL2093      ECNZF   Y432         2/10    01APR26 1245\r\n" +
			"WEIGHT           DISTRIBUTION\r\n" +
			"LOAD IN COMPARTMENTS      7691 2/710 3/3130 4/3789 5/62\r\n" +
			".\r\n" +
			"PASSENGER/CABIN BAG      17082 109/101/12/2    TTL 224 CAB 0\r\n" +
			"PAX 222         SOC\r\n" +
			"TOTAL TRAFFIC LOAD       24773\r\n" +
			"DRY OPERATING WEIGHT    135525\r\n" +
			"ZERO FUEL WEIGHT ACTUAL 160298 MAX 194000      ADJ\r\n" +
			"TAKE OFF FUEL            73400\r\n" +
			"TAKE OFF WEIGHT  ACTUAL 233698 MAX 272000      ADJ\r\n",
	}

	parsed := parser.Parse(msg)
	if parsed == nil {
		t.Fatal("Parse() returned nil")
	}

	result, ok := parsed.(*Result)
	if !ok {
		t.Fatalf("Parse() returned %T, want *Result", parsed)
	}

	assertStringEqual(t, "flight", result.Flight, "WFL2093")
	assertStringEqual(t, "origin", result.Origin, "LEMD")
	assertStringEqual(t, "destination", result.Destination, "SKCL")
	assertIntEqual(t, "pax", result.PAX, 224)
	assertIntEqual(t, "zfw", result.ZFW, 160298)
	assertIntEqual(t, "tow", result.TOW, 233698)
	assertIntEqual(t, "tof", result.TOF, 73400)
}

func TestLoadsheetParsesEtihadLeadingZeroFlight(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		Label: "C1",
		Text: "QUAUHASEY~1WAB01192148\r\n" +
			"LOADSHEET FINAL   001 0148\r\n" +
			"EY031/19 20MAR26 B789\r\n" +
			"AUH CDG A6BLA    3/12\r\n" +
			"ZFW 170864 MAX 181436  L\r\n" +
			"TOF 52540\r\n" +
			"TOW 223404 MAX 250836\r\n" +
			"TIF 42919\r\n" +
			"LAW 180485 MAX 192776\r\n" +
			"UNDLD 10572\r\n" +
			"PAX/8/28/189 TTL 226\r\n" +
			"PAX 225 PLUS 1\r\n" +
			"BI      430.2\r\n" +
			"DOI     444.0\r\n" +
			"LIZFW   696.7\r\n" +
			"LITOW   717.8\r\n" +
			"MACZFW   29.3\r\n" +
			"MACTOW   27.9\r\n" +
			"A16 B20 C62 D127\r\n" +
			"SEATROW TRIM\r\n" +
			"CDG FRE 15852 POS    0 BAG 5520 TRA    0\r\n" +
			"SI**************************************\r\n" +
			"LOAD IN CPTS 0/20 1/4559 2/4875 3/7549\r\n" +
			"4/6028 5/100\r\n" +
			"B787-9H3C\r\n" +
			"BW  125078\r\n" +
			"PANTRY CODE  C\r\n" +
			"SERVICE WEIGHT ADJUSTMENT WEIGHT/INDEX\r\n" +
			"DEDUCTIONSNILADD\r\n" +
			"CDG POTABLE WATER\r\n" +
			"454    19.2\r\n" +
			"CDG BLANKETS\r\n" +
			"0      0.0\r\n" +
			"CDG HEADSETS\r\n" +
			"0      0.0\r\n" +
			"DOW 130703\r\n" +
			"LIZFW LIMITS +  338.5/+  812.9\r\n" +
			"NOW: +  696.7\r\n" +
			"LITOW LIMITS +  261.0/+  916.4\r\n" +
			"NOW: +  717.8\r\n" +
			"PIC 13455\r\n" +
			"PREPARED BY KAMESH/KUMAR 971 280 53065\r\n" +
			"NOTOC: YES\r\n",
	}

	parsed := parser.Parse(msg)
	if parsed == nil {
		t.Fatal("Parse() returned nil")
	}

	result, ok := parsed.(*Result)
	if !ok {
		t.Fatalf("Parse() returned %T, want *Result", parsed)
	}

	assertStringEqual(t, "flight", result.Flight, "ETD31")
	assertStringEqual(t, "origin", result.Origin, "OMAA")
	assertStringEqual(t, "destination", result.Destination, "LFPG")
	assertStringEqual(t, "crew", result.Crew, "3/12")
	assertIntEqual(t, "pax", result.PAX, 226)
	assertIntEqual(t, "zfw", result.ZFW, 170864)
	assertIntEqual(t, "tow", result.TOW, 223404)
	assertIntEqual(t, "law", result.LAW, 180485)
	assertIntEqual(t, "tof", result.TOF, 52540)
}

func TestLoadsheetQuickCheckAcceptsSpacedKeyword(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		Label: "H1",
		Text:  "L O A D S H E E T FINAL EDNO 1 ZFW 60158 TOW 70000 TTL: 180",
	}

	if !parser.QuickCheck(msg.Text) {
		t.Fatal("QuickCheck() = false, want true for spaced LOADSHEET keyword")
	}

	parsed := parser.Parse(msg)
	if parsed == nil {
		t.Fatal("Parse() returned nil for spaced LOADSHEET keyword")
	}

	result, ok := parsed.(*Result)
	if !ok {
		t.Fatalf("Parse() returned %T, want *Result", parsed)
	}
	assertStringEqual(t, "msg_type", result.MsgType, "LOADSHEET")
	assertIntEqual(t, "zfw", result.ZFW, 60158)
	assertIntEqual(t, "tow", result.TOW, 70000)
	assertIntEqual(t, "pax", result.PAX, 180)
}

func TestLoadsheetLabelsAreGlobal(t *testing.T) {
	parser := &Parser{}
	if labels := parser.Labels(); len(labels) != 0 {
		t.Fatalf("Labels() = %v, want global parser with no labels", labels)
	}
}

func TestLoadsheetRejectsWithoutUsefulData(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{
		Label: "C1",
		Text:  "LOADSHEET HEADER ONLY",
	}

	if parsed := parser.Parse(msg); parsed != nil {
		t.Fatalf("Parse() returned %T, want nil", parsed)
	}
}

func assertStringEqual(t *testing.T, field, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %q, want %q", field, got, want)
	}
}

func assertIntEqual(t *testing.T, field string, got, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %d, want %d", field, got, want)
	}
}
