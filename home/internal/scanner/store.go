package scanner

import (
	"sync"

	"github.com/hhftechnology/vps-monitor/internal/models"
)

const maxResultsPerImage = 10

// ScanResultStore stores scan results in memory, keyed by host:imageRef.
type ScanResultStore struct {
	mu      sync.RWMutex
	results map[string][]models.ScanResult // key: "host:imageRef"
}

// NewScanResultStore creates a new in-memory scan result store.
func NewScanResultStore() *ScanResultStore {
	return &ScanResultStore{
		results: make(map[string][]models.ScanResult),
	}
}

func resultKey(host, imageRef string) string {
	return host + ":" + imageRef
}

// Add stores a scan result, keeping at most maxResultsPerImage per image.
func (s *ScanResultStore) Add(result models.ScanResult) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := resultKey(result.Host, result.ImageRef)
	// Prepend for newest-first ordering
	s.results[key] = append([]models.ScanResult{result}, s.results[key]...)

	if len(s.results[key]) > maxResultsPerImage {
		s.results[key] = s.results[key][:maxResultsPerImage]
	}
}

// GetResults returns all scan results for a specific image on a host.
func (s *ScanResultStore) GetResults(host, imageRef string) []models.ScanResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := resultKey(host, imageRef)
	results := s.results[key]
	if results == nil {
		return []models.ScanResult{}
	}
	out := make([]models.ScanResult, len(results))
	copy(out, results)
	return out
}

// GetLatest returns the most recent scan result for an image on a host.
func (s *ScanResultStore) GetLatest(host, imageRef string) *models.ScanResult {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := resultKey(host, imageRef)
	if len(s.results[key]) == 0 {
		return nil
	}
	result := s.results[key][0]
	return &result
}
