package ilnge7x

import (
	"testing"

	"acars_parser/internal/acars"
)

func TestParseILNGE7XSummary(t *testing.T) {
	parser := &Parser{}
	tests := []struct {
		name            string
		text            string
		wantTail        string
		wantFlight      string
		wantTakeOffDate string
		wantTakeOffTime string
		wantOrigin      string
		wantDestination string
		wantRoute       string
	}{
		{
			name:            "tail with separator digit two",
			text:            "/ILNGE7X.CLB006TK TC-LLT2THY69W 260304054817LTFMKIAD+056060295877174000013000022447M85P04B205B2050406010B2122M32P08 2462M81P05 2462M82P05 -00434646+04153090+02880721007NQVK1UNANS01SVRVL8:::0FF0HGG",
			wantTail:        "TC-LLT",
			wantFlight:      "THY69W",
			wantTakeOffDate: "26-03-04",
			wantTakeOffTime: "05:48:17",
			wantOrigin:      "LTFM",
			wantDestination: "KIAD",
			wantRoute:       "LTFM-KIAD",
		},
		{
			name:            "tail with four suffix characters",
			text:            "/ILNGE7X.CLB006VNVN-A8642HVN36 260304131843EDDFVVNB+022060295655674000013000022447M85P02B205B205040605092122M32P08 2462M81P05 2462M82P05 +11747475+04983830+00888276007NQVK1UNANS01SVRVL8:::0FF0HGG00000481P810BHA01L01J02M02J00V00U0090000018211210Q000299R9BM9HU9GP9F99CT97F9A59AO1TJ01I0310V70S20P60P40P50LS0MQ0M602PR0V90DP31OJ1GQR0V60F40EG111C03M0VQ96F95Q0AFV0S21R41FG1HCV14S103GM1O80RO6DD1K16LJ0N80QR10R001N1GI1IJ0V901F12G18S12C1910V70RV0010FCD0FCG0F72::::::::::::::::::::::)))181G107291I1J2",
			wantTail:        "VN-A864",
			wantFlight:      "HVN36",
			wantTakeOffDate: "26-03-04",
			wantTakeOffTime: "13:18:43",
			wantOrigin:      "EDDF",
			wantDestination: "VVNB",
			wantRoute:       "EDDF-VVNB",
		},
		{
			name:            "HEG9 TA format",
			text:            "/ILNGE7X.<102>HEG9 4 B-1293CSN668 LYBEZGGG 216TA040326100313 415 CSN55ACMF2707488256 L958579 2447M85P04GEC42-2124-2325A0 2GE747 R956528 2447M85P04GEC42-2124-2325A0 2GE747 748 1040 133 99095",
			wantTail:        "B-1293",
			wantFlight:      "CSN668",
			wantTakeOffDate: "04-03-26",
			wantTakeOffTime: "10:03:13",
			wantOrigin:      "LYBE",
			wantDestination: "ZGGG",
			wantRoute:       "LYBE-ZGGG",
		},
		{
			name:            "FEG8 ER format",
			text:            "/ILNGE7X.<101>FEG8 C 4 JY-BAG RJA263 OJAIKORD 278ER04/03/2609:40:12 6201 BCG48ACMFGE27480551ENG 0905 L956403 2447M85P04GEC45-2124-2322B0 2GE7075 R956785 2447M85P04GEC42-2124-2325A0 2GE7075 09:40:12CLSD 553OFFOFF 01 -2918CLSD 539OFFOFF 01 -2969 L 0 3N2 R 0 3N1 L00000000000000000000000000000000000000000000 R00000000000000000000000000000000000000000000 L00000000000000000000000000000000000000000000 R00000000000000000000000000000000000000000000 L 1111 111 11VALID R 111",
			wantTail:        "JY-BAG",
			wantFlight:      "RJA263",
			wantTakeOffDate: "04-03-26",
			wantTakeOffTime: "09:40:12",
			wantOrigin:      "OJAI",
			wantDestination: "KORD",
			wantRoute:       "OJAI-KORD",
		},
		{
			name:            "SCR103 embedded separator format",
			text:            "/ILNGE7X.SCR103.N852GT5Y3GTI9771 260304173200LHBPVHHH+00008 8F959307670001000072124M70P04C090C0900406020B2401M43P04 2405M75P07 2405M76P07 NOT LOADED +045+023+11901I5OFS1ES1181501100T00L00000T00S0",
			wantTail:        "N852GT",
			wantFlight:      "GTI9771",
			wantTakeOffDate: "26-03-04",
			wantTakeOffTime: "17:32:00",
			wantOrigin:      "LHBP",
			wantDestination: "VHHH",
			wantRoute:       "LHBP-VHHH",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msg := &acars.Message{
				ID:        acars.FlexInt64(501),
				Timestamp: "2026-03-14T00:00:00Z",
				Label:     "_",
				Text:      tc.text,
			}

			parsed := parser.Parse(msg)
			if parsed == nil {
				t.Fatal("Parse() returned nil")
			}

			result, ok := parsed.(*Result)
			if !ok {
				t.Fatalf("Parse() returned %T, want *Result", parsed)
			}

			if result.Tail != tc.wantTail {
				t.Errorf("Tail = %q, want %q", result.Tail, tc.wantTail)
			}
			if result.MsgType != "ILNGE" {
				t.Errorf("MsgType = %q, want %q", result.MsgType, "ILNGE")
			}
			if result.Flight != tc.wantFlight {
				t.Errorf("Flight = %q, want %q", result.Flight, tc.wantFlight)
			}
			if result.TakeOffDate != tc.wantTakeOffDate {
				t.Errorf("TakeOffDate = %q, want %q", result.TakeOffDate, tc.wantTakeOffDate)
			}
			if result.TakeOffTime != tc.wantTakeOffTime {
				t.Errorf("TakeOffTime = %q, want %q", result.TakeOffTime, tc.wantTakeOffTime)
			}
			if result.Origin != tc.wantOrigin {
				t.Errorf("Origin = %q, want %q", result.Origin, tc.wantOrigin)
			}
			if result.Destination != tc.wantDestination {
				t.Errorf("Destination = %q, want %q", result.Destination, tc.wantDestination)
			}
			if result.Route != tc.wantRoute {
				t.Errorf("Route = %q, want %q", result.Route, tc.wantRoute)
			}
		})
	}
}

func TestQuickCheckILNGE7X(t *testing.T) {
	parser := &Parser{}

	if !parser.QuickCheck("/ILNGE7X.CLB006TK TC-LLT2THY69W 260304054817LTFMKIAD") {
		t.Fatal("QuickCheck() = false, want true")
	}
	if parser.QuickCheck("/OTHER.TEST DATA") {
		t.Fatal("QuickCheck() = true, want false")
	}
}
