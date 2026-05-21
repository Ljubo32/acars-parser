# ACARS Parser

A Go tool for parsing ACARS (Aircraft Communications Addressing and Reporting System) messages. It extracts structured flight data from various message types including Pre-Departure Clearances, flight plans, position reports, and wind forecasts.

## Installation

```bash
go build -o acars_parser ./cmd/acars_parser
```

## Project Structure

```
acars_parser/
├── cmd/acars_parser/          # Command-line entry point
│   ├── main.go
│   ├── extract.go          # Extract command
│   └── live.go             # Live NATS command
├── internal/
│   ├── acars/              # ACARS message types
│   ├── registry/           # Parser registry
│   ├── patterns/           # Shared regex patterns and extractors
│   └── parsers/            # Individual parser implementations
│       ├── adsc/           # ADS-C (B6)
  │       ├── abs/            # ABS0 route hints from H1
│       ├── agfsr/          # AGFSR flight status (4T)
│       ├── atncm/          # ATN CM logon route hints from VDL2
│       ├── cpdlc/          # CPDLC FANS-1/A (AA)
  │       ├── dis/            # RA DIS OFP info
│       ├── eta/            # ETA/timing (5Z)
│       ├── fst/            # FST reports (15)
│       ├── h1/             # H1 FPN/POS/PWI
│       ├── h2wind/         # Wind data (H2)
│       ├── label10/        # Rich position (10)
│       ├── label16/        # Waypoint position (16)
│       ├── label21/        # Position reports (21)
│       ├── label22/        # Detailed position (22)
│       ├── label44/        # Runway info (44)
│       ├── label4j/        # Position+weather (4J)
│       ├── label5l/        # Routes (5L)
│       ├── label80/        # Position (80)
│       ├── label83/        # Position reports (83)
│       ├── labelb2/        # Oceanic clearances (B2)
│       ├── labelb3/        # Gate info (B3)
│       ├── pdc/            # Pre-departure clearances
│       ├── rep301/         # REP301 compact position reports
│       └── sq/             # ARINC position (SQ)
└── README.md
```

## Commands

## GUI Viewer

The standalone HTML viewer in [gui/acars_viewer_fast_kv_map_v8.html](gui/acars_viewer_fast_kv_map_v8.html) now supports a waypoint lookup table from [gui/Waypoints.txt](gui/Waypoints.txt). When the parsed JSON contains named waypoints without embedded coordinates, the viewer can resolve those names to latitude/longitude and plot them on the Leaflet map for FPN route waypoints and H1 PWI route-wind waypoint data.

When `Waypoints.txt` contains multiple fixes with the same name in different regions, the viewer now keeps all candidates and chooses the most plausible coordinate using nearby route anchors instead of blindly taking the last duplicate. For H1 PWI route-wind data, repeated fixes across multiple flight levels are also merged into a single map marker popup that lists all available wind and temperature layers for that waypoint.

When the viewer is served over HTTP from the `gui/` directory, it will try to load `Waypoints.txt` automatically. When the HTML file is opened directly from disk and the browser blocks sibling-file fetches, use the `Load Waypoints` picker in the viewer to load the same file manually.

The viewer can also load airport names from [internal/airports/airports.json](internal/airports/airports.json). When that lookup is available, the Flight column shows one airport-name line for each ICAO code in the saved route between the route string and the saved-route date. For example, a saved route such as `GUCY-GOBD-OMDB` renders `Conakry`, `Blaise Diagne International`, and `Dubai International` on separate lines underneath the route. Those display labels omit `Od:` and `Do:` prefixes, and they also strip a trailing `Airport` suffix to keep the table narrower. Parsed-only rows do not show airport names until the route has been saved, which keeps unsaved rows visually simpler. When the viewer is served over HTTP, it tries to auto-load that airport file; when the HTML file is opened directly from disk, use the `Load Airports` picker if the browser blocks auto-load in file mode.

When JSON output is produced through [gui/acars_parser_gui_dnd_fix2.py](gui/acars_parser_gui_dnd_fix2.py), the GUI can also enrich each JSON row with route lookup data from [gui/flightroute.sqb](gui/flightroute.sqb) when that database is present. The `Route lookup from flightroute.sqb` checkbox controls that behaviour, so route enrichment can be turned off when faster JSON generation is preferred. When enabled, the lookup uses the already normalised ICAO-style flight value, queries the `FlightRoute` table by the `flight` column, and stores the matched `route` plus the latest `updatetime` on the JSON row. In the HTML viewer, those fields are now rendered directly underneath the Flight value instead of using a separate Route column. If no lookup row is found, or the database is unavailable, extraction continues and only the Flight value is shown.

When that GUI runs in merge mode with JSON output enabled, it now reuses the same merged output filename instead of creating incremented names such as `_1`. This matches the single-file extraction path and allows the existing merged JSON to be overwritten directly.

The standalone HTML viewer can now also write to that same `FlightRoute` table through the local Go `routeapi` bridge. Run `acars_parser routeapi -db gui/flightroute.sqb -port 8765`, then the viewer shows two small buttons underneath each row's `Prikaži na mapi` action when the row has a usable flight key: `Upiši` writes the parsed ICAO route in `XXXX-XXXX` format with the row's log date as `updatetime` in `YYYY-MM-DD`, and `Reload` refreshes the saved route for that flight from the database. That refresh is applied to all currently loaded rows with the same flight key so manually synced routes do not keep showing as mismatches in other message types for the same flight.

In that same Flight column, the viewer now labels route lines with `parsed` or `saved` badges so it is obvious whether the displayed route comes from the current parsed row or from the `FlightRoute` lookup database. When a row has only a parsed route candidate and nothing saved yet, the viewer shows that parsed ICAO route directly instead of leaving the route area blank.

