package threadcalc

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"counting-embroidery-threads/internal/dmc"
	"counting-embroidery-threads/internal/dmccode"

	"golang.org/x/text/encoding/charmap"
)

const DefaultSkeinLengthMeters = 8.0

// ImportFile reads, parses and calculates DMC usage from a third-party TXT report.
func ImportFile(path string, skeinLengthMeters float64) (*ImportResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	text, encoding, err := decodeReport(data)
	if err != nil {
		return nil, err
	}

	palette, err := dmc.LoadEffectivePalette()
	if err != nil {
		return nil, err
	}
	settings, err := LoadTransformationSettings()
	if err != nil {
		return nil, err
	}

	result := parseAndCalculateWithSettings(text, palette, skeinLengthMeters, settings)
	result.FilePath = path
	result.FileName = filepath.Base(path)
	result.Encoding = encoding
	return result, nil
}

// RecalculateWithCorrections applies user code edits to an aggregated result and
// recalculates palette matches and skein counts.
func RecalculateWithCorrections(source *ImportResult, corrections []CodeCorrection, skeinLengthMeters float64) (*ImportResult, error) {
	if source == nil {
		return nil, errors.New("нет результата для пересчета")
	}
	if skeinLengthMeters <= 0 {
		skeinLengthMeters = DefaultSkeinLengthMeters
	}

	palette, err := dmc.LoadEffectivePalette()
	if err != nil {
		return nil, err
	}

	correctionByCode := map[string]string{}
	for _, correction := range corrections {
		from := normalizeCode(correction.From)
		to := normalizeCode(correction.To)
		if from == "" || to == "" || from == to {
			continue
		}
		correctionByCode[from] = to
	}

	metersByCode := map[string]float64{}
	notesByCode := map[string]map[string]struct{}{}
	for _, item := range source.Items {
		from := normalizeCode(item.Code)
		if from == "" {
			continue
		}

		to := from
		if corrected, ok := correctionByCode[from]; ok {
			to = corrected
		}

		metersByCode[to] += item.Meters
		ensureNoteSet(notesByCode, to)
		for _, note := range item.Notes {
			if isRecalculatedNote(note) {
				continue
			}
			notesByCode[to][note] = struct{}{}
		}
		if to != from {
			notesByCode[to][fmt.Sprintf("исправлено: %s -> %s", from, to)] = struct{}{}
		} else if from != item.Code {
			notesByCode[to][fmt.Sprintf("%s -> %s", item.Code, from)] = struct{}{}
		}
	}

	result := buildResultFromMeters(metersByCode, notesByCode, palette, skeinLengthMeters)
	result.FilePath = source.FilePath
	result.FileName = source.FileName
	result.Encoding = source.Encoding
	result.RowsImported = source.RowsImported
	result.BeadRowsIgnored = source.BeadRowsIgnored
	return result, nil
}

func decodeReport(data []byte) (string, string, error) {
	if utf8.Valid(data) {
		return string(data), "UTF-8", nil
	}

	reader := charmap.Windows1251.NewDecoder().Reader(bytes.NewReader(data))
	decoded, err := ioReadAll(reader)
	if err != nil {
		return "", "", err
	}
	return string(decoded), "Windows-1251", nil
}

func ioReadAll(reader interface {
	Read([]byte) (int, error)
}) ([]byte, error) {
	var buffer bytes.Buffer
	_, err := buffer.ReadFrom(reader)
	return buffer.Bytes(), err
}

func parseAndCalculate(text string, palette dmc.Palette, skeinLengthMeters float64) *ImportResult {
	return parseAndCalculateWithSettings(text, palette, skeinLengthMeters, DefaultTransformationSettings())
}

