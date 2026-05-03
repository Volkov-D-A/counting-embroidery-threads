package threadcalc

// ImportResult is returned to the frontend after processing a report.
type ImportResult struct {
	Cancelled         bool           `json:"cancelled"`
	FilePath          string         `json:"filePath"`
	FileName          string         `json:"fileName"`
	Encoding          string         `json:"encoding"`
	RowsImported      int            `json:"rowsImported"`
	BeadRowsIgnored   int            `json:"beadRowsIgnored"`
	TotalMeters       float64        `json:"totalMeters"`
	TotalSkeins       int            `json:"totalSkeins"`
	SkeinLengthMeters float64        `json:"skeinLengthMeters"`
	Items             []ThreadResult `json:"items"`
	Warnings          []string       `json:"warnings"`
}

// CodeCorrection describes a user edit of an aggregated DMC code.
type CodeCorrection struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// ThreadResult is one aggregated DMC row in the final calculation.
type ThreadResult struct {
	Code         string   `json:"code"`
	ColorName    string   `json:"colorName"`
	ColorHex     string   `json:"colorHex"`
	PaletteFound bool     `json:"paletteFound"`
	Meters       float64  `json:"meters"`
	Skeins       int      `json:"skeins"`
	Notes        []string `json:"notes"`
}