The viewer now also ignores numeric-only values when selecting a Flight display value or a route-save flight key. That prevents unrelated payload numbers from being shown in the Flight column or from being used as a lookup key.

That route-write path now also accepts the Unix-seconds timestamps emitted by `acarsdec`, so ACARS JSON rows with values such as `1777761826.271997` can be written to `flightroute.sqb` the same way as ISO-timestamped VDL2 rows.

To make route troubleshooting less ambiguous when more than one `flightroute.sqb` exists, the `routeapi` bridge now resolves the database path to an absolute path and exposes it through `/api/flightroute/config`. The HTML viewer shows that exact database path in the `Route API` status line, so it is immediately obvious which SQLite file the `Upiši` and `Reload` buttons are using.

When that rendered `route` value is in the ICAO `XXXX-XXXX` format and it does not match the best parsed origin/destination pair already present in the JSON, the viewer now highlights that route line in dark red under the Flight value. The viewer also highlights the `updatetime` line in dark red when it begins with `1`, which helps suspicious Unix-style timestamps stand out during review.

The viewer now always shows a `raw text` details block for non-empty ACARS payload text, including single-line messages. That block no longer relies on any symbol-limit style truncation in the details panel; instead it wraps long lines to the available panel width.

The main search box in the HTML viewer now also has a `srch txt` checkbox. When it is ticked, the search is limited to the JSON `text` or `message.text` payload fields instead of the broader combined row summary. When it is unticked, search behaves as before and scans the wider rendered row content.

The main `Summary / Details` column in the HTML viewer also no longer truncates the summary text to a fixed 500-character limit for normal row rendering paths. Long summaries now rely on the existing word wrap in that column instead of being cut off.

### extract

Extracts structured data from JSONL files and JAERO TXT logs containing ACARS messages.

```bash
./acars_parser extract -input messages.jsonl [-output output.json] [-pretty] [-all] [-format json|text]
```

The `extract` command autodetects JSONL and JAERO TXT input. For JAERO logs, the CLI converts each timestamped block into a normal ACARS message, keeps only the raw ACARS payload in `message.text`, preserves legitimate multiline payload text, strips JAERO line-wrap artefacts such as inserted `- #MD` continuations, and skips empty blocks.

The extractor handles both the original JAERO L-Band log format and the C-Band JAERO format produced by a different decoder. C-Band headers use the same `HH:MM:SS DD-MM-YY UTC AES: GES: ... ! <label> <prio> [description]` structure but append a `FLIGHT <callsign>` token to the aircraft description and use a digit for the priority character. The flight number is extracted from that suffix and normalised to its ICAO equivalent via the airline translator, then placed in `message.flight`. The `FLIGHT <callsign>` token is stripped from `message.airframe.manufacturer_model`.

For label `MA` MIAM messages, the decoded MIAM block that JAERO prints below the compressed payload is extracted separately and dispatched through the dedicated `miam` parser. The original compressed payload is kept in `message.text` while the structured MIAM fields (`message_type`, `transfer_type`, `pdu_length`, `aircraft_id`, `msg_num`, `ack_required`, `compression`, `encoding`, `inner_label`, `inner_sublabel`, `inner_message`, `formatted_text`) appear in the `results[]` array. In the HTML viewer the MA raw text expansion shows the compressed payload followed by the full decoded MIAM block. The parser handles both the original JAERO title-case field names and the ALL CAPS field names produced by the C-Band decoder. Lines annotated by libacars with `-- DECOMPRESSION FAILED` or `-- CRC CHECK FAILED` are excluded from `inner_message` but retained in `formatted_text`.

For AFN payloads that contain segments such as `/FMH<flight>` and `/FAK0,<destination>`, the extractor also infers the flight number, a clean tail fallback, and `destination_airport` directly from the raw message text before serialising the output JSON. It also infers the flight from FPN headers in the form `FPN/FN<flight>/...`, so values such as `FPN/FNSVA1047/...` populate `message.flight` in the emitted JSON.

For RA payloads carrying `INI01` initialisation messages such as `QUDXBEGEK~1INI01091501 UAE810 /09/OEMA/OMDB/...`, the extractor now infers `message.flight`, `message.departing_airport`, and `message.destination_airport` from the raw message text. The parsed result for those rows now also exposes `msg_type: "INI"`.

For RA acknowledgement and failure payloads that include literals such as `FLIGHT NUMBER: EK0806/UAE806 SECTOR: OEJN-OMDB` or `FLIGHT NUMBER: UAE394 SECTOR: OMDB-VVNB`, the extractor now also infers `message.flight`, `message.departing_airport`, and `message.destination_airport` directly from the raw message text. When the `FLIGHT NUMBER:` field carries both an IATA-style and ICAO-style identifier separated by `/`, the extractor prefers the ICAO-style flight value after the slash.

For RA `DIS` payloads that contain an `OFP INFO` block such as `EK512 DXB-DEL A6ENM: CURRENT OFP NUMBER 15/0/0`, the extractor now also infers `message.flight`, `message.departing_airport`, and `message.destination_airport` from that operational flight-plan summary. The dedicated `dis` parser preserves the displayed flight and route from the message body, while the message-level airport fields are normalised to ICAO codes for downstream route handling.

The same `dis` parser now also returns a dedicated parsed result for `LDSHT ACCEPT ACK` messages such as `FLIGHT NUMBER: EK0615/UAE6W` with `SECTOR: OPIS-OMDB`. Those rows are now grouped under the `DIS` type in the viewer filter instead of staying untyped.

