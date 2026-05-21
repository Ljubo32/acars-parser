package cpdlc

import (
	"fmt"
	"math"
	"strings"
)

// MessageDirection indicates whether the message is uplink (ground to air) or downlink (air to ground).
type MessageDirection int

const (
	// DirectionUnknown indicates the direction could not be determined.
	DirectionUnknown MessageDirection = iota
	// DirectionUplink is a ground-to-air message from ATC to the aircraft.
	DirectionUplink
	// DirectionDownlink is an air-to-ground message from the aircraft to ATC.
	DirectionDownlink
)

func (d MessageDirection) String() string {
	switch d {
	case DirectionUplink:
		return "uplink"
	case DirectionDownlink:
		return "downlink"
	default:
		return "unknown"
	}
}

// MessageHeader contains the CPDLC message header fields.
type MessageHeader struct {
	MsgID     int   `json:"msg_id"`              // Message identification number.
	MsgRef    *int  `json:"msg_ref,omitempty"`   // Reference number (optional).
	Timestamp *Time `json:"timestamp,omitempty"` // Timestamp (optional).
}

// Time represents a FANS timestamp (hours, minutes).
type Time struct {
	Hours   int  `json:"hours"`
	Minutes int  `json:"minutes"`
	Seconds *int `json:"seconds,omitempty"`
}

func (t *Time) String() string {
	if t == nil {
		return ""
	}
	if t.Seconds != nil {
		return fmt.Sprintf("%02d:%02d:%02d", t.Hours, t.Minutes, *t.Seconds)
	}
	return fmt.Sprintf("%02d:%02d", t.Hours, t.Minutes)
}

// Altitude represents an altitude value with its type.
type Altitude struct {
	Type  string `json:"type"`  // "flight_level", "feet", "meters", etc.
	Value int    `json:"value"` // The altitude value.
}

func (a *Altitude) String() string {
	if a == nil {
		return ""
	}
	switch a.Type {
	case "flight_level":
		return fmt.Sprintf("FL%d", a.Value)
	case "flight_level_metric":
		return fmt.Sprintf("FL%dm", a.Value*10) // Value is in 10s of metres.
	case "feet":
		return fmt.Sprintf("%d ft", a.Value)
	case "meters":
		return fmt.Sprintf("%d m", a.Value)
	default:
		return fmt.Sprintf("%d %s", a.Value, a.Type)
	}
}

// Speed represents a speed value with its type.
type Speed struct {
	Type  string `json:"type"`  // "knots", "mach", etc.
	Value int    `json:"value"` // The speed value (mach is scaled by 1000).
}

func (s *Speed) String() string {
	if s == nil {
		return ""
	}
	switch s.Type {
	case "mach":
		return fmt.Sprintf("M.%02d", s.Value) // Value is mach * 100.
	case "knots":
		return fmt.Sprintf("%d kt", s.Value)
	case "kph":
		return fmt.Sprintf("%d km/h", s.Value)
	default:
		return fmt.Sprintf("%d %s", s.Value, s.Type)
	}
}

// Position represents a geographic position.
type Position struct {
	Type         string   `json:"type"`                    // "latlon", "fix", "navaid", "place_bearing_distance", etc.
	Latitude     *float64 `json:"latitude,omitempty"`      // Decimal degrees.
	Longitude    *float64 `json:"longitude,omitempty"`     // Decimal degrees.
	Name         string   `json:"name,omitempty"`          // Fix/navaid name.
	Bearing      *int     `json:"bearing,omitempty"`       // Bearing in degrees (for place_bearing_distance).
	Distance     *int     `json:"distance,omitempty"`      // Distance value (for place_bearing_distance).
	DistanceUnit string   `json:"distance_unit,omitempty"` // "nm" or "km" (for place_bearing_distance).
}

