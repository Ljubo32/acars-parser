package fst

import (
	"acars_parser/internal/acars"
	"fmt"
	"testing"
)

func TestFSTPrintAllFields(t *testing.T) {
	parser := &Parser{}
	msg := &acars.Message{Text: "FST01EGLLOMAAN418071E0214075390 245 145M 57C 3828713613851713682504540050"}
	res := parser.Parse(msg)
	result, ok := res.(*Result)
	if !ok {
		t.Fatalf("Expected *Result, got %T", res)
	}
	fmt.Printf("Sequence: %s\n", result.Sequence)
	fmt.Printf("Route: %s\n", result.Route)
	fmt.Printf("Latitude: %f\n", result.Latitude)
	fmt.Printf("Longitude: %f\n", result.Longitude)
	fmt.Printf("FlightLevel: %d\n", result.FlightLevel)
	fmt.Printf("Heading: %d\n", result.Heading)
	fmt.Printf("GroundSpeedKts: %d\n", result.GroundSpeedKts)
	fmt.Printf("GroundSpeedKmh: %d\n", result.GroundSpeedKmh)
	fmt.Printf("WindSpeedKts: %d\n", result.WindSpeedKts)
	fmt.Printf("WindSpeedKmh: %d\n", result.WindSpeedKmh)
	fmt.Printf("WindDirection: %d\n", result.WindDirection)
	fmt.Printf("Track: %d\n", result.Track)
	fmt.Printf("Temperature: %d\n", result.Temperature)
	fmt.Printf("RawData: %s\n", result.RawData)
}