It also returns a dedicated parsed result for `FLT SUMM ACK` / `FLIGHT SUMMARY ACK` messages that carry `FLIGHT NUMBER:` and `SECTOR:` lines. Those acknowledgement rows are also grouped under the same `DIS` viewer type. For rows where the flight field contains both variants separated by `/`, such as `EK0651/UAE7P`, the parsed `flight` value prefers the trailing token, which matches the existing RA metadata extraction behaviour.

For PDC-style clearances, including label `A3` payloads such as `/...DC1/CLD ... ZSSS PDC ... CES239 CLRD TO VMMC ...`, the extractor also populates `message.flight`, `message.departing_airport`, and `message.destination_airport` from the raw clearance text.

For label `A4` `FS1/FSM` payloads such as `/CDGATYA.FS1/FSM 0546 260314 LFPG SV0143 ...`, the extractor also populates `message.flight` and `message.departing_airport` from the raw message text. Two-character IATA airline prefixes continue to be normalised in the backend to their ICAO equivalents, so values such as `SV0143` become `SVA143` and `AY99` becomes `FIN99` in the emitted JSON.

For CPDLC label `AA` and `BA` payloads, the backend now preserves richer structured element data for contact and monitor instructions, connection-management payloads, full facility names, facility designations, facility functions, frequencies, free-text elements, TP4 table values, and header timestamp seconds when present. The uplink decoder now also preserves compound climb/descent target data for `uM26` to `uM29`, so instructions such as `CLIMB TO REACH [altitude] BY [time]` retain structured altitude and time or position values instead of collapsing to the bare label template. Route-clearance uplinks now also tolerate the short empty `route_info_additional` tail encoding seen in some FANS-1/A messages, which prevents later CPDLC elements from being swallowed into the first clearance element.

For ADS-C label `A6` contract requests, the backend now decodes periodic reporting intervals from the actual request tag encoding instead of treating the third payload byte as a raw `* 64` modulus. This fixes periodic intervals such as `0xD9 -> 1664 seconds` and `0x00 -> 0 seconds`. The emitted `contract_request` JSON also now includes a `kind` and structured `groups` entries for periodic and event request tags such as reporting interval, report moduli, aircraft-intent projection time, lateral-deviation thresholds, vertical-speed thresholds, altitude ranges, and waypoint-change triggers.

The ADS-C backend now also decodes air-reference Mach values with the correct 0.0005 Mach resolution. This fixes downlink displays where aircraft such as `9V-SKU` were previously shown at exactly double the real Mach value in the HTML viewer.

In the HTML viewer, ADS-C label `B6` downlink messages now get the same expanded raw-text treatment as `A6`: instead of showing only the opaque hex block, the viewer renders a libacars-style indented text view built from the parsed ADS-C result, including the downlink type, basic report, flight IDs, airframe ID, earth-reference data, air-reference data, meteo data, and predicted route when those tags are present.

For nested `dumphfdl` JSON lines carrying HFDL `hfnpdu` data types such as `Frequency data` or `Performance data`, the extractor now preserves a synthetic `HFDL` message when `hfnpdu.flight_id` and `hfnpdu.pos.lat/lon` are present. This allows such rows to survive extraction with `flight_id`, coordinates, ICAO address, and a parsed `hfdl_data` result that the HTML viewer can place on the map.

The compact H1 `EB00` and `SB01` parsers now also emit `msg_type: "EBSB"` in their parsed JSON results while keeping their existing parser identities unchanged.

The HTML viewer also gives `hfdl_data` rows a dedicated summary, parsed-details block, and map popup so `hfnpdu_type`, `flight_id`, ICAO address, ground station, synthetic HFDL text, and coordinates are readable without digging through the flattened raw JSON. On the map, direct HFDL points also use a distinct teal marker so they stand out from the generic ACARS position markers.

For H2 wind messages that begin with `02A` or `02D`, the backend now parses the short start-position wind blocks before the coordinate-bearing points as structured `initial_layers` instead of misreading them as flight levels. Each layer preserves the altitude in feet, a converted altitude in metres, signed temperature in Celsius with one decimal place, and wind direction and speed with a derived km/h value. In the HTML viewer map popup, those `initial_layers` are shown only on the direct start-position marker for the H2 row, while the later coordinate-bearing route points keep their own shorter per-point wind displays. Route-point popups in the HTML viewer now also display altitude in metres alongside flight levels for H2 wind messages.

For FST01 fixed-layout flight status reports, the backend now also accepts the observed variant where the third field before the temperature carries a suffix such as `145M`. Those rows continue through the compact decoder, which preserves separate `track` and `heading` values in the observed field order instead of falling back to the older heuristic path that could conflate them. For the combined-temperature A350-style compact variant, the backend also now decodes the reported wind direction using the same `track`/`heading` disambiguation rule used in the viewer script, so samples such as `M51C045098113111537` no longer expose the raw encoded bearing as the final wind direction.

FST coordinate decoding now also respects the actual coordinate field width instead of always assuming four decimal places. This fixes five-digit latitude samples such as `N46976`, which should decode to `46.976` rather than `4.6976`.

H1 messages that carry `ABS0` blocks now also have a dedicated `abs` parser. It extracts `origin`, `destination`, `route`, and an optional three-digit `level` from the ABS0 line or the following line, which lets the state tracker learn route hints from messages such as `ABS001DA_T       EBBRLTFJ537`. It also decodes ABS0 position rows where the second line starts with compact `lat lon` values such as `44391  19636`, exposing `latitude: 44.391`, `longitude: 19.636`, the five-digit `altitude_ft` immediately before the first signed temperature token, that `temperature_c` value, and a `positions` array when the block carries multiple position rows.