func (p *Position) String() string {
	if p == nil {
		return ""
	}
	if p.Type == "place_bearing_distance" && p.Name != "" && p.Bearing != nil && p.Distance != nil {
		return fmt.Sprintf("%s %03d/%d%s", p.Name, *p.Bearing, *p.Distance, p.DistanceUnit)
	}
	if p.Name != "" {
		return p.Name
	}
	if p.Latitude != nil && p.Longitude != nil {
		return fmt.Sprintf("%.4f,%.4f", *p.Latitude, *p.Longitude)
	}
	return ""
}

// PlaceBearingDistance represents a position defined by fix, bearing, and distance.
type PlaceBearingDistance struct {
	FixName      string   `json:"fix_name"`
	Latitude     *float64 `json:"latitude,omitempty"`
	Longitude    *float64 `json:"longitude,omitempty"`
	Bearing      *int     `json:"bearing"`       // Degrees 1-360.
	Distance     *int     `json:"distance"`      // Distance value.
	DistanceUnit string   `json:"distance_unit"` // "nm" or "km".
	Magnetic     bool     `json:"magnetic"`      // True if bearing is magnetic.
}

// RouteClearance represents a route clearance with departure/arrival information.
type RouteClearance struct {
	AirportDeparture    string                    `json:"airport_departure,omitempty"`
	AirportDestination  string                    `json:"airport_destination,omitempty"`
	RunwayDeparture     *Runway                   `json:"runway_departure,omitempty"`
	ProcedureDeparture  *ProcedureName            `json:"procedure_departure,omitempty"`
	RunwayArrival       *Runway                   `json:"runway_arrival,omitempty"`
	ProcedureApproach   *ProcedureName            `json:"procedure_approach,omitempty"`
	ProcedureArrival    *ProcedureName            `json:"procedure_arrival,omitempty"`
	AirwayIntercept     string                    `json:"airway_intercept,omitempty"`
	RouteInformation    []RouteInformationElement  `json:"route_information,omitempty"`
	RouteInfoAdditional []WaypointSpeedAltitude    `json:"route_info_additional,omitempty"`
}

// ATWAltitude represents a single altitude constraint with a tolerance qualifier.
// It corresponds to the FANSATWAltitude ASN.1 type in the FANS-1/A specification.
type ATWAltitude struct {
	Tolerance string    `json:"tolerance"` // "at", "atorabove", or "atorbelow".
	Altitude  *Altitude `json:"altitude,omitempty"`
}

// WaypointSpeedAltitude represents a waypoint with optional speed and altitude
// constraints. It corresponds to FANSWaypointSpeedAltitude in FANS-1/A.
type WaypointSpeedAltitude struct {
	Position  *Position     `json:"position,omitempty"`
	Speed     *Speed        `json:"speed,omitempty"`
	Altitudes []ATWAltitude `json:"altitudes,omitempty"`
}

func (w *WaypointSpeedAltitude) String() string {
	if w == nil {
		return ""
	}
	parts := []string{}
	if w.Position != nil {
		parts = append(parts, w.Position.String())
	}
	if w.Speed != nil {
		parts = append(parts, w.Speed.String())
	}
	for _, a := range w.Altitudes {
		if a.Altitude != nil {
			parts = append(parts, a.Tolerance+" "+a.Altitude.String())
		}
	}
	return strings.Join(parts, " ")
}

// FormatHierarchical returns a multi-line hierarchical representation of the
// waypoint speed/altitude entry.
func (w *WaypointSpeedAltitude) FormatHierarchical(indent string) string {
	if w == nil {
		return ""
	}
	sub := indent + " "
	var sb strings.Builder
	if w.Position != nil {
		sb.WriteString(indent + "POSITION: " + w.Position.String() + "\n")
	}
	if w.Speed != nil {
		sb.WriteString(indent + "SPEED: " + w.Speed.String() + "\n")
	}
	for _, a := range w.Altitudes {
		if a.Altitude != nil {
			sb.WriteString(indent + "ALTITUDE CONSTRAINT:\n")
			sb.WriteString(sub + "TOLERANCE: " + strings.ToUpper(a.Tolerance) + "\n")
			sb.WriteString(sub + "ALTITUDE: " + a.Altitude.String() + "\n")
		}
	}
	return sb.String()
}

