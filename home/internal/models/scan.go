package models

// ScannerType represents the type of vulnerability scanner
type ScannerType string

const (
	ScannerGrype ScannerType = "grype"
	ScannerTrivy ScannerType = "trivy"
)

// SeverityLevel represents the severity of a vulnerability
type SeverityLevel string

const (
	SeverityCritical   SeverityLevel = "Critical"
	SeverityHigh       SeverityLevel = "High"
	SeverityMedium     SeverityLevel = "Medium"
	SeverityLow        SeverityLevel = "Low"
	SeverityNegligible SeverityLevel = "Negligible"
	SeverityUnknown    SeverityLevel = "Unknown"
)

// Vulnerability represents a single vulnerability finding
type Vulnerability struct {
	ID               string        `json:"id"`
	Severity         SeverityLevel `json:"severity"`
	Package          string        `json:"package"`
	InstalledVersion string        `json:"installed_version"`
	FixedVersion     string        `json:"fixed_version,omitempty"`
	Description      string        `json:"description,omitempty"`
	DataSource       string        `json:"data_source,omitempty"`
}

// SeveritySummary summarizes vulnerability counts by severity
type SeveritySummary struct {
	Critical   int `json:"critical"`
	High       int `json:"high"`
	Medium     int `json:"medium"`
	Low        int `json:"low"`
	Negligible int `json:"negligible"`
	Unknown    int `json:"unknown"`
	Total      int `json:"total"`
}

// ScanResult holds the results of a vulnerability scan
type ScanResult struct {
	ID              string          `json:"id"`
	ImageRef        string          `json:"image_ref"`
	Host            string          `json:"host"`
	Scanner         ScannerType     `json:"scanner"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
	Summary         SeveritySummary `json:"summary"`
	StartedAt       int64           `json:"started_at"`
	CompletedAt     int64           `json:"completed_at"`
	DurationMs      int64           `json:"duration_ms"`
	Error           string          `json:"error,omitempty"`
}

// ScanJobStatus represents the status of a scan job
type ScanJobStatus string

const (
	ScanJobPending   ScanJobStatus = "pending"
	ScanJobPulling   ScanJobStatus = "pulling_scanner"
	ScanJobScanning  ScanJobStatus = "scanning"
	ScanJobComplete  ScanJobStatus = "complete"
	ScanJobFailed    ScanJobStatus = "failed"
	ScanJobCancelled ScanJobStatus = "cancelled"
)

// ScanJob represents an individual scan job
type ScanJob struct {
	ID        string        `json:"id"`
	ImageRef  string        `json:"image_ref"`
	Host      string        `json:"host"`
	Scanner   ScannerType   `json:"scanner"`
	Status    ScanJobStatus `json:"status"`
	Progress  string        `json:"progress,omitempty"`
	Result    *ScanResult   `json:"result,omitempty"`
	CreatedAt int64         `json:"created_at"`
	Error     string        `json:"error,omitempty"`
}

// BulkScanJob represents a bulk scan of multiple images
type BulkScanJob struct {
	ID          string        `json:"id"`
	Jobs        []*ScanJob    `json:"jobs"`
	TotalImages int           `json:"total_images"`
	Completed   int           `json:"completed"`
	Failed      int           `json:"failed"`
	Status      ScanJobStatus `json:"status"`
	CreatedAt   int64         `json:"created_at"`
}

// SBOMFormat represents the output format for SBOM
type SBOMFormat string

const (
	SBOMFormatSPDX      SBOMFormat = "spdx-json"
	SBOMFormatCycloneDX SBOMFormat = "cyclonedx-json"
)

// SBOMJob represents a SBOM generation job
type SBOMJob struct {
	ID        string        `json:"id"`
	ImageRef  string        `json:"image_ref"`
	Host      string        `json:"host"`
	Format    SBOMFormat    `json:"format"`
	Status    ScanJobStatus `json:"status"`
	FilePath  string        `json:"-"`
	CreatedAt int64         `json:"created_at"`
	Error     string        `json:"error,omitempty"`
}

// ScannerConfig holds scanner configuration
type ScannerConfig struct {
	GrypeImage     string             `json:"grypeImage"`
	TrivyImage     string             `json:"trivyImage"`
	SyftImage      string             `json:"syftImage"`
	DefaultScanner ScannerType        `json:"defaultScanner"`
	GrypeArgs      string             `json:"grypeArgs"`
	TrivyArgs      string             `json:"trivyArgs"`
	Notifications  NotificationConfig `json:"notifications"`
}

// NotificationConfig holds notification webhook configuration
type NotificationConfig struct {
	DiscordWebhookURL string        `json:"discordWebhookURL,omitempty"`
	SlackWebhookURL   string        `json:"slackWebhookURL,omitempty"`
	OnScanComplete    bool          `json:"onScanComplete"`
	OnBulkComplete    bool          `json:"onBulkComplete"`
	MinSeverity       SeverityLevel `json:"minSeverity,omitempty"`
}
