// Package fst provides grok-style pattern definitions for FST (Label 15) message parsing.
package fst

import "acars_parser/internal/patterns"

// Formats defines the known FST message formats.
var Formats = []patterns.Format{
	// FST format with 7-digit longitude (for coordinates with leading zeros).
	// Example: FST01EGLLWSSSN452140E0249275330...
	// Groups: seq, origin, dest, lat_dir, lat, lon_dir, lon, rest
	{
		Name: "fst_7digit_lon",
		Pattern: `FST(?P<seq>\d{2})(?P<origin>{ICAO})(?P<dest>{ICAO})` +
			`(?P<lat_dir>{LAT_DIR})(?P<lat>\d{6})` +
			`(?P<lon_dir>{LON_DIR})(?P<lon>\d{7})(?P<rest>.+)`,
		Fields: []string{"seq", "origin", "dest", "lat_dir", "lat", "lon_dir", "lon", "rest"},
	},
	// FST format with 6-digit longitude.
	// Example: FST01EGGDEGLL51420N00312W...
	// Groups: seq, origin, dest, lat_dir, lat, lon_dir, lon, rest
	{
		Name: "fst_6digit",
		Pattern: `FST(?P<seq>\d{2})(?P<origin>{ICAO})(?P<dest>{ICAO})` +
			`(?P<lat_dir>{LAT_DIR})(?P<lat>\d{6})` +
			`(?P<lon_dir>{LON_DIR})(?P<lon>\d{6})(?P<rest>.+)`,
		Fields: []string{"seq", "origin", "dest", "lat_dir", "lat", "lon_dir", "lon", "rest"},
	},
	// FST format with 5-digit longitude (older format).
	// Groups: seq, origin, dest, lat_dir, lat, lon_dir, lon, rest
	{
		Name: "fst_5digit_lon",
		Pattern: `FST(?P<seq>\d{2})(?P<origin>{ICAO})(?P<dest>{ICAO})` +
			`(?P<lat_dir>{LAT_DIR})(?P<lat>\d{5})` +
			`(?P<lon_dir>{LON_DIR})(?P<lon>\d{5})(?P<rest>.+)`,
		Fields: []string{"seq", "origin", "dest", "lat_dir", "lat", "lon_dir", "lon", "rest"},
	},
}