type RouteInformationElement struct {
	Kind     string    `json:"kind"`
	Position *Position `json:"position,omitempty"`
	Airway   string    `json:"airway,omitempty"`
	Text     string    `json:"text,omitempty"`
}

func (r RouteInformationElement) String() string {
	if r.Position != nil {
		return r.Position.String()
	}
	if r.Airway != "" {
		return r.Airway
	}
	return r.Text
}

func (r *RouteClearance) String() string {
	if r == nil {
		return ""
	}
	parts := []string{}
	if r.AirportDeparture != "" {
		parts = append(parts, "DEP:"+r.AirportDeparture)
	}
	if r.AirportDestination != "" {
		parts = append(parts, "DEST:"+r.AirportDestination)
	}
	if r.RunwayDeparture != nil {
		parts = append(parts, "RWY:"+r.RunwayDeparture.String())
	}
	if r.ProcedureDeparture != nil {
		parts = append(parts, "SID:"+r.ProcedureDeparture.String())
	}
	if r.AirwayIntercept != "" {
		parts = append(parts, "AWY:"+r.AirwayIntercept)
	}
	if len(r.RouteInformation) > 0 {
		routeParts := make([]string, 0, len(r.RouteInformation))
		for _, routeInfo := range r.RouteInformation {
			if text := routeInfo.String(); text != "" {
				routeParts = append(routeParts, text)
			}
		}
		if len(routeParts) > 0 {
			parts = append(parts, "ROUTE:"+strings.Join(routeParts, " "))
		}
	}
	if len(parts) == 0 {
		return "(route clearance)"
	}
	return strings.Join(parts, " ")
}

// Runway represents a runway designation.
type Runway struct {
	Direction     int    `json:"direction"`     // 1-36.
	Configuration string `json:"configuration"` // "left", "right", "center", "none".
}

func (r *Runway) String() string {
	if r == nil {
		return ""
	}
	dir := fmt.Sprintf("%02d", r.Direction)
	switch r.Configuration {
	case "left":
		return dir + "L"
	case "right":
		return dir + "R"
	case "center":
		return dir + "C"
	default:
		return dir
	}
}

// ProcedureName represents a procedure (SID/STAR/approach).
type ProcedureName struct {
	Type       string `json:"type"`                 // "arrival", "approach", "departure".
	Name       string `json:"name"`                 // Procedure name.
	Transition string `json:"transition,omitempty"` // Optional transition.
}

func (p *ProcedureName) String() string {
	if p == nil {
		return ""
	}
	if p.Transition != "" {
		return p.Name + "." + p.Transition
	}
	return p.Name
}

// Frequency represents a radio frequency.
type Frequency struct {
	Type  string  `json:"type"`  // "vhf", "uhf", "hf", "satcom".
	Value float64 `json:"value"` // For VHF/UHF value is MHz; for HF value is kHz; for satcom channel it's an integer channel number.
}

func (f *Frequency) String() string {
	if f == nil {
		return ""
	}
	switch f.Type {
	case "vhf":
		return fmt.Sprintf("%.3f MHz", f.Value)
	case "uhf":
		return fmt.Sprintf("%.3f MHz", f.Value)
	case "hf":
		// HF in kHz
		return fmt.Sprintf("%.0f kHz", f.Value)
	case "satcom":
		return fmt.Sprintf("SATCOM ch %.0f", f.Value)
	default:
		return fmt.Sprintf("%v", f.Value)
	}
}

// Degrees represents a heading or track value.
type Degrees struct {
	Magnetic bool `json:"magnetic,omitempty"` // True if magnetic, false if true.
	Value    int  `json:"value"`              // Degrees 0-359.
}

func (d *Degrees) String() string {
	if d == nil {
		return ""
	}
	suffix := "T"
	if d.Magnetic {
		suffix = "M"
	}
	return fmt.Sprintf("%03d%s", d.Value, suffix)
}

