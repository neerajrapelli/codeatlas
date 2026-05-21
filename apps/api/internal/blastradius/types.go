package blastradius

// Result is the blast-radius API response.
type Result struct {
	Target      TargetInfo      `json:"target"`
	BlastRadius BlastSummary    `json:"blast_radius"`
	Files       []AffectedFile  `json:"files"`
	Warnings    []string        `json:"warnings"`
}

type TargetInfo struct {
	FilePath        string  `json:"file_path"`
	Symbol          string  `json:"symbol,omitempty"`
	Owner           string  `json:"owner,omitempty"`
	BusFactorScore  float64 `json:"bus_factor_score"`
}

type BlastSummary struct {
	DirectDependents      int      `json:"direct_dependents"`
	TransitiveDependents  int      `json:"transitive_dependents"`
	TotalFilesAffected    int      `json:"total_files_affected"`
	RiskScore             float64  `json:"risk_score"`
	TeamsAffected         []string `json:"teams_affected"`
}

type AffectedFile struct {
	FilePath     string `json:"file_path"`
	Depth        int    `json:"depth"`
	Relationship string `json:"relationship"`
	Owner        string `json:"owner,omitempty"`
	HasTests     bool   `json:"has_tests"`
	RiskLevel    string `json:"risk_level"`
}
