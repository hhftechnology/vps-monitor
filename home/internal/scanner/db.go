package scanner

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/hhftechnology/vps-monitor/internal/models"
	_ "modernc.org/sqlite"
)

// ScanDB manages the SQLite database for persisting scan results and settings.
type ScanDB struct {
	db *sql.DB
}

// HistoryQuery defines parameters for querying scan history.
type HistoryQuery struct {
	ImageRef    string `json:"image,omitempty"`
	Host        string `json:"host,omitempty"`
	MinSeverity string `json:"min_severity,omitempty"`
	StartDate   int64  `json:"start_date,omitempty"`
	EndDate     int64  `json:"end_date,omitempty"`
	Page        int    `json:"page,omitempty"`
	PageSize    int    `json:"page_size,omitempty"`
	SortBy      string `json:"sort_by,omitempty"`
	SortDir     string `json:"sort_dir,omitempty"`
}

// HistoryPage holds paginated scan history results.
type HistoryPage struct {
	Results    []models.ScanResult `json:"results"`
	Total      int                 `json:"total"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"page_size"`
	TotalPages int                 `json:"total_pages"`
}

// ScannedImage represents a distinct image+host pair with scan count.
type ScannedImage struct {
	ImageRef    string `json:"image_ref"`
	Host        string `json:"host"`
	ScanCount   int    `json:"scan_count"`
	LastScanned int64  `json:"last_scanned"`
}

// ImageScanState tracks the last known image ID for rescan gating and auto-scan.
type ImageScanState struct {
	Host       string `json:"host"`
	ImageRef   string `json:"image_ref"`
	ImageID    string `json:"image_id"`
	LastScanAt int64  `json:"last_scan_at"`
	LastScanID string `json:"last_scan_id"`
}

// SBOMHistoryQuery defines parameters for querying persisted SBOM history.
type SBOMHistoryQuery struct {
	ImageRef  string `json:"image,omitempty"`
	Host      string `json:"host,omitempty"`
	Format    string `json:"format,omitempty"`
	StartDate int64  `json:"start_date,omitempty"`
	EndDate   int64  `json:"end_date,omitempty"`
	Page      int    `json:"page,omitempty"`
	PageSize  int    `json:"page_size,omitempty"`
	SortBy    string `json:"sort_by,omitempty"`
	SortDir   string `json:"sort_dir,omitempty"`
}

// SBOMHistoryPage holds paginated SBOM history results.
type SBOMHistoryPage struct {
	Results    []models.SBOMResult `json:"results"`
	Total      int                 `json:"total"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"page_size"`
	TotalPages int                 `json:"total_pages"`
}

// SBOMedImage represents a distinct image+host pair with generated SBOM count.
type SBOMedImage struct {
	ImageRef   string `json:"image_ref"`
	Host       string `json:"host"`
	SBOMCount  int    `json:"sbom_count"`
	LastSBOMAt int64  `json:"last_sbom_at"`
}

// ImageSBOMState tracks the last known image ID for SBOM rescan gating.
type ImageSBOMState struct {
	Host       string `json:"host"`
	ImageRef   string `json:"image_ref"`
	Format     string `json:"format"`
	ImageID    string `json:"image_id"`
	LastSBOMAt int64  `json:"last_sbom_at"`
	LastSBOMID string `json:"last_sbom_id"`
}