// DistanceOffset represents a lateral offset from route.
type DistanceOffset struct {
	Distance  int    `json:"distance"`  // Distance value.
	Unit      string `json:"unit"`      // "nm" or "km".
	Direction string `json:"direction"` // "left" or "right".
}

func (d *DistanceOffset) String() string {
	if d == nil {
		return ""
	}
	return fmt.Sprintf("%d %s %s", d.Distance, d.Unit, d.Direction)
}

// BeaconCode represents a transponder code.
type BeaconCode struct {
	Code string `json:"code"` // 4-digit octal code.
}

func (b *BeaconCode) String() string {
	if b == nil {
		return ""
	}
	return b.Code
}

// FreeText represents free-form text.
type FreeText struct {
	Text string `json:"text"`
}

// Temperature represents an air temperature.
type Temperature struct {
	Type  string  `json:"type"`  // "C" or "F"
	Value float64 `json:"value"` // degrees
}

func (t *Temperature) String() string {
	if t == nil {
		return ""
	}
	unit := t.Type
	if unit == "" {
		unit = "C"
	}
	// Keep one decimal max, but avoid trailing .0 for integer-ish values.
	if t.Value == float64(int(t.Value)) {
		return fmt.Sprintf("%d %s", int(t.Value), unit)
	}
	return fmt.Sprintf("%.1f %s", t.Value, unit)
}

// WindSpeed represents wind speed.
type WindSpeed struct {
	Type  string `json:"type"`  // "kts" or "kmh"
	Value int    `json:"value"` // speed
}

func (w *WindSpeed) String() string {
	if w == nil {
		return ""
	}
	suffix := w.Type
	if suffix == "" {
		suffix = "kts"
	}
	return fmt.Sprintf("%d %s", w.Value, suffix)
}

// Winds represents wind direction and speed.
type Winds struct {
	Direction int        `json:"direction"` // degrees
	Speed     *WindSpeed `json:"speed,omitempty"`
}

func (w *Winds) String() string {
	if w == nil {
		return ""
	}
	if w.Speed != nil {
		return fmt.Sprintf("%d\u00b0/%s", w.Direction, w.Speed.String())
	}
	return fmt.Sprintf("%d\u00b0", w.Direction)
}

// PositionReport represents a downlink POSITION REPORT (dM48).
// Field set matches what is commonly seen in FANS-1/A position reports.
type PositionReport struct {
	PosCurrent       *Position    `json:"pos_current,omitempty"`
	TimeAtPosCurrent *Time        `json:"time_at_pos_current,omitempty"`
	Alt              *Altitude    `json:"alt,omitempty"`
	NextFix          *Position    `json:"next_fix,omitempty"`
	EtaAtFixNext     *Time        `json:"eta_at_fix_next,omitempty"`
	NextNextFix      *Position    `json:"next_next_fix,omitempty"`
	EtaAtDest        *Time        `json:"eta_at_dest,omitempty"`
	Temp             *Temperature `json:"temp,omitempty"`
	Winds            *Winds       `json:"winds,omitempty"`
	Speed            *Speed       `json:"speed,omitempty"`
	SpeedGround      *Speed          `json:"speed_gnd,omitempty"`
	VertChange       *VerticalChange `json:"vert_change,omitempty"`
	TrackAngle       *Degrees        `json:"trk_angle,omitempty"`
	TrueHeading      *Degrees        `json:"true_hdg,omitempty"`
	ReportedWptPos   *Position    `json:"reported_wpt_pos,omitempty"`
	ReportedWptTime  *Time        `json:"reported_wpt_time,omitempty"`
	ReportedWptAlt   *Altitude    `json:"reported_wpt_alt,omitempty"`
}

