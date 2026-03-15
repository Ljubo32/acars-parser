// Package fst provides grok-style pattern definitions for FST (Label 15) message parsing.
package fst

import "acars_parser/internal/patterns"

// Formats defines the known FST message formats.
var Formats = []patterns.Format{
	// Fixed-layout FST01 format with space-delimited fields after the coordinates.
	// Example: FST01VTBSEGKKN465853E0210031360 197 825 M50C 1226229129746211600005180304
	{
		Name: "fst01_fixed",
		Pattern: `FST(?P<seq>01)(?P<origin>{ICAO})(?P<dest>{ICAO})` +
			`(?P<lat_dir>[NS])(?P<lat>\d{5,7})` +
			`(?P<lon_dir>[EW])(?P<lon>\d{5,7})` +
			`(?P<rest>\d{3}\s+\d{1,3}\s+\d{1,3}\s+(?:M?\d{1,2}C\s+\d{27,28}|M?\d{1,2}C\d{14,15}\s+\d{8}))$`,
		Fields: []string{"seq", "origin", "dest", "lat_dir", "lat", "lon_dir", "lon", "rest"},
	},
	// FST format with 5-digit longitude (more common for European coordinates).
	// Example: FST01EGGDEGLL51420N00312W...
	// Groups: seq, origin, dest, lat_dir, lat, lon_dir, lon, rest
	{
		Name: "fst_5digit_lon",
		Pattern: `FST(?P<seq>\d{2})(?P<origin>{ICAO})(?P<dest>{ICAO})` +
			`(?P<lat_dir>[NS])(?P<lat>\d{5,7})` +
			`(?P<lon_dir>[EW])(?P<lon>\d{5,7})(?P<rest>\d.+)`,
		Fields: []string{"seq", "origin", "dest", "lat_dir", "lat", "lon_dir", "lon", "rest"},
	},
	// FST format with 6-digit longitude.
	// Groups: seq, origin, dest, lat_dir, lat, lon_dir, lon, rest
	{
		Name: "fst_6digit",
		Pattern: `FST(?P<seq>\d{2})(?P<origin>{ICAO})(?P<dest>{ICAO})` +
			`(?P<lat_dir>[NS])(?P<lat>{LON_6D})` +
			`(?P<lon_dir>[EW])(?P<lon>{LON_6D})(?P<rest>.+)`,
		Fields: []string{"seq", "origin", "dest", "lat_dir", "lat", "lon_dir", "lon", "rest"},
	},
}
