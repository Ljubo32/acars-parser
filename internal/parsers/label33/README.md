# Label 33 Parser

Parser za ACARS Label 33 poruke koje sadrže informacije o poziciji aviona u CSV formatu.

## Format Poruke

Label 33 poruke koriste CSV (Comma-Separated Values) format sa sledećim poljima:

```
DATUM,VREME,ORIGIN,DEST,NEPOZNATO,KOORDINATE,GS,FL,FOB,TEMP,WIND_DIR,WIND_SPEED,NEXT_WPT,ETA,FOLLOW_WPT,...
```

### Primer Poruke

```
2026-01-21,10:01:56,LLBG,EGLL,0315,N44064E018441,492,FL360,0180,-65,161, 19,BEDOX ,10:23,SIMBA ,-36,846,282,474,-049,-048,-043,024,210126
```

## Parsirana Polja

Parser ekstraktuje sledeće podatke:

| Polje | Opis | Jedinica | Primer |
|-------|------|----------|--------|
| `date` | Datum izveštaja | YYYY-MM-DD | 2026-01-21 |
| `time` | Vreme izveštaja | HH:MM:SS | 10:01:56 |
| `origin_icao` | ICAO kod polaznog aerodroma | - | LLBG |
| `origin_name` | Naziv polaznog aerodroma | - | Ben Gurion |
| `dest_icao` | ICAO kod odredišnog aerodroma | - | EGLL |
| `dest_name` | Naziv odredišnog aerodroma | - | London Heathrow |
| `latitude` | Geografska širina | stepeni | 44.107 |
| `longitude` | Geografska dužina | stepeni | 18.735 |
| `ground_speed_kts` | Brzina po zemlji | čvorovi (knots) | 492 |
| `flight_level` | Nivo leta | FL | 360 (36000 ft) |
| `fuel_on_board` | Gorivo na brodu | kg/lbs | 180 |
| `temperature_c` | Temperatura vazduha | °C | -65 |
| `wind_dir` | Pravac vetra | stepeni | 161 |
| `wind_speed_kts` | Brzina vetra | čvorovi | 19 |
| `wind_speed_kmh` | Brzina vetra | km/h | 35 |
| `next_waypoint` | Sledeći waypoint | - | BEDOX |
| `next_wpt_eta` | ETA do sledećeg waypointa | HH:MM | 10:23 |
| `follow_waypoint` | Waypoint posle sledećeg | - | SIMBA |

## Format Koordinata

Koordinate su kodirane u formatu:
- **Širina (Latitude)**: `NDDMMM` ili `SDDMMM` gde je:
  - `N`/`S` - smer (North/South)
  - `DD` - stepeni (2 cifre)
  - `MMM` - minute * 10 (3 cifre, npr. 350 = 35.0 minuta)

- **Dužina (Longitude)**: `EDDDMMM` ili `WDDDMMM` gde je:
  - `E`/`W` - smer (East/West)
  - `DDD` - stepeni (3 cifre)
  - `MMM` - minute * 10 (3 cifre) ili `MMMM` (4 cifre, format MMDD)

### Primeri Koordinata

- `N43350` = 43°35.0' = 43.583° N
- `E021400` = 21°40.0' = 21.667° E
- `N44064` = 44°06.4' = 44.107° N
- `E018441` = 18°44.1' = 18.735° E

## Konverzije

Parser automatski vrši sledeće konverzije:
- **Brzina vetra**: čvorovi → km/h (1 knot = 1.852 km/h)
- **Flight Level**: FL360 → 360 (broj se parsira)
- **Naziv aerodroma**: ICAO kod → pun naziv iz baze aerodroma

## Testiranje

Za testiranje parsera:

```bash
go test ./internal/parsers/label33/... -v
```

## Primer Korišćenja

Parser se automatski registruje i koristi kroz glavni parser sistem:

```go
import (
    "acars_parser/internal/acars"
    "acars_parser/internal/registry"
    _ "acars_parser/internal/parsers/label33"  // registruje parser
)

msg := &acars.Message{
    Label: "33",
    Text:  "2026-01-21,10:01:56,LLBG,EGLL,0315,N44064E018441,492,FL360,0180,-65,161,19,BEDOX,10:23,SIMBA",
}

results := registry.Default().Dispatch(msg)
for _, result := range results {
    if posResult, ok := result.(*label33.Result); ok {
        fmt.Printf("Position: %.4f, %.4f\n", posResult.Latitude, posResult.Longitude)
        fmt.Printf("Speed: %d kts\n", posResult.GroundSpeed)
        fmt.Printf("Next waypoint: %s (ETA: %s)\n", posResult.NextWaypoint, posResult.NextWptETA)
    }
}
```

## Napomene

- Polje 5 (nakon odredišta) je nepoznato i preskače se
- Dodatna polja posle waypointa nisu dekodirane jer njihovo značenje nije poznato
- Gorivo može biti u različitim jedinicama u zavisnosti od aviona (kg ili lbs)
- Parser podržava i kraće verzije poruka sa manje polja