func (p *PositionReport) String() string {
	if p == nil {
		return ""
	}
	// Compact summary for label substitution.
	parts := []string{}
	if p.PosCurrent != nil {
		parts = append(parts, p.PosCurrent.String())
	}
	if p.TimeAtPosCurrent != nil {
		parts = append(parts, p.TimeAtPosCurrent.String())
	}
	if p.Alt != nil {
		parts = append(parts, p.Alt.String())
	}
	if p.NextFix != nil && p.EtaAtFixNext != nil {
		parts = append(parts, fmt.Sprintf("next %s %s", p.NextFix.String(), p.EtaAtFixNext.String()))
	} else if p.NextFix != nil {
		parts = append(parts, fmt.Sprintf("next %s", p.NextFix.String()))
	}
	if p.Winds != nil {
		parts = append(parts, "wind "+p.Winds.String())
	}
	if p.Speed != nil {
		parts = append(parts, "spd "+p.Speed.String())
	}
	if p.Temp != nil {
		parts = append(parts, "temp "+p.Temp.String())
	}
	if len(parts) == 0 {
		return "(position report)"
	}
	// Use newline as the field separator so callers (e.g. the viewer raw text
	// panel) can display each field on its own line without further parsing.
	return joinNonEmpty(parts, "\n")
}

// FormatHierarchical returns a multi-line hierarchical representation of the position report
// in the libacars display style. The indent string is prepended to each line.
func (p *PositionReport) FormatHierarchical(indent string) string {
	if p == nil {
		return ""
	}
	var sb strings.Builder

	// Current position.
	if p.PosCurrent != nil {
		if p.PosCurrent.Latitude != nil && p.PosCurrent.Longitude != nil {
			sb.WriteString(indent + formatCoordLat(*p.PosCurrent.Latitude) + "\n")
			sb.WriteString(indent + formatCoordLon(*p.PosCurrent.Longitude) + "\n")
		} else if p.PosCurrent.Name != "" {
			sb.WriteString(indent + "FIX: " + p.PosCurrent.Name + "\n")
		}
	}

	// Time at current position (HH:MM).
	if p.TimeAtPosCurrent != nil {
		sb.WriteString(indent + "TIME AT CURRENT POSITION: " + p.TimeAtPosCurrent.String() + "\n")
	}

	// Altitude.
	if p.Alt != nil {
		sb.WriteString(indent + formatAltitudeHierarchical(p.Alt) + "\n")
	}

	// Next fix and its ETA.
	if p.NextFix != nil {
		sb.WriteString(indent + "NEXT FIX:\n")
		sb.WriteString(indent + " FIX: " + p.NextFix.Name + "\n")
	}
	if p.EtaAtFixNext != nil {
		sb.WriteString(indent + "ETA AT NEXT FIX: " + p.EtaAtFixNext.String() + "\n")
	}

	// Next+1 fix.
	if p.NextNextFix != nil {
		sb.WriteString(indent + "NEXT+1 FIX:\n")
		sb.WriteString(indent + " FIX: " + p.NextNextFix.Name + "\n")
	}

	// ETA at destination.
	if p.EtaAtDest != nil {
		sb.WriteString(indent + "ETA AT DESTINATION: " + p.EtaAtDest.String() + "\n")
	}

	// Temperature.
	if p.Temp != nil {
		sb.WriteString(indent + fmt.Sprintf("TEMPERATURE: %d %s\n", int(p.Temp.Value), strings.ToUpper(p.Temp.Type)))
	}

	// Winds: direction and speed on separate lines.
	if p.Winds != nil {
		sb.WriteString(indent + fmt.Sprintf("WIND DIRECTION: %d DEG\n", p.Winds.Direction))
		if p.Winds.Speed != nil {
			sb.WriteString(indent + fmt.Sprintf("WIND SPEED: %d %s\n", p.Winds.Speed.Value, strings.ToUpper(p.Winds.Speed.Type)))
		}
	}

	// Speed (e.g. Mach number).
	if p.Speed != nil {
		sb.WriteString(indent + formatSpeedHierarchical(p.Speed) + "\n")
	}

	// Ground speed (FANSGroundSpeedKnots, always in knots).
	if p.SpeedGround != nil {
		sb.WriteString(indent + fmt.Sprintf("SPEED GROUND: %d KTS\n", p.SpeedGround.Value))
	}

	// Vertical change (direction and rate).
	if p.VertChange != nil {
		sb.WriteString(indent + "VERTICAL CHANGE: " + p.VertChange.String() + "\n")
	}

	// Track angle.
	if p.TrackAngle != nil {
		typeStr := "TRUE"
		if p.TrackAngle.Magnetic {
			typeStr = "MAGNETIC"
		}
		sb.WriteString(indent + fmt.Sprintf("TRACK ANGLE: %d DEG %s\n", p.TrackAngle.Value, typeStr))
	}

	// True heading.
	if p.TrueHeading != nil {
		typeStr := "TRUE"
		if p.TrueHeading.Magnetic {
			typeStr = "MAGNETIC"
		}
		sb.WriteString(indent + fmt.Sprintf("TRUE HEADING: %d DEG %s\n", p.TrueHeading.Value, typeStr))
	}

	// Reported waypoint position, time, and altitude.
	if p.ReportedWptPos != nil {
		sb.WriteString(indent + "REPORTED WAYPOINT POSITION:\n")
		sb.WriteString(indent + " FIX: " + p.ReportedWptPos.Name + "\n")
	}
	if p.ReportedWptTime != nil {
		sb.WriteString(indent + "REPORTED WAYPOINT TIME: " + p.ReportedWptTime.String() + "\n")
	}
	if p.ReportedWptAlt != nil {
		sb.WriteString(indent + "REPORTED WAYPOINT ALTITUDE:\n")
		sb.WriteString(indent + " " + formatAltitudeHierarchical(p.ReportedWptAlt) + "\n")
	}

	return sb.String()
}