const schema = `
CREATE TABLE IF NOT EXISTS scan_results (
    id              TEXT PRIMARY KEY,
    image_ref       TEXT NOT NULL,
    host            TEXT NOT NULL,
    scanner         TEXT NOT NULL,
    summary_critical   INTEGER NOT NULL DEFAULT 0,
    summary_high       INTEGER NOT NULL DEFAULT 0,
    summary_medium     INTEGER NOT NULL DEFAULT 0,
    summary_low        INTEGER NOT NULL DEFAULT 0,
    summary_negligible INTEGER NOT NULL DEFAULT 0,
    summary_unknown    INTEGER NOT NULL DEFAULT 0,
    summary_total      INTEGER NOT NULL DEFAULT 0,
    started_at      INTEGER NOT NULL,
    completed_at    INTEGER NOT NULL,
    duration_ms     INTEGER NOT NULL,
    error           TEXT NOT NULL DEFAULT '',
    created_at      INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sr_image_host ON scan_results(image_ref, host);
CREATE INDEX IF NOT EXISTS idx_sr_completed ON scan_results(completed_at DESC);
CREATE INDEX IF NOT EXISTS idx_sr_host ON scan_results(host);

CREATE TABLE IF NOT EXISTS vulnerabilities (
    id                TEXT NOT NULL,
    scan_result_id    TEXT NOT NULL REFERENCES scan_results(id) ON DELETE CASCADE,
    severity          TEXT NOT NULL,
    package           TEXT NOT NULL,
    installed_version TEXT NOT NULL,
    fixed_version     TEXT NOT NULL DEFAULT '',
    description       TEXT NOT NULL DEFAULT '',
    data_source       TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (scan_result_id, id, package)
);

CREATE INDEX IF NOT EXISTS idx_vuln_scan ON vulnerabilities(scan_result_id);

CREATE TABLE IF NOT EXISTS image_scan_state (
    host         TEXT NOT NULL,
    image_ref    TEXT NOT NULL,
    image_id     TEXT NOT NULL,
    last_scan_at INTEGER NOT NULL DEFAULT 0,
    last_scan_id TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (host, image_ref)
);

CREATE TABLE IF NOT EXISTS sbom_results (
    id              TEXT PRIMARY KEY,
    image_ref       TEXT NOT NULL,
    host            TEXT NOT NULL,
    format          TEXT NOT NULL,
    component_count INTEGER NOT NULL DEFAULT 0,
    file_size       INTEGER NOT NULL DEFAULT 0,
    file_path       TEXT NOT NULL,
    started_at      INTEGER NOT NULL,
    completed_at    INTEGER NOT NULL,
    duration_ms     INTEGER NOT NULL,
    error           TEXT NOT NULL DEFAULT '',
    created_at      INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sbomr_image_host ON sbom_results(image_ref, host);
CREATE INDEX IF NOT EXISTS idx_sbomr_completed ON sbom_results(completed_at DESC);
CREATE INDEX IF NOT EXISTS idx_sbomr_host ON sbom_results(host);

CREATE TABLE IF NOT EXISTS sbom_components (
    sbom_result_id TEXT NOT NULL REFERENCES sbom_results(id) ON DELETE CASCADE,
    name           TEXT NOT NULL,
    version        TEXT NOT NULL DEFAULT '',
    type           TEXT NOT NULL DEFAULT '',
    purl           TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_sbomc_result ON sbom_components(sbom_result_id);

CREATE TABLE IF NOT EXISTS image_sbom_state (
    host         TEXT NOT NULL,
    image_ref    TEXT NOT NULL,
    format       TEXT NOT NULL DEFAULT '',
    image_id     TEXT NOT NULL,
    last_sbom_at INTEGER NOT NULL DEFAULT 0,
    last_sbom_id TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (host, image_ref, format)
);

CREATE TABLE IF NOT EXISTS settings (
    key        TEXT PRIMARY KEY,
    value      TEXT NOT NULL,
    updated_at INTEGER NOT NULL
);
`

// NewScanDB opens (or creates) the SQLite database and runs migrations.
func NewScanDB(dbPath string) (*ScanDB, error) {
	// Ensure the parent directory exists
	if dir := filepath.Dir(dbPath); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory %q: %w", dir, err)
		}
	}

	db, err := sql.Open("sqlite", "file:"+dbPath+"?_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("failed to open scan database: %w", err)
	}

	// Set PRAGMAs for performance and correctness
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to set pragma %q: %w", p, err)
		}
	}

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	scanDB := &ScanDB{db: db}
	if err := scanDB.migrateImageSBOMStateTable(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate image_sbom_state table: %w", err)
	}

	return scanDB, nil
}

// Close closes the database connection.
func (s *ScanDB) Close() error {
	return s.db.Close()
}

func (s *ScanDB) migrateImageSBOMStateTable() error {
	rows, err := s.db.Query(`PRAGMA table_info(image_sbom_state)`)
	if err != nil {
		return err
	}
	defer rows.Close()

	hasFormat := false
	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal sql.NullString
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &primaryKey); err != nil {
			return err
		}
		if name == "format" {
			hasFormat = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if hasFormat {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`CREATE TABLE image_sbom_state_new (
		host         TEXT NOT NULL,
		image_ref    TEXT NOT NULL,
		format       TEXT NOT NULL DEFAULT '',
		image_id     TEXT NOT NULL,
		last_sbom_at INTEGER NOT NULL DEFAULT 0,
		last_sbom_id TEXT NOT NULL DEFAULT '',
		PRIMARY KEY (host, image_ref, format)
	)`); err != nil {
		return fmt.Errorf("create image_sbom_state_new: %w", err)
	}

	if _, err := tx.Exec(`INSERT INTO image_sbom_state_new (host, image_ref, format, image_id, last_sbom_at, last_sbom_id)
		SELECT host, image_ref, '', image_id, last_sbom_at, last_sbom_id
		FROM image_sbom_state`); err != nil {
		return fmt.Errorf("copy image_sbom_state rows: %w", err)
	}

	if _, err := tx.Exec(`DROP TABLE image_sbom_state`); err != nil {
		return fmt.Errorf("drop legacy image_sbom_state: %w", err)
	}
	if _, err := tx.Exec(`ALTER TABLE image_sbom_state_new RENAME TO image_sbom_state`); err != nil {
		return fmt.Errorf("rename image_sbom_state_new: %w", err)
	}

	return tx.Commit()
}

