# Quickstart (Windows)

## 1) Install Go
Install a recent Go version and ensure `go version` works in PowerShell.

## 2) Build the parser
Open PowerShell in the project folder and run:

```powershell
go mod download
go build -o acars_parser.exe .\cmd\acars_parser
```

## 3) Run extract (CLI)
```powershell
.\acars_parser.exe extract -input messages.jsonl -output out.json -pretty -all
```

## 4) Run GUI wrapper
```powershell
python .\gui\acars_parser_gui.py
```

## 5) HTML viewer waypoint lookup

The standalone viewer [gui/acars_viewer_fast_kv_map_v8.html](gui/acars_viewer_fast_kv_map_v8.html) can resolve named FPN and PWI route waypoints through [gui/Waypoints.txt](gui/Waypoints.txt), so those points can be drawn on the map even when the JSON only contains waypoint names.

The viewer also supports filtering rows that have a Flight value and sorting the Summary / Details column by the amount of message content, which is useful for surfacing the densest ACARS messages first.

If the viewer is opened via a local `file:` URL, some browsers block automatic loading of sibling files. In that case, use the `Load Waypoints` control in the viewer and select `gui/Waypoints.txt` manually.

Notes:
- Input must be JSONL (one JSON object per line) matching either:
  - `internal/acars.Message`, or
  - NATS wrapper (`internal/acars.NATSWrapper`) with `message{...}` inside.