// formatAltitudeHierarchical returns a single-line libacars-style altitude label such as
// "FLIGHT LEVEL: 320" or "ALTITUDE: 35000 FT".
func formatAltitudeHierarchical(a *Altitude) string {
	if a == nil {
		return ""
	}
	switch a.Type {
	case "flight_level":
		return fmt.Sprintf("FLIGHT LEVEL: %d", a.Value)
	case "feet":
		return fmt.Sprintf("ALTITUDE: %d FT", a.Value)
	case "meters":
		return fmt.Sprintf("ALTITUDE: %d M", a.Value)
	default:
		return fmt.Sprintf("ALTITUDE: %d %s", a.Value, strings.ToUpper(a.Type))
	}
}

// formatSpeedHierarchical returns a single-line libacars-style speed label such as
// "MACH NUMBER: 0.83" or "SPEED: 250 KTS".
func formatSpeedHierarchical(s *Speed) string {
	if s == nil {
		return ""
	}
	switch s.Type {
	case "mach":
		// Value is Mach * 100 (e.g. 83 → M.83).
		return fmt.Sprintf("MACH NUMBER: 0.%02d", s.Value)
	case "knots":
		return fmt.Sprintf("SPEED: %d KTS", s.Value)
	case "kph":
		return fmt.Sprintf("SPEED: %d KPH", s.Value)
	default:
		return fmt.Sprintf("SPEED: %d %s", s.Value, strings.ToUpper(s.Type))
	}
}

func joinNonEmpty(parts []string, sep string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return ""
	}
	res := out[0]
	for i := 1; i < len(out); i++ {
		res += sep + out[i]
	}
	return res
}

// ErrorInfo represents CPDLC error information.
type ErrorInfo struct {
	Code int    `json:"code"`
	Desc string `json:"description,omitempty"`
}

// VerticalRate represents a climb/descent rate.
type VerticalRate struct {
	Value int `json:"value"` // ft/min.
}

func (v *VerticalRate) String() string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%d ft/min", v.Value)
}

// VerticalChange represents a FANSVerticalChange: a direction (up/down) and an optional rate.
type VerticalChange struct {
	Direction string       `json:"direction"`         // "up" or "down"
	Rate      *VerticalRate `json:"rate,omitempty"`
}

