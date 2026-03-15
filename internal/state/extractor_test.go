package state

import (
	"database/sql"
	"path/filepath"
	"testing"

	"acars_parser/internal/acars"
	"acars_parser/internal/parsers/sb01"
	"acars_parser/internal/registry"
	_ "modernc.org/sqlite"
)

func TestExtractAndUpdateFromSB01(t *testing.T) {
	tracker, err := NewTracker(":memory:")
	if err != nil {
		t.Fatalf("NewTracker: %v", err)
	}
	defer func() { _ = tracker.Close() }()

	msg := &acars.Message{
		ID:        42,
		Timestamp: "2026-03-13T18:32:00Z",
		Tail:      "F-GZNG",
		Label:     "H1",
		Text:      "SB0122BA_F-GZNG LFPOFMEE195 42703 0184101832 31001-550356015010GMY012015",
	}
	result := (&sb01.Parser{}).Parse(msg)
	if result == nil {
		t.Fatal("SB01 parser returned nil")
	}

	ExtractAndUpdate(tracker, msg, []registry.Result{result})

	flight := tracker.GetFlight("F-GZNG")
	if flight == nil {
		t.Fatal("expected flight state for F-GZNG")
	}
	if flight.Registration != "F-GZNG" {
		t.Fatalf("Registration = %q, want %q", flight.Registration, "F-GZNG")
	}
	if flight.Origin != "LFPO" {
		t.Fatalf("Origin = %q, want %q", flight.Origin, "LFPO")
	}
	if flight.Destination != "FMEE" {
		t.Fatalf("Destination = %q, want %q", flight.Destination, "FMEE")
	}
	if flight.ReportTime != "18:32" {
		t.Fatalf("ReportTime = %q, want %q", flight.ReportTime, "18:32")
	}
	if flight.Altitude != 31001 {
		t.Fatalf("Altitude = %d, want %d", flight.Altitude, 31001)
	}
	if flight.Latitude < 42.7029 || flight.Latitude > 42.7031 {
		t.Fatalf("Latitude = %.6f, want 42.703", flight.Latitude)
	}
	if flight.Longitude < 18.4099 || flight.Longitude > 18.4101 {
		t.Fatalf("Longitude = %.6f, want 18.410", flight.Longitude)
	}
}

func TestNewTrackerMigratesFlightStateReportTimeColumn(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "state.db")
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE flight_state (
			key           TEXT PRIMARY KEY,
			icao_hex      TEXT,
			registration  TEXT,
			flight_number TEXT,
			origin        TEXT,
			destination   TEXT,
			latitude      REAL,
			longitude     REAL,
			altitude      INTEGER,
			ground_speed  INTEGER,
			track         INTEGER,
			waypoints     TEXT,
			first_seen    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_seen     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			msg_count     INTEGER NOT NULL DEFAULT 1
		)
	`)
	if err != nil {
		_ = db.Close()
		t.Fatalf("create legacy flight_state: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close legacy db: %v", err)
	}

	tracker, err := NewTracker(dbPath)
	if err != nil {
		t.Fatalf("NewTracker migration: %v", err)
	}
	defer func() { _ = tracker.Close() }()

	rows, err := tracker.db.Query("PRAGMA table_info(flight_state)")
	if err != nil {
		t.Fatalf("PRAGMA table_info: %v", err)
	}
	defer func() { _ = rows.Close() }()

	hasReportTime := false
	for rows.Next() {
		var cid int
		var name string
		var dataType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			t.Fatalf("scan pragma row: %v", err)
		}
		if name == "report_time" {
			hasReportTime = true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("pragma rows: %v", err)
	}
	if !hasReportTime {
		t.Fatal("expected report_time column to be added to flight_state")
	}
}