VDL2 logs that carry a nested ATN CM logon request now also emit a synthetic `ATNCM` message when the nested context-management block contains a `departure_airport` and `destination_airport`. The dedicated `atncm` parser exposes `flight_id`, `origin`, `destination`, and `route`, so route learning can use ATN CM logon data even when there is no useful ACARS free text to parse.

When the input contains `message.flight` with a leading two-character IATA airline designator from the embedded mapping followed by digits, the emitted JSON normalises that value to the matching three-letter ICAO airline code. This includes alphanumeric designators such as `2C -> CMA` and `2G -> HUA`. The backend also strips leading zeros from the numeric part of `flight` values, so `AEE01BS` becomes `AEE1BS`. The `flight_id` field is preserved as received.

The loadsheet parser now applies the same backend normalisation to the route tuple it extracts from IATA-formatted loadsheet text. Inputs such as `U21234/... LTN DUB` now emit `flight: "EZY1234"`, `origin: "EGGW"`, and `destination: "EIDW"`, which lets the parsed result compare directly against ICAO-style route rows in `flightroute.sqb`.

The loadsheet parser also now recognises the multiline header variant where the flight and date are on one line and the IATA route is on the next line, for example `WFL2093/26  26APR26` followed by `MAD CLO ...`. In that case the parser emits `flight: "WFL2093"`, `origin: "LEMD"`, and `destination: "SKCL"`, and it prefers the explicit `PAX` line over earlier generic `TTL` totals such as `TTL 24217`.

The loadsheet parser now also accepts two additional header variants that occur in airline-specific formats: a slash-separated route line such as `AMS/MCT A4OSH CREW 2/8` after a flight header like `WY172/26 26APR26 2044`, and a multiline header where the flight line carries an extra aircraft token such as `EY844/26 26APR26 B789` before the next line starts with `SVO AUH ...`. For passenger totals, explicit forms such as `PAX TTL 187`, `PAX/24/253 TTL 281`, and `PAX 277 PLUS 4` now take precedence over earlier generic `TTL` cargo or payload totals.

For multiline Etihad-style headers such as `EY031/19 20MAR26 B789` followed by `AUH CDG ...`, the parser now keeps that explicit header flight and route even when later cargo or service rows contain airport-like triplets such as `CDG FRE 15852`. That prevents payload counts from overwriting the true flight and route.

The loadsheet parser also accepts compact headers where the flight, date, and IATA route are concatenated on one line, such as `AF0274/26/26APR26CDGHNDF-GSQK4/13`. In that case it extracts `flight: "AFR274"`, `origin: "LFPG"`, and `destination: "RJTT"`, and it honours explicit four-segment passenger totals such as `PAX/4/60/33/200 TTL 300.`.

The parser also accepts `FINAL01`-style loadsheet headers where the flight and compact six-letter IATA route share the same line, such as `FINAL01 EK784/26    LOSDXB A6ENN 26APR26`. In that case it extracts `flight: "UAE784"`, `origin: "DNMM"`, and `destination: "OMDB"`, and it prefers the explicit total in lines such as `PAX  7/26/238       TTL 273` over the partial category sum.

The loadsheet parser also accepts airline headers where the `FLIGHT:` field carries the flight and date together, followed by a compact six-letter IATA route, such as `FLIGHT:MU2076/04APR26   SVOPKX B6083`. In that case it extracts `flight: "CES2076"`, `origin: "UUEE"`, and `destination: "ZBAD"` while keeping the usual airline and airport normalisation.

The loadsheet parser also accepts compact inline headers where the flight and a six-letter IATA route share the same line without an explicit date token after the slash count, such as `UU969/04 RUNCDG F-OLRD 3/12`. In that case it extracts `flight: "REU969"`, `origin: "FMEE"`, and `destination: "LFPG"` from the same header line.

The loadsheet parser also accepts multiline headers where the flight line carries a slash-date plus a repeated date token before the route appears on the next line, such as `ET863/01APR26 01APR26 EDNO-2` followed by `ADD LUN ...`. In that case it extracts `flight: "ETH863"`, `origin: "HAAB"`, and `destination: "FLLS"`, and it also tolerates edition markers such as `EDNO-2`.

The loadsheet parser also accepts slash-delimited operational summaries where the flight reference appears at the end of the first line and the sector is carried in a dedicated `SEC/` field, such as `/BA0085` with `SEC/LHR-YVR`. In that case it extracts `flight: "BAW85"`, `origin: "EGLL"`, and `destination: "CYVR"`, and it also reads slash-delimited values such as `ZFW/181123`, `TOW/248325`, `FWT/068024`, and `CRW/03/10`.

The parser also accepts tabular loadsheet formats where the route and flight appear in a `FROM/TO FLIGHT` row such as `BCN SCL LL 2605 ...`, and where passenger totals are carried on a `PASSENGER/CABIN BAG ... TTL 236` line rather than a plain `PAX TTL` line. In that case it extracts `flight: "LVL2605"`, `origin: "LEBL"`, `destination: "SCEL"`, and it also reads verbose weight labels such as `ZERO FUEL WEIGHT ACTUAL`, `TAKE OFF FUEL`, and `TAKE OFF WEIGHT ACTUAL`.

The same tabular logic also accepts rows where the flight token is already combined, such as `MAD CLO WFL2093 ...`, instead of being split as an airline code plus a separate flight number. In that case it extracts `flight: "WFL2093"`, `origin: "LEMD"`, and `destination: "SKCL"` from the table row.

**Options:**
- `-input FILE` - Input JSONL file (default: stdin)
- `-output FILE` - Output JSON file (default: stdout)
- `-format FORMAT` - Output format: `json` (default) or `text`
- `-pretty` - Pretty print JSON output
- `-all` - Include all parsed data types

### live

Connects to a live NATS feed and displays parsed messages in real-time.