func (vc *VerticalChange) String() string {
	if vc == nil {
		return ""
	}
	if vc.Rate == nil {
		return vc.Direction
	}
	return fmt.Sprintf("%s %s", vc.Direction, vc.Rate.String())
}

// formatCoordLat formats a decimal latitude into the libacars-style display format.
// Whole-degree values are rendered as "LATITUDE:   dd DEG NORTH/SOUTH".
// Sub-degree values are rendered as "LATITUDE:   dd mm.m NORTH/SOUTH".
func formatCoordLat(lat float64) string {
	direction := "NORTH"
	if lat < 0 {
		direction = "SOUTH"
		lat = -lat
	}
	degrees := int(lat)
	minutes := (lat - float64(degrees)) * 60.0
	if math.Abs(minutes) < 0.05 {
		return fmt.Sprintf("LATITUDE: %4d DEG %s", degrees, direction)
	}
	return fmt.Sprintf("LATITUDE: %4d %4.1f %s", degrees, minutes, direction)
}

// formatCoordLon formats a decimal longitude into the libacars-style display format.
// Whole-degree values are rendered as "LONGITUDE: ddd DEG EAST/WEST".
// Sub-degree values are rendered as "LONGITUDE: ddd mm.m EAST/WEST".
func formatCoordLon(lon float64) string {
	direction := "EAST"
	if lon < 0 {
		direction = "WEST"
		lon = -lon
	}
	degrees := int(lon)
	minutes := (lon - float64(degrees)) * 60.0
	if math.Abs(minutes) < 0.05 {
		return fmt.Sprintf("LONGITUDE: %03d DEG %s", degrees, direction)
	}
	return fmt.Sprintf("LONGITUDE: %03d %4.1f %s", degrees, minutes, direction)
}

// FormatHierarchical returns a multi-line hierarchical representation of the route clearance.
// The indent string is prepended to each top-level line; sub-items use indent with one additional space.
func (r *RouteClearance) FormatHierarchical(indent string) string {
	if r == nil {
		return ""
	}
	sub := indent + " "
	var sb strings.Builder

	if r.AirportDeparture != "" {
		sb.WriteString(indent + "DEPARTURE AIRPORT: " + r.AirportDeparture + "\n")
	}
	if r.AirportDestination != "" {
		sb.WriteString(indent + "DESTINATION AIRPORT: " + r.AirportDestination + "\n")
	}
	if r.RunwayDeparture != nil {
		sb.WriteString(indent + "DEPARTURE RUNWAY:\n")
		sb.WriteString(sub + fmt.Sprintf("RUNWAY DIRECTION: %d\n", r.RunwayDeparture.Direction))
		if r.RunwayDeparture.Configuration != "none" && r.RunwayDeparture.Configuration != "" {
			sb.WriteString(sub + "RUNWAY CONFIGURATION: " + strings.ToUpper(r.RunwayDeparture.Configuration) + "\n")
		}
	}
	if r.ProcedureDeparture != nil {
		sb.WriteString(indent + "DEPARTURE PROCEDURE:\n")
		sb.WriteString(r.ProcedureDeparture.FormatHierarchical(sub))
	}
	if r.RunwayArrival != nil {
		sb.WriteString(indent + "ARRIVAL RUNWAY:\n")
		sb.WriteString(sub + fmt.Sprintf("RUNWAY DIRECTION: %d\n", r.RunwayArrival.Direction))
		if r.RunwayArrival.Configuration != "none" && r.RunwayArrival.Configuration != "" {
			sb.WriteString(sub + "RUNWAY CONFIGURATION: " + strings.ToUpper(r.RunwayArrival.Configuration) + "\n")
		}
	}
	if r.ProcedureApproach != nil {
		sb.WriteString(indent + "APPROACH PROCEDURE:\n")
		sb.WriteString(r.ProcedureApproach.FormatHierarchical(sub))
	}
	if r.ProcedureArrival != nil {
		sb.WriteString(indent + "ARRIVAL PROCEDURE:\n")
		sb.WriteString(r.ProcedureArrival.FormatHierarchical(sub))
	}
	if r.AirwayIntercept != "" {
		sb.WriteString(indent + "AIRWAY INTERCEPT: " + r.AirwayIntercept + "\n")
	}
	if len(r.RouteInformation) > 0 {
		sb.WriteString(indent + "ROUTE:\n")
		for _, elem := range r.RouteInformation {
			sb.WriteString(elem.FormatHierarchical(sub))
		}
	}
	if len(r.RouteInfoAdditional) > 0 {
		sb.WriteString(indent + "ROUTE ADDITIONAL CONSTRAINTS:\n")
		for _, w := range r.RouteInfoAdditional {
			w := w
			sb.WriteString(w.FormatHierarchical(sub))
		}
	}
	return sb.String()
}

