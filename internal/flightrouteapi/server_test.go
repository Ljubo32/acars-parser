package flightrouteapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestFlightRouteAPIExposesResolvedDBPath(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "flightroute.sqb")
	server, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer server.Close()

	req := httptest.NewRequest(http.MethodGet, "/api/flightroute/config", nil)
	resp := httptest.NewRecorder()
	server.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var payload map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if !strings.HasSuffix(payload["db_path"], filepath.Join("flightroute.sqb")) {
		t.Fatalf("db_path = %q, want suffix %q", payload["db_path"], filepath.Join("flightroute.sqb"))
	}
}

func TestFlightRouteAPIWritesAndReadsRoute(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "flightroute.sqb")
	server, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer server.Close()

	handler := server.Handler()
	body, err := json.Marshal(map[string]any{
		"flight":     "MAS020",
		"route":      "WMKK-LFPG",
		"updatetime": "2026-03-31",
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	writeReq := httptest.NewRequest(http.MethodPost, "/api/flightroute", bytes.NewReader(body))
	writeReq.Header.Set("Content-Type", "application/json")
	writeResp := httptest.NewRecorder()
	handler.ServeHTTP(writeResp, writeReq)
	if writeResp.Code != http.StatusOK {
		t.Fatalf("POST status = %d, want %d, body=%s", writeResp.Code, http.StatusOK, writeResp.Body.String())
	}

	lookupReq := httptest.NewRequest(http.MethodGet, "/api/flightroute?flight=MAS20", nil)
	lookupResp := httptest.NewRecorder()
	handler.ServeHTTP(lookupResp, lookupReq)
	if lookupResp.Code != http.StatusOK {
		t.Fatalf("GET status = %d, want %d, body=%s", lookupResp.Code, http.StatusOK, lookupResp.Body.String())
	}

	var record Record
	if err := json.Unmarshal(lookupResp.Body.Bytes(), &record); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if record.Flight != "MAS20" {
		t.Fatalf("Flight = %q, want %q", record.Flight, "MAS20")
	}
	if record.Route != "WMKK-LFPG" {
		t.Fatalf("Route = %q, want %q", record.Route, "WMKK-LFPG")
	}
	if record.UpdateTime != "2026-03-31" {
		t.Fatalf("UpdateTime = %q, want %q", record.UpdateTime, "2026-03-31")
	}
	if record.Origin != "WMKK" || record.Destination != "LFPG" {
		t.Fatalf("pair = %q-%q, want %q-%q", record.Origin, record.Destination, "WMKK", "LFPG")
	}
}

func TestFlightRouteAPIUpdatesSameFlightAndDate(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "flightroute.sqb")
	server, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer server.Close()

	handler := server.Handler()
	for _, route := range []string{"WMKK-LFPG", "WSSS-LFPG"} {
		body, err := json.Marshal(map[string]any{
			"flight":     "MAS20",
			"route":      route,
			"updatetime": "2026-03-31",
		})
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/api/flightroute", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("POST status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
		}
	}

	record, err := server.lookupLatest("MAS20")
	if err != nil {
		t.Fatalf("lookupLatest() error = %v", err)
	}
	if record == nil {
		t.Fatal("lookupLatest() = nil, want row")
	}
	if record.Route != "WSSS-LFPG" {
		t.Fatalf("Route = %q, want %q", record.Route, "WSSS-LFPG")
	}
}

func TestFlightRouteAPIReplacesUnixUpdateTimeForExistingFlight(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "flightroute.sqb")
	server, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer server.Close()

	if _, err := server.db.Exec(`INSERT INTO FlightRoute (flight, route, updatetime) VALUES (?, ?, ?)`, "MAS20", "WMKK-LFPG", 1712092800); err != nil {
		t.Fatalf("seed insert error = %v", err)
	}

	body, err := json.Marshal(map[string]any{
		"flight":     "MAS20",
		"route":      "WMKK-LFPG",
		"updatetime": "2026-03-31",
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/flightroute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	server.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("POST status = %d, want %d, body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}

	var updatedAt string
	if err := server.db.QueryRow(`SELECT CAST(updatetime AS TEXT) FROM FlightRoute WHERE UPPER(TRIM(flight)) = UPPER(TRIM(?)) LIMIT 1`, "MAS20").Scan(&updatedAt); err != nil {
		t.Fatalf("lookup updatetime error = %v", err)
	}
	if updatedAt != "2026-03-31" {
		t.Fatalf("updatetime = %q, want %q", updatedAt, "2026-03-31")
	}
}

func TestFlightRouteAPIRejectsInvalidRoute(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "flightroute.sqb")
	server, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer server.Close()

	body := []byte(`{"flight":"MAS20","route":"WMKKLFPG","updatetime":"2026-03-31"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/flightroute", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	server.Handler().ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body=%s", resp.Code, http.StatusBadRequest, resp.Body.String())
	}
}
