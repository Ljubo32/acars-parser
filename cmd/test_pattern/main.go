package main

import (
	"fmt"
	"regexp"
)

func main() {
	text := "POSN46283E022271,,140610,370,N44052E026499,143250,ARTAT,M56,34863,1040,849/TS140610,120126D038"

	fmt.Println("Testing text:", text)
	fmt.Println()

	pattern := `^POS(?P<lat_dir>[NS])(?P<lat>\d{5})(?P<lon_dir>[EW])(?P<lon>\d{6}),` +
		`(?P<curr_wpt>[A-Z0-9/-]+),(?P<report_time>\d{6}),(?P<altitude>\d+),` +
		`(?P<next_wpt>[A-Z0-9/-]+),(?P<eta>\d+),(?P<wpt3>[A-Z0-9/-]+),(?P<temp>[MP]\d+)` +
		`(?:,(?P<wind>\d{5,6}))?(?:,(?P<extra>.+))?$`

	re := regexp.MustCompile(pattern)
	fmt.Println("Current pattern match:", re.MatchString(text))

	patternFixed := `^POS(?P<lat_dir>[NS])(?P<lat>\d{5})(?P<lon_dir>[EW])(?P<lon>\d{6}),` +
		`(?P<curr_wpt>[A-Z0-9/-]*),(?P<report_time>\d{6}),(?P<altitude>\d+),` +
		`(?P<next_wpt>[A-Z0-9/-]+),(?P<eta>\d+),(?P<wpt3>[A-Z0-9/-]+),(?P<temp>[MP]\d+)` +
		`(?:,(?P<wind>\d{5,6}))?(?:,(?P<extra>.+))?$`

	reFixed := regexp.MustCompile(patternFixed)
	fmt.Println("Fixed pattern match (allow empty curr_wpt):", reFixed.MatchString(text))

	if reFixed.MatchString(text) {
		matches := reFixed.FindStringSubmatch(text)
		for i, name := range reFixed.SubexpNames() {
			if i > 0 && i < len(matches) {
				fmt.Printf("  %s = '%s'\n", name, matches[i])
			}
		}
	}
}