```bash
./acars_parser live -creds credentials.creds [options]
```

**Options:**
- `-creds FILE` - Path to NATS credentials file (required)
- `-server URL` - NATS server URL (default: `nats://157.90.242.138:4222`)
- `-subject SUBJ` - NATS subject to subscribe to (default: `v1.aircraft.ingest.*.message.*.created`)
- `-output FILE` - Optional JSONL output file
- `-db FILE` - SQLite database for message storage (default: `messages.db`)
- `-state FILE` - SQLite database for flight state tracking (default: `state.db`)
- `-no-store` - Disable all database storage
- `-all` - Show all messages with text, not just parsed ones
- `-raw` - Show raw message text
- `-empty` - Show empty/missing fields to identify unparsed data
- `-exclude TYPES` - Exclude result types from display (default: `sq_position`). Use `-exclude ""` to show all.
- `-debug LABELS` - Debug specific labels (comma-separated, e.g. `80,B6,H1`)
- `-v` - Verbose output

### query

Query stored messages in SQLite database.

```bash
./acars_parser query [options]
```

**Options:**
- `-db FILE` - SQLite database file (default: `messages.db`)
- `-id N` - Fetch a specific message by database row ID
- `-msg-id N` - Fetch by ACARS message ID (from parsed JSON)
- `-type TYPE` - Filter by parser type (e.g. `h1_position`, `pdc`)
- `-label LABEL` - Filter by ACARS label (e.g. `H1`, `16`)
- `-flight TEXT` - Filter by flight number (partial match)
- `-missing FIELD` - Filter by specific missing field
- `-has-missing` - Only show messages with any missing fields
- `-search TEXT` - Full-text search on raw message text
- `-limit N` - Max results to return (default: 20)
- `-offset N` - Pagination offset
- `-order FIELD` - Sort by field: id, timestamp, parser_type, confidence (default: `id`)
- `-desc` - Sort descending (default: true)
- `-raw` - Show raw message text
- `-json` - Output as JSON
- `-stats` - Show database statistics only
- `-list-types` - List all parser types in the database
- `-list-missing` - List top missing fields across all messages

### reparse

Re-parse stored messages to compare old vs new parsing results.

```bash
./acars_parser reparse [options]
```

**Options:**
- `-db FILE` - SQLite database file (default: `messages.db`)
- `-type TYPE` - Filter by parser type
- `-label LABEL` - Filter by ACARS label
- `-v` - Verbose output: show detailed diffs
- `-regressions-only` - Show only messages that regressed
- `-improvements-only` - Show only messages that improved
- `-limit N` - Limit number of messages to process (0 = all)
- `-json` - Output as JSON
- `-update` - Update database with new parsed results

### debug

Debug why a message didn't parse correctly.

```bash
./acars_parser debug -id N [options]
./acars_parser debug -text "MESSAGE TEXT" [-label LABEL] [options]
```

**Options:**
- `-db FILE` - SQLite database file (default: `messages.db`)
- `-id N` - Message ID to debug
- `-text TEXT` - Raw message text to debug (instead of -id)
- `-label LABEL` - ACARS label for raw text (e.g. `H1`, `16`)
- `-all` - Show all pattern attempts, not just matches
- `-type TYPE` - Only show trace for specific parser type (e.g. `pdc`)

### backfill

Populate state tracker from existing parsed messages.

```bash
./acars_parser backfill [options]
```

**Options:**
- `-db FILE` - SQLite database with parsed messages (default: `messages.db`)
- `-state FILE` - SQLite database for flight state (default: `state.db`)
- `-type TYPE` - Filter by parser type
- `-limit N` - Limit number of messages (0 = all)
- `-v` - Verbose output

### review

Launch web UI for reviewing and annotating messages.

```bash
./acars_parser review [options]
```

**Options:**
- `-db FILE` - SQLite database file (default: `messages.db`)
- `-port N` - HTTP port (default: 8080)
- `-type TYPE` - Pre-filter to specific parser type

### templates

Discover message format templates by normalising messages.

```bash
./acars_parser templates [options]
```

**Options:**
- `-db FILE` - SQLite database file (default: `messages.db`)
- `-type TYPE` - Filter by parser type
- `-label LABEL` - Filter by ACARS label
- `-limit N` - Limit number of messages (0 = all)
- `-min N` - Minimum messages per template to show (default: 2)
- `-examples N` - Number of example messages per template (default: 1)
- `-v` - Verbose output: show full template strings

## Supported Message Types

### PDC (Pre-Departure Clearance)
Extracts flight number, origin/destination, runway, SID, squawk code, and frequencies from pre-departure clearances.

### Route (5L)
Parses route messages containing callsign, origin/destination airports (IATA/ICAO), and scheduling data.

### Label 16 Position
Parses classic waypoint position reports, `POSA` position reports, and multiline `POS02` position reports. `POS02` parsing now extracts the flight, start date, end date, header message time, an ICAO origin-destination route in `XXXX-XXXX` format, coordinates, altitude in feet, derived flight level, Mach number, and outside-air temperature while classifying the parsed result with `message_type: "pos"`.

### ILNGE7X Summary
Parses `/ILNGE7X.` summary messages and extracts the tail, flight, take-off date/time, and origin-destination route. The parsed JSON result also emits `msg_type: "ILNGE"`.

### REP301
Parses compact `REP301` reports regardless of the ACARS label. Extracts the route, origin/destination, latitude/longitude, report time, flight level in tenths, temperature in Celsius, and wind direction/speed in knots and km/h. The parsed JSON result also emits `msg_type: "REP301"`.

### Position (80)
Extracts current position (lat/lon), altitude, ground speed, and flight routing.

