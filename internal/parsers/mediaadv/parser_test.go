package mediaadv

import (
	"testing"

	"acars_parser/internal/acars"
)

func TestParse(t *testing.T) {
	parser := &Parser{}

	tests := []struct {
		name       string
		text       string
		wantMatch  bool
		wantEstab  bool
		wantLink   string
		wantTime   string
		wantAvail  int
		wantText   string
		wantFormat string
	}{
		{
			name:       "VHF established",
			text:       "0EV095905V",
			wantMatch:  true,
			wantEstab:  true,
			wantLink:   "V",
			wantTime:   "09:59:05",
			wantAvail:  1,
			wantFormat: " Media Advisory, version 0:\n  Link VHF ACARS established at 09:59:05 UTC\n  Available links: VHF ACARS",
		},
		{
			name:       "SATCOM lost with VDL2 available",
			text:       "0LS0959482",
			wantMatch:  true,
			wantEstab:  false,
			wantLink:   "S",
			wantTime:   "09:59:48",
			wantAvail:  1,
			wantFormat: " Media Advisory, version 0:\n  Link Default SATCOM lost at 09:59:48 UTC\n  Available links: VDL2",
		},
		{
			name:       "VDL2 established",
			text:       "0E21000102",
			wantMatch:  true,
			wantEstab:  true,
			wantLink:   "2",
			wantTime:   "10:00:10",
			wantAvail:  1, // Just "2" at end.
			wantFormat: " Media Advisory, version 0:\n  Link VDL2 established at 10:00:10 UTC\n  Available links: VDL2",
		},
		{
			name:       "With text suffix and multiple links",
			text:       "0E21031582SH/test",
			wantMatch:  true,
			wantEstab:  true,
			wantLink:   "2",
			wantTime:   "10:31:58",
			wantAvail:  3, // "2SH" = VDL2, SATCOM, HF.
			wantText:   "test",
			wantFormat: " Media Advisory, version 0:\n  Link VDL2 established at 10:31:58 UTC\n  Available links: VDL2, Default SATCOM, HF\n  Text: test",
		},
		{
			name:       "HF established with VHF SATCOM HF available",
			text:       "0EH103440VSH/",
			wantMatch:  true,
			wantEstab:  true,
			wantLink:   "H",
			wantTime:   "10:34:40",
			wantAvail:  3,
			wantFormat: " Media Advisory, version 0:\n  Link HF established at 10:34:40 UTC\n  Available links: VHF ACARS, Default SATCOM, HF",
		},
		{
			name:       "VDL2 lost with HF available",
			text:       "0L2055427H",
			wantMatch:  true,
			wantEstab:  false,
			wantLink:   "2",
			wantTime:   "05:54:27",
			wantAvail:  1,
			wantFormat: " Media Advisory, version 0:\n  Link VDL2 lost at 05:54:27 UTC\n  Available links: HF",
		},
		{
			name:       "VHF lost with HF and SATCOM available",
			text:       "0LV061446HS",
			wantMatch:  true,
			wantEstab:  false,
			wantLink:   "V",
			wantTime:   "06:14:46",
			wantAvail:  2,
			wantFormat: " Media Advisory, version 0:\n  Link VHF ACARS lost at 06:14:46 UTC\n  Available links: HF, Default SATCOM",
		},
		{
			name:       "HF established with HF available",
			text:       "0EH063130H",
			wantMatch:  true,
			wantEstab:  true,
			wantLink:   "H",
			wantTime:   "06:31:30",
			wantAvail:  1,
			wantFormat: " Media Advisory, version 0:\n  Link HF established at 06:31:30 UTC\n  Available links: HF",
		},
		{
			name:       "HF established with trailing slash",
			text:       "0EH075757VSH/",
			wantMatch:  true,
			wantEstab:  true,
			wantLink:   "H",
			wantTime:   "07:57:57",
			wantAvail:  3,
			wantFormat: " Media Advisory, version 0:\n  Link HF established at 07:57:57 UTC\n  Available links: VHF ACARS, Default SATCOM, HF",
		},
		{
			name:      "Invalid version",
			text:      "1EV095905V",
			wantMatch: false,
		},
		{
			name:      "Invalid state",
			text:      "0XV095905V",
			wantMatch: false,
		},
		{
			name:      "Too short",
			text:      "0EV09590",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &acars.Message{Text: tt.text, Label: "SA"}

			if got := parser.QuickCheck(tt.text); got != tt.wantMatch {
				t.Errorf("QuickCheck() = %v, want %v", got, tt.wantMatch)
			}

			result := parser.Parse(msg)

			if tt.wantMatch {
				if result == nil {
					t.Fatal("Parse() returned nil, want result")
				}
				r := result.(*Result)
				if r.Established != tt.wantEstab {
					t.Errorf("Established = %v, want %v", r.Established, tt.wantEstab)
				}
				if r.CurrentLink.Code != tt.wantLink {
					t.Errorf("CurrentLink.Code = %v, want %v", r.CurrentLink.Code, tt.wantLink)
				}
				if r.MessageType != "media_advisory" {
					t.Errorf("MessageType = %q, want %q", r.MessageType, "media_advisory")
				}
				if r.LinkTime != tt.wantTime {
					t.Errorf("LinkTime = %v, want %v", r.LinkTime, tt.wantTime)
				}
				if len(r.AvailableLinks) != tt.wantAvail {
					t.Errorf("AvailableLinks count = %d, want %d", len(r.AvailableLinks), tt.wantAvail)
				}
				if r.Text != tt.wantText {
					t.Errorf("Text = %q, want %q", r.Text, tt.wantText)
				}
				if r.FormattedText != tt.wantFormat {
					t.Errorf("FormattedText = %q, want %q", r.FormattedText, tt.wantFormat)
				}
				if got := r.HumanReadableText(); got != tt.wantFormat {
					t.Errorf("HumanReadableText() = %q, want %q", got, tt.wantFormat)
				}
			} else {
				if result != nil {
					t.Errorf("Parse() = %v, want nil", result)
				}
			}
		})
	}
}
