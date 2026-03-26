package h1

import (
	"encoding/json"
	"testing"

	"acars_parser/internal/acars"
)

func TestParseWaypointCoords(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantLat float64
		wantLon float64
	}{
		{
			name:    "VELOX coordinates",
			input:   "N33490E034050",
			wantLat: 33.816666666666666, // 33° 49.0' N
			wantLon: 34.083333333333336, // 034° 05.0' E
		},
		{
			name:    "MUVIN coordinates",
			input:   "N31490E035327",
			wantLat: 31.816666666666666, // 31° 49.0' N
			wantLon: 35.545,             // 035° 32.7' E
		},
		{
			name:    "Western hemisphere",
			input:   "N37312W102468",
			wantLat: 37.52,   // 37° 31.2' N
			wantLon: -102.78, // 102° 46.8' W
		},
		{
			name:    "Southern hemisphere",
			input:   "S33520E151180",
			wantLat: -33.866666666666667, // 33° 52.0' S
			wantLon: 151.3,               // 151° 18.0' E
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLat, gotLon := parseWaypointCoords(tt.input)

			// Allow small floating point tolerance.
			if abs(gotLat-tt.wantLat) > 0.01 {
				t.Errorf("parseWaypointCoords(%q) lat = %v, want %v", tt.input, gotLat, tt.wantLat)
			}
			if abs(gotLon-tt.wantLon) > 0.01 {
				t.Errorf("parseWaypointCoords(%q) lon = %v, want %v", tt.input, gotLon, tt.wantLon)
			}
		})
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func TestFPNParseWithCoordinates(t *testing.T) {
	testText := `FPN/FNRJA111/RP:DA:OJAI:AA:EGLL:F:MUVIN,N31490E035327.L53..TAPUZ,N32020E034314.W13..VELOX,N33490E034050.N71..DESPO,N34269E034229`

	msg := &acars.Message{
		ID:    1,
		Label: "H1",
		Text:  testText,
	}

	parser := &FPNParser{}
	result := parser.Parse(msg)
	if result == nil {
		t.Fatal("Failed to parse FPN message")
	}

	fpn, ok := result.(*FPNResult)
	if !ok {
		t.Fatal("Result is not FPNResult")
	}
	if fpn.MsgType != "FPN" {
		t.Fatalf("MsgType = %q, want %q", fpn.MsgType, "FPN")
	}

	// Check that we got waypoints with coordinates.
	if len(fpn.Waypoints) == 0 {
		t.Fatal("No waypoints parsed")
	}

	// Find VELOX and check its coordinates.
	var velox *RouteWaypoint
	for i := range fpn.Waypoints {
		if fpn.Waypoints[i].Name == "VELOX" {
			velox = &fpn.Waypoints[i]
			break
		}
	}

	if velox == nil {
		t.Fatal("VELOX waypoint not found")
	}

	// VELOX should be at N33° 49.0' E034° 05.0'
	expectedLat := 33.816666666666666
	expectedLon := 34.083333333333336

	if abs(velox.Latitude-expectedLat) > 0.01 {
		t.Errorf("VELOX latitude = %v, want %v", velox.Latitude, expectedLat)
	}
	if abs(velox.Longitude-expectedLon) > 0.01 {
		t.Errorf("VELOX longitude = %v, want %v", velox.Longitude, expectedLon)
	}

	// Print full result for visual inspection.
	jsonBytes, _ := json.MarshalIndent(fpn, "", "  ")
	t.Logf("Parsed result:\n%s", string(jsonBytes))
}