### ADS-C (B6)
Parses ADS-C (Automatic Dependent Surveillance - Contract) position reports using tag-based binary parsing based on libacars. Extracts:
- **Position data**: latitude, longitude, altitude, report timestamp, position accuracy (0-7)
- **Meteorological data** (tag 16): wind speed, wind direction, temperature
- **Earth reference** (tag 14): true track, ground speed, vertical speed
- **Air reference** (tag 15): true heading, mach number, vertical speed
- **Predicted route** (tag 13): next waypoint lat/lon/alt/ETA, next+1 waypoint coordinates
- **Flight ID** (tag 12): ISO5-encoded flight identifier
- **Airframe ID** (tag 17): ICAO hex address

### Flight Plan (H1 FPN)
Extracts flight plan data including waypoints, origin/destination, and route information. The parsed JSON result also emits `msg_type: "FPN"`.

### H1 Position (H1 POS)
Parses H1 position reports with current/next waypoint, altitude, and coordinates.
The parsed JSON result also emits `msg_type: "POS"`.

### SB01 (H1)
Parses compact `SB01` status messages carried on label `H1`. Extracts the registration, route, latitude/longitude, report time, altitude in feet and metres, temperature in Celsius, and wind direction/speed in knots and km/h.
```
SB0122BA_F-GZNG LFPOFMEE195 42703 0184101832 31001-550356015010GMY012015
```

### EB00 (H1)
Parses compact `EB00` status messages carried on label `H1`. Extracts the aircraft registration, route, message number, latitude/longitude, report time, altitude in feet and metres, temperature in Celsius, and wind direction/speed in knots and km/h.
```
EB0032AA_ D-ABPR VABBEDDF 44 45570 0236890440 35998-633193015090W/X014022
```

### PWI - Predicted Wind Information (H1)
Extracts wind and temperature forecasts along the route:
- **Climb winds (CB)**: Wind direction/speed at various altitudes during climb
- **Route winds (WD)**: Wind direction/speed/temperature at waypoints for each flight level
- **Descent winds (DD)**: Wind direction/speed at various altitudes during descent

The parsed JSON result also emits `msg_type: "PWI"`.
Route-wind waypoints may be named fixes or compact coordinates such as `N24012E078331`; coordinate waypoints are exported with latitude/longitude so the viewer can draw them on the map.

Example PWI data structure:
```json
{
  "climb_winds": [
    {"flight_level": 100, "wind_dir": 252, "wind_speed": 39},
    {"flight_level": 310, "wind_dir": 261, "wind_speed": 84}
  ],
  "route_winds": [
    {
      "flight_level": 360,
      "waypoints": [
        {"waypoint": "DOLEV", "wind_dir": 321, "wind_speed": 74, "temperature": -57},
        {"waypoint": "ROTAR", "wind_dir": 303, "wind_speed": 85, "temperature": -63}
      ]
    }
  ],
  "descent_winds": [
    {"flight_level": 100, "wind_dir": 305, "wind_speed": 22},
    {"flight_level": 350, "wind_dir": 300, "wind_speed": 76}
  ]
}
```

### Waypoint Position (16)
Extracts waypoint crossing reports with position and timing.

### Position Report (21)
Parses position reports with coordinates, altitude, and destination.

### Oceanic Clearance (B2)
Extracts oceanic clearance data including track, flight level, and Mach number.
The parsed JSON result also emits `msg_type: "OCEANIC_CLEARANCE"`.

### Gate Info (B3)
Parses gate information messages with flight number and gate assignment.
The parsed JSON result also emits `msg_type: "GATEINFO"`.

### Position + Weather (4J)
Extracts combined position and weather data.

### SQ - ARINC Position (96k messages)
Parses squitter messages containing airport IATA/ICAO mapping and position data.
```
02XAORDKORD54158N08754WV136975/ARINC
```

### Label 10 - Rich Position/Route (10k messages)
Parses position reports with full route picture including waypoint timing.
```
/N40.024/W073.100/10/0.72/230/430/KISM/2057/0064/00015/ZIZZI/TBONN/1831/
```

### Label 4T - AGFSR Flight Status (2.6k messages)
Parses comprehensive flight status messages with route, position, fuel, wind, and ETA.
```
AGFSR AC1204/29/29/YULMIA/1829Z/110/3457.3N07711.0W/300/CRUISE/0067/0052/M37/248095/0300/202/02/1432/1640/
```
The parsed JSON result also emits `msg_type: "AGFSR"`.

### Label 22 - Detailed Position (13k messages)
Parses detailed position reports in degrees/minutes/seconds format.
```
N 325338W 971058,-------,182836,9977, ,      , ,M  3,31104  41,  64,
```

### Label 5Z - ETA/Timing (21k messages)
Parses ETA and timing messages in various formats (ET, IR, B6, OS, C3).
```
/ET EXP TIME       / KSNA KIAH 29 182901/EON 1908 AUTO
```

### Label 15 - FST Reports (14k messages)
Parses flight status reports with route, position, temperature, and FST01 fixed-layout wind and speed data.
```
FST01EGLCEIDWN51420W00049317803270072M020C014331258256370
```

For the space-delimited FST01 layout, the parser extracts the route, coordinates, flight level, temperature, wind direction, track, heading, and ground speed. Wind speed and ground speed are exposed via `wind_speed_kts` / `wind_speed_kmh` and `ground_speed_kts` / `ground_speed_kmh`, while the JSON output keeps the route as a single field instead of repeating separate origin and destination keys. The parsed JSON result also emits `msg_type: "FST"`.

### Label 83 - Position Reports (3.6k messages)
Parses PR and ZSPD position report formats.
```
001PR29182854N5106.0W11400.4035000----
```

