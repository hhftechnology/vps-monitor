package scanner

import (
	"log"

	"github.com/hhftechnology/vps-monitor/internal/models"
)

// ScanResultStore wraps ScanDB to persist scan results in SQLite.
type ScanResultStore struct {
	db *ScanDB
}

// NewScanResultStore creates a new SQLite-backed scan result store.
func NewScanResultStore(db *ScanDB) *ScanResultStore {
	return &ScanResultStore{db: db}
}

// Add stores a scan result in the database.
func (s *ScanResultStore) Add(result models.ScanResult) error {
	if err := s.db.InsertResult(result); err != nil {
		log.Printf("Failed to persist scan result: %v", err)
		return err
	}
	return nil
}

// AddSBOM stores a persisted SBOM result in the database.
func (s *ScanResultStore) AddSBOM(result models.SBOMResult, imageID string) error {
	if err := s.db.InsertSBOMResult(result, imageID); err != nil {
		log.Printf("Failed to persist SBOM result: %v", err)
		return err
	}
	return nil
}

// GetResults returns all scan results for a specific image on a host.
func (s *ScanResultStore) GetResults(host, imageRef string) []models.ScanResult {
	results, err := s.db.GetResults(host, imageRef)
	if err != nil {
		log.Printf("Failed to get scan results: %v", err)
		return []models.ScanResult{}
	}
	return results
}

// GetLatest returns the most recent scan result for an image on a host.
func (s *ScanResultStore) GetLatest(host, imageRef string) *models.ScanResult {
	result, err := s.db.GetLatest(host, imageRef)
	if err != nil {
		log.Printf("Failed to get latest scan result: %v", err)
		return nil
	}
	return result
}

// DB returns the underlying ScanDB for direct queries (history, settings, etc.).
func (s *ScanResultStore) DB() *ScanDB {
	return s.db
}