func parseAndCalculateWithSettings(text string, palette dmc.Palette, skeinLengthMeters float64, settings *TransformationSettings) *ImportResult {
	if skeinLengthMeters <= 0 {
		skeinLengthMeters = DefaultSkeinLengthMeters
	}
	settings = normalizeTransformationSettings(settings)

	parseResult := &ImportResult{
		SkeinLengthMeters: skeinLengthMeters,
		Items:             []ThreadResult{},
		Warnings:          []string{},
	}

	lines := normalizeNewlines(text)
	metersByCode := map[string]float64{}
	notesByCode := map[string]map[string]struct{}{}
	section := ""

	for lineIndex, line := range strings.Split(lines, "\n") {
		lineNumber := lineIndex + 1
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		switch {
		case strings.HasPrefix(trimmed, "Thread lengths"), strings.HasPrefix(trimmed, "Длины нитей"):
			section = "thread-title"
			continue
		case strings.HasPrefix(trimmed, "Thread (DMC)"), strings.HasPrefix(trimmed, "Нить (DMC)"):
			section = "threads"
			continue
		case strings.HasPrefix(trimmed, "Bead Count"):
			section = "beads"
			continue
		case section == "beads":
			if strings.HasPrefix(trimmed, "Bead colour") || trimmed == "Quantity" {
				continue
			}
			if strings.Contains(line, "\t") && firstNumber.MatchString(trimmed) {
				parseResult.BeadRowsIgnored++
			}
			continue
		}

		if section != "threads" {
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) < 7 {
			parseResult.Warnings = append(parseResult.Warnings, fmt.Sprintf("Строка %d пропущена: ожидалось 7 колонок", lineNumber))
			continue
		}

		rawCode := strings.TrimSpace(fields[0])
		description := strings.TrimSpace(fields[1])
		totalMeters, err := strconv.ParseFloat(strings.TrimSpace(fields[6]), 64)
		if err != nil {
			parseResult.Warnings = append(parseResult.Warnings, fmt.Sprintf("Строка %d пропущена: не удалось прочитать метраж %q", lineNumber, fields[6]))
			continue
		}

		parts := []string{rawCode}
		share := totalMeters
		isBlend := strings.Contains(rawCode, "+") || strings.HasPrefix(description, "+")
		if isBlend {
			parts = blendParts(rawCode, description)
			share = totalMeters / 2
		}
		parseResult.RowsImported++

		for _, part := range parts {
			transformed, ruleNotes := applyTransformRules(part, description, settings)
			normalized := normalizeCode(transformed)
			if normalized == "" {
				continue
			}
			metersByCode[normalized] += share

			ensureNoteSet(notesByCode, normalized)
			cleanPart := strings.TrimSpace(transformed)
			for _, ruleNote := range ruleNotes {
				notesByCode[normalized][ruleNote] = struct{}{}
			}
			if shouldShowNormalizationNote(cleanPart, normalized) {
				notesByCode[normalized][fmt.Sprintf("%s -> %s", cleanPart, normalized)] = struct{}{}
			}
		}
	}

	result := buildResultFromMeters(metersByCode, notesByCode, palette, skeinLengthMeters)
	result.RowsImported = parseResult.RowsImported
	result.BeadRowsIgnored = parseResult.BeadRowsIgnored
	result.Warnings = append(result.Warnings, parseResult.Warnings...)
	sort.Strings(result.Warnings)
	return result
}

func buildResultFromMeters(metersByCode map[string]float64, notesByCode map[string]map[string]struct{}, palette dmc.Palette, skeinLengthMeters float64) *ImportResult {
	if skeinLengthMeters <= 0 {
		skeinLengthMeters = DefaultSkeinLengthMeters
	}

	result := &ImportResult{
		SkeinLengthMeters: skeinLengthMeters,
		Items:             []ThreadResult{},
		Warnings:          []string{},
	}

	for code, meters := range metersByCode {
		lookupCode := paletteLookupCode(code, palette)
		color, found := palette[lookupCode]
		notes := sortedNoteSet(notesByCode[code])
		if !found {
			color = dmc.Color{Hex: "#D1D5DB"}
			notes = append(notes, "цвет не найден в палитре")
			result.Warnings = append(result.Warnings, fmt.Sprintf("Для кода %s не найден цвет палитры", code))
		} else if lookupCode != code {
			notes = append(notes, fmt.Sprintf("цвет взят из %s", lookupCode))
		}

		skeins := 0
		if meters > 0 {
			skeins = int(math.Ceil(meters / skeinLengthMeters))
		}
		result.TotalMeters += meters
		result.TotalSkeins += skeins
		result.Items = append(result.Items, ThreadResult{
			Code:         code,
			ColorName:    color.Name,
			ColorHex:     color.Hex,
			PaletteFound: found,
			Meters:       roundMeters(meters),
			Skeins:       skeins,
			Notes:        notes,
		})
	}

	result.TotalMeters = roundMeters(result.TotalMeters)
	sort.Slice(result.Items, func(i, j int) bool {
		return lessDMCCode(result.Items[i].Code, result.Items[j].Code)
	})
	sort.Strings(result.Warnings)

	return result
}

