// Package label83 provides grok-style pattern definitions for Label 83 message parsing.
package label83

import "acars_parser/internal/patterns"

// Formats defines the known Label 83 message formats.
var Formats = []patterns.Format{
	// PR (Position Report) format.
	// Example: 001PR29182854N5106.0W11400.4035000----
	// Groups: day, time, lat_dir, lat, lon_dir, lon, altitude
	{
		Name: "pr_position",
		Pattern: `\d{3}PR(?P<day>\d{2})(?P<time>{TIME6})` +
			`(?P<lat_dir>{LAT_DIR})(?P<lat>\d{4}\.\d)` +
			`(?P<lon_dir>{LON_DIR})(?P<lon>\d{5}\.\d)(?P<altitude>{LON_6D})`,
		Fields: []string{"day", "time", "lat_dir", "lat", "lon_dir", "lon", "altitude"},
	},
	// ZSPD/CSV position format.
	// Example (with ZSPD prefix): M77AUA0199ZSPD,KLAX,291829, 53.83, 176.53,37999,271,  97.1, 85400
	// Example (plain CSV):        KORD,EGLL,130317, 56.01,- 29.34,39001,265,  93.2, 16400
	// Groups: origin, dest, time, lat, lon, altitude, heading, ground_speed
	// Signed coordinates may carry an internal space after the minus sign
	// (e.g. "- 29.34"); the parser normalises these before converting to float.
	{
		Name: "zspd_position",
		Pattern: `(?P<origin>{ICAO}),(?P<dest>{ICAO}),(?P<time>{TIME6}),\s*` +
			`(?P<lat>-?\s*\d+\.\d+),\s*` +
			`(?P<lon>-?\s*\d+\.\d+),` +
			`(?P<altitude>\d+),(?P<heading>{HEADING}),\s*` +
			`(?P<ground_speed>-?\s*[\d.]+)`,
		Fields: []string{"origin", "dest", "time", "lat", "lon", "altitude", "heading", "ground_speed"},
	},
}
