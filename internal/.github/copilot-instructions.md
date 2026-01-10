# ACARS Parser - AI Coding Agent Instructions

## Project Overview

This is an ACARS (Aircraft Communications Addressing and Reporting System) message parser written in Go. It implements a plugin-style registry system where specialized parsers decode aviation messages from aircraft. Each parser targets specific ACARS label types (H1, B2, PDC, etc.) and extracts structured data like flight plans, positions, clearances, and weather.

## Architecture: Plugin-Based Registry Pattern

**Core Design**: Parsers self-register during `init()` to a global registry, which dispatches messages based on labels and content patterns.

### Registration Flow
1. Each parser package calls `registry.Register(&Parser{})` in its `init()` function
2. Registry organizes parsers by label (e.g., "H1", "B2") or marks them as content-based (global)
3. `parsers/parsers.go` imports all parser packages with blank imports (`_ "acars_parser/internal/parsers/pdc"`) to trigger registration

### Dispatch Pipeline (from [registry/registry.go](registry/registry.go))
```
Message → Registry.Dispatch() →
  1. Try label-specific parsers (e.g., msg.Label="H1" → H1Parser)
  2. Try global/content-based parsers (e.g., PDC parser checks all labels)
  3. Try catch-all parsers (envelope parser for unmatched messages)
```

**Performance**: Each parser implements `QuickCheck(text string) bool` using cheap string operations (`strings.Contains`, `HasPrefix`) before expensive regex. Lower `Priority()` = checked first.

## Creating a New Parser

### Required Interface (from [registry/registry.go](registry/registry.go#L18-L39))
```go
type Parser interface {
    Name() string               // Unique identifier, e.g., "pdc"
    Labels() []string           // Target labels ["H1", "4A"] or nil for content-based
    QuickCheck(text string) bool // Fast pre-filter, NO REGEX HERE
    Priority() int              // Lower = checked first (10-100 for label, 500+ for content-based)
    Parse(msg *acars.Message) Result
}

type Result interface {
    Type() string               // e.g., "flight_plan", "pdc"
    MessageID() int64
}
```

