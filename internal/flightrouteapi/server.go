package flightrouteapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"acars_parser/internal/airlines"

	_ "modernc.org/sqlite"
)

var routePattern = regexp.MustCompile(`^([A-Z]{4})-([A-Z]{4})$`)

// Record represents one FlightRoute database row.
type Record struct {
	Flight      string `json:"flight"`
	Route       string `json:"route"`
	UpdateTime  string `json:"updatetime"`
	Origin      string `json:"origin,omitempty"`
	Destination string `json:"destination,omitempty"`
}

// Server exposes a small HTTP API for FlightRoute lookups and writes.
type Server struct {
	db     *sql.DB
	dbPath string
}

// Open opens or creates the local FlightRoute database and ensures the table exists.
func Open(dbPath string) (*Server, error) {
	if strings.TrimSpace(dbPath) == "" {
		return nil, errors.New("database path is required")
	}

	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		return nil, fmt.Errorf("resolve database path: %w", err)
	}

	db, err := sql.Open("sqlite", absPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	server := &Server{db: db, dbPath: absPath}
	if err := server.ensureSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return server, nil
}

// Close closes the underlying database handle.
func (s *Server) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Run starts the HTTP server.
func Run(dbPath string, port int) error {
	server, err := Open(dbPath)
	if err != nil {
		return err
	}
	defer server.Close()

	addr := fmt.Sprintf(":%d", port)
	log.Printf("FlightRoute API listening on http://127.0.0.1%s using %s", addr, server.dbPath)
	return http.ListenAndServe(addr, server.Handler())
}

// Handler returns the HTTP handler for the FlightRoute API.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/flightroute", s.handleFlightRoute)
	mux.HandleFunc("/api/flightroute/config", s.handleConfig)
	mux.HandleFunc("/healthz", s.handleHealth)
	return withCORS(mux)
}

func (s *Server) ensureSchema() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS FlightRoute (
			flight TEXT,
			route TEXT,
			updatetime TEXT
		);
		CREATE INDEX IF NOT EXISTS idx_FlightRoute_flight ON FlightRoute(flight);
		CREATE INDEX IF NOT EXISTS idx_FlightRoute_update ON FlightRoute(updatetime);
	`)
	if err != nil {
		return fmt.Errorf("ensure FlightRoute schema: %w", err)
	}
	return nil
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"db_path": s.dbPath,
	})
}

func (s *Server) handleFlightRoute(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleLookup(w, r)
	case http.MethodPost:
		s.handleUpsert(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleLookup(w http.ResponseWriter, r *http.Request) {
	flight := normaliseFlight(r.URL.Query().Get("flight"))
	if flight == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "flight is required"})
		return
	}

	record, err := s.lookupLatest(flight)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if record == nil {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "route not found", "flight": flight})
		return
	}

	writeJSON(w, http.StatusOK, record)
}

func (s *Server) handleUpsert(w http.ResponseWriter, r *http.Request) {
	var request Record
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid JSON body"})
		return
	}

	flight := normaliseFlight(request.Flight)
	if flight == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "flight is required"})
		return
	}

	route, origin, destination, err := normaliseRoute(request.Route)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	updateTime, err := normaliseUpdateDate(request.UpdateTime)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	record, err := s.upsert(flight, route, updateTime)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	record.Origin = origin
	record.Destination = destination

	writeJSON(w, http.StatusOK, record)
}

func (s *Server) lookupLatest(flight string) (*Record, error) {
	row := s.db.QueryRow(`
		SELECT flight, route, updatetime
		FROM FlightRoute
		WHERE UPPER(TRIM(flight)) = UPPER(TRIM(?))
		ORDER BY LENGTH(TRIM(updatetime)) DESC, TRIM(updatetime) DESC, rowid DESC
		LIMIT 1
	`, flight)

	var record Record
	if err := row.Scan(&record.Flight, &record.Route, &record.UpdateTime); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("lookup latest route: %w", err)
	}

	origin, destination := splitRoute(record.Route)
	record.Origin = origin
	record.Destination = destination
	return &record, nil
}

func (s *Server) upsert(flight, route, updateTime string) (*Record, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.Exec(`
		UPDATE FlightRoute
		SET flight = ?, route = ?, updatetime = ?
		WHERE UPPER(TRIM(flight)) = UPPER(TRIM(?))
	`, flight, route, updateTime, flight)
	if err != nil {
		return nil, fmt.Errorf("update route rows: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("read updated route row count: %w", err)
	}

	if rowsAffected == 0 {
		if _, err := tx.Exec(`INSERT INTO FlightRoute (flight, route, updatetime) VALUES (?, ?, ?)`, flight, route, updateTime); err != nil {
			return nil, fmt.Errorf("insert route row: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return s.lookupLatest(flight)
}

func normaliseFlight(value string) string {
	return airlines.TranslateFlight(strings.TrimSpace(value))
}

func normaliseRoute(value string) (string, string, string, error) {
	route := strings.ToUpper(strings.TrimSpace(value))
	match := routePattern.FindStringSubmatch(route)
	if len(match) != 3 {
		return "", "", "", errors.New("route must be in XXXX-XXXX format")
	}
	return route, match[1], match[2], nil
}

func normaliseUpdateDate(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", errors.New("updatetime is required")
	}
	parsed, err := time.Parse("2006-01-02", trimmed)
	if err != nil {
		return "", errors.New("updatetime must be in YYYY-MM-DD format")
	}
	return parsed.Format("2006-01-02"), nil
}

func splitRoute(value string) (string, string) {
	match := routePattern.FindStringSubmatch(strings.ToUpper(strings.TrimSpace(value)))
	if len(match) != 3 {
		return "", ""
	}
	return match[1], match[2]
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