### H2 - Wind Data
Parses wind/weather data with multiple altitude layers.
```
02A291829EDDKLSZHN50529E007101291809   6M005   48P002290008G
```
The parsed JSON result also emits `msg_type: "H2WIND"`.

### Label 44 - Runway/Procedure Info (3k messages)
Parses runway takeoff information, FB positions, and POS reports.
```
KLGA T/O RWYS,04                  7002
```

### ATIS (A9)
Parses ATIS (Automatic Terminal Information Service) weather reports with runway, wind, visibility, and QNH data.

### Envelope (AA, A6)
Parses envelope-formatted messages containing aircraft position and status data.

### Gate Assignment (RA)
Parses gate assignment messages with flight and gate information.
The parsed JSON result also emits `msg_type: "GATEASSIGN"`.

### Landing Data (C1)
Parses landing performance data including runway, approach, and configuration.
The parsed JSON result also emits `msg_type: "LANDINGDATA"`.

### Loadsheet (All Labels)
Parses aircraft loadsheet messages with weight and balance information regardless of ACARS label. The parser recognises both `LOADSHEET` and spaced `L O A D S H E E T` headers, and the parsed JSON result also emits `msg_type: "LOADSHEET"`.

### Turbulence (C1)
Parses turbulence reports with severity and location data.

### Weather (RA, C1)
Parses general weather observation messages with temperature, wind, and conditions.

### Media Advisory (SA)
Parses data link status messages reporting which communication links (VHF, SATCOM, HF, VDL2, etc) are available or unavailable. Based on libacars media-adv format.
```
0EV095905V
```
Extracts: link status (established/lost), current link type, timestamp, available links, and a human-readable `formatted_text` rendering.

Example decoded text:
```
0EH103440VSH/
 Media Advisory, version 0:
  Link HF established at 10:34:40 UTC
  Available links: VHF ACARS, Default SATCOM, HF
```

In `extract -format text`, SA payloads are rendered in this human-readable form after the raw ACARS payload. The HTML viewer raw-text pane also expands SA messages in the same style.

### CPDLC - Controller-Pilot Data Link Communications (AA)
Parses FANS-1/A CPDLC messages using pure Go ASN.1 PER decoding (no libacars dependency). Supports:
- **Downlink messages** (dM0-dM80): Pilot responses/requests to ATC
- **Uplink messages** (uM0-uM182): ATC instructions/requests to aircraft
- **Connection management**: Connect requests (CR1), connect confirms (CC1), disconnect (DR1)

Message format:
```
/BOMCAYA.AT1.A4O-SI005080204A
```
Structure: `/<station>.<type>.<registration><hex_data>`

**Decoded element types include:**
- Altitudes (flight level, feet, metres, QNH/QFE/GNSS)
- Speeds (knots, Mach, km/h)
- Positions (fix, navaid, airport, lat/lon, place-bearing-distance)
- Route clearances (departure/arrival airports, runways, SIDs/STARs, airways)
- Frequencies (VHF, UHF, HF, SATCOM)
- Free text messages
- Error information
- Vertical rates, beacon codes, ATIS codes, and more

Example decoded output:
```json
{
  "message_type": "cpdlc",
  "direction": "downlink",
  "header": {"msg_id": 0},
  "elements": [{
    "id": 80,
    "label": "DEVIATING [distanceoffset] [direction] OF ROUTE",
    "text": "DEVIATING 1 km south OF ROUTE"
  }]
}
```

**Limitations:**
- Multi-element messages (containing 2-5 elements) currently only decode the primary element
- Some complex route information types (placeBearingPlaceBearing, trackDetail, holdAtWaypoint) return placeholder text

## Output Format

The `extract` command outputs JSON by default. When `-format text` is selected, it prints the raw ACARS payload followed by any available human-readable parser rendering, such as the expanded SA Media Advisory text.

JSON output example:

```json
{
  "stats": {
    "total_messages": 794302,
    "parsed_pdcs": 1234,
    "parsed_pwi": 2706,
    ...
  },
  "pwi_reports": [...],
  "pdcs": [...]
}
```

The live command outputs human-readable summaries:
```
[UAL123 N12345 737-800] [PWI] CB:FL100-350 WD:FL360 (3 wpts) DD:FL100-390
[DAL456 N67890] [PDC] DAL456 KJFK->KLAX RWY 31L SID DEEZZ5 SQK 1234
```

---

## Developer Guide