// isValidProcedureIdentifier returns true if the string contains only alphanumeric characters
// (A–Z, 0–9). Procedure names and transitions must be alphanumeric; non-alphanumeric characters
// indicate a decode error in the bit stream.
func isValidProcedureIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for _, ch := range s {
		if !((ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')) {
			return false
		}
	}
	return true
}

// FormatHierarchical returns a multi-line hierarchical representation of a procedure name.
// The indent string is prepended to each line. Procedure names or transitions containing
// non-alphanumeric characters are omitted, as they indicate a bit-level decode error.
func (p *ProcedureName) FormatHierarchical(indent string) string {
	if p == nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(indent + "PROCEDURE TYPE: " + strings.ToUpper(p.Type) + "\n")
	if isValidProcedureIdentifier(p.Name) {
		sb.WriteString(indent + "PROCEDURE NAME: " + p.Name + "\n")
	}
	if p.Transition != "" && isValidProcedureIdentifier(p.Transition) {
		sb.WriteString(indent + "PROCEDURE TRANSITION: " + p.Transition + "\n")
	}
	return sb.String()
}

// FormatHierarchical returns a multi-line hierarchical representation of a route information element.
// The indent string is prepended to each top-level line; sub-items use indent with one additional space.
func (r RouteInformationElement) FormatHierarchical(indent string) string {
	sub := indent + " "
	var sb strings.Builder
	switch r.Kind {
	case "latlon":
		if r.Position != nil && r.Position.Latitude != nil && r.Position.Longitude != nil {
			sb.WriteString(indent + formatCoordLat(*r.Position.Latitude) + "\n")
			sb.WriteString(indent + formatCoordLon(*r.Position.Longitude) + "\n")
		}
	case "published_identifier":
		sb.WriteString(indent + "PUBLISHED IDENTIFIER:\n")
		if r.Position != nil {
			if r.Position.Name != "" {
				sb.WriteString(sub + "FIX: " + r.Position.Name + "\n")
			}
			if r.Position.Latitude != nil {
				sb.WriteString(sub + formatCoordLat(*r.Position.Latitude) + "\n")
			}
			if r.Position.Longitude != nil {
				sb.WriteString(sub + formatCoordLon(*r.Position.Longitude) + "\n")
			}
		}
	case "airway":
		sb.WriteString(indent + "AIRWAY ID: " + r.Airway + "\n")
	case "place_bearing_distance":
		if r.Position != nil {
			sb.WriteString(indent + "PLACE BEARING DISTANCE:\n")
			if r.Position.Name != "" {
				sb.WriteString(sub + "FIX: " + r.Position.Name + "\n")
			}
			if r.Position.Bearing != nil {
				sb.WriteString(sub + fmt.Sprintf("BEARING: %d\n", *r.Position.Bearing))
			}
			if r.Position.Distance != nil {
				sb.WriteString(sub + fmt.Sprintf("DISTANCE: %d %s\n", *r.Position.Distance, strings.ToUpper(r.Position.DistanceUnit)))
			}
		}
	default:
		if r.Text != "" {
			sb.WriteString(indent + r.Text + "\n")
		}
	}
	return sb.String()
}