func blendParts(rawCode, description string) []string {
	parts := strings.Split(rawCode, "+")
	if !strings.HasSuffix(rawCode, "+") {
		return parts
	}

	descriptionPart := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(description), "+"))
	if descriptionPart != "" {
		parts = append(parts, descriptionPart)
	}
	return parts
}

func applyTransformRules(code, description string, settings *TransformationSettings) (string, []string) {
	cleanCode := strings.TrimSpace(code)
	if cleanCode == "" {
		return cleanCode, nil
	}

	transformed := dmccodeBase(cleanCode)
	notes := []string{}
	for _, rule := range settings.Rules {
		matchValue := description
		if rule.MatchColumn == "code" {
			matchValue = transformed
		}
		if !rule.Enabled || !transformRuleMatches(matchValue, rule) {
			continue
		}

		before := transformed
		if rule.StripCodePrefix != "" {
			transformed = strings.TrimPrefix(transformed, rule.StripCodePrefix)
		}
		if rule.CodePrefix != "" && !strings.HasPrefix(transformed, rule.CodePrefix) {
			transformed = rule.CodePrefix + transformed
		}
		if rule.CodeSuffix != "" && !strings.HasSuffix(transformed, rule.CodeSuffix) {
			transformed += rule.CodeSuffix
		}

		if transformed != before {
			notes = append(notes, fmt.Sprintf("%s: %s -> %s", transformRuleLabel(rule), before, transformed))
		}
	}
	return transformed, notes
}

func dmccodeBase(code string) string {
	return strings.TrimSpace(dmccode.BaseNormalize(code))
}

func transformRuleMatches(value string, rule DescriptionTransformRule) bool {
	actual := strings.ToLower(strings.TrimSpace(value))
	expected := strings.ToLower(strings.TrimSpace(rule.Description))
	if expected == "" {
		return false
	}

	switch rule.MatchMode {
	case "contains":
		return strings.Contains(actual, expected)
	case "prefix":
		return strings.HasPrefix(actual, expected)
	case "suffix":
		return strings.HasSuffix(actual, expected)
	default:
		return actual == expected
	}
}

func transformRuleLabel(rule DescriptionTransformRule) string {
	if rule.MatchColumn == "code" {
		return "код: " + rule.Description
	}
	return rule.Description
}

func normalizeNewlines(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return text
}

func sortedNoteSet(noteSet map[string]struct{}) []string {
	notes := make([]string, 0, len(noteSet))
	for note := range noteSet {
		notes = append(notes, note)
	}
	sort.Strings(notes)
	return notes
}

func ensureNoteSet(notesByCode map[string]map[string]struct{}, code string) {
	if _, ok := notesByCode[code]; !ok {
		notesByCode[code] = map[string]struct{}{}
	}
}

func isRecalculatedNote(note string) bool {
	return note == "цвет не найден в палитре" ||
		note == "только в смесях" ||
		note == "самостоятельная нить + смесь" ||
		note == "смесь: добавлена половина метража строки" ||
		strings.HasPrefix(note, "цвет взят из ") ||
		strings.HasPrefix(note, "исправлено: ")
}

func roundMeters(value float64) float64 {
	return math.Round(value*100) / 100
}

func lessDMCCode(left, right string) bool {
	leftNumber, leftOK := firstNumberValue(left)
	rightNumber, rightOK := firstNumberValue(right)
	if leftOK && rightOK && leftNumber != rightNumber {
		return leftNumber < rightNumber
	}
	return left < right
}

func firstNumberValue(code string) (int, bool) {
	match := firstNumber.FindString(code)
	if match == "" {
		return 0, false
	}
	value, err := strconv.Atoi(match)
	return value, err == nil
}
