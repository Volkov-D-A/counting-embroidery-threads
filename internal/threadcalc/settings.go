package threadcalc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"counting-embroidery-threads/internal/dmccode"
)

const transformationSettingsFilename = "transform_settings.json"

// DefaultTransformationSettings returns the built-in starter rules.
func DefaultTransformationSettings() *TransformationSettings {
	return &TransformationSettings{
		Rules: []DescriptionTransformRule{
			{
				Enabled:         true,
				MatchColumn:     "code",
				MatchMode:       "prefix",
				Description:     "DMC",
				StripCodePrefix: "DMC",
			},
			{
				Enabled:     true,
				MatchColumn: "description",
				MatchMode:   "equals",
				Description: "DMC:E",
				CodePrefix:  "E",
			},
			{
				Enabled:     true,
				MatchColumn: "description",
				MatchMode:   "equals",
				Description: "DMC_light_effects:E",
				CodePrefix:  "E",
			},
			{
				Enabled:     true,
				MatchColumn: "description",
				MatchMode:   "equals",
				Description: "DMC_light_effects:B",
				CodePrefix:  "B",
			},
			{
				Enabled:     true,
				MatchColumn: "description",
				MatchMode:   "prefix",
				Description: "DMCS:S",
				CodePrefix:  "S",
			},
			{
				Enabled:     true,
				MatchColumn: "description",
				MatchMode:   "equals",
				Description: "Etoile:C",
				CodePrefix:  "C",
			},
		},
	}
}

// TransformationSettingsPath returns the JSON file path near the executable.
func TransformationSettingsPath() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(executable), transformationSettingsFilename), nil
}

// LoadTransformationSettings reads the saved settings or returns defaults.
func LoadTransformationSettings() (*TransformationSettings, error) {
	path, err := TransformationSettingsPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return DefaultTransformationSettings(), nil
	}
	if err != nil {
		return nil, err
	}

	var settings TransformationSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, err
	}
	normalized := normalizeTransformationSettings(&settings)
	if len(normalized.Rules) == 0 {
		return DefaultTransformationSettings(), nil
	}
	return normalized, nil
}

// SaveTransformationSettings stores settings in the JSON file near the executable.
func SaveTransformationSettings(settings *TransformationSettings) (*TransformationSettings, error) {
	settings = normalizeTransformationSettings(settings)
	path, err := TransformationSettingsPath()
	if err != nil {
		return nil, err
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return nil, err
	}
	return settings, nil
}

func normalizeTransformationSettings(settings *TransformationSettings) *TransformationSettings {
	if settings == nil {
		return DefaultTransformationSettings()
	}

	normalized := &TransformationSettings{Rules: []DescriptionTransformRule{}}
	for _, rule := range settings.Rules {
		rule.MatchColumn = normalizeMatchColumn(rule.MatchColumn)
		rule.MatchMode = normalizeMatchMode(rule.MatchMode)
		rule.Description = strings.TrimSpace(rule.Description)
		rule.StripCodePrefix = dmccode.BaseNormalize(rule.StripCodePrefix)
		rule.CodePrefix = dmccode.BaseNormalize(rule.CodePrefix)
		rule.CodeSuffix = dmccode.BaseNormalize(rule.CodeSuffix)
		if rule.Description == "" || rule.StripCodePrefix == "" && rule.CodePrefix == "" && rule.CodeSuffix == "" {
			continue
		}
		normalized.Rules = append(normalized.Rules, rule)
	}
	return normalized
}

func normalizeMatchColumn(matchColumn string) string {
	switch strings.ToLower(strings.TrimSpace(matchColumn)) {
	case "code":
		return "code"
	default:
		return "description"
	}
}

func normalizeMatchMode(matchMode string) string {
	switch strings.ToLower(strings.TrimSpace(matchMode)) {
	case "contains", "prefix", "suffix":
		return strings.ToLower(strings.TrimSpace(matchMode))
	default:
		return "equals"
	}
}