### Implementation Checklist
1. **Create package** in `parsers/<name>/parser.go`
2. **Define Result struct** with `Type()` and `MessageID()` methods
3. **Implement Parser interface** with fast `QuickCheck()` (see [parsers/pdc/parser.go](parsers/pdc/parser.go#L68-L105))
4. **Call `registry.Register()`** in `init()` function
5. **Add blank import** to [parsers/parsers.go](parsers/parsers.go)

### Example Pattern (from PDC parser)
```go
func init() {
    registry.Register(&Parser{})
}

func (p *Parser) QuickCheck(text string) bool {
    upper := strings.ToUpper(text)
    // Reject negative cases first
    if strings.Contains(upper, "NO PDC ON FILE") {
        return false
    }
    // Require word boundaries to avoid false positives in route strings
    return strings.Contains(upper, " PDC") || strings.HasPrefix(upper, "PDC")
}
```

## Pattern Matching: Grok-Style Compiler

**Location**: [patterns/compiler.go](patterns/compiler.go), [patterns/patterns.go](patterns/patterns.go)

Instead of raw regex, use the Grok-style pattern system with `{PLACEHOLDER}` syntax:

### Defining Patterns (example pattern format)
```go
var Formats = []patterns.Format{
    {
        Name: "fst_5digit_lon",
        Pattern: `FST(?P<seq>\d{2})(?P<origin>{ICAO})(?P<dest>{ICAO})` +
                 `(?P<lat_dir>{LAT_DIR})(?P<lat>\d{5,6})`,
        Fields: []string{"seq", "origin", "dest", "lat_dir", "lat"},
    },
}

// In parser init or sync.Once
compiler := patterns.NewCompiler(Formats, localPatterns)
compiler.Compile()  // Expands {ICAO} → actual regex from BasePatterns
```

### BasePatterns Reference (from [patterns/base_patterns.go](patterns/base_patterns.go))
- `{ICAO}` → 4-letter airport codes with valid prefixes
- `{LAT_DIR}` → N/S, `{LON_DIR}` → E/W
- `{FLIGHT}` → Airline code + digits (e.g., UAL123)
- `{WAYPOINT}` → Aviation fix names

**When to Use**: Message formats with fixed structures (FST, Label16, Label21). For freeform text (PDC), use shared patterns from [patterns/patterns.go](patterns/patterns.go).

## H1 Parsing: Tokeniser-Based Approach

**Unique Pattern**: H1 messages use ARINC 622/633 format with section markers (`:DA:`, `:AA:`, `:F:`). The [parsers/h1/tokeniser.go](parsers/h1/tokeniser.go) splits messages into sections before parsing.

### Tokeniser Example
```go
tokens := TokeniseFPN(msg.Text)  // Splits on :XX: markers
origin := tokens.GetOrigin()     // From :DA: section
dest := tokens.GetDestination()  // From :AA: section
route := tokens.GetRoute()       // From :F: section
```

**Key Function**: `NormaliseFPN()` strips transmission artifacts (`\r`, `\n`, `\t`) that can appear mid-field in ACARS messages.

### Coordinate Parsing
H1 waypoints embed coordinates in format `N31490E035327` (degrees + decimal minutes):
- Extract with regex, parse via `parseWaypointCoords()` (see [parsers/h1/parser.go](parsers/h1/parser.go))
- Convert ddmm.m format to decimal degrees

## Testing Conventions

### Test Structure (from [parsers/pdc/parser_test.go](parsers/pdc/parser_test.go))
```go
func TestParser(t *testing.T) {
    testCases := []struct {
        name string
        text string  // Raw ACARS message text
        want struct {
            flightNum string
            origin    string
            // ... expected fields
        }
    }{
        {
            name: "Jetstar YBBN to YMML PDC",  // Descriptive, includes carrier
            text: `.MELOJJQ 301036\nAGM\nAN VH-OFW/MA 511A...`,
            want: struct{...}{
                flightNum: "JST577",
                origin: "YBBN",
                ...
            },
        },
    }
}
```

**Test Data**: Use real-world examples with carrier names in test case descriptions. Include edge cases like truncated messages, missing fields, and format variations.

### Running Tests
```bash
go test ./internal/parsers/...           # All parsers
go test ./internal/parsers/pdc -v       # Specific parser with verbose
```

## Common Patterns & Conventions

### Message Metadata Extraction
```go
result.Tail = msg.Tail
if result.Tail == "" && msg.Airframe != nil {
    result.Tail = msg.Airframe.Tail  // Fallback to airframe data
}
if msg.Airframe != nil {
    result.AircraftICAO = msg.Airframe.ICAO
}
```

### ICAO Code Validation
Always use `ICAOBlocklist` from [patterns/patterns.go](patterns/patterns.go) to filter false positives:
```go
if ICAOBlocklist[code] {
    continue  // Skip "WILL", "WHEN", "PUSH", etc.
}
```

### Singleton Pattern for Grok Compilers
Compilation is expensive, use `sync.Once` (see [parsers/pdc/parser.go](parsers/pdc/parser.go#L47-L57)):
```go
var (
    grokCompiler *Compiler
    grokOnce     sync.Once
)

func getCompiler() (*Compiler, error) {
    grokOnce.Do(func() {
        grokCompiler = NewCompiler()
        grokErr = grokCompiler.Compile()
    })
    return grokCompiler, grokErr
}
```

## Key Files Reference

- **Registry core**: [registry/registry.go](registry/registry.go) - Dispatcher and parser interface
- **Pattern library**: [patterns/patterns.go](patterns/patterns.go) - Shared regex for runways, SIDs, squawk codes
- **Grok compiler**: [patterns/compiler.go](patterns/compiler.go) - Template-based pattern matching
- **Message types**: [acars/message.go](acars/message.go) - ACARS message structs
- **Parser registration**: [parsers/parsers.go](parsers/parsers.go) - Auto-registers all parsers
- **Reference implementations**: [parsers/pdc/parser.go](parsers/pdc/parser.go), [parsers/h1/parser.go](parsers/h1/parser.go)

## Common Pitfalls

1. **QuickCheck with Regex**: Use only string operations. Regex defeats the performance optimization.
2. **ICAO False Positives**: Always check `ICAOBlocklist`. Words like "WHEN", "WITH" match ICAO patterns.
3. **Missing Blank Import**: New parsers won't register without import in [parsers/parsers.go](parsers/parsers.go).
4. **Priority Conflicts**: Label-specific parsers (10-100), content-based (500+). PDC at 500 runs after all label parsers.
5. **H1 Transmission Artifacts**: Always normalize H1 messages with `NormaliseFPN()` before parsing.
6. **Thread Safety**: Registry dispatch is concurrent. Don't mutate shared state in `Parse()`.