// InsertResult inserts a scan result and its vulnerabilities in a single transaction.
func (s *ScanDB) InsertResult(result models.ScanResult) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`INSERT INTO scan_results
		(id, image_ref, host, scanner, summary_critical, summary_high, summary_medium,
		 summary_low, summary_negligible, summary_unknown, summary_total,
		 started_at, completed_at, duration_ms, error, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		result.ID, result.ImageRef, result.Host, string(result.Scanner),
		result.Summary.Critical, result.Summary.High, result.Summary.Medium,
		result.Summary.Low, result.Summary.Negligible, result.Summary.Unknown, result.Summary.Total,
		result.StartedAt, result.CompletedAt, result.DurationMs, result.Error,
		time.Now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("insert scan_results: %w", err)
	}

	if len(result.Vulnerabilities) > 0 {
		stmt, err := tx.Prepare(`INSERT OR IGNORE INTO vulnerabilities
			(id, scan_result_id, severity, package, installed_version, fixed_version, description, data_source)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
		if err != nil {
			return fmt.Errorf("prepare vuln insert: %w", err)
		}
		defer stmt.Close()

		for _, v := range result.Vulnerabilities {
			_, err = stmt.Exec(v.ID, result.ID, string(v.Severity), v.Package,
				v.InstalledVersion, v.FixedVersion, v.Description, v.DataSource)
			if err != nil {
				return fmt.Errorf("insert vulnerability %s: %w", v.ID, err)
			}
		}
	}

	// Update image_scan_state
	_, err = tx.Exec(`INSERT INTO image_scan_state (host, image_ref, image_id, last_scan_at, last_scan_id)
		VALUES (?, ?, '', ?, ?)
		ON CONFLICT(host, image_ref) DO UPDATE SET
			last_scan_at = excluded.last_scan_at,
			last_scan_id = excluded.last_scan_id`,
		result.Host, result.ImageRef, result.CompletedAt, result.ID,
	)
	if err != nil {
		return fmt.Errorf("upsert image_scan_state: %w", err)
	}

	return tx.Commit()
}