func TestDetectTruncation(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		route string
		want  bool
	}{
		{
			name:  "normal complete message",
			text:  "FPN/SN123:DA:KSFO:AA:KLAX:F:WAYP1..WAYP2",
			route: "WAYP1..WAYP2",
			want:  false,
		},
		{
			name:  "multi-part without terminator",
			text:  "FPN/SN123#M1:DA:KSFO:AA:KLAX",
			route: "",
			want:  true,
		},
		{
			name:  "multi-part with terminator",
			text:  "FPN/SN123#M1:DA:KSFO#MD:AA:KLAX",
			route: "",
			want:  false,
		},
		{
			name:  "all waypoints have coords",
			text:  "FPN:DA:KSFO:AA:KLAX:F:WPT1,N33490E034050..WPT2,N34000E035000",
			route: "WPT1,N33490E034050..WPT2,N34000E035000",
			want:  false,
		},
		{
			name:  "ends with colon",
			text:  "FPN:DA:KSFO:AA:KLAX:",
			route: "",
			want:  true,
		},
		{
			name:  "ends with comma",
			text:  "FPN:DA:KSFO:AA:KLAX:F:WAYP1,",
			route: "WAYP1,",
			want:  true,
		},
		{
			name:  "ends with double period",
			text:  "FPN:DA:KSFO:AA:KLAX:F:WAYP1..",
			route: "WAYP1..",
			want:  true,
		},
		{
			name:  "incomplete coordinate after comma",
			text:  "FPN:DA:KSFO:AA:KLAX:F:WAYP1,N334",
			route: "WAYP1,N334",
			want:  true,
		},
		{
			name:  "complete coordinate after comma",
			text:  "FPN:DA:KSFO:AA:KLAX:F:WAYP1,N33490E034050",
			route: "WAYP1,N33490E034050",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectTruncation(tt.text, tt.route)
			if got != tt.want {
				t.Errorf("detectTruncation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestH1PosParse_WindSpeedThreeDigits(t *testing.T) {
	// Example where wind is encoded as DDDSSS (3-digit speed with leading zero): 255044 => 255° / 44 kts.
	msg := &acars.Message{
		ID:        1,
		Label:     "H1",
		Text:      "POSN43451E017323,VRANA,032901,370,PETAK,034717,PINDO,M46,255044,141C21C",
		Timestamp: "2025-10-06T04:45:10.137514Z",
		Tail:      "TEST",
	}

	parser := &H1PosParser{}
	res := parser.Parse(msg)
	if res == nil {
		t.Fatalf("expected parse result, got nil")
	}

	pos, ok := res.(*H1PosResult)
	if !ok {
		t.Fatalf("expected *H1PosResult, got %T", res)
	}

	if pos.FlightLevel != 370 {
		t.Fatalf("flight level = %d, want 370", pos.FlightLevel)
	}
	if pos.Temperature != -46 {
		t.Fatalf("temperature = %d, want -46", pos.Temperature)
	}
	if pos.WindDir != 255 {
		t.Fatalf("wind_dir = %d, want 255", pos.WindDir)
	}
	if pos.WindSpeed != 44 {
		t.Fatalf("wind_speed = %d, want 44", pos.WindSpeed)
	}
	if pos.MsgType != "POS" {
		t.Fatalf("MsgType = %q, want %q", pos.MsgType, "POS")
	}
}

func TestH1PosParse_SimpleWaypoints(t *testing.T) {
	// Test message with simple letter-only waypoints
	msg := &acars.Message{
		ID:        1,
		Label:     "H1",
		Text:      "POSN43596E017223,REKTI,121745,350,RINAV,123631,BONZI,M61,334080,198D8E5",
		Timestamp: "2025-10-06T12:00:00Z",
		Tail:      "TEST1",
	}

	parser := &H1PosParser{}
	res := parser.Parse(msg)
	if res == nil {
		t.Fatalf("expected parse result for simple waypoints, got nil")
	}

	pos, ok := res.(*H1PosResult)
	if !ok {
		t.Fatalf("expected *H1PosResult, got %T", res)
	}

	if pos.CurrentWaypoint != "REKTI" {
		t.Errorf("current_waypoint = %s, want REKTI", pos.CurrentWaypoint)
	}
	if pos.NextWaypoint != "RINAV" {
		t.Errorf("next_waypoint = %s, want RINAV", pos.NextWaypoint)
	}
	if pos.ThirdWaypoint != "BONZI" {
		t.Errorf("third_waypoint = %s, want BONZI", pos.ThirdWaypoint)
	}
	if pos.FlightLevel != 350 {
		t.Errorf("flight_level = %d, want 350", pos.FlightLevel)
	}
	if pos.MsgType != "POS" {
		t.Fatalf("MsgType = %q, want %q", pos.MsgType, "POS")
	}
}

func TestPWIParseEmitsMsgType(t *testing.T) {
	msg := &acars.Message{
		ID:        77,
		Label:     "H1",
		Text:      "PWI/WD360,TITIL,267041,360M58.RELGE,248035,360M58.ARLIF,241034,360M58.GASBI,209032,360M56.ABTEK,203033,360M55.LEKBA,201034,360M54.LEYLA,199033,360M52.ADEKI,198030,360M51.BADIR,196026,360M50.LAGAS,196020,360M49.AGISO",
		Timestamp: "2026-03-20T00:00:00Z",
		Tail:      "TESTPWI",
	}

	parser := &PWIParser{}
	res := parser.Parse(msg)
	if res == nil {
		t.Fatalf("expected parse result, got nil")
	}

	pwi, ok := res.(*PWIResult)
	if !ok {
		t.Fatalf("expected *PWIResult, got %T", res)
	}

	if pwi.MsgType != "PWI" {
		t.Fatalf("MsgType = %q, want %q", pwi.MsgType, "PWI")
	}
	if len(pwi.RouteWinds) != 1 {
		t.Fatalf("len(RouteWinds) = %d, want 1", len(pwi.RouteWinds))
	}
	if pwi.RouteWinds[0].FlightLevel != 360 {
		t.Fatalf("RouteWinds[0].FlightLevel = %d, want 360", pwi.RouteWinds[0].FlightLevel)
	}
	if len(pwi.RouteWinds[0].Waypoints) == 0 {
		t.Fatal("expected parsed route wind waypoints, got none")
	}
}

func TestPWIParseCoordinateLeadingWaypointsWithoutTemperature(t *testing.T) {
	msg := &acars.Message{
		ID:        78,
		Label:     "H1",
		Text:      "PWI/WD390,N24012E078331,292064.N24425E077165,284075.IBANI,284073.NIKOT,274080.NOKOK,274080.ISKEN,270080.LATOS,268090.VIKIT,264095.RK,262094.HILAL,260079.ZB,260081.REGET,258082.SAJAN,258082.LAKRA,258082.HANGU,262082.PS,262082.ATROL,254102.GERRY,254102.MOTMO,254102.FIRUZ,254102.VAJEN,252108.DAROW,248106.BUTRA,248106.USETU,246108.OGNOB,246107.TOLIB,246107.BUPOR,244105/WD340,N24012E078331,296067,340M41.N24425E077165,292070,340M42.IBANI,286072,340M42.NIKOT,282081,340M43.NOKOK,282081,340M43.ISKEN,2780",
		Timestamp: "2026-03-20T00:00:00Z",
		Tail:      "TESTPWI1",
	}

	parser := &PWIParser{}
	res := parser.Parse(msg)
	if res == nil {
		t.Fatal("expected parse result, got nil")
	}

	pwi, ok := res.(*PWIResult)
	if !ok {
		t.Fatalf("expected *PWIResult, got %T", res)
	}
	if len(pwi.RouteWinds) != 2 {
		t.Fatalf("len(RouteWinds) = %d, want 2", len(pwi.RouteWinds))
	}
	if len(pwi.RouteWinds[0].Waypoints) < 2 {
		t.Fatalf("expected first route wind layer to have coordinate waypoints, got %d", len(pwi.RouteWinds[0].Waypoints))
	}

	first := pwi.RouteWinds[0].Waypoints[0]
	if first.Waypoint != "N24012E078331" {
		t.Fatalf("first waypoint = %q, want %q", first.Waypoint, "N24012E078331")
	}
	if abs(first.Latitude-24.02) > 0.01 {
		t.Fatalf("first latitude = %v, want about 24.02", first.Latitude)
	}
	if abs(first.Longitude-78.55166666666666) > 0.01 {
		t.Fatalf("first longitude = %v, want about 78.5517", first.Longitude)
	}
	if first.WindDir != 292 || first.WindSpeed != 64 {
		t.Fatalf("first wind = %d/%d, want 292/64", first.WindDir, first.WindSpeed)
	}

	second := pwi.RouteWinds[0].Waypoints[1]
	if second.Waypoint != "N24425E077165" {
		t.Fatalf("second waypoint = %q, want %q", second.Waypoint, "N24425E077165")
	}
	if abs(second.Latitude-24.708333333333332) > 0.01 {
		t.Fatalf("second latitude = %v, want about 24.7083", second.Latitude)
	}
	if abs(second.Longitude-77.275) > 0.01 {
		t.Fatalf("second longitude = %v, want about 77.275", second.Longitude)
	}
	if second.WindDir != 284 || second.WindSpeed != 75 {
		t.Fatalf("second wind = %d/%d, want 284/75", second.WindDir, second.WindSpeed)
	}
}

func TestPWIParseCoordinateLeadingWaypointsWithTemperature(t *testing.T) {
	msg := &acars.Message{
		ID:        79,
		Label:     "H1",
		Text:      "PWI/WD410,S15000E090000,080012,410M58.S20000E095000,021025,410M57.S25000E101000,333012,410M56.S28000E105000,247028,410M55/WD390,S15000E090000,074010,390M53.S20000E095000,010019,390M51.S25000E101000,312010,390M51.S28000E105000,246031,390M50/WD370,S15000E090000,056011,370M47.S20000E095000,015013,370M45.S25000E101000,304006,370M45.S28000E105000,257030,370M45/WD350,S15000E090000,040013,350M42.S20000E095000,028013,350M40.S25000E101000,313004,350M40.S28000E105000,266031,350M40/DD100161037.200176050.31",
		Timestamp: "2026-03-20T00:00:00Z",
		Tail:      "TESTPWI2",
	}

	parser := &PWIParser{}
	res := parser.Parse(msg)
	if res == nil {
		t.Fatal("expected parse result, got nil")
	}

	pwi, ok := res.(*PWIResult)
	if !ok {
		t.Fatalf("expected *PWIResult, got %T", res)
	}
	if len(pwi.RouteWinds) != 4 {
		t.Fatalf("len(RouteWinds) = %d, want 4", len(pwi.RouteWinds))
	}
	if len(pwi.DescentWinds) == 0 {
		t.Fatal("expected descent winds, got none")
	}

	first := pwi.RouteWinds[0].Waypoints[0]
	if first.Waypoint != "S15000E090000" {
		t.Fatalf("first waypoint = %q, want %q", first.Waypoint, "S15000E090000")
	}
	if abs(first.Latitude-(-15.0)) > 0.01 {
		t.Fatalf("first latitude = %v, want about -15.0", first.Latitude)
	}
	if abs(first.Longitude-90.0) > 0.01 {
		t.Fatalf("first longitude = %v, want about 90.0", first.Longitude)
	}
	if first.WindDir != 80 || first.WindSpeed != 12 {
		t.Fatalf("first wind = %d/%d, want 80/12", first.WindDir, first.WindSpeed)
	}
	if first.Temperature != -58 {
		t.Fatalf("first temperature = %d, want -58", first.Temperature)
	}
}

func TestH1PosParse_WaypointsWithDashAndNumbers(t *testing.T) {
	// Test message with waypoints containing dashes and numbers (BUD-01, IRL-02, OPT-03)
	msg := &acars.Message{
		ID:        2,
		Label:     "H1",
		Text:      "POSN45574E023238,BUD-01,123151,310,IRL-02,123604,OPT-03,M50,34178,2043/TS123151,1201264BF7",
		Timestamp: "2025-10-06T12:30:00Z",
		Tail:      "TEST2",
	}

	parser := &H1PosParser{}
	res := parser.Parse(msg)
	if res == nil {
		t.Fatalf("expected parse result for dash/number waypoints, got nil")
	}

	pos, ok := res.(*H1PosResult)
	if !ok {
		t.Fatalf("expected *H1PosResult, got %T", res)
	}

	if pos.CurrentWaypoint != "BUD-01" {
		t.Errorf("current_waypoint = %s, want BUD-01", pos.CurrentWaypoint)
	}
	if pos.NextWaypoint != "IRL-02" {
		t.Errorf("next_waypoint = %s, want IRL-02", pos.NextWaypoint)
	}
	if pos.ThirdWaypoint != "OPT-03" {
		t.Errorf("third_waypoint = %s, want OPT-03", pos.ThirdWaypoint)
	}
	if pos.FlightLevel != 310 {
		t.Errorf("flight_level = %d, want 310", pos.FlightLevel)
	}
	if pos.Temperature != -50 {
		t.Errorf("temperature = %d, want -50", pos.Temperature)
	}
	if pos.MsgType != "POS" {
		t.Fatalf("MsgType = %q, want %q", pos.MsgType, "POS")
	}

	// Print full result for verification
	jsonBytes, _ := json.MarshalIndent(pos, "", "  ")
	t.Logf("Parsed position with dash/number waypoints:\n%s", string(jsonBytes))
}

func TestH1PosParse_EmptyWaypoint(t *testing.T) {
	// Test message with empty current waypoint (double comma) and coordinates in next_wpt field
	msg := &acars.Message{
		ID:        3,
		Label:     "H1",
		Text:      "POSN46283E022271,,140610,370,N44052E026499,143250,ARTAT,M56,34863,1040,849/TS140610,120126D038",
		Timestamp: "2025-10-06T14:00:00Z",
		Tail:      "TEST3",
	}

	parser := &H1PosParser{}
	res := parser.Parse(msg)
	if res == nil {
		t.Fatalf("expected parse result for empty waypoint, got nil")
	}

	pos, ok := res.(*H1PosResult)
	if !ok {
		t.Fatalf("expected *H1PosResult, got %T", res)
	}

	// Current waypoint should be empty
	if pos.CurrentWaypoint != "" {
		t.Errorf("current_waypoint = %s, want empty string", pos.CurrentWaypoint)
	}
	// Next waypoint contains coordinates
	if pos.NextWaypoint != "N44052E026499" {
		t.Errorf("next_waypoint = %s, want N44052E026499", pos.NextWaypoint)
	}
	if pos.ThirdWaypoint != "ARTAT" {
		t.Errorf("third_waypoint = %s, want ARTAT", pos.ThirdWaypoint)
	}
	if pos.FlightLevel != 370 {
		t.Errorf("flight_level = %d, want 370", pos.FlightLevel)
	}
	if pos.Temperature != -56 {
		t.Errorf("temperature = %d, want -56", pos.Temperature)
	}
	if pos.WindDir != 348 {
		t.Errorf("wind_dir = %d, want 348", pos.WindDir)
	}
	if pos.WindSpeed != 63 {
		t.Errorf("wind_speed = %d, want 63", pos.WindSpeed)
	}
	if pos.MsgType != "POS" {
		t.Fatalf("MsgType = %q, want %q", pos.MsgType, "POS")
	}

	// Print full result for verification
	jsonBytes, _ := json.MarshalIndent(pos, "", "  ")
	t.Logf("Parsed position with empty waypoint:\n%s", string(jsonBytes))
}