### Application Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│  cmd/acars_parser/main.go                                               │
│  - Entry point, imports internal/parsers for side-effect registration  │
│  - Calls registry.Default().Sort() to prepare parsers                  │
│  - Routes to extract.go or live.go based on subcommand                 │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┴───────────────┐
                    ▼                               ▼
    ┌──────────────────────────┐    ┌──────────────────────────┐
    │  cmd/.../extract.go      │    │  cmd/.../live.go         │
    │  - Reads JSONL files     │    │  - Connects to NATS      │
    │  - Batch processing      │    │  - Real-time streaming   │
    │  - JSON output           │    │  - Console output        │
    └──────────────────────────┘    └──────────────────────────┘
                    │                               │
                    └───────────────┬───────────────┘
                                    ▼
    ┌─────────────────────────────────────────────────────────────────────┐
    │  internal/registry/registry.go                                      │
    │  - Dispatch(msg) routes messages to matching parsers                │
    │  - Parsers registered via init() in each parser package            │
    │  - Label-based routing (fast) + global parsers (content-based)     │
    └─────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
    ┌─────────────────────────────────────────────────────────────────────┐
    │  internal/parsers/*/parser.go                                       │
    │  - Each parser implements: Name(), Labels(), QuickCheck(), Parse() │
    │  - Returns a Result struct with Type() and MessageID()             │
    └─────────────────────────────────────────────────────────────────────┘
```

### Key Files

| File | Purpose |
|------|---------|
| `cmd/acars_parser/main.go` | Entry point, subcommand routing |
| `cmd/acars_parser/extract.go` | Batch extraction from JSONL files |
| `cmd/acars_parser/live.go` | Real-time NATS streaming, console output |
| `internal/acars/message.go` | ACARS message types (`Message`, `NATSWrapper`, `Airframe`, `Flight`) |
| `internal/registry/registry.go` | Parser registry, `Dispatch()` routing logic |
| `internal/parsers/parsers.go` | Blank import to trigger all parser `init()` registrations |
| `internal/patterns/patterns.go` | Shared regex patterns (coordinates, flight numbers, etc.) |
| `internal/patterns/extractors.go` | Shared extraction functions |

### Parser Locations

Each parser lives in `internal/parsers/<name>/parser.go`:

| Parser | Label(s) | Result Type | File |
|--------|----------|-------------|------|
| ADS-C | `B6` | `adsc` | `internal/parsers/adsc/parser.go` |
| AGFSR | `4T` | `agfsr` | `internal/parsers/agfsr/parser.go` |
| ATIS | `A9` | `atis` | `internal/parsers/atis/parser.go` |
| CPDLC | `AA` | `cpdlc`, `connect_request`, `connect_confirm`, `disconnect` | `internal/parsers/cpdlc/parser.go` |
| Envelope | `AA`, `A6` | `envelope` | `internal/parsers/envelope/parser.go` |
| ETA | `5Z` | `eta` | `internal/parsers/eta/parser.go` |
| FST | `15` | `fst` | `internal/parsers/fst/parser.go` |
| Gate Assignment | `RA` | `gate_assignment` | `internal/parsers/gateassign/parser.go` |
| H1 FPN | `H1`, `4A`, `HX` | `flight_plan` | `internal/parsers/h1/parser.go` |
| H1 POS | `H1` | `h1_position` | `internal/parsers/h1/parser.go` |
| H1 PWI | `H1` | `pwi` | `internal/parsers/h1/parser.go` |
| H2 Wind | `H2` | `h2_wind` | `internal/parsers/h2wind/parser.go` |
| Label 10 | `10` | `label10_position` | `internal/parsers/label10/parser.go` |
| Label 16 | `16` | `waypoint_position` | `internal/parsers/label16/parser.go` |
| Label 21 | `21` | `position_report` | `internal/parsers/label21/parser.go` |
| Label 22 | `22` | `label22_position` | `internal/parsers/label22/parser.go` |
| Label 44 | `44` | `label44` | `internal/parsers/label44/parser.go` |
| Label 4J | `4J` | `pos_weather` | `internal/parsers/label4j/parser.go` |
| Label 5L | `5L` | `route` | `internal/parsers/label5l/parser.go` |
| Label 80 | `80` | `position` | `internal/parsers/label80/parser.go` |
| Label 83 | `83` | `label83_position` | `internal/parsers/label83/parser.go` |
| Label B2 | `B2` | `oceanic_clearance` | `internal/parsers/labelb2/parser.go` |
| Label B3 | `B3` | `gate_info` | `internal/parsers/labelb3/parser.go` |
| Landing Data | `C1` | `landing_data` | `internal/parsers/landingdata/parser.go` |
| Loadsheet | `all` | `loadsheet` | `internal/parsers/loadsheet/parser.go` |
| Media Advisory | `SA` | `media_advisory` | `internal/parsers/mediaadv/parser.go` |
| PDC | *(content-based)* | `pdc` | `internal/parsers/pdc/parser.go` |
| SQ | `SQ` | `sq_position` | `internal/parsers/sq/parser.go` |
| Turbulence | `C1` | `turbulence` | `internal/parsers/turbulence/parser.go` |
| Weather | `RA`, `C1` | `weather` | `internal/parsers/weather/parser.go` |

### Adding a New Parser

1. Create directory: `internal/parsers/<name>/`
2. Create `parser.go` implementing the `registry.Parser` interface:

```go
package myparser

import (
    "acars_parser/internal/acars"
    "acars_parser/internal/registry"
)

type Result struct {
    MsgID     int64  `json:"message_id"`
    Timestamp string `json:"timestamp"`
    // ... your fields
}

func (r *Result) Type() string     { return "my_type" }
func (r *Result) MessageID() int64 { return r.MsgID }

type Parser struct{}

func init() {
    registry.Register(&Parser{})
}

func (p *Parser) Name() string     { return "myparser" }
func (p *Parser) Labels() []string { return []string{"XX"} } // or empty for content-based
func (p *Parser) Priority() int    { return 100 }

func (p *Parser) QuickCheck(text string) bool {
    return strings.Contains(text, "MYPREFIX") // fast string check, no regex
}

func (p *Parser) Parse(msg *acars.Message) registry.Result {
    // Parse logic here
    // Return nil if message doesn't match
    return &Result{...}
}
```

3. Add import to `internal/parsers/parsers.go`:
```go
_ "acars_parser/internal/parsers/myparser"
```

### Parser Interface

```go
type Parser interface {
    Name() string           // Unique identifier
    Labels() []string       // ACARS labels to match (empty = content-based, checks all)
    QuickCheck(text string) bool  // Fast pre-filter (use strings.Contains, not regex)
    Priority() int          // Lower = checked first
    Parse(msg *acars.Message) Result  // Returns nil if not applicable
}
```

### Registry Dispatch Order

1. **Label-specific parsers** - Matched by `msg.Label`, sorted by priority
2. **Global parsers** - Content-based parsers (empty `Labels()`), check all messages
3. **Catch-all parsers** - Only run if nothing else matched

Multiple parsers can return results for the same message.