// GetResults returns all scan results for a specific image on a host (backward compat).
func (s *ScanDB) GetResults(host, imageRef string) ([]models.ScanResult, error) {
	rows, err := s.db.Query(`SELECT id, image_ref, host, scanner,
		summary_critical, summary_high, summary_medium, summary_low,
		summary_negligible, summary_unknown, summary_total,
		started_at, completed_at, duration_ms, error
		FROM scan_results WHERE host = ? AND image_ref = ?
		ORDER BY completed_at DESC`, host, imageRef)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []models.ScanResult
	for rows.Next() {
		r, err := scanResultRow(rows)
		if err != nil {
			return nil, err
		}
		// Load vulnerabilities for backward compat
		r.Vulnerabilities, err = s.loadVulnerabilities(r.ID)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	if results == nil {
		results = []models.ScanResult{}
	}
	return results, rows.Err()
}

// GetLatest returns the most recent scan result for an image on a host (backward compat).
func (s *ScanDB) GetLatest(host, imageRef string) (*models.ScanResult, error) {
	row := s.db.QueryRow(`SELECT id, image_ref, host, scanner,
		summary_critical, summary_high, summary_medium, summary_low,
		summary_negligible, summary_unknown, summary_total,
		started_at, completed_at, duration_ms, error
		FROM scan_results WHERE host = ? AND image_ref = ?
		ORDER BY completed_at DESC LIMIT 1`, host, imageRef)

	r, err := scanResultFromRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	r.Vulnerabilities, err = s.loadVulnerabilities(r.ID)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// GetResultByID returns a full scan result with vulnerabilities.
func (s *ScanDB) GetResultByID(id string) (*models.ScanResult, error) {
	row := s.db.QueryRow(`SELECT id, image_ref, host, scanner,
		summary_critical, summary_high, summary_medium, summary_low,
		summary_negligible, summary_unknown, summary_total,
		started_at, completed_at, duration_ms, error
		FROM scan_results WHERE id = ?`, id)

	r, err := scanResultFromRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	r.Vulnerabilities, err = s.loadVulnerabilities(r.ID)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// DeleteScanResult removes a scan result and its vulnerabilities by ID.
func (s *ScanDB) DeleteScanResult(id string) error {
	_, err := s.db.Exec(`DELETE FROM scan_results WHERE id = ?`, id)
	return err
}

// InsertSBOMResult inserts a persisted SBOM result, its components, and image state.
func (s *ScanDB) InsertSBOMResult(result models.SBOMResult, imageID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`INSERT INTO sbom_results
		(id, image_ref, host, format, component_count, file_size, file_path,
		 started_at, completed_at, duration_ms, error, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		result.ID, result.ImageRef, result.Host, string(result.Format), result.ComponentCount, result.FileSize,
		result.FilePath, result.StartedAt, result.CompletedAt, result.DurationMs, result.Error, time.Now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("insert sbom_results: %w", err)
	}

	if len(result.Components) > 0 {
		stmt, err := tx.Prepare(`INSERT INTO sbom_components
			(sbom_result_id, name, version, type, purl)
			VALUES (?, ?, ?, ?, ?)`)
		if err != nil {
			return fmt.Errorf("prepare sbom component insert: %w", err)
		}
		defer stmt.Close()

		for _, component := range result.Components {
			_, err = stmt.Exec(result.ID, component.Name, component.Version, component.Type, component.PURL)
			if err != nil {
				return fmt.Errorf("insert sbom component %q: %w", component.Name, err)
			}
		}
	}

	_, err = tx.Exec(`INSERT INTO image_sbom_state (host, image_ref, format, image_id, last_sbom_at, last_sbom_id)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(host, image_ref, format) DO UPDATE SET
			image_id = excluded.image_id,
			last_sbom_at = excluded.last_sbom_at,
			last_sbom_id = excluded.last_sbom_id`,
		result.Host, result.ImageRef, result.Format, imageID, result.CompletedAt, result.ID,
	)
	if err != nil {
		return fmt.Errorf("upsert image_sbom_state: %w", err)
	}

	return tx.Commit()
}

// GetSBOMResultByID returns a full SBOM result with normalized components.
func (s *ScanDB) GetSBOMResultByID(id string) (*models.SBOMResult, error) {
	row := s.db.QueryRow(`SELECT id, image_ref, host, format, component_count, file_size, file_path,
		started_at, completed_at, duration_ms, error
		FROM sbom_results WHERE id = ?`, id)

	result, err := sbomResultFromRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	result.Components, err = s.loadSBOMComponents(result.ID)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// GetLatestSBOMResult returns the most recent persisted SBOM for an image on a host.
func (s *ScanDB) GetLatestSBOMResult(host, imageRef string) (*models.SBOMResult, error) {
	row := s.db.QueryRow(`SELECT id, image_ref, host, format, component_count, file_size, file_path,
		started_at, completed_at, duration_ms, error
		FROM sbom_results WHERE host = ? AND image_ref = ?
		ORDER BY completed_at DESC LIMIT 1`, host, imageRef)

	result, err := sbomResultFromRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	result.Components, err = s.loadSBOMComponents(result.ID)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// DeleteSBOMResult removes a persisted SBOM result and cascaded components by ID.
func (s *ScanDB) DeleteSBOMResult(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM sbom_results WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete sbom_results: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM image_sbom_state WHERE last_sbom_id = ?`, id); err != nil {
		return fmt.Errorf("delete image_sbom_state: %w", err)
	}

	return tx.Commit()
}

// GetPreviousResult returns the scan result immediately before the given scan ID
// for the same image and host.
func (s *ScanDB) GetPreviousResult(host, imageRef, beforeID string) (*models.ScanResult, error) {
	// Get the completed_at of the reference scan
	var refCompletedAt int64
	err := s.db.QueryRow(`SELECT completed_at FROM scan_results WHERE id = ?`, beforeID).Scan(&refCompletedAt)
	if err != nil {
		return nil, err
	}

	row := s.db.QueryRow(`SELECT id, image_ref, host, scanner,
		summary_critical, summary_high, summary_medium, summary_low,
		summary_negligible, summary_unknown, summary_total,
		started_at, completed_at, duration_ms, error
		FROM scan_results
		WHERE host = ? AND image_ref = ? AND completed_at < ? AND id != ?
		ORDER BY completed_at DESC LIMIT 1`,
		host, imageRef, refCompletedAt, beforeID)

	r, err := scanResultFromRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	r.Vulnerabilities, err = s.loadVulnerabilities(r.ID)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

// QueryHistory returns paginated scan history with optional filters.
// Vulnerabilities are NOT loaded for performance (use GetResultByID for detail).
func (s *ScanDB) QueryHistory(params HistoryQuery) (*HistoryPage, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}

	// Map user input to safe column names (prevents SQL injection)
	sortColumnMap := map[string]string{
		"completed_at":     "completed_at",
		"summary_total":    "summary_total",
		"summary_critical": "summary_critical",
	}
	sortColumn := sortColumnMap[params.SortBy]
	if sortColumn == "" {
		sortColumn = "completed_at"
	}
	sortDir := "DESC"
	if params.SortDir == "asc" {
		sortDir = "ASC"
	}

	where, args := buildHistoryWhere(params)

	// Count total
	var total int
	countQuery := "SELECT COUNT(*) FROM scan_results" + where
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count query: %w", err)
	}

	totalPages := (total + params.PageSize - 1) / params.PageSize
	offset := (params.Page - 1) * params.PageSize

	query := "SELECT id, image_ref, host, scanner," +
		" summary_critical, summary_high, summary_medium, summary_low," +
		" summary_negligible, summary_unknown, summary_total," +
		" started_at, completed_at, duration_ms, error" +
		" FROM scan_results" + where +
		" ORDER BY " + sortColumn + " " + sortDir +
		" LIMIT ? OFFSET ?"

	args = append(args, params.PageSize, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("history query: %w", err)
	}
	defer rows.Close()

	results := make([]models.ScanResult, 0)
	for rows.Next() {
		r, err := scanResultRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &HistoryPage{
		Results:    results,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}

// QuerySBOMHistory returns paginated SBOM history without loading component rows.
func (s *ScanDB) QuerySBOMHistory(params SBOMHistoryQuery) (*SBOMHistoryPage, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 || params.PageSize > 100 {
		params.PageSize = 20
	}

	sortColumnMap := map[string]string{
		"completed_at":    "completed_at",
		"component_count": "component_count",
	}
	sortColumn := sortColumnMap[params.SortBy]
	if sortColumn == "" {
		sortColumn = "completed_at"
	}
	sortDir := "DESC"
	if params.SortDir == "asc" {
		sortDir = "ASC"
	}

	where, args := buildSBOMHistoryWhere(params)

	var total int
	countQuery := "SELECT COUNT(*) FROM sbom_results" + where
	if err := s.db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count sbom history: %w", err)
	}

	totalPages := (total + params.PageSize - 1) / params.PageSize
	offset := (params.Page - 1) * params.PageSize

	query := "SELECT id, image_ref, host, format, component_count, file_size, file_path," +
		" started_at, completed_at, duration_ms, error" +
		" FROM sbom_results" + where +
		" ORDER BY " + sortColumn + " " + sortDir +
		" LIMIT ? OFFSET ?"

	args = append(args, params.PageSize, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("sbom history query: %w", err)
	}
	defer rows.Close()

	results := make([]models.SBOMResult, 0)
	for rows.Next() {
		result, err := sbomResultRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &SBOMHistoryPage{
		Results:    results,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, nil
}

// ListScannedImages returns distinct image+host pairs with scan counts.
func (s *ScanDB) ListScannedImages() ([]ScannedImage, error) {
	rows, err := s.db.Query(`SELECT image_ref, host, COUNT(*) as scan_count, MAX(completed_at) as last_scanned
		FROM scan_results GROUP BY image_ref, host ORDER BY last_scanned DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []ScannedImage
	for rows.Next() {
		var img ScannedImage
		if err := rows.Scan(&img.ImageRef, &img.Host, &img.ScanCount, &img.LastScanned); err != nil {
			return nil, err
		}
		images = append(images, img)
	}
	if images == nil {
		images = []ScannedImage{}
	}
	return images, rows.Err()
}

// ListSBOMedImages returns distinct image+host pairs with SBOM counts.
func (s *ScanDB) ListSBOMedImages() ([]SBOMedImage, error) {
	rows, err := s.db.Query(`SELECT image_ref, host, COUNT(*) as sbom_count, MAX(completed_at) as last_sbom_at
		FROM sbom_results GROUP BY image_ref, host ORDER BY last_sbom_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []SBOMedImage
	for rows.Next() {
		var img SBOMedImage
		if err := rows.Scan(&img.ImageRef, &img.Host, &img.SBOMCount, &img.LastSBOMAt); err != nil {
			return nil, err
		}
		images = append(images, img)
	}
	if images == nil {
		images = []SBOMedImage{}
	}
	return images, rows.Err()
}

// GetImageScanState returns the scan state for a specific image on a host.
func (s *ScanDB) GetImageScanState(host, imageRef string) (*ImageScanState, error) {
	var state ImageScanState
	err := s.db.QueryRow(`SELECT host, image_ref, image_id, last_scan_at, last_scan_id
		FROM image_scan_state WHERE host = ? AND image_ref = ?`, host, imageRef).
		Scan(&state.Host, &state.ImageRef, &state.ImageID, &state.LastScanAt, &state.LastScanID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &state, nil
}

// UpsertImageScanState updates the image scan state for auto-scan and rescan gating.
func (s *ScanDB) UpsertImageScanState(host, imageRef, imageID string, scannedAt int64, scanID string) error {
	_, err := s.db.Exec(`INSERT INTO image_scan_state (host, image_ref, image_id, last_scan_at, last_scan_id)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(host, image_ref) DO UPDATE SET
			image_id = excluded.image_id,
			last_scan_at = excluded.last_scan_at,
			last_scan_id = excluded.last_scan_id`,
		host, imageRef, imageID, scannedAt, scanID)
	return err
}

// GetImageSBOMState returns the latest persisted SBOM state for an image on a host.
func (s *ScanDB) GetImageSBOMState(host, imageRef, format string) (*ImageSBOMState, error) {
	var state ImageSBOMState
	err := s.db.QueryRow(`SELECT host, image_ref, format, image_id, last_sbom_at, last_sbom_id
		FROM image_sbom_state WHERE host = ? AND image_ref = ? AND format = ?`, host, imageRef, format).
		Scan(&state.Host, &state.ImageRef, &state.Format, &state.ImageID, &state.LastSBOMAt, &state.LastSBOMID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &state, nil
}

// UpsertImageSBOMState updates the SBOM image state for rescan gating.
func (s *ScanDB) UpsertImageSBOMState(host, imageRef, format, imageID string, generatedAt int64, sbomID string) error {
	_, err := s.db.Exec(`INSERT INTO image_sbom_state (host, image_ref, format, image_id, last_sbom_at, last_sbom_id)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(host, image_ref, format) DO UPDATE SET
			image_id = excluded.image_id,
			last_sbom_at = excluded.last_sbom_at,
			last_sbom_id = excluded.last_sbom_id`,
		host, imageRef, format, imageID, generatedAt, sbomID)
	return err
}

// CanRescan checks if an image has changed since the last scan (for rescan gating).
// Returns true if a rescan is allowed (image changed or never scanned).
func (s *ScanDB) CanRescan(host, imageRef, currentImageID string) (bool, error) {
	state, err := s.GetImageScanState(host, imageRef)
	if err != nil {
		return false, err
	}
	if state == nil {
		return true, nil // never scanned
	}
	if state.ImageID == "" {
		return true, nil // no image ID recorded
	}
	return state.ImageID != currentImageID, nil
}

// CanRegenerateSBOM checks if an image has changed since the last persisted SBOM.
func (s *ScanDB) CanRegenerateSBOM(host, imageRef, format, currentImageID string) (bool, error) {
	state, err := s.GetImageSBOMState(host, imageRef, format)
	if err != nil {
		return false, err
	}
	if state == nil || state.ImageID == "" {
		return true, nil
	}
	return state.ImageID != currentImageID, nil
}

// --- Settings ---

// GetSetting returns a setting value by key.
func (s *ScanDB) GetSetting(key string) (string, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// SetSetting sets a setting value.
func (s *ScanDB) SetSetting(key, value string) error {
	_, err := s.db.Exec(`INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, value, time.Now().Unix())
	return err
}

// GetAllSettings returns all settings as a map.
func (s *ScanDB) GetAllSettings() (map[string]string, error) {
	rows, err := s.db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		settings[key] = value
	}
	return settings, rows.Err()
}

// SaveScannerSettings saves scanner configuration to the settings table.
func (s *ScanDB) SaveScannerSettings(cfg *models.ScannerConfig) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().Unix()
	stmt, err := tx.Prepare(`INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	pairs := []struct{ key, val string }{
		{"grype_image", cfg.GrypeImage},
		{"trivy_image", cfg.TrivyImage},
		{"syft_image", cfg.SyftImage},
		{"default_scanner", string(cfg.DefaultScanner)},
		{"grype_args", cfg.GrypeArgs},
		{"trivy_args", cfg.TrivyArgs},
		{"discord_webhook_url", cfg.Notifications.DiscordWebhookURL},
		{"slack_webhook_url", cfg.Notifications.SlackWebhookURL},
		{"notify_on_complete", boolToStr(cfg.Notifications.OnScanComplete)},
		{"notify_on_bulk", boolToStr(cfg.Notifications.OnBulkComplete)},
		{"notify_on_new_cves", boolToStr(cfg.Notifications.OnNewCVEs)},
		{"notify_min_severity", string(cfg.Notifications.MinSeverity)},
		{"auto_scan_enabled", boolToStr(cfg.AutoScan.Enabled)},
		{"auto_scan_poll_interval", fmt.Sprintf("%d", cfg.AutoScan.PollInterval)},
		{"force_rescan_enabled", boolToStr(cfg.ForceRescan)},
		{"scan_timeout_minutes", fmt.Sprintf("%d", cfg.ScanTimeoutMinutes)},
		{"bulk_timeout_minutes", fmt.Sprintf("%d", cfg.BulkTimeoutMinutes)},
		{"scanner_memory_mb", fmt.Sprintf("%d", cfg.ScannerMemoryMB)},
		{"scanner_pids_limit", fmt.Sprintf("%d", cfg.ScannerPidsLimit)},
	}

	for _, p := range pairs {
		if _, err := stmt.Exec(p.key, p.val, now); err != nil {
			return fmt.Errorf("save setting %s: %w", p.key, err)
		}
	}

	return tx.Commit()
}

// LoadScannerSettings loads scanner configuration from DB, using envCfg as override.
// Priority: env vars (if set) > DB values > defaults.
func (s *ScanDB) LoadScannerSettings(envCfg *models.ScannerConfig) *models.ScannerConfig {
	dbSettings, err := s.GetAllSettings()
	if err != nil {
		log.Printf("Warning: failed to load settings from DB: %v", err)
		return envCfg
	}

	cfg := &models.ScannerConfig{
		GrypeImage:     getSettingWithDefault(dbSettings, "grype_image", "anchore/grype:v0.110.0"),
		TrivyImage:     getSettingWithDefault(dbSettings, "trivy_image", "aquasec/trivy:0.69.3"),
		SyftImage:      getSettingWithDefault(dbSettings, "syft_image", "anchore/syft:v1.42.3"),
		DefaultScanner: models.ScannerType(getSettingWithDefault(dbSettings, "default_scanner", "grype")),
		GrypeArgs:      getSettingWithDefault(dbSettings, "grype_args", ""),
		TrivyArgs:      getSettingWithDefault(dbSettings, "trivy_args", ""),
		Notifications: models.NotificationConfig{
			DiscordWebhookURL: getSettingWithDefault(dbSettings, "discord_webhook_url", ""),
			SlackWebhookURL:   getSettingWithDefault(dbSettings, "slack_webhook_url", ""),
			OnScanComplete:    getSettingWithDefault(dbSettings, "notify_on_complete", "true") == "true",
			OnBulkComplete:    getSettingWithDefault(dbSettings, "notify_on_bulk", "true") == "true",
			OnNewCVEs:         getSettingWithDefault(dbSettings, "notify_on_new_cves", "true") == "true",
			MinSeverity:       models.SeverityLevel(getSettingWithDefault(dbSettings, "notify_min_severity", "High")),
		},
		AutoScan: models.AutoScanConfig{
			Enabled:      getSettingWithDefault(dbSettings, "auto_scan_enabled", "false") == "true",
			PollInterval: parseIntSetting(getSettingWithDefault(dbSettings, "auto_scan_poll_interval", "15"), 15),
		},
		ForceRescan:        getSettingWithDefault(dbSettings, "force_rescan_enabled", "false") == "true",
		ScanTimeoutMinutes: parseIntSetting(getSettingWithDefault(dbSettings, "scan_timeout_minutes", "20"), 20),
		BulkTimeoutMinutes: parseIntSetting(getSettingWithDefault(dbSettings, "bulk_timeout_minutes", "120"), 120),
		ScannerMemoryMB:    parseIntSetting(getSettingWithDefault(dbSettings, "scanner_memory_mb", "2048"), 2048),
		ScannerPidsLimit:   parseIntSetting(getSettingWithDefault(dbSettings, "scanner_pids_limit", "512"), 512),
	}

	// Apply env overrides (non-empty env values take precedence)
	if envCfg != nil {
		if envCfg.GrypeImage != "" {
			cfg.GrypeImage = envCfg.GrypeImage
		}
		if envCfg.TrivyImage != "" {
			cfg.TrivyImage = envCfg.TrivyImage
		}
		if envCfg.SyftImage != "" {
			cfg.SyftImage = envCfg.SyftImage
		}
		if envCfg.DefaultScanner != "" {
			cfg.DefaultScanner = envCfg.DefaultScanner
		}
		if envCfg.GrypeArgs != "" {
			cfg.GrypeArgs = envCfg.GrypeArgs
		}
		if envCfg.TrivyArgs != "" {
			cfg.TrivyArgs = envCfg.TrivyArgs
		}
		if envCfg.Notifications.DiscordWebhookURL != "" {
			cfg.Notifications.DiscordWebhookURL = envCfg.Notifications.DiscordWebhookURL
		}
		if envCfg.Notifications.SlackWebhookURL != "" {
			cfg.Notifications.SlackWebhookURL = envCfg.Notifications.SlackWebhookURL
		}
		if envCfg.Notifications.MinSeverity != "" {
			cfg.Notifications.MinSeverity = envCfg.Notifications.MinSeverity
		}
		if envCfg.ScanTimeoutMinutes > 0 {
			cfg.ScanTimeoutMinutes = envCfg.ScanTimeoutMinutes
		}
		if envCfg.BulkTimeoutMinutes > 0 {
			cfg.BulkTimeoutMinutes = envCfg.BulkTimeoutMinutes
		}
		if envCfg.ScannerMemoryMB > 0 {
			cfg.ScannerMemoryMB = envCfg.ScannerMemoryMB
		}
		if envCfg.ScannerPidsLimit > 0 {
			cfg.ScannerPidsLimit = envCfg.ScannerPidsLimit
		}
	}

	return cfg
}

// MigrateFromFileConfig imports scanner settings from config.json into DB
// if the settings table is empty (first run after migration).
func (s *ScanDB) MigrateFromFileConfig(cfg *models.ScannerConfig) error {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM settings`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil // already have settings
	}

	log.Println("Migrating scanner settings from config file to database...")
	return s.SaveScannerSettings(cfg)
}

// --- Helpers ---

func scanResultRow(rows *sql.Rows) (models.ScanResult, error) {
	var r models.ScanResult
	var scannerStr, errStr string
	err := rows.Scan(&r.ID, &r.ImageRef, &r.Host, &scannerStr,
		&r.Summary.Critical, &r.Summary.High, &r.Summary.Medium,
		&r.Summary.Low, &r.Summary.Negligible, &r.Summary.Unknown, &r.Summary.Total,
		&r.StartedAt, &r.CompletedAt, &r.DurationMs, &errStr)
	if err != nil {
		return r, err
	}
	r.Scanner = models.ScannerType(scannerStr)
	r.Error = errStr
	return r, nil
}

func scanResultFromRow(row *sql.Row) (models.ScanResult, error) {
	var r models.ScanResult
	var scannerStr, errStr string
	err := row.Scan(&r.ID, &r.ImageRef, &r.Host, &scannerStr,
		&r.Summary.Critical, &r.Summary.High, &r.Summary.Medium,
		&r.Summary.Low, &r.Summary.Negligible, &r.Summary.Unknown, &r.Summary.Total,
		&r.StartedAt, &r.CompletedAt, &r.DurationMs, &errStr)
	if err != nil {
		return r, err
	}
	r.Scanner = models.ScannerType(scannerStr)
	r.Error = errStr
	return r, nil
}

func sbomResultRow(rows *sql.Rows) (models.SBOMResult, error) {
	var result models.SBOMResult
	var formatStr, errStr string
	err := rows.Scan(&result.ID, &result.ImageRef, &result.Host, &formatStr, &result.ComponentCount,
		&result.FileSize, &result.FilePath, &result.StartedAt, &result.CompletedAt, &result.DurationMs, &errStr)
	if err != nil {
		return result, err
	}
	result.Format = models.SBOMFormat(formatStr)
	result.Error = errStr
	return result, nil
}

func sbomResultFromRow(row *sql.Row) (models.SBOMResult, error) {
	var result models.SBOMResult
	var formatStr, errStr string
	err := row.Scan(&result.ID, &result.ImageRef, &result.Host, &formatStr, &result.ComponentCount,
		&result.FileSize, &result.FilePath, &result.StartedAt, &result.CompletedAt, &result.DurationMs, &errStr)
	if err != nil {
		return result, err
	}
	result.Format = models.SBOMFormat(formatStr)
	result.Error = errStr
	return result, nil
}

func (s *ScanDB) loadVulnerabilities(scanResultID string) ([]models.Vulnerability, error) {
	rows, err := s.db.Query(`SELECT id, severity, package, installed_version,
		fixed_version, description, data_source
		FROM vulnerabilities WHERE scan_result_id = ?`, scanResultID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vulns []models.Vulnerability
	for rows.Next() {
		var v models.Vulnerability
		var sevStr string
		if err := rows.Scan(&v.ID, &sevStr, &v.Package, &v.InstalledVersion,
			&v.FixedVersion, &v.Description, &v.DataSource); err != nil {
			return nil, err
		}
		v.Severity = models.SeverityLevel(sevStr)
		vulns = append(vulns, v)
	}
	if vulns == nil {
		vulns = []models.Vulnerability{}
	}
	return vulns, rows.Err()
}

func (s *ScanDB) loadSBOMComponents(sbomResultID string) ([]models.SBOMComponent, error) {
	rows, err := s.db.Query(`SELECT name, version, type, purl
		FROM sbom_components WHERE sbom_result_id = ?
		ORDER BY name ASC, version ASC, purl ASC`, sbomResultID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var components []models.SBOMComponent
	for rows.Next() {
		var component models.SBOMComponent
		if err := rows.Scan(&component.Name, &component.Version, &component.Type, &component.PURL); err != nil {
			return nil, err
		}
		components = append(components, component)
	}
	if components == nil {
		components = []models.SBOMComponent{}
	}
	return components, rows.Err()
}

func buildHistoryWhere(params HistoryQuery) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	if params.ImageRef != "" {
		conditions = append(conditions, "image_ref LIKE ?")
		args = append(args, "%"+params.ImageRef+"%")
	}
	if params.Host != "" {
		conditions = append(conditions, "host = ?")
		args = append(args, params.Host)
	}
	if params.StartDate > 0 {
		conditions = append(conditions, "completed_at >= ?")
		args = append(args, params.StartDate)
	}
	if params.EndDate > 0 {
		conditions = append(conditions, "completed_at <= ?")
		args = append(args, params.EndDate)
	}
	if params.MinSeverity != "" {
		switch models.SeverityLevel(params.MinSeverity) {
		case models.SeverityCritical:
			conditions = append(conditions, "summary_critical > 0")
		case models.SeverityHigh:
			conditions = append(conditions, "(summary_critical > 0 OR summary_high > 0)")
		case models.SeverityMedium:
			conditions = append(conditions, "(summary_critical > 0 OR summary_high > 0 OR summary_medium > 0)")
		case models.SeverityLow:
			conditions = append(conditions, "(summary_critical > 0 OR summary_high > 0 OR summary_medium > 0 OR summary_low > 0)")
		}
	}

	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func buildSBOMHistoryWhere(params SBOMHistoryQuery) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	if params.ImageRef != "" {
		conditions = append(conditions, "image_ref LIKE ?")
		args = append(args, "%"+params.ImageRef+"%")
	}
	if params.Host != "" {
		conditions = append(conditions, "host = ?")
		args = append(args, params.Host)
	}
	if params.Format != "" {
		conditions = append(conditions, "format = ?")
		args = append(args, params.Format)
	}
	if params.StartDate > 0 {
		conditions = append(conditions, "completed_at >= ?")
		args = append(args, params.StartDate)
	}
	if params.EndDate > 0 {
		conditions = append(conditions, "completed_at <= ?")
		args = append(args, params.EndDate)
	}

	if len(conditions) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conditions, " AND "), args
}

func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func getSettingWithDefault(settings map[string]string, key, defaultVal string) string {
	if v, ok := settings[key]; ok && v != "" {
		return v
	}
	return defaultVal
}

// parseIntSetting parses a settings value as a positive int. If the value is
// missing, malformed, or non-positive, def is returned. The per-setting default
// must be supplied by the caller — there is no universal fallback.
func parseIntSetting(s string, def int) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n <= 0 {
		return def
	}
	return n
}
